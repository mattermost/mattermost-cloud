# Deploying a Provisioner and Importing an External Cluster in New Infrastructure

This guide covers how to deploy a new mattermost-cloud provisioner in a new VPC/infrastructure within the same AWS account and import an existing EKS cluster into it.

## Prerequisites

- A running EKS cluster in the new VPC
- RDS Aurora PostgreSQL cluster for the provisioner's own metadata database
- AWS Secrets Manager access
- `cloud` CLI configured to point to the new provisioner

---

## 1. Prepare the Provisioner Database

Connect to the new RDS Aurora cluster as the master user and run:

```sql
CREATE USER provisioner WITH PASSWORD '<password>';
CREATE DATABASE cloud OWNER provisioner;
GRANT ALL PRIVILEGES ON DATABASE cloud TO provisioner;
```

The `DATABASE` secret connection string format:
```
postgres://provisioner:<password>@<rds-endpoint>:5432/cloud?sslmode=require
```

Create a Kubernetes secret with this value:
```bash
kubectl create secret generic mattermost-cloud-secret \
  --from-literal=DATABASE="postgres://provisioner:<password>@<rds-endpoint>:5432/cloud?sslmode=require"
```

---

## 2. Deploy the Provisioner

The provisioner runs as a Kubernetes deployment. The `init-database` init container runs `schema migrate` on startup to apply DB migrations before the server starts.

Key server flags relevant to a new infra deployment:
- `--state-store` — S3 bucket for state storage
- `--cluster-supervisor=false` / `--installation-supervisor=false` — disable supervisors not needed initially
- `--always-schedule-external-clusters=true` — required when using external clusters
- `--max-installations-rds-postgres-pgbouncer` — max installations per PgBouncer RDS cluster

---

## 3. Tag AWS Resources

mattermost-cloud discovers infrastructure via AWS tags. All of the following must be in place before importing a cluster.

### VPC Tags

Add these tags directly on the VPC:

| Tag | Value |
|-----|-------|
| `Available` | `true` |
| `CloudClusterID` | `none` |
| `CloudSecondaryClusterID` | `none` |

### Subnet Tags

Tag every **private** subnet:

| Tag | Value |
|-----|-------|
| `SubnetType` | `private` |

Tag every **public** subnet:

| Tag | Value |
|-----|-------|
| `SubnetType` | `public` |

### Security Group Tags

Create (or tag existing) security groups in the VPC with the `NodeType` tag. All three are required:

| Security Group | Tag | Value |
|---|---|---|
| EKS node/control plane SG | `NodeType` | `master` |
| EKS worker nodes SG | `NodeType` | `worker` |
| Calls/RTCD SG | `NodeType` | `calls` |

> If you don't use RTCD, create an empty security group in the VPC and tag it `NodeType=calls`.

---

## 4. Store the Kubeconfig in AWS Secrets Manager

The provisioner accesses the external cluster via a kubeconfig stored in Secrets Manager:

```bash
aws secretsmanager create-secret \
  --name <secret-name> \
  --secret-string "$(cat kubeconfig.yaml)"
```

The secret name is what you'll pass to `cloud cluster import`.

---

## 5. Authorize the Provisioner IAM Role on the EKS Cluster

The provisioner pod uses an IAM role (via IRSA). That role must be authorized on the target EKS cluster.

Using EKS access entries (recommended for newer clusters):
```bash
aws eks create-access-entry \
  --cluster-name <cluster-name> \
  --principal-arn arn:aws:iam::<account-id>:role/<provisioner-role-name> \
  --type STANDARD

aws eks associate-access-policy \
  --cluster-name <cluster-name> \
  --principal-arn arn:aws:iam::<account-id>:role/<provisioner-role-name> \
  --policy-arn arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy \
  --access-scope type=cluster
```

Or via `aws-auth` ConfigMap (older clusters):
```bash
kubectl edit configmap aws-auth -n kube-system
```
```yaml
mapRoles:
  - rolearn: arn:aws:iam::<account-id>:role/<provisioner-role-name>
    username: <provisioner-role-name>
    groups:
      - system:masters
```

---

## 6. Import the External Cluster

