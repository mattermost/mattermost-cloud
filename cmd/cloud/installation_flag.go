// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type installationCreateRequestOptions struct {
	name                      string
	ownerID                   string
	groupID                   string
	version                   string
	image                     string
	size                      string
	license                   string
	affinity                  string
	database                  string
	filestore                 string
	mattermostEnv             []string
	priorityEnv               []string
	annotations               []string
	groupSelectionAnnotations []string
}

func (flags *installationCreateRequestOptions) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.name, "name", "", "Unique human-readable installation name. It should be the same as first segment of domain name.")
	command.Flags().StringVar(&flags.ownerID, "owner", "", "An opaque identifier describing the owner of the installation.")
	command.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to join")
	command.Flags().StringVar(&flags.version, "version", "stable", "The Mattermost version to install.")
	command.Flags().StringVar(&flags.image, "image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	command.Flags().StringVar(&flags.size, "size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	command.Flags().StringVar(&flags.license, "license", "", "The Mattermost License to use in the server.")
	command.Flags().StringVar(&flags.affinity, "affinity", model.InstallationAffinityIsolated, "Whether the installation can be scheduled on a cluster with other installations. Accepts isolated or multitenant.")
	command.Flags().StringVar(&flags.database, "database", model.InstallationDatabaseMysqlOperator, "The Mattermost server database type. Accepts mysql-operator, aws-rds, aws-rds-postgres, aws-multitenant-rds, or aws-multitenant-rds-postgres")
	command.Flags().StringVar(&flags.filestore, "filestore", model.InstallationFilestoreMinioOperator, "The Mattermost server filestore type. Accepts minio-operator, aws-s3, bifrost, or aws-multitenant-s3")
	command.Flags().StringArrayVar(&flags.mattermostEnv, "mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	command.Flags().StringArrayVar(&flags.priorityEnv, "priority-env", []string{}, "Env vars to add to the Mattermost App that take priority over group config. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	command.Flags().StringArrayVar(&flags.annotations, "annotation", []string{}, "Additional annotations for the installation. Accepts multiple values, for example: '... --annotation abc --annotation def'")
	command.Flags().StringArrayVar(&flags.groupSelectionAnnotations, "group-selection-annotation", []string{}, "Annotations for automatic group selection. Accepts multiple values, for example: '... --group-selection-annotation abc --group-selection-annotation def'")

	_ = command.MarkFlagRequired("owner")
}

type rdsOptions struct {
	rdsPrimaryInstance string
	rdsReplicaInstance string
	rdsReplicasCount   int
}

func (flags *rdsOptions) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.rdsPrimaryInstance, "rds-primary-instance", "", "The machine instance type used for primary replica of database cluster. Works only with single tenant RDS databases.")
	command.Flags().StringVar(&flags.rdsReplicaInstance, "rds-replica-instance", "", "The machine instance type used for reader replicas of database cluster. Works only with single tenant RDS databases.")
	command.Flags().IntVar(&flags.rdsReplicasCount, "rds-replicas-count", 0, "The number of reader replicas of database cluster. Min: 0, Max: 15. Works only with single tenant RDS databases.")
}

type installationCreateFlags struct {
	clusterFlags
	installationCreateRequestOptions
	rdsOptions
	dns                        []string
	externalDatabaseSecretName string
}

func (flags *installationCreateFlags) addFlags(command *cobra.Command) {
	flags.installationCreateRequestOptions.addFlags(command)
	flags.rdsOptions.addFlags(command)

	command.Flags().StringSliceVar(&flags.dns, "dns", []string{}, "URLs at which the Mattermost server will be available.")
	command.Flags().StringVar(&flags.externalDatabaseSecretName, "external-database-secret-name", "", "The AWS secret name where the external database DSN is stored. Works only with external databases.")

	_ = command.MarkFlagRequired("dns")
}

type installationPatchRequestChanges struct {
	ownerIDChanged          bool
	versionChanged          bool
	imageChanged            bool
	sizeChanged             bool
	licenseChanged          bool
	allowedIPRangesChanged  bool
	overrideIPRangesChanged bool
}

func (flags *installationPatchRequestChanges) addFlags(command *cobra.Command) {
	flags.ownerIDChanged = command.Flags().Changed("owner")
	flags.versionChanged = command.Flags().Changed("version")
	flags.imageChanged = command.Flags().Changed("image")
	flags.sizeChanged = command.Flags().Changed("size")
	flags.licenseChanged = command.Flags().Changed("license")
	flags.allowedIPRangesChanged = command.Flags().Changed("allowed-ip-ranges")
	flags.overrideIPRangesChanged = command.Flags().Changed("override-ip-ranges")
}

