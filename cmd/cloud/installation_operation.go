// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import "github.com/spf13/cobra"

func init() {
	installationOperationCmd.AddCommand(installationRestorationOperationCmd)
	installationOperationCmd.AddCommand(installationDBMigrationOperationCmd)
}

var installationOperationCmd = &cobra.Command{
	Use:   "operation",
	Short: "Manipulate installation operations managed by the provisioning server.",
}
