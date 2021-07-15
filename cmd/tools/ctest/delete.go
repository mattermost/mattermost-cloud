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

func init() {
	deleteCmd.PersistentFlags().String("group", "", "Specify the group to delete installations from.")
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Generate multiple installations with a common configuration.",
	RunE: func(command *cobra.Command, args []string) error {
		return runDeleteCommand(command)
	},
}

func runDeleteCommand(command *cobra.Command) error {
	serverAddress, _ := command.Flags().GetString("server")
	group, _ := command.Flags().GetString("group")

	client := model.NewClient(serverAddress)

	installations, err := client.GetInstallations(&model.GetInstallationsRequest{
		GroupID: group,
		Paging:  model.AllPagesNotDeleted(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to get installations")
	}

	if len(installations) == 0 {
		logger.Infof("No installations in group %s", group)
		return nil
	}

	printSeparator()
	logger.Infof("Deleting %d installations", len(installations))
	printSeparator()

	for _, installation := range installations {
		logger.Debugf("%s - %s", installation.ID, installation.DNS)
	}

	logger.Info("Proceeding in 10 seconds...")

	time.Sleep(10 * time.Second)

	for _, installation := range installations {
		logger.Debugf("Deleting: %s - %s", installation.ID, installation.DNS)

		err = client.DeleteInstallation(installation.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to delete installation %s", installation.ID)
		}

		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("Installation deletion complete")

	return nil
}