type installationPatchRequestOptions struct {
	installationPatchRequestChanges
	ownerID            string
	version            string
	image              string
	size               string
	license            string
	allowedIPRanges    string
	mattermostEnv      []string
	mattermostEnvClear bool
	overrideIPRanges   bool
}

func (flags *installationPatchRequestOptions) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.ownerID, "owner", "", "The new owner value of this installation.")
	command.Flags().StringVar(&flags.version, "version", "stable", "The Mattermost version to target.")
	command.Flags().StringVar(&flags.image, "image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	command.Flags().StringVar(&flags.size, "size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	command.Flags().StringVar(&flags.license, "license", "", "The Mattermost License to use in the server.")
	command.Flags().StringVar(&flags.allowedIPRanges, "allowed-ip-ranges", "", "JSON Encoded list of IP Ranges that are allowed to access the workspace.")
	command.Flags().StringArrayVar(&flags.mattermostEnv, "mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	command.Flags().BoolVar(&flags.mattermostEnvClear, "mattermost-env-clear", false, "Clears all env var data.")
	command.Flags().BoolVar(&flags.overrideIPRanges, "override-ip-ranges", true, "Overrides Allowed IP ranges and force ignoring any previous value.")
}

func (flags *installationPatchRequestOptions) GetPatchInstallationRequest() *model.PatchInstallationRequest {
	request := model.PatchInstallationRequest{}

	if flags.ownerIDChanged {
		request.OwnerID = &flags.ownerID
	}

	if flags.versionChanged {
		request.Version = &flags.version
	}

	if flags.imageChanged {
		request.Image = &flags.image
	}

	if flags.sizeChanged {
		request.Size = &flags.size
	}

	if flags.licenseChanged {
		request.License = &flags.license
	}

	if flags.overrideIPRangesChanged {
		request.OverrideIPRanges = &flags.overrideIPRanges
	}

	return &request
}

type installationUpdateFlags struct {
	clusterFlags
	installationPatchRequestOptions
	priorityEnv      []string
	priorityEnvClear bool
	installationID   string
}

func (flags *installationUpdateFlags) addFlags(command *cobra.Command) {
	flags.installationPatchRequestOptions.addFlags(command)

	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be updated.")
	command.Flags().StringArrayVar(&flags.priorityEnv, "priority-env", []string{}, "Env vars to add to the Mattermost App that take priority over group config. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	command.Flags().BoolVar(&flags.priorityEnvClear, "priority-env-clear", false, "Clears all priority env var data.")

	_ = command.MarkFlagRequired("installation")
}

func (flags *installationUpdateFlags) GetPatchInstallationRequest() (*model.PatchInstallationRequest, error) {
	request := flags.installationPatchRequestOptions.GetPatchInstallationRequest()

	mattermostEnv, err := parseEnvVarInput(flags.mattermostEnv, flags.mattermostEnvClear)
	if err != nil {
		return nil, err
	}

	priorityEnv, err := parseEnvVarInput(flags.priorityEnv, flags.priorityEnvClear)
	if err != nil {
		return nil, err
	}

	if flags.allowedIPRangesChanged {
		allowedIPRanges := &model.AllowedIPRanges{}
		_, jsonErr := allowedIPRanges.FromJSONString(flags.allowedIPRanges)
		if jsonErr != nil {
			return nil, jsonErr
		}
		allowedIPRanges, err := allowedIPRanges.FromJSONString(flags.allowedIPRanges)
		if err != nil {
			return nil, err
		}
		request.AllowedIPRanges = allowedIPRanges
	}

	request.MattermostEnv = mattermostEnv
	request.PriorityEnv = priorityEnv

	return request, nil
}

type installationDeleteFlags struct {
	clusterFlags
	installationID string
}

func (flags *installationDeleteFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be deleted.")
	_ = command.MarkFlagRequired("installation")
}

type baseVolumeFlags struct {
	mountPath string
	readOnly  bool
	filename  string
	data      string
}
type installationCreateVolumeFlags struct {
	clusterFlags
	baseVolumeFlags
	installationID string
	volumeName     string
}

func (flags *installationCreateVolumeFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to create a volume for.")
	command.Flags().StringVar(&flags.volumeName, "volume-name", "", "The name of the volume to create.")
	command.Flags().StringVar(&flags.mountPath, "mount-path", "", "The container path to mount the volume in.")
	command.Flags().BoolVar(&flags.readOnly, "read-only", true, "Whether the volume should be read only or not.")

	command.Flags().StringVar(&flags.filename, "filename", "", "The name of the file that will be mounted in the volume mount path.")
	command.Flags().StringVar(&flags.data, "data", "", "The data contained in the file.")

	_ = command.MarkFlagRequired("installation")
	_ = command.MarkFlagRequired("volume-name")
	_ = command.MarkFlagRequired("filename")
	_ = command.MarkFlagRequired("data")
}

func (flags *installationCreateVolumeFlags) GetCreateInstallationVolumeRequest() *model.CreateInstallationVolumeRequest {
	return &model.CreateInstallationVolumeRequest{
		Name: flags.volumeName,
		Volume: &model.Volume{
			Type:      model.VolumeTypeSecret,
			MountPath: flags.mountPath,
			ReadOnly:  flags.readOnly,
		},
		Data: map[string][]byte{
			flags.filename: []byte(flags.data),
		},
	}
}

type installationUpdateVolumeChanges struct {
	mountPathChanged bool
	readOnlyChanged  bool
	filenameChanged  bool
	dataChanged      bool
}

func (flags *installationUpdateVolumeChanges) addFlags(command *cobra.Command) {
	flags.mountPathChanged = command.Flags().Changed("mount-path")
	flags.readOnlyChanged = command.Flags().Changed("read-only")
	flags.filenameChanged = command.Flags().Changed("filename")
	flags.dataChanged = command.Flags().Changed("data")
}

type installationUpdateVolumeFlags struct {
	installationUpdateVolumeChanges
	clusterFlags
	baseVolumeFlags
	installationID string
	volumeName     string
}

func (flags *installationUpdateVolumeFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to update a volume for.")
	command.Flags().StringVar(&flags.volumeName, "volume-name", "", "The name of the volume to update.")
	command.Flags().StringVar(&flags.mountPath, "mount-path", "", "The container path to mount the volume in.")
	command.Flags().BoolVar(&flags.readOnly, "read-only", true, "Whether the volume should be read only or not.")

	command.Flags().StringVar(&flags.filename, "filename", "", "The name of the file that will be mounted in the volume mount path.")
	command.Flags().StringVar(&flags.data, "data", "", "The data contained in the file.")

	_ = command.MarkFlagRequired("installation")
	_ = command.MarkFlagRequired("volume-name")
}

func (flags *installationUpdateVolumeFlags) Validate() error {
	if flags.filenameChanged && !flags.dataChanged {
		return errors.New("must provide --data when changing --filename")
	}
	if !flags.filenameChanged && flags.dataChanged {
		return errors.New("must provide --filename when changing --data")
	}

	return nil
}

func (flags *installationUpdateVolumeFlags) GetUpdateInstallationVolumeRequest() *model.PatchInstallationVolumeRequest {
	patch := &model.PatchInstallationVolumeRequest{}

	if flags.mountPathChanged {
		patch.MountPath = &flags.mountPath
	}
	if flags.readOnlyChanged {
		patch.ReadOnly = &flags.readOnly
	}
	if flags.filenameChanged && flags.dataChanged {
		patch.Data = map[string][]byte{
			flags.filename: []byte(flags.data),
		}
	}

	return patch
}

type installationDeleteVolumeFlags struct {
	clusterFlags
	installationID string
	volumeName     string
}

func (flags *installationDeleteVolumeFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to delete a volume from.")
	command.Flags().StringVar(&flags.volumeName, "volume-name", "", "The name of the volume to delete.")

	_ = command.MarkFlagRequired("installation")
	_ = command.MarkFlagRequired("volume-name")
}

type installationDeletionPatchRequestOptions struct {
	futureDeletionTime time.Duration
}

func (flags *installationDeletionPatchRequestOptions) addFlags(command *cobra.Command) {
	command.Flags().DurationVar(&flags.futureDeletionTime, "future-expiry", 0, "The amount of time from now when the installation can be deleted (0s for immediate deletion)")
}

type installationDeletionPatchRequestOptionsChanged struct {
	futureDeletionTimeChanged bool
}

func (flags *installationDeletionPatchRequestOptionsChanged) addFlags(command *cobra.Command) {
	flags.futureDeletionTimeChanged = command.Flags().Changed("future-expiry")
}

type installationUpdateDeletionFlags struct {
	clusterFlags
	installationDeletionPatchRequestOptions
	installationDeletionPatchRequestOptionsChanged
	installationID string
}

func (flags *installationUpdateDeletionFlags) addFlags(command *cobra.Command) {
	flags.installationDeletionPatchRequestOptions.addFlags(command)
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to update pending deletion parameters for.")
	_ = command.MarkFlagRequired("installation")
}

type installationCancelDeletionFlags struct {
	clusterFlags
	installationID string
}

func (flags *installationCancelDeletionFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to cancel pending deletion for.")
	_ = command.MarkFlagRequired("installation")
}

type installationHibernateFlags struct {
	clusterFlags
	installationID string
}

func (flags *installationHibernateFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to put into hibernation.")
	_ = command.MarkFlagRequired("installation")
}

type installationWakeupFlags struct {
	clusterFlags
	installationPatchRequestOptions
	installationID string
}

func (flags *installationWakeupFlags) addFlags(command *cobra.Command) {
	flags.installationPatchRequestOptions.addFlags(command)

	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to wake up from hibernation.")
	_ = command.MarkFlagRequired("installation")
}

func (flags *installationWakeupFlags) GetPatchInstallationRequest() (*model.PatchInstallationRequest, error) {
	request := flags.installationPatchRequestOptions.GetPatchInstallationRequest()

	envVarMap, err := parseEnvVarInput(flags.mattermostEnv, flags.mattermostEnvClear)
	if err != nil {
		return nil, err
	}

	request.MattermostEnv = envVarMap

	return request, nil
}

type installationGetFlags struct {
	clusterFlags
	installationID              string
	includeGroupConfig          bool
	includeGroupConfigOverrides bool
	hideLicense                 bool
	hideEnv                     bool
}

func (flags *installationGetFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be fetched.")
	command.Flags().BoolVar(&flags.includeGroupConfig, "include-group-config", true, "Whether to include group configuration in the installation or not.")
	command.Flags().BoolVar(&flags.includeGroupConfigOverrides, "include-group-config-overrides", true, "Whether to include a group configuration override summary in the installation or not.")
	command.Flags().BoolVar(&flags.hideLicense, "hide-license", true, "Whether to hide the license value in the output or not.")
	command.Flags().BoolVar(&flags.hideEnv, "hide-env", true, "Whether to hide env vars in the output or not.")

	_ = command.MarkFlagRequired("installation")
}

type installationGetRequestOptions struct {
	owner                       string
	group                       string
	state                       string
	dns                         string
	deletionLocked              bool
	includeGroupConfig          bool
	includeGroupConfigOverrides bool
}

func (flags *installationGetRequestOptions) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.owner, "owner", "", "The owner ID to filter installations by.")
	command.Flags().StringVar(&flags.group, "group", "", "The group ID to filter installations.")
	command.Flags().StringVar(&flags.state, "state", "", "The state to filter installations by.")
	command.Flags().StringVar(&flags.dns, "dns", "", "The dns name to filter installations by.")
	command.Flags().BoolVar(&flags.deletionLocked, "deletion-locked", false, "Filter installations by deletion-locked configuration.")
	command.Flags().BoolVar(&flags.includeGroupConfig, "include-group-config", true, "Whether to include group configuration in the installations or not.")
	command.Flags().BoolVar(&flags.includeGroupConfigOverrides, "include-group-config-overrides", true, "Whether to include a group configuration override summary in the installations or not.")
}

