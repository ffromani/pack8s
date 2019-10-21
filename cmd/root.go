package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCommand returns entrypoint command to interact with all other commands
func NewRootCommand() *cobra.Command {

	root := &cobra.Command{
		Use:   "pack8s",
		Short: "pack8s helps you creating ephemeral kubernetes and openshift clusters for testing",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringP("prefix", "p", "kubevirt", "Prefix to identify containers")

	root.AddCommand(
		NewPortCommand(),
		NewRemoveCommand(),
		NewRunCommand(),
	)

	return root

}

// Execute executes root command
func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
