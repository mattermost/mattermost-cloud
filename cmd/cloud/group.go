package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	groupCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")

	groupCreateCmd.Flags().String("name", "", "A unique name describing this group of installations.")
	groupCreateCmd.Flags().String("description", "", "An optional description for this group of installations.")
	groupCreateCmd.Flags().String("version", "stable", "The Mattermost version for installations in this group to target.")
	groupCreateCmd.Flags().StringArray("mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	groupCreateCmd.MarkFlagRequired("name")

	groupUpdateCmd.Flags().String("group", "", "The id of the group to be updated.")
	groupUpdateCmd.Flags().String("name", "", "A unique name describing this group of installations.")
	groupUpdateCmd.Flags().String("description", "", "An optional description for this group of installations.")
	groupUpdateCmd.Flags().String("version", "", "The Mattermost version for installations in this group to target.")
	groupUpdateCmd.Flags().StringArray("mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	groupUpdateCmd.MarkFlagRequired("group")

	groupDeleteCmd.Flags().String("group", "", "The id of the group to be deleted.")
	groupDeleteCmd.MarkFlagRequired("group")

	groupGetCmd.Flags().String("group", "", "The id of the group to be fetched.")
	groupGetCmd.MarkFlagRequired("group")

	groupListCmd.Flags().Int("page", 0, "The page of groups to fetch, starting at 0.")
	groupListCmd.Flags().Int("per-page", 100, "The number of groups to fetch per page.")
	groupListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted groups.")

	groupJoinCmd.Flags().String("group", "", "The id of the group to which the installation will be added.")
	groupJoinCmd.Flags().String("installation", "", "The id of the installation to add to the group.")
	groupJoinCmd.MarkFlagRequired("group")
	groupJoinCmd.MarkFlagRequired("installation")

	groupLeaveCmd.Flags().String("installation", "", "The id of the installation to leave its currently configured group.")
	groupLeaveCmd.MarkFlagRequired("installation")

	groupCmd.AddCommand(groupCreateCmd)
	groupCmd.AddCommand(groupUpdateCmd)
	groupCmd.AddCommand(groupDeleteCmd)
	groupCmd.AddCommand(groupGetCmd)
	groupCmd.AddCommand(groupListCmd)
	groupCmd.AddCommand(groupJoinCmd)
	groupCmd.AddCommand(groupLeaveCmd)
}

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manipulate groups managed by the provisioning server.",
}

var groupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a group.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		name, _ := command.Flags().GetString("name")
		description, _ := command.Flags().GetString("description")
		version, _ := command.Flags().GetString("version")
		mattermostEnv, _ := command.Flags().GetStringArray("mattermost-env")

		envVarMap, err := parseEnvVarInput(mattermostEnv)
		if err != nil {
			return err
		}

		group, err := client.CreateGroup(&model.CreateGroupRequest{
			Name:          name,
			Description:   description,
			Version:       version,
			MattermostEnv: envVarMap,
		})

		if err != nil {
			return errors.Wrap(err, "failed to create group")
		}

		err = printJSON(group)
		if err != nil {
			return err
		}

		return nil
	},
}

var groupUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the group metadata.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		groupID, _ := command.Flags().GetString("group")
		name, _ := command.Flags().GetString("name")
		description, _ := command.Flags().GetString("description")
		version, _ := command.Flags().GetString("version")
		mattermostEnv, _ := command.Flags().GetStringArray("mattermost-env")

		envVarMap, err := parseEnvVarInput(mattermostEnv)
		if err != nil {
			return err
		}

		patchPointer := func(s string) *string {
			if s == "" {
				return nil
			}

			return &s
		}

		err = client.UpdateGroup(&model.PatchGroupRequest{
			ID:            groupID,
			Name:          patchPointer(name),
			Description:   patchPointer(description),
			Version:       patchPointer(version),
			MattermostEnv: envVarMap,
		})
		if err != nil {
			return errors.Wrap(err, "failed to update group")
		}

		return nil
	},
}

var groupDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a group.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		groupID, _ := command.Flags().GetString("group")

		err := client.DeleteGroup(groupID)
		if err != nil {
			return errors.Wrap(err, "failed to delete group")
		}

		return nil
	},
}

var groupGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular group.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		groupID, _ := command.Flags().GetString("group")
		group, err := client.GetGroup(groupID)
		if err != nil {
			return errors.Wrap(err, "failed to query group")
		}
		if group == nil {
			return nil
		}

		err = printJSON(group)
		if err != nil {
			return err
		}

		return nil
	},
}

var groupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List created groups.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")
		groups, err := client.GetGroups(&model.GetGroupsRequest{
			Page:           page,
			PerPage:        perPage,
			IncludeDeleted: includeDeleted,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query groups")
		}

		err = printJSON(groups)
		if err != nil {
			return err
		}

		return nil
	},
}

var groupJoinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join an installation to the given group, leaving any existing group.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		groupID, _ := command.Flags().GetString("group")
		installationID, _ := command.Flags().GetString("installation")

		err := client.JoinGroup(groupID, installationID)
		if err != nil {
			return errors.Wrap(err, "failed to join group")
		}

		return nil
	},
}

var groupLeaveCmd = &cobra.Command{
	Use:   "leave",
	Short: "Remove an installation from its group, if any.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")

		err := client.LeaveGroup(installationID)
		if err != nil {
			return errors.Wrap(err, "failed to leave group")
		}

		return nil
	},
}
