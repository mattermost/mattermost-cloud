// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	eventsCmd.AddCommand(stateChangeEventsCmd)
	eventsCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")

	listStateChangeEventsCmd.Flags().String("resource-type", "", "Type of a resource for which to list events.")
	listStateChangeEventsCmd.Flags().String("resource-id", "", "ID of a resource for which to list events.")
	registerPagingFlags(listStateChangeEventsCmd)
	registerTableOutputFlags(listStateChangeEventsCmd)

	stateChangeEventsCmd.AddCommand(listStateChangeEventsCmd)
}

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Groups event commands managed by the provisioning server.",
}

var stateChangeEventsCmd = &cobra.Command{
	Use:   "state-change",
	Short: "Groups state change event commands managed by the provisioning server.",
}

var listStateChangeEventsCmd = &cobra.Command{
	Use:   "list",
	Short: "Fetch state change events managed by the provisioning server.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		resourceType, _ := command.Flags().GetString("resource-type")
		resourceID, _ := command.Flags().GetString("resource-id")
		paging := parsePagingFlags(command)

		req := model.ListStateChangeEventsRequest{
			Paging:       paging,
			ResourceType: model.ResourceType(resourceType),
			ResourceID:   resourceID,
		}

		events, err := client.ListStateChangeEvents(&req)
		if err != nil {
			return err
		}

		if enabled, customCols := tableOutputEnabled(command); enabled {
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
	},
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
