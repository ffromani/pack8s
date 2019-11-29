package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
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
	root.PersistentFlags().StringP("podman-socket", "s", podman.DefaultSocket, "Path to podman-socket")
	root.PersistentFlags().IntP("verbose", "v", 3, "verbosiness level [1,5)")

	root.AddCommand(
		NewPortCommand(),
		NewPullCommand(),
		NewRemoveCommand(),
		NewRunCommand(),
		NewSCPCommand(),
		NewSSHCommand(),
		NewShowCommand(),
		NewPruneVolumesCommand(),
		NewExecCommand(),
		NewVersionCommand(),
	)

	return root

}

func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Println(podman.SprintError("pack8s", err)) //XXX specific method
		os.Exit(1)
	}
}
