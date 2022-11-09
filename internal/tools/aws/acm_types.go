// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/mattermost/mattermost-cloud/model"
)

// ACMAPI represents the series of calls we require from the AWS SDK v2 ACM Client
type ACMAPI interface {
	ListCertificates(ctx context.Context, params *acm.ListCertificatesInput, optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error)
	ListTagsForCertificate(ctx context.Context, params *acm.ListTagsForCertificateInput, optFns ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error)
}

// newCertificateFromACMCertificateSummary converts an ACM's certificate summary into our own certificate type
func newCertificateFromACMCertificateSummary(c types.CertificateSummary) *model.Certificate {
	return &model.Certificate{
		ARN: c.CertificateArn,
	}
}
