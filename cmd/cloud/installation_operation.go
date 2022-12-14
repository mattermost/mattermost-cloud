// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import "github.com/spf13/cobra"

func newCmdInstallationOperation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operation",
		Short: "Manipulate installation operations managed by the provisioning server.",
	}

	cmd.AddCommand(newCmdInstallationRestorationOperation())
	cmd.AddCommand(newCmdInstallationDBMigrationOperation())

	return cmd
}
