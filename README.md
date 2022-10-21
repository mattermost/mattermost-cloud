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
2. Install [Terraform](https://learn.hashicorp.com/terraform/getting-started/install.html) version v1.0.7
   1. Try using [tfswitch](https://warrensbox.github.io/terraform-switcher/) for switching easily between versions
3. Install [kops](https://github.com/kubernetes/kops/blob/master/docs/install.md) version 1.23.X
4. Install [Helm](https://helm.sh/docs/intro/install/) version 3.5.X
5. Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
6. Install [golang/mock](https://github.com/golang/mock#installation) version 1.4.x

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
  ```bash
  aws s3api create-bucket --bucket cloud-<yourname>-kops-state --region us-east-1
  ```
4. Clone this repository into your GOPATH (or anywhere if you have Go Modules enabled)

5. Generate a Gitlab Token for access to the utities Helm values repo.  Then export:
```
export GITLAB_OAUTH_TOKEN=YOURTOKEN
```
This is the option when a remote git repo is used for utility values. In case you want to use local values for local testing you can export an empty token value and add the values in the relevant file in `helm-charts` directory and pass it in the cluster creation step.

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

- Make sure you have `helm version` different than 2.16.4 as there is an issue with
  nginx: https://stackoverflow.com/questions/60836127/error-validation-failed-serviceaccounts-nginx-ingress-not-found-serviceacc

### Building

Simply run the following:

```bash
go install ./cmd/cloud
alias cloud='$HOME/go/bin/cloud'
```


### Running
Before running the server the first time you must set up the DB with:

```bash
$ cloud schema migrate
```

Run the server with:

```bash
cloud server --state-store=<your-s3-bucket> --utilities-git-url=<https://gitlab.example.com>
```
tip: if you want to debug, enable `--dev` flag


In a different terminal/window, to create a cluster:
```bash
cloud cluster create --zones <availabiity-zone> --size SizeAlef500
i.e.
cloud cluster create --zones us-east-1c --size SizeAlef500
```
Note: Provisioner's default network provider is **amazon-vpc-routed-eni** which can be overridden using `--networking` flag. Supported network providers are weave, canal, calico, amazon-vpc-routed-eni.e.g
```bash
    cloud cluster create --networking weave
```
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
Check its creation progress on the first window where the API runs or run `cloud cluster list`
 to check cluster status

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
$ go test ./...
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

#### Manual Terraform 0.12 Migration Steps

Kops version 1.18.0 should introduce terraform 0.12 as the default version. Before migrating to the new terraform version, manual terraform state migration may be required. To perform a check if migration is needed, follow these instructions provided by kops:

```
kops update cluster --target terraform ...
terraform plan
# Observe any aws_route or aws_vpc_ipv4_cidr_block_association resources being destroyed and recreated
# Run these commands as necessary. The exact names may differ; use what is outputted by terraform plan
terraform state mv aws_route.0-0-0-0--0 aws_route.route-0-0-0-0--0
terraform state mv aws_vpc_ipv4_cidr_block_association.10-1-0-0--16 aws_vpc_ipv4_cidr_block_association.cidr-10-1-0-0--16
terraform state list | grep aws_autoscaling_attachment | xargs -L1 terraform state rm
terraform plan
# Ensure these resources are no longer being destroyed and recreated
terraform apply
```

Tip: a quick and reliable way to get access to a cluster's terraform files and state is to use the `cloud workbench cluster` command. This will checkout the correct files locally in the same manner that the provisioning process uses.

For more information on this change and reasoning for it, check out the [kops release notes](https://github.com/kubernetes/kops/releases/tag/v1.18.3).

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
