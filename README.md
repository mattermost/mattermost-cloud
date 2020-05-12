# Mattermost Private Cloud ![CircleCI branch](https://img.shields.io/circleci/project/github/mattermost/mattermost-cloud/master.svg)

Mattermost Private Cloud is a SaaS offering meant to smooth and accelerate the customer journey from trial to full adoption. There is a significant amount of friction for a customer to set up a trial of Mattermost, and even more friction to run an extended length proof of concept. Both require hardware and technical personnel resources that create a significant barrier to potential customers. Mattermost Cloud aims to remove this barrier by providing a service to provision and host Mattermost instances that can be used by customers who have no technical experience. This will accelerate the customer journey to a full adoption of Mattermost either in the form of moving to a self-hosted instance or by continuing to use the cloud service.

Read more about the [Mattermost Private Cloud Architecture](https://docs.google.com/document/d/1DZRrJ4LymdNA-D130i44VICLKmTzwAZMTMjiYNYIfiM/edit#).

## Other Resources

This repository houses the open-source components of Mattermost Private Cloud. Other resources are linked below:

- [Mattermost the server and user interface](https://github.com/mattermost/mattermost-server)
- [Helm chart for Mattermost Enterprise Edition](https://github.com/mattermost/mattermost-kubernetes)
- [Experimental Mattermost operator for Kubernetes](https://github.com/mattermost/mattermost-operator)

## Get Involved

- [Join the discussion on Private Cloud](https://community.mattermost.com/core/channels/cloud)

## Developing

### Environment Setup

1. Install [Go](https://golang.org/doc/install)
2. Install [Terraform](https://learn.hashicorp.com/terraform/getting-started/install.html) version v0.11.14
   1. Try using [tfswitch](https://warrensbox.github.io/terraform-switcher/) for switching easily between versions
3. Install [kops](https://github.com/kubernetes/kops/blob/master/docs/install.md) version 1.15.X
4. Install [Helm](https://helm.sh/docs/using_helm/) version 2.14.X
5. Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
6. Install [mockgen](github.com/golang/mock/mockgen) version 1.4.x
7. Specify the region in your AWS config, e.g. `~/.aws/config`:
```
[profile mm-cloud]
region = us-east-1
```
7. Generate an AWS Access and Secret key pair, then export them in your bash profile:
  ```
  export AWS_ACCESS_KEY_ID=YOURACCESSKEYID
  export AWS_SECRET_ACCESS_KEY=YOURSECRETACCESSKEY
  export AWS_PROFILE=mm-cloud
  ```
8. Create an S3 bucket to store the kops state
9. Clone this repository into your GOPATH (or anywhere if you have Go Modules enabled)

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
cloud server --state-store=<your-s3-bucket>
```
tip: if you want to debug, enable `--dev` flag


In a different terminal/window, to create a cluster:
```bash
cloud cluster create --zones <availabiity-zone> --size SizeAlef500
i.e.
cloud cluster create --zones us-east-1c --size SizeAlef500
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
cloud clusterprovision --cluster tetw1yt3yinjdbhctsstcrybch
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
cloud installation create --owner stelios --dns stelios.test.mattermost.cloud --size 100users --affinity multitenant
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
        "DNS": "stelios.test.mattermost.cloud",
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

Then procced to cluster deletion:
```bash
cloud cluster delete --cluster <cluster-ID>
```

If a cluster cannot be deleted due to absense of ssh key, need to run:
```bash
kops create secret --name <cluster-ID>-kops.k8s.local sshpublickey admin -i ~/.ssh/id_rsa.pub --state s3://<your-s3-bucket>
```

### Deprecation Instructions

#### Terraform remote state and the ./clusters directory

The cloud server used to store all terraform generated by kops locally in the `./clusters` directory.

This has been updated and the server now uses remote state in S3 for storing terraform state. The `./clusters` directory is now completely ignored. To manually update your terraform to use remote state, do the following:

```
cd clusters/CLUSTER_ID
terraform init -backend-config=bucket=NAME_OF_KOPS_STATE_BUCKET -backend-config=region=us-east-1 -backend-config=key=terraform/KOPS_CLUSTER_NAME -force-copy
```

Repeat for every cluster in `./clusters` dir.
Note that `KOPS_CLUSTER_NAME` is not just the cluster ID. Run `kops get clusters` to get the full kops cluster names.
