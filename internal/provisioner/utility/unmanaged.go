// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"fmt"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

type unmanaged struct {
	cluster            *model.Cluster
	provisioner        string
	kubeconfigPath     string
	logger             log.FieldLogger
	actualVersion      *model.HelmUtilityVersion
	desiredVersion     *model.HelmUtilityVersion
	allowCIDRRangeList []string
	awsClient          aws.AWS
}

// newUnmanagedHandle creates a new instance of the unmanaged utility.
func newUnmanagedHandle(name, kubeconfigPath string, allowCIDRRangeList []string, cluster *model.Cluster, awsClient aws.AWS, logger log.FieldLogger) *unmanaged {
	return &unmanaged{
		cluster:            cluster,
		provisioner:        name,
		kubeconfigPath:     kubeconfigPath,
		logger:             logger,
		allowCIDRRangeList: allowCIDRRangeList,
		awsClient:          awsClient,
	}
}

// ValuesPath returns the path to the values files for this utility.
func (u *unmanaged) ValuesPath() string {
	return fmt.Sprintf("helm-values/%s/%s", u.cluster.ID, u.provisioner)
}

// CreateOrUpgrade creates or upgrades an unmanaged utility.
func (u *unmanaged) CreateOrUpgrade() error {
	logger := u.logger.WithField("unmanaged-action", "create")

	logger.Info("Utility is unmanaged; no create or upgrade action will be taken")

	return nil
}

// DesiredVersion returns the desired version for the unmanaged utility.
func (u *unmanaged) DesiredVersion() *model.HelmUtilityVersion {
	return u.desiredVersion
}

// ActualVersion returns the actual version for the unmanaged utility.
func (u *unmanaged) ActualVersion() *model.HelmUtilityVersion {
	return u.actualVersion
}

// Destroy destroys an unmanaged utility.
func (u *unmanaged) Destroy() error {
	logger := u.logger.WithField("unmanaged-action", "delete")

	logger.Info("Utility is unmanaged; no destroy action will be taken")

	return nil
}

// Migrate migrates an unmanaged utility.
func (u *unmanaged) Migrate() error {
	logger := u.logger.WithField("unmanaged-action", "migrate")

	logger.Info("Utility is unmanaged; no migrate action will be taken")

	return nil
}

// Name returns the name of the unmanaged utility.
func (u *unmanaged) Name() string {
	return u.provisioner
}
