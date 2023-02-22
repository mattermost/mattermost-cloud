package api

import "github.com/mattermost/mattermost-cloud/internal/provisioner"

func GetProvisionerOption(provisioner provisioner.Provisioner) Provisioner {
	return provisioner
}
