// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"os"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Manipulate groups managed by the provisioning server.",
	}

	setClusterFlags(cmd)

	cmd.AddCommand(newCmdGroupCreate())
	cmd.AddCommand(newCmdGroupUpdate())
	cmd.AddCommand(newCmdGroupDelete())
	cmd.AddCommand(newCmdGroupGet())
	cmd.AddCommand(newCmdGroupList())
	cmd.AddCommand(newCmdGroupGetStatus())
	cmd.AddCommand(newCmdGroupJoin())
	cmd.AddCommand(newCmdGroupAssign())
	cmd.AddCommand(newCmdGroupLeave())
	cmd.AddCommand(newCmdGroupListStatus())
	cmd.AddCommand(newCmdGroupAnnotation())

	return cmd
}

func newCmdGroupCreate() *cobra.Command {

	var flags groupCreateFlags

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a group.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			envVarMap, err := parseEnvVarInput(flags.mattermostEnv, false)
			if err != nil {
				return err
			}

			request := &model.CreateGroupRequest{
				Name:          flags.name,
				MaxRolling:    flags.maxRolling,
				Description:   flags.description,
				Version:       flags.version,
				Image:         flags.image,
				MattermostEnv: envVarMap,
				Annotations:   flags.annotations,
			}

			if flags.dryRun {
				return runDryRun(request)
			}

			group, err := client.CreateGroup(request)
			if err != nil {
				return errors.Wrap(err, "failed to create group")
			}

			return printJSON(group)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdGroupUpdate() *cobra.Command {
	var flags groupUpdateFlags

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the group metadata.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeGroupUpdateCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			flags.groupUpgradeFlagChanged.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeGroupUpdateCmd(flags groupUpdateFlags) error {
	client := model.NewClient(flags.serverAddress)

	envVarMap, err := parseEnvVarInput(flags.mattermostEnv, flags.mattermostEnvClear)
	if err != nil {
		return errors.Wrap(err, "failed to parse env var input")
	}

	request := &model.PatchGroupRequest{
		ID:                        flags.groupID,
		MattermostEnv:             envVarMap,
		ForceSequenceUpdate:       flags.forceSequenceUpdate,
		ForceInstallationsRestart: flags.forceInstallationRestart,
	}

	if flags.isNameChanged {
		request.Name = &flags.name
	}
	if flags.isDescriptionChanged {
		request.Description = &flags.description
	}
	if flags.isMaxRollingChanged {
		request.MaxRolling = &flags.maxRolling
	}
	if flags.isVersionChanged {
		request.Version = &flags.version
	}
	if flags.isImageChanged {
		request.Image = &flags.image
	}

	if flags.dryRun {
		return runDryRun(request)
	}

	group, err := client.UpdateGroup(request)
	if err != nil {
		return errors.Wrap(err, "failed to update group")
	}

	return printJSON(group)
}

func newCmdGroupDelete() *cobra.Command {
	var flags groupDeleteFlags

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a group.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.DeleteGroup(flags.groupID); err != nil {
				return errors.Wrap(err, "failed to delete group")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdGroupGet() *cobra.Command {
	var flags groupGetFlags

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a particular group.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			group, err := client.GetGroup(flags.groupID)
			if err != nil {
				return errors.Wrap(err, "failed to query group")
			}
			if group == nil {
				return nil
			}

			return printJSON(group)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdGroupList() *cobra.Command {
	var flags groupListFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List created groups.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeGroupListCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeGroupListCmd(flags groupListFlags) error {
	client := model.NewClient(flags.serverAddress)

	paging := getPaging(flags.pagingFlags)
	groups, err := client.GetGroups(&model.GetGroupsRequest{
		Paging:                paging,
		WithInstallationCount: flags.withInstallationCount,
	})
	if err != nil {
		return errors.Wrap(err, "failed to query groups")
	}
	if flags.outputToTable {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetHeader([]string{"ID", "NAME", "SEQ", "ROL", "IMAGE", "VERSION", "ENV?"})

		for _, group := range groups {
			hasEnv := "no"
			if len(group.MattermostEnv) > 0 {
				hasEnv = "yes"
			}
			table.Append([]string{group.ID, group.Name, fmt.Sprintf("%d", group.Sequence), fmt.Sprintf("%d", group.MaxRolling), group.Image, group.Version, hasEnv})
		}
		table.Render()
		return nil
	}

	return printJSON(groups)
}

func newCmdGroupGetStatus() *cobra.Command {
	var flags groupGetStatusFlags

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get a particular group's status.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			groupStatus, err := client.GetGroupStatus(flags.groupID)
			if err != nil {
				return errors.Wrap(err, "failed to query group status")
			}
			if groupStatus == nil {
				return nil
			}

			return printJSON(groupStatus)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdGroupJoin() *cobra.Command {
	var flags groupJoinFlags

	cmd := &cobra.Command{
		Use:   "join",
		Short: "Join an installation to the given group, leaving any existing group.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			err := client.JoinGroup(flags.groupID, flags.installationID)
			if err != nil {
				return errors.Wrap(err, "failed to join group")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdGroupAssign() *cobra.Command {
	var flags groupAssignFlags

	cmd := &cobra.Command{
		Use:   "assign",
		Short: "Assign an installation to the group based on annotations, leaving any existing group.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			err := client.AssignGroup(flags.installationID, model.AssignInstallationGroupRequest{GroupSelectionAnnotations: flags.annotations})
			if err != nil {
				return errors.Wrap(err, "failed to assign group")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdGroupLeave() *cobra.Command {
	var flags groupLeaveFlags

	cmd := &cobra.Command{
		Use:   "leave",
		Short: "Remove an installation from its group, if any.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)
			request := &model.LeaveGroupRequest{RetainConfig: flags.retainConfig}
			if err := client.LeaveGroup(flags.installationID, request); err != nil {
				return errors.Wrap(err, "failed to leave group")
			}

			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdGroupListStatus() *cobra.Command {
	var flags clusterFlags

	cmd := &cobra.Command{
		Use:   "statuses",
		Short: "Get Status from all groups.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			groupStatus, err := client.GetGroupsStatus()
			if err != nil {
				return errors.Wrap(err, "failed to query group status")
			}
			if groupStatus == nil {
				return nil
			}

			return printJSON(groupStatus)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
		},
	}

	return cmd
}
