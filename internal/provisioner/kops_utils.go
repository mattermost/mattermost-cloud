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

const (
	// SkipAMIUpdateLabelKey is the label key that can be applied to a kops instance
	// group to indicate to the provisioner that it should skip updating the AMI.
	// The label value can be any string.
	SkipAMIUpdateLabelKey string = "mattermost/cloud-provisioner-ami-skip"
	// AMISuffixLabelKey is the label key that can be applied to a kops instance
	// group to indicate to the provisioner that it should use a different AMI.
	// The label value will be applied as a suffix to whatever AMI value is
	// passed in for the update.
	// Example:
	//   Cluster AMI is "custom-ubuntu"
	//   Instance group label is set to "mattermost/cloud-provisioner-ami-suffix=-arm64"
	//   Final AMI for that instance group is "custom-ubuntu-arm64"
	AMISuffixLabelKey string = "mattermost/cloud-provisioner-ami-suffix"
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
			if val, ok := ig.Metadata.Labels[SkipAMIUpdateLabelKey]; ok {
				logger.Infof("Instance group has label %s=%s; skipping AMI update...", SkipAMIUpdateLabelKey, val)
				continue
			}
			if kopsMetadata.ChangeRequest.AMI == "latest" {
				// Setting the image value to "" leads kops to autoreplace it with
				// the default image for that kubernetes release.
				logger.Infof("Updating instance group '%s' image value to the default kops image", ig.Metadata.Name)
				ami = ""
			} else {
				ami = kopsMetadata.ChangeRequest.AMI
				if suffix, ok := ig.Metadata.Labels[AMISuffixLabelKey]; ok {
					logger.Infof("Instance group has label %s=%s; applying custom AMI suffix...", AMISuffixLabelKey, suffix)
					ami = kopsMetadata.ChangeRequest.AMI + suffix
				}
				logger.Infof("Updating instance group '%s' image value to '%s'", ig.Metadata.Name, ami)
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
