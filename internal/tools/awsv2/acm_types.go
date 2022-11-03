// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
// 

package awsv2

import (
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/mattermost/mattermost-cloud/model"
)

// newCertificateFromACMCertificateSummary converts an ACM's certificate summary into our own certificate type
func newCertificateFromACMCertificateSummary(c types.CertificateSummary) *model.Certificate {
	return &model.Certificate{
		ARN: c.CertificateArn,
	}
}
