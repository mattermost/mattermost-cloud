# Mattermost Cloud ![CircleCI branch](https://img.shields.io/circleci/project/github/mattermost/mattermost-cloud/master.svg)

Mattermost Cloud is a SaaS offering meant to smooth and accelerate the customer journey from trial to full adoption. There is a significant amount of friction for a customer to set up a trial of Mattermost, and even more friction to run an extended length proof of concept. Both require hardware and technical personnel resources that create a significant barrier to potential customers. Mattermost Cloud aims to remove this barrier by providing a service to provision and host Mattermost instances that can be used by customers who have no technical experience. This will accelerate the customer journey to a full adoption of Mattermost either in the form of moving to a self-hosted instance or by continuing to use the cloud service.

Read more about the [Mattermost Cloud Architecture](https://docs.google.com/document/u/1/d/1L11OcIlm6YJcPbnQyTsczwm4M3DcyKDx3ahmHN-DcmQ/edit).

## Other Resources

This repository houses the open-source components of Mattermost Private Cloud. Other resources are linked below:

- [Mattermost the server and user interface](https://github.com/mattermost/mattermost-server)
- [Helm chart for Mattermost Enterprise Edition](https://github.com/mattermost/mattermost-kubernetes)
- [Experimental Mattermost operator for Kubernetes](https://github.com/mattermost/mattermost-operator)

## Get Involved

- [Join the discussion on Mattermost Cloud](https://community.mattermost.com/core/channels/cloud)

## Developing

### Environment Setup

#### Required Software

The following is required to properly run the cloud server.

##### Note: when versions are specified, it is extremely important to follow the requirement. Newer versions will often not work as expected

1. Install [Go](https://golang.org/doc/install)
2. Install [Terraform](https://learn.hashicorp.com/terraform/getting-started/install.html) version v1.5.5
   1. Try using [tfswitch](https://warrensbox.github.io/terraform-switcher/) for switching easily between versions
3. Install [kops](https://github.com/kubernetes/kops/blob/master/docs/install.md) version 1.27.X
4. Install [Helm](https://helm.sh/docs/intro/install/) version 3.11.X
5. Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
6. Install [golang/mock](https://github.com/golang/mock#installation) version 1.4.x
7. (macOS only) Upgrade [diffutils](https://www.gnu.org/software/diffutils/) with `brew install diffutils`. Used by the linters.

#### Other Setup

1. Specify the region in your AWS config, e.g. `~/.aws/config`:
```
[profile mm-cloud]
region = us-east-1
```

2. Generate an AWS Access and Secret key pair, then export them in your bash profile:
```
export AWS_ACCESS_KEY_ID=YOURACCESSKEYID
export AWS_SECRET_ACCESS_KEY=YOURSECRETACCESSKEY
export AWS_PROFILE=mm-cloud
```
3. Create an S3 bucket to store the kops state. The name of the bucket **MUST** start with `cloud-` prefix and
  has to be created using the provisioner account credentials. To do it run the following:
```
aws s3api create-bucket --bucket cloud-<yourname>-kops-state --region us-east-1
```
4. Clone this repository into your GOPATH (or anywhere if you have Go Modules enabled)

5. Generate a Gitlab Token for access to the utilities Helm values repo. Then export:
```
export GITLAB_OAUTH_TOKEN=<token>
```
This is the option when a remote git repo is used for utility values. In case you want to use local values for local testing you can export an empty token value and add the values in the relevant file in `helm-charts` directory and pass it in the cluster creation step.

6. Get a CloudFlare API Key, and export it with
```
export CLOUDFLARE_API_KEY=<key>
```

Also:
- Make sure you have a key in your ~/.ssh/
    such as:
    ```
    id_rsa
    id_rsa.pub
    ```
    If not you need to run:
    ```
    ssh-keygen -t rsa -C "<your-email>"
    ```

### Building

The provisioning server can be built and installed by running the following:

```bash
go install ./cmd/cloud
```

This will install the cloud binary to your `$GOPATH/bin` directory. If you haven't already, it's generally a good idea to add this directory to your `$PATH` to easily reference.

### Notes on the Cloud Server Database

The cloud server makes use of a PostgreSQL database to store information on currently-deployed resources. Maintaining this database is important so that the provisioner can update and delete resources in the future.

NOTE: the cloud server recently deprecated SQLite support. If you were using a SQLite database for development then review the deprecation notes at the end of the README.

### Running
Before running the server the first time you must set up the DB with:
```bash
make dev-start
```

This will start a docker PostgreSQL container `cloud-postgres` for your provisioning server to store resource information in.

Before using the new database we need to ensure it has the latest schema by running the following:
```bash
$ cloud schema migrate --database 'postgres://provisioner:provisionerdev@localhost:5430/cloud?sslmode=disable'
```

You are now ready to start the provisioning server. Here is a command with some required flags as well as some flags which help with development.

```bash
cloud server --dev --database=postgres://provisioner:provisionerdev@localhost:5430/cloud?sslmode=disable --state-store=<your-s3-bucket> --utilities-git-url=<https://gitlab.example.com>
```

### Creating your First Cluster

Before you can create a Mattermost installation you will need a kubernetes cluster. You can create one by running the following command in a new terminal window.
```bash
cloud cluster create --zones <availabiity-zone> --size SizeAlef500 --networking calico
i.e.
cloud cluster create --zones=us-east-1a --networking calico
```
Note: Provisioner's default network provider is **amazon-vpc-routed-eni** which can be overridden using `--networking` flag. Supported network providers are weave, canal, calico, amazon-vpc-routed-eni.

You will get a response like this one:
```bash
[
    {
        "ID": "tetw1yt3yinjdbhctsstcrybch",
        "Provider": "aws",
        "Provisioner": "kops",
        "ProviderMetadata": "eyJab25lcyI6WyJ1cy1lYXN0LTFjIl19",
        "ProvisionerMetadata": "eyJOYW1lIjoidGV0dzF5dDN5aW5qZGJoY3Rzc3RjcnliY2gta29wcy5rOHMubG9jYWwiLCJWZXJzaW9uIjoibGF0ZXN0IiwiQU1JIjoiIn0=",
        "AllowInstallations": true,
        "Version": "1.16.9",
        "Size": "SizeAlef500",
        "State": "creation-requested",
        "CreateAt": 1589188584607,
        "DeleteAt": 0,
        "LockAcquiredBy": "4z4f3xf6sfgnbrxfze94zrbb5y",
        "LockAcquiredAt": 1589211779835,
        "UtilityMetadata": "eyJkZXNpcmVkVmVyc2lvbnMiOnsiUHJvbWV0aGV1cyI6IjEwLjQuMCIsIk5naW54IjoiMS4zMC4wIiwiRmx1ZW50Yml0IjoiMi44LjciLCJDZXJ0TWFuYWdlciI6InYwLjEzLjEiLCJQdWJsaWNOZ2lueCI6IjEuMzAuMCJ9LCJhY3R1YWxWZXJzaW9ucyI6eyJQcm9tZXRoZXVzIjoiMTAuNC4wIiwiTmdpbngiOiIxLjMwLjAiLCJGbHVlbnRiaXQiOiIyLjguNyIsIkNlcnRNYW5hZ2VyIjoidjAuMTMuMSIsIlB1YmxpY05naW54IjoiMS4zMC4wIn19"
    }
]
```

The terminal window where the cloud server is running should produce logs as it creates the cluster. At any time you can run `cloud cluster list` to check on the status of the cluster.

If something breaks and reprovisioning is needed, run
```bash
cloud cluster provision --cluster <cluster-ID>
i.e.
cloud cluster provision --cluster tetw1yt3yinjdbhctsstcrybch
```

Then, when the `state` will be `stable`, export the kubeconfig:
```bash
kops export kubecfg <cluster-ID>-kops.k8s.local --state s3://<your-s3-bucket>
i.e.
kops export kubecfg tetw1yt3yinjdbhctsstcrybch-kops.k8s.local --state s3://angelos-kops-state
```

Now check if you can get the pods from the cluster with: `kubectl get pods -A`

#### Installation
To create an installation, run:
```bash
cloud installation create --owner <your-name> --dns <your-dns-record> --size 100users --affinity multitenant
i.e. in test account
cloud installation create --owner stelios --dns stelios.test.cloud.mattermost.com --size 100users --affinity multitenant
```

Check its creation progress on the first window where the API runs or run `cloud installation list`
 to check installation status

After the installation has finished(stable) you will be able to access your installation
on your <your-dns-record>

### Testing

Run the go tests to test:

```bash
$ make unittest
```

### End-to-end tests

There are some end-to-end tests located in `./e2e` directory.

#### DB Migration end-to-end tests

DB Migration e2e tests can be run with `make e2e-db-migration`.

E2e test for DB migration requires the following setup:
- Local instance of Provisioner.
- Workload cluster able to handle at least 2 installations of size `1000users`.
- 2 Multi tenant Postgres databases provisioned in the same VPC as the workload cluster.
- The destination database to which the migration can be performed should be exported as environment variable `DESTINATION_DB`.
- Kubeconfig of workload cluster exported locally.

#### Cluster end-to-end tests

Cluster provisioning e2e tests can be run with `make e2e-cluster`.

### Deleting a cluster and installations
Before deleting a cluster you will **have** to delete the installations first on it.

First, check how many installations are running:
```bash
cloud installation list
[
    {
        "ID": "8npnfpbiitygxbsrpg1p5i1sse",
        "OwnerID": "stelios",
        "GroupID": "",
        "Version": "stable",
        "Image": "mattermost/mattermost-enterprise-edition",
        "DNS": "stelios.test.cloud.mattermost.com",
        "Database": "mysql-operator",
        "Filestore": "minio-operator",
        "License": "",
        "MattermostEnv": null,
        "Size": "100users",
        "Affinity": "multitenant",
        "State": "creation-final-tasks",
        "CreateAt": 1589359071894,
        "DeleteAt": 0,
        "LockAcquiredBy": "m6sj6oc7fg93jkfgh3tjqw",
        "LockAcquiredAt": 1589359923924
    }
]
```
Get their IDs, and delete them
```bash
cloud installation delete --installation <installation-ID>
i.e.
cloud installation delete --installation 8npnfpbiitygxbsrpg1p5i1sse
```

Then proceed to cluster deletion:
```bash
cloud cluster delete --cluster <cluster-ID>
```

If a cluster cannot be deleted due to absence of ssh key, need to run:
```bash
kops create secret --name <cluster-ID>-kops.k8s.local sshpublickey admin -i ~/.ssh/id_rsa.pub --state s3://<your-s3-bucket>
```

### Deprecation Instructions

#### Cloud Server SQLite to PostgreSQL Migration

The cloud server recently deprecated SQLite database support.

The simplest migration method to PostgreSQL is to remove all running resources and then migrate to PostgreSQL by doing the following:
1. Run `cloud installation delete` for all active installations.
2. Run `cloud cluster delete` for all active clusters.
3. Run `make dev-start` to build a new PG docker container

Your existing `cloud.db` SQLite database will remain and can by kept for historical purposes if you want to lookup old resource IDs or events.

#### Cloud Server Postgres Version 12.5 -> 14.8 Migration

When upgrading the Postgres image in the docker-compose.yaml file for local development, the following steps can be followed:

1. Run `docker exec cloud-postgres pg_dumpall -U provisioner > backup.sql` to perform a backup of your existing database
2. Run `docker-compose down`
3. Run `docker-compose up -d`
4. Open `backup.sql` created in step 1 with a text editor, and remove the line `ALTER ROLE provisioner WITH SUPERUSER INHERIT CREATEROLE CREATEDB LOGIN REPLICATION BYPASSRLS PASSWORD '<some hash>'; ` so the provisioner user's credentials aren't overwritten
5. Run `cat backup.sql | docker exec -i cloud-postgres psql -U provisioner`

After completing the above steps, you should be up and running with your existing data in place on pg 14.8, persisted to a volume at `~/.cloud`

#### Cluster reprovisioning steps for new NGINX deployment

This is related to the changes introduced in [PR-263](https://github.com/mattermost/mattermost-cloud/pull/263)

Please follow the steps below for the reprovisioning of existing clusters:
- Reprovision the cluster by running ```cloud cluster provision --cluster <cluster_id> --nginx-version 4.0.18```.
- Check that new nginx deployed both internal and public Load Balancers (nginx-ingress-nginx-controller-internal and nginx-ingress-nginx-controller).
- Manually update Prometheus Route53 record to use the new private Load Balancer (nginx-ingress-nginx-controller-internal).
- Manually update cluster installations Route53 records one by one to use the new public Load balancer (nginx-ingress-nginx-controller).
  - Update clusterinstallation ingress annotation to use Nginx class nginx-controller instead of nginx.
  - Manually update network policy to target nginx instead of public-nginx.
- Confirm that all services are up and running.
- Delete old NGINX helm charts.
  - ```helm del --purge public-nginx```
  - ```helm del --purge private-nginx```
