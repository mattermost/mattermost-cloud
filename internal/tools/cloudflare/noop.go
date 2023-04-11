// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package cloudflare

import "github.com/sirupsen/logrus"

// NoopCloudflarer is used as a dummy Cloudflarer interface
type NoopCloudflarer struct{}

// NoopClient returns an empty noopCloudflarer struct
func NoopClient() *NoopCloudflarer {
	return &NoopCloudflarer{}
}

// CreateDNSRecords returns an empty dummy func for noopCloudflarer
func (*NoopCloudflarer) CreateDNSRecords(_ []string, _ []string, logger logrus.FieldLogger) error {
	logger.Debug("Using noop Cloudflare client, CreateDNSRecords function")
	return nil
}

// DeleteDNSRecords returns an empty dummy func for noopCloudflarer
func (*NoopCloudflarer) DeleteDNSRecords(_ []string, logger logrus.FieldLogger) error {
	logger.Debug("Using noop Cloudflare client, DeleteDNSRecords function")
	return nil
}
