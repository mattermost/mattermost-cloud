package main

import "github.com/spf13/cobra"

type groupCreateFlags struct {
	clusterFlags
	name          string
	description   string
	version       string
	image         string
	maxRolling    int64
	mattermostEnv []string
	annotations   []string
}

func (flags *groupCreateFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.name, "name", "", "A unique name describing this group of installations.")
	command.Flags().StringVar(&flags.description, "description", "", "An optional description for this group of installations.")
	command.Flags().StringVar(&flags.version, "version", "", "The Mattermost version for installations in this group to target.")
	command.Flags().StringVar(&flags.image, "image", "", "The Mattermost container image to use.")
	command.Flags().Int64Var(&flags.maxRolling, "max-rolling", 1, "The maximum number of installations that can be updated at one time when a group is updated")
	command.Flags().StringArrayVar(&flags.mattermostEnv, "mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	command.Flags().StringArrayVar(&flags.annotations, "annotation", []string{}, "Annotations for a group used for automatic group selection. Accepts multiple values, for example: '... --annotation abc --annotation def'")

	_ = command.MarkFlagRequired("name")
}

type groupUpgradeFlagChanged struct {
	isNameChanged        bool
	isDescriptionChanged bool
	isVersionChanged     bool
	isImageChanged       bool
	isMaxRollingChanged  bool
}

func (flags *groupUpgradeFlagChanged) addFlags(command *cobra.Command) {
	flags.isNameChanged = command.Flags().Changed("name")
	flags.isDescriptionChanged = command.Flags().Changed("description")
	flags.isVersionChanged = command.Flags().Changed("version")
	flags.isImageChanged = command.Flags().Changed("image")
	flags.isMaxRollingChanged = command.Flags().Changed("max-rolling")
}

type groupUpdateFlags struct {
	clusterFlags
	groupUpgradeFlagChanged
	groupID                  string
	name                     string
	description              string
	version                  string
	image                    string
	maxRolling               int64
	mattermostEnv            []string
	mattermostEnvClear       bool
	forceSequenceUpdate      bool
	forceInstallationRestart bool
}

func (flags *groupUpdateFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to be updated.")
	command.Flags().StringVar(&flags.name, "name", "", "A unique name describing this group of installations.")
	command.Flags().StringVar(&flags.description, "description", "", "An optional description for this group of installations.")
	command.Flags().StringVar(&flags.version, "version", "", "The Mattermost version for installations in this group to target.")
	command.Flags().StringVar(&flags.image, "image", "", "The Mattermost container image to use.")
	command.Flags().Int64Var(&flags.maxRolling, "max-rolling", 0, "The maximum number of installations that can be updated at one time when a group is updated")
	command.Flags().StringArrayVar(&flags.mattermostEnv, "mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	command.Flags().BoolVar(&flags.mattermostEnvClear, "mattermost-env-clear", false, "Clear all Mattermost env vars.")
	command.Flags().BoolVar(&flags.forceSequenceUpdate, "force-sequence-update", false, "Forces the group version sequence to be increased by 1 even when no updates are present.")
	command.Flags().BoolVar(&flags.forceInstallationRestart, "force-installation-restart", false, "Forces the restart of all installations in the group even if Mattermost CR does not change.")

	_ = command.MarkFlagRequired("group")
}

type groupDeleteFlags struct {
	clusterFlags
	groupID string
}

func (flags *groupDeleteFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to be deleted.")

	_ = command.MarkFlagRequired("group")
}

type groupGetFlags struct {
	clusterFlags
	groupID string
}

func (flags *groupGetFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to be fetched.")

	_ = command.MarkFlagRequired("group")
}

type groupListFlags struct {
	clusterFlags
	pagingFlags
	outputToTable         bool
	withInstallationCount bool
}

func (flags *groupListFlags) addFlags(command *cobra.Command) {
	flags.pagingFlags.addFlags(command)

	command.Flags().BoolVar(&flags.outputToTable, "table", false, "Whether to display the returned group list in a table or not")
	command.Flags().BoolVar(&flags.withInstallationCount, "include-installation-count", false, "Whether to retrieve the installation count for the groups")
}

type groupGetStatusFlags struct {
	clusterFlags
	groupID string
}

func (flags *groupGetStatusFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.groupID, "group", "", "The id of the group of which the status should be fetched.")

	_ = command.MarkFlagRequired("group")
}

type groupJoinFlags struct {
	clusterFlags
	groupID        string
	installationID string
}

func (flags *groupJoinFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to which the installation will be added.")
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to join the group.")

	_ = command.MarkFlagRequired("group")
	_ = command.MarkFlagRequired("installation")
}

type groupAssignFlags struct {
	clusterFlags
	installationID string
	annotations    []string
}

func (flags *groupAssignFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to assign to the group.")
	command.Flags().StringArrayVar(&flags.annotations, "group-selection-annotation", []string{}, "Group annotations based on which the installation should be assigned.")

	_ = command.MarkFlagRequired("installation")
	_ = command.MarkFlagRequired("group-selection-annotation")
}

type groupLeaveFlags struct {
	clusterFlags
	installationID string
	retainConfig   bool
}

func (flags *groupLeaveFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to leave its currently configured group.")
	command.Flags().BoolVar(&flags.retainConfig, "retain-config", true, "Whether to retain the group configuration values or not.")

	_ = command.MarkFlagRequired("installation")
}
