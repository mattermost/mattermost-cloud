// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// verifyTerraformAndKopsMatch looks at terraform output and verifies that the
// given kops name matches. This should only catch errors where terraform output
// was incorrectly created from kops or if the terraform client is targeting the
// wrong directory, but should be used as a final sanity check before invoking
// terraform commands.
func verifyTerraformAndKopsMatch(kopsName string, terraformClient *terraform.Cmd, logger log.FieldLogger) error {
	out, ok, err := terraformClient.Output("cluster_name")
	if err != nil {
		return err
	}
	if !ok {
		logger.Warn("No cluster_name in terraform config, skipping check")
		return nil
	}
	if out != kopsName {
		return errors.Errorf("terraform cluster_name (%s) does not match kops_name from provided ID (%s)", out, kopsName)
	}

	return nil
}

func updateKopsInstanceGroupAMIs(kops *kops.Cmd, kopsMetadata *model.KopsMetadata, logger log.FieldLogger) error {
	if len(kopsMetadata.ChangeRequest.AMI) == 0 {
		logger.Info("Skipping cluster AMI update")
		return nil
	}

	instanceGroups, err := kops.GetInstanceGroupsJSON(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get instance groups")
	}

	var ami string
	for _, ig := range instanceGroups {
		if ig.Spec.Image != kopsMetadata.ChangeRequest.AMI {
			if kopsMetadata.ChangeRequest.AMI == "latest" {
				// Setting the image value to "" leads kops to autoreplace it with
				// the default image for that kubernetes release.
				logger.Infof("Updating instance group '%s' image value the default kops image", ig.Metadata.Name)
				ami = ""
			} else {
				logger.Infof("Updating instance group '%s' image value to '%s'", ig.Metadata.Name, kopsMetadata.ChangeRequest.AMI)
				ami = kopsMetadata.ChangeRequest.AMI
			}

			err = kops.SetInstanceGroup(kopsMetadata.Name, ig.Metadata.Name, fmt.Sprintf("spec.image=%s", ami))
			if err != nil {
				return errors.Wrap(err, "failed to update instance group ami")
			}
		}
	}

	return nil
}

func updateKopsInstanceGroupValue(kops *kops.Cmd, kopsMetadata *model.KopsMetadata, value string) error {

	instanceGroups, err := kops.GetInstanceGroupsJSON(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get instance groups")
	}

	for _, ig := range instanceGroups {

		err = kops.SetInstanceGroup(kopsMetadata.Name, ig.Metadata.Name, value)
		if err != nil {
			return errors.Wrapf(err, "failed to update value %s", value)
		}
	}

	return nil
}

func updateWorkersKopsInstanceGroupValue(kops *kops.Cmd, kopsMetadata *model.KopsMetadata, value string) error {

	instanceGroups, err := kops.GetInstanceGroupsJSON(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get instance groups")
	}

	for _, ig := range instanceGroups {
		if strings.Contains(ig.Metadata.Name, "nodes-") {
			err = kops.SetInstanceGroup(kopsMetadata.Name, ig.Metadata.Name, value)
			if err != nil {
				return errors.Wrapf(err, "failed to update value %s", value)
			}
		}
	}

	return nil
}
