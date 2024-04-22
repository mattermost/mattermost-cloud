// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"regexp"
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

	for _, ig := range instanceGroups {
		if ig.Spec.Image != kopsMetadata.ChangeRequest.AMI {
			if val, ok := ig.Metadata.Labels[SkipAMIUpdateLabelKey]; ok {
				logger.Infof("Instance group has label %s=%s; skipping AMI update...", SkipAMIUpdateLabelKey, val)
				continue
			}

			ami := kopsMetadata.ChangeRequest.AMI // Default AMI from metadata

			// Handle the "latest" special case
			if ami == "latest" {
				ami = "" // Setting to default kops image
			} else {
				// Determine if a suffix modification is needed
				var archLabel string
				if suffix, ok := ig.Metadata.Labels[AMISuffixLabelKey]; ok && suffix != "" {
					archLabel = strings.TrimPrefix(suffix, "=")
				}
				ami = ModifyAMISuffix(ami, archLabel)
			}

			logger.Infof("Updating instance group '%s' image value to '%s'", ig.Metadata.Name, ami)
			err = kops.SetInstanceGroup(kopsMetadata.Name, ig.Metadata.Name, fmt.Sprintf("spec.image=%s", ami))
			if err != nil {
				return errors.Wrap(err, "failed to update instance group AMI")
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

// ModifyAMISuffix updates the AMI name based on the provided architecture label.
// If the AMI is in "ami-*" format, it returns it unmodified.
// If the AMI does not include an architecture suffix, it adds the default "amd64" or the provided arch label.
// If the AMI includes a different architecture suffix than the label, it replaces the suffix with the label.
func ModifyAMISuffix(ami, archLabel string) string {
	if strings.HasPrefix(ami, "ami-") {
		// Case 1: AMI starts with "ami-", return as is.
		return ami
	}

	// Regex to find existing architecture suffix in the AMI value.
	re := regexp.MustCompile(`-(amd64|arm64)$`)

	if !re.MatchString(ami) {
		// Case 2: No architecture suffix, append default "amd64" or the provided label.
		suffix := "amd64"
		if archLabel == "arm64" {
			suffix = archLabel
		}
		return ami + "-" + suffix
	}

	// Case 3: AMI already includes an architecture suffix.
	if archLabel != "" && !strings.HasSuffix(ami, "-"+archLabel) {
		// If a different architecture label is provided, replace the existing suffix with the label.
		return re.ReplaceAllString(ami, "-"+archLabel)
	}

	// If the existing suffix matches the label, or no label is provided, return the AMI unmodified.
	return ami
}
