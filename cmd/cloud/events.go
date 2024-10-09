// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdEvents() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Groups event commands managed by the provisioning server.",
	}

	setEventFlags(cmd)

	cmd.AddCommand(newCmdStateChangeEvents())

	return cmd
}

func newCmdStateChangeEvents() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "state-change",
		Short: "Groups state change event commands managed by the provisioning server.",
	}

	cmd.AddCommand(newCmdStateChangeEventList())

	return cmd
}

func newCmdStateChangeEventList() *cobra.Command {
	var flags stateChangeEventListFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Fetch state change events managed by the provisioning server.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeStateChangeEventListCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.eventFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeStateChangeEventListCmd(ctx context.Context, flags stateChangeEventListFlags) error {
	client := createClient(ctx, flags.clusterFlags)

	paging := getPaging(flags.pagingFlags)

	req := model.ListStateChangeEventsRequest{
		Paging:       paging,
		ResourceType: model.ResourceType(flags.resourceType),
		ResourceID:   flags.resourceID,
	}

	events, err := client.ListStateChangeEvents(&req)
	if err != nil {
		return err
	}

	if enabled, customCols := getTableOutputOption(flags.tableOptions); enabled {
		var keys []string
		var vals [][]string

		if len(customCols) > 0 {
			data := make([]interface{}, 0, len(events))
			for _, elem := range events {
				data = append(data, elem)
			}
			keys, vals, err = prepareTableData(customCols, data)
			if err != nil {
				return errors.Wrap(err, "failed to prepare table output")
			}
		} else {
			keys, vals = defaultEventsTableData(events)
		}

		printTable(keys, vals)
		return nil
	}

	return printJSON(events)

}

func defaultEventsTableData(events []*model.StateChangeEventData) ([]string, [][]string) {
	keys := []string{"ID", "RESOURCE TYPE", "RESOURCE ID", "OLD STATE", "NEW STATE", "TIMESTAMP"}
	vals := make([][]string, 0, len(events))

	for _, event := range events {
		vals = append(vals, []string{
			event.Event.ID,
			event.StateChange.ResourceType.String(),
			event.StateChange.ResourceID,
			event.StateChange.OldState,
			event.StateChange.NewState,
			model.TimeFromMillis(event.Event.Timestamp).Format("2006-01-02 15:04:05 -0700 MST"),
		})
	}

	return keys, vals
}
