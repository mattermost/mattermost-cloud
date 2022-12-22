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

func (flags *groupCreateFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.name, "name", "", "A unique name describing this group of installations.")
	cmd.Flags().StringVar(&flags.description, "description", "", "An optional description for this group of installations.")
	cmd.Flags().StringVar(&flags.version, "version", "", "The Mattermost version for installations in this group to target.")
	cmd.Flags().StringVar(&flags.image, "image", "", "The Mattermost container image to use.")
	cmd.Flags().Int64Var(&flags.maxRolling, "max-rolling", 1, "The maximum number of installations that can be updated at one time when a group is updated")
	cmd.Flags().StringArrayVar(&flags.mattermostEnv, "mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	cmd.Flags().StringArrayVar(&flags.annotations, "annotation", []string{}, "Annotations for a group used for automatic group selection. Accepts multiple values, for example: '... --annotation abc --annotation def'")

	_ = cmd.MarkFlagRequired("name")
}

type groupUpgradeFlagChanged struct {
	isNameChanged        bool
	isDescriptionChanged bool
	isVersionChanged     bool
	isImageChanged       bool
	isMaxRollingChanged  bool
}

func (flags *groupUpgradeFlagChanged) addFlags(cmd *cobra.Command) {
	flags.isNameChanged = cmd.Flags().Changed("name")
	flags.isDescriptionChanged = cmd.Flags().Changed("description")
	flags.isVersionChanged = cmd.Flags().Changed("version")
	flags.isImageChanged = cmd.Flags().Changed("image")
	flags.isMaxRollingChanged = cmd.Flags().Changed("max-rolling")
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

func (flags *groupUpdateFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to be updated.")
	cmd.Flags().StringVar(&flags.name, "name", "", "A unique name describing this group of installations.")
	cmd.Flags().StringVar(&flags.description, "description", "", "An optional description for this group of installations.")
	cmd.Flags().StringVar(&flags.version, "version", "", "The Mattermost version for installations in this group to target.")
	cmd.Flags().StringVar(&flags.image, "image", "", "The Mattermost container image to use.")
	cmd.Flags().Int64Var(&flags.maxRolling, "max-rolling", 0, "The maximum number of installations that can be updated at one time when a group is updated")
	cmd.Flags().StringArrayVar(&flags.mattermostEnv, "mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	cmd.Flags().BoolVar(&flags.mattermostEnvClear, "mattermost-env-clear", false, "Clears all env var data.")
	cmd.Flags().BoolVar(&flags.forceSequenceUpdate, "force-sequence-update", false, "Forces the group version sequence to be increased by 1 even when no updates are present.")
	cmd.Flags().BoolVar(&flags.forceInstallationRestart, "force-installation-restart", false, "Forces the restart of all installations in the group even if Mattermost CR does not change.")

	_ = cmd.MarkFlagRequired("group")
}

type groupDeleteFlags struct {
	clusterFlags
	groupID string
}

func (flags *groupDeleteFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to be deleted.")

	_ = cmd.MarkFlagRequired("group")
}

type groupGetFlags struct {
	clusterFlags
	groupID string
}

func (flags *groupGetFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to be fetched.")

	_ = cmd.MarkFlagRequired("group")
}

type groupListFlags struct {
	clusterFlags
	pagingFlags
	outputToTable         bool
	withInstallationCount bool
}

func (flags *groupListFlags) addFlags(cmd *cobra.Command) {
	flags.pagingFlags.addFlags(cmd)

	cmd.Flags().BoolVar(&flags.outputToTable, "table", false, "Whether to display the returned group list in a table or not")
	cmd.Flags().BoolVar(&flags.withInstallationCount, "include-installation-count", false, "Whether to retrieve the installation count for the groups")
}

type groupGetStatusFlags struct {
	clusterFlags
	groupID string
}

func (flags *groupGetStatusFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.groupID, "group", "", "The id of the group of which the status should be fetched.")

	_ = cmd.MarkFlagRequired("group")
}

type groupJoinFlags struct {
	clusterFlags
	groupID        string
	installationID string
}

func (flags *groupJoinFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to which the installation will be added.")
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to add to the group.")

	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("installation")
}

type groupAssignFlags struct {
	clusterFlags
	installationID string
	annotations    []string
}

func (flags *groupAssignFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to assign to the group.")
	cmd.Flags().StringArrayVar(&flags.annotations, "group-selection-annotation", []string{}, "Group annotations based on which the installation should be assigned.")

	_ = cmd.MarkFlagRequired("installation")
	_ = cmd.MarkFlagRequired("group-selection-annotation")
}

type groupLeaveFlags struct {
	clusterFlags
	installationID string
	retainConfig   bool
}

func (flags *groupLeaveFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to leave its currently configured group.")
	cmd.Flags().BoolVar(&flags.retainConfig, "retain-config", true, "Whether to retain the group configuration values or not.")

	_ = cmd.MarkFlagRequired("installation")
}
