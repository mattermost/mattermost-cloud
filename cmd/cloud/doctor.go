// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
)

func newCmdDoctor() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Helpful tools.",
	}

	setSchemaFlags(cmd)

	cmd.AddCommand(newCmdDoctorCreateID())

	return cmd
}

func newCmdDoctorCreateID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-id",
		Short: "Creates a Z-Base-32 ID",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			command.Print(model.NewID())
			return nil
		},
	}

	return cmd

}
