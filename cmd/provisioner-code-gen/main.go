// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "provisioner-code-gen",
		Short: "Code generator CLI for Provisioner",
		// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}

	rootCmd.PersistentFlags().String("out-file", "", "Generated code output file")
	rootCmd.PersistentFlags().Bool("stdout", true, "Instructs generator to print generated code to standard output.")
	rootCmd.PersistentFlags().String("boilerplate-file", "", "Path to file containing boilerplate to include in generated code.")

	rootCmd.AddCommand(newGenerateCmd())

	cobra.OnInitialize(func() {
		bindFlags(rootCmd)

		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		viper.SetEnvPrefix("PROVISIONER_CODE_GEN")
		viper.AutomaticEnv()
	})

	return rootCmd
}

// Binds all flags as viper values
func bindFlags(cmd *cobra.Command) {
	viper.BindPFlags(cmd.PersistentFlags())
	viper.BindPFlags(cmd.Flags())
	for _, c := range cmd.Commands() {
		bindFlags(c)
	}
}

func main() {
	rootCmd := newRootCmd()

	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Error("command failed")
		os.Exit(1)
	}
}