type installationGetRequestChanges struct {
	deletionLockedChanged bool
}

func (flags *installationGetRequestChanges) addFlags(command *cobra.Command) {
	flags.deletionLockedChanged = command.Flags().Changed("deletion-locked")
}

type installationListFlags struct {
	clusterFlags
	installationGetRequestOptions
	installationGetRequestChanges
	pagingFlags
	tableOptions
	hideLicense bool
	hideEnv     bool
}

func (flags *installationListFlags) addFlags(command *cobra.Command) {
	flags.installationGetRequestOptions.addFlags(command)
	flags.pagingFlags.addFlags(command)
	flags.tableOptions.addFlags(command)

	command.Flags().BoolVar(&flags.hideLicense, "hide-license", true, "Whether to hide the license value in the output or not.")
	command.Flags().BoolVar(&flags.hideEnv, "hide-env", true, "Whether to hide env vars in the output or not.")
}

func (flags *installationListFlags) deletionLockedFilterValue() *bool {
	if flags.deletionLockedChanged {
		return &flags.deletionLocked
	}

	return nil
}

type installationGetStatusesFlags struct {
	clusterFlags
}

func (flags *installationGetStatusesFlags) addFlags(command *cobra.Command) {

}

type installationRecoveryFlags struct {
	clusterFlags
	installationID string
	databaseID     string
	database       string
}

func (flags *installationRecoveryFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be recovered.")
	command.Flags().StringVar(&flags.databaseID, "installation-database", "", "The original multitenant database id of the installation to be recovered.")
	command.Flags().StringVar(&flags.database, "database", "", "The database backing the provisioning server.")

	_ = command.MarkFlagRequired("installation")
	_ = command.MarkFlagRequired("installation-database")
}

type installationDeploymentReportFlags struct {
	clusterFlags
	installationID string
	eventCount     int
}

func (flags *installationDeploymentReportFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to report on.")
	command.Flags().IntVar(&flags.eventCount, "event-count", 10, "The number of recent installation events to include in the report.")

	_ = command.MarkFlagRequired("installation")
}

type installationDeletionReportFlags struct {
	clusterFlags
	days int
}

func (flags *installationDeletionReportFlags) addFlags(command *cobra.Command) {
	command.Flags().IntVar(&flags.days, "days", 7, "The number of days include in the deletion report.")
}
