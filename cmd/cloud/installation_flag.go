// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"time"

	"github.com/mattermost/mattermost-cloud/model"
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
	command.Flags().StringVar(&flags.affinity, "affinity", model.InstallationAffinityIsolated, "How other installations may be co-located in the same cluster.")
	command.Flags().StringVar(&flags.database, "database", model.DefaultDatabaseEngine, "The Mattermost server database type. Accepts aws-rds, aws-rds-postgres, aws-multitenant-rds, or aws-multitenant-rds-postgres")
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
	ownerIDChanged bool
	versionCHanged bool
	imageChanged   bool
	sizeChanged    bool
	licenseChanged bool
}

func (flags *installationPatchRequestChanges) addFlags(command *cobra.Command) {
	flags.ownerIDChanged = command.Flags().Changed("owner")
	flags.versionCHanged = command.Flags().Changed("version")
	flags.imageChanged = command.Flags().Changed("image")
	flags.sizeChanged = command.Flags().Changed("size")
	flags.licenseChanged = command.Flags().Changed("license")
}

type installationPatchRequestOptions struct {
	installationPatchRequestChanges
	ownerID            string
	version            string
	image              string
	size               string
	license            string
	mattermostEnv      []string
	mattermostEnvClear bool
}

func (flags *installationPatchRequestOptions) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.ownerID, "owner", "", "The new owner value of this installation.")
	command.Flags().StringVar(&flags.version, "version", "stable", "The Mattermost version to target.")
	command.Flags().StringVar(&flags.image, "image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	command.Flags().StringVar(&flags.size, "size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	command.Flags().StringVar(&flags.license, "license", "", "The Mattermost License to use in the server.")
	command.Flags().StringArrayVar(&flags.mattermostEnv, "mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	command.Flags().BoolVar(&flags.mattermostEnvClear, "mattermost-env-clear", false, "Clears all env var data.")
}

func (flags *installationPatchRequestOptions) GetPatchInstallationRequest() *model.PatchInstallationRequest {
	request := model.PatchInstallationRequest{}

	if flags.ownerIDChanged {
		request.OwnerID = &flags.ownerID
	}

	if flags.versionCHanged {
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
	includeGroupConfig          bool
	includeGroupConfigOverrides bool
}

func (flags *installationGetRequestOptions) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.owner, "owner", "", "The owner ID to filter installations by.")
	command.Flags().StringVar(&flags.group, "group", "", "The group ID to filter installations.")
	command.Flags().StringVar(&flags.state, "state", "", "The state to filter installations by.")
	command.Flags().StringVar(&flags.dns, "dns", "", "The dns name to filter installations by.")
	command.Flags().BoolVar(&flags.includeGroupConfig, "include-group-config", true, "Whether to include group configuration in the installations or not.")
	command.Flags().BoolVar(&flags.includeGroupConfigOverrides, "include-group-config-overrides", true, "Whether to include a group configuration override summary in the installations or not.")

}

type installationListFlags struct {
	clusterFlags
	installationGetRequestOptions
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
	command.Flags().StringVar(&flags.database, "database", "sqlite://cloud.db", "The database backing the provisioning server.")

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
