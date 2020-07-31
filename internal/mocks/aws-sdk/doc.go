// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package mocks to create the mocks run go generate to regenerate this package.
//go:generate ../../../bin/mockgen -package=mocks -destination ./ec2.go github.com/aws/aws-sdk-go/service/ec2/ec2iface EC2API
//go:generate ../../../bin/mockgen -package=mocks -destination ./rds.go github.com/aws/aws-sdk-go/service/rds/rdsiface RDSAPI
//go:generate ../../../bin/mockgen -package=mocks -destination ./s3.go github.com/aws/aws-sdk-go/service/s3/s3iface S3API
//go:generate ../../../bin/mockgen -package=mocks -destination ./acm.go github.com/aws/aws-sdk-go/service/acm/acmiface ACMAPI
//go:generate ../../../bin/mockgen -package=mocks -destination ./iam.go github.com/aws/aws-sdk-go/service/iam/iamiface IAMAPI
//go:generate ../../../bin/mockgen -package=mocks -destination ./route53.go github.com/aws/aws-sdk-go/service/route53/route53iface Route53API
//go:generate ../../../bin/mockgen -package=mocks -destination ./kms.go github.com/aws/aws-sdk-go/service/kms/kmsiface KMSAPI
//go:generate ../../../bin/mockgen -package=mocks -destination ./secrets_manager.go github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface SecretsManagerAPI
//go:generate ../../../bin/mockgen -package=mocks -destination ./resource_tagging.go github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface ResourceGroupsTaggingAPIAPI
//go:generate ../../../bin/mockgen -package=mocks -destination ./sts.go github.com/aws/aws-sdk-go/service/sts/stsiface STSAPI
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt ec2.go > _ec2.go && mv _ec2.go ec2.go"
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt rds.go > _rds.go && mv _rds.go rds.go"
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt s3.go > _s3.go && mv _s3.go s3.go"
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt acm.go > _acm.go && mv _acm.go acm.go"
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt iam.go > _iam.go && mv _iam.go iam.go"
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt route53.go > _route53.go && mv _route53.go route53.go"
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt kms.go > _kms.go && mv _kms.go kms.go"
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt secrets_manager.go > _secrets_manager.go && mv _secrets_manager.go secrets_manager.go"
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt resource_tagging.go > _resource_tagging.go && mv _resource_tagging.go resource_tagging.go"
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate/boilerplate.generatego.txt sts.go > _sts.go && mv _sts.go sts.go"
package mocks //nolint