```bash
cloud cluster import \
  --secret-name <aws-secrets-manager-secret-name> \
  --vpc-id <vpc-id> \
  --allow-installations true \
  --annotation multi-tenant \
  --annotation <cluster-name>
```

Optionally set a human-readable name after importing:
```bash
cloud cluster update --cluster <cluster-id> --name <cluster-name>
```

Verify the cluster is imported and stable:
```bash
cloud cluster list --table
```

---

## 7. VPC Isolation Between Provisioners

When running multiple provisioners in the same AWS account (different VPCs), isolation is automatic:

- Each provisioner only discovers RDS clusters tagged with its own `VpcID`
- Each provisioner only claims VPCs for Kubernetes clusters it manages
- Each provisioner has its own separate metadata database

**The only risk:** if a Kubernetes cluster from the new VPC is accidentally added to the old provisioner, it would search for RDS tagged with the new VPC ID. Avoid registering clusters in the wrong provisioner.

---

## 8. Adding a New Multitenant RDS PgBouncer Cluster

The provisioner discovers RDS clusters automatically on first use (when an installation needs a database). To make a new Aurora cluster discoverable, tag it in AWS:

| Tag | Value |
|-----|-------|
| `DatabaseType` | `multitenant-rds-dbproxy` |
| `MattermostCloudInstallationDatabase` | `PostgreSQL/Aurora` |
| `VpcID` | the VPC ID (e.g. `vpc-0e21107e8d3c6699e`) |
| `Purpose` | `provisioning` |
| `Owner` | `cloud-team` |
| `Terraform` | `true` |
| `MultitenantDatabaseID` | the RDS cluster identifier |
| `Counter` | `0` |

> **Critical**: `DatabaseType` must be exactly `multitenant-rds-dbproxy` (not `multitenant-rds`). The provisioner filters using this exact value when discovering PgBouncer-type clusters.

The cluster name must contain the prefix `rds-cluster-multitenant`, e.g.:
```
rds-cluster-multitenant-<vpc-id-hash>-pgbouncer
```

Once tagged, the provisioner will discover and register the cluster automatically the next time an installation in that VPC needs a database.

---

## 9. Setting Up the Multitenant RDS Master Secret

The provisioner connects to the RDS cluster as its master user to provision logical databases. The secret **must be a plain password string** — not JSON.

```bash
# Generate an alphanumeric-only password (required — password goes directly into a postgres:// URL)
PASSWORD=$(openssl rand -hex 20)

# Create the secret as a plain string
aws secretsmanager create-secret \
  --name rds-cluster-multitenant-<vpc-id-hash>-pgbouncer \
  --secret-string "$PASSWORD"
```

> **Why alphanumeric only?** The provisioner passes `*SecretString` directly as the password in a `postgres://` URL without URL-encoding. Special characters (`:`, `@`, `/`, `+`, `]`, etc.) will break URL parsing and cause `net/url: invalid userinfo` errors.

> **Why plain string?** The provisioner code does `url.Password = *masterSecretValue.SecretString` directly. If the secret is JSON (e.g. `{"MasterPassword":"..."}`), the entire JSON blob is used as the password, causing auth failures.

The secret name format is: `rds-cluster-multitenant-<vpc-id-hash>-pgbouncer`
(matches the RDS cluster identifier with `-pgbouncer` suffix).

### Granting the Master User CREATEDB

The master user must have the `CREATEDB` attribute to create logical databases per installation:

```sql
ALTER ROLE <master-user> CREATEDB;
```

This is separate from `rds_superuser` membership — the attribute must be set explicitly.

---

## 10. Setting Up the PgBouncer Auth User Secret

PgBouncer uses a dedicated `pgbouncer` database user to authenticate client connections via the `get_auth` function. Its credentials must be stored in Secrets Manager.

The secret name format is:
```
rds-multitenant-pgbouncer-authuser-<vpc-id>
```

where `<vpc-id>` includes the `vpc-` prefix (e.g. `vpc-0e21107e8d3c6699e`):

```bash
AUTH_PASSWORD=$(openssl rand -hex 20)

aws secretsmanager create-secret \
  --name rds-multitenant-pgbouncer-authuser-vpc-0e21107e8d3c6699e \
  --secret-string "$AUTH_PASSWORD"
```

> **Note**: The VPC ID in the secret name includes the `vpc-` prefix. The provisioner calls `cluster.VpcID()` which returns the full ID with prefix.

