// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

type unmanaged struct {
	utilityName string
	logger      log.FieldLogger
}

func newUnmanagedHandle(utilityName string, logger log.FieldLogger) *unmanaged {
	return &unmanaged{
		utilityName: utilityName,
		logger: logger.WithFields(log.Fields{
			"cluster-utility": utilityName,
			"unmanaged":       true,
		}),
	}
}

func (u *unmanaged) ValuesPath() string {
	return ""
}

func (u *unmanaged) CreateOrUpgrade() error {
	u.logger.WithField("unmanaged-action", "create").Info("Utility is unmanaged; skippping...")

	return nil
}

func (u *unmanaged) Destroy() error {
	u.logger.WithField("unmanaged-action", "destroy").Info("Utility is unmanaged; skippping...")

	return nil
}

func (u *unmanaged) Migrate() error {
	return nil
}

func (u *unmanaged) Name() string {
	return u.utilityName
}

func (u *unmanaged) DesiredVersion() *model.HelmUtilityVersion {
	return &model.HelmUtilityVersion{Chart: model.UnmanagedUtilityVersion}
}

func (u *unmanaged) ActualVersion() *model.HelmUtilityVersion {
	return &model.HelmUtilityVersion{Chart: model.UnmanagedUtilityVersion}
}
