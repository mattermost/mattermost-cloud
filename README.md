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
3. Install [kops](https://github.com/kubernetes/kops/blob/master/docs/install.md) version 1.13.X
4. Install [Helm](https://helm.sh/docs/using_helm/)
5. Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
6. Generate an AWS Access and Secret key pair, then export them in your bash profile:
  ```
  export AWS_ACCESS_KEY_ID=YOURACCESSKEYID
  export AWS_SECRET_ACCESS_KEY=YOURSECRETACCESSKEY
  ```
7. Create an S3 bucket to store the kops state
8. Clone this repository into your GOPATH (or anywhere if you have Go Modules enabled)

### Building

Simply run the following:

```
$ go install ./cmd/cloud
```

### Running

Before running the server the first time you must set up the DB with:

```
$ cloud schema migrate
```

To run the server you will need a certificate ARN, private and public Route53 IDs, and a private DNS from AWS. For staff developers you can get these in our dev AWS account.

Run the server with:

```
$ cloud server --state-store=<your-s3-bucket> --private-dns dev.cloud.internal.mattermost.com --private-route53-id=<private-route53-id> --route53-id=<public-route53-id> --certificate-aws-arn <certifcate-aws-arn>
```

### Testing

Run the go tests to test:

```
$ go test ./...
```