Then create the `pgbouncer` user in RDS with that password:

```sql
CREATE USER pgbouncer WITH PASSWORD '<AUTH_PASSWORD>';
```

---

## 11. S3 State Bucket

The provisioner uses an S3 bucket for state storage. The bucket name is derived from the AWS account alias and VPC ID:

```
mattermost-cloud-<env>-provisioning-<vpc-id>
```

where `<env>` comes from the AWS account alias. The alias must follow the format `mattermost-cloud-<env>` (e.g. `mattermost-cloud-staging` → `env=staging`).

```bash
aws s3api create-bucket \
  --bucket mattermost-cloud-<env>-provisioning-<vpc-id> \
  --region us-east-1
```

> If you share an AWS account with another environment (e.g. `mattermost-cloud-test`), you cannot change the account alias. The bucket name will use whatever environment name the alias resolves to — plan accordingly.

---

## 12. Troubleshooting: PgBouncer `bouncer config error (08P01)`

For **new** logical databases, `ensurePGBouncerDatabasePrep` creates the `pgbouncer` schema with the auth role as owner and `get_auth` as `SECURITY DEFINER`, so access is set up automatically.

If you see `bouncer config error (08P01)` on a **legacy or manually-created** logical database where the `pgbouncer` user lacks schema access, connect as the master user and run:

```sql
GRANT USAGE ON SCHEMA pgbouncer TO pgbouncer;
GRANT SELECT ON TABLE pgbouncer.pgbouncer_users TO pgbouncer;
GRANT EXECUTE ON FUNCTION pgbouncer.get_auth(text) TO pgbouncer;
```

Verify the grants are working:

```bash
# Connect as the pgbouncer user to the logical database
psql -h <rds-endpoint> -U pgbouncer -d cloud_<logical-db-id>
```

```sql
SELECT * FROM pgbouncer.get_auth('id_<installation-id>');
-- Should return a row; "permission denied for schema pgbouncer" means grants are still missing
```

After granting, restart PgBouncer pods to reload config:

```bash
kubectl rollout restart deployment pgbouncer -n pgbouncer
```

---

## 13. Security Group: Provisioner → RDS Access

The provisioner pod must be able to reach the multitenant RDS cluster on port 5432. Add an inbound rule to the RDS security group:

- **Type**: PostgreSQL (port 5432)
- **Source**: the security group attached to the provisioner pod (or its node group)

Without this, the provisioner will time out when trying to provision logical databases, and installations will remain in `creation-pre-provisioning`.

---

## Troubleshooting

| Error | Cause | Fix |
|-------|-------|-----|
| `private subnet list is empty` | Subnets missing `SubnetType` tag | Tag subnets with `SubnetType=private` / `public` |
| `master security group list is empty` | SGs missing `NodeType` tag | Tag SGs with `NodeType=master/worker/calls` |
| `couldn't claim VPC as primary nor secondary` | VPC missing availability tags | Add `Available=true`, `CloudClusterID=none`, `CloudSecondaryClusterID=none` to VPC |
| `failed to create namespace: Unauthorized` | Provisioner IAM role not authorized on EKS | Add role to `aws-auth` or EKS access entries |
| `init-database` exit 1 / DNS timeout | Provisioner can't reach its RDS | Check RDS endpoint, security groups, and VPC DNS resolution |
| `no multitenant proxy databases are currently available` | `DatabaseType` tag wrong on RDS | Tag must be `multitenant-rds-dbproxy`, not `multitenant-rds` |
| `ResourceNotFoundException` for pgbouncer auth user secret | Secret name missing `vpc-` prefix | Secret must be named `rds-multitenant-pgbouncer-authuser-vpc-<id>` |
| `net/url: invalid userinfo` | Master RDS secret is JSON, not plain password | Store only the password string in the secret, not a JSON object |
| `bouncer config error (08P01)` | `pgbouncer` user lacks schema permissions in logical DB | Grant `USAGE`, `SELECT`, `EXECUTE` on pgbouncer schema objects to `pgbouncer` user |
| Installation stuck in `deletion-final-cleanup` | Logical database or pgbouncer objects missing | Create stub database and pgbouncer schema objects manually, then retry deletion |
