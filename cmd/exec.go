package cmd

import (
	"context"
	"os"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
	"github.com/spf13/cobra"
)

type execOptions struct {
	commands []string
}

var execOpt execOptions

// NewExecCommand runs given command inside container
func NewExecCommand() *cobra.Command {
	exec := &cobra.Command{
		Use:   "exec",
		Short: "exec runs given command in container",
		RunE:  execCommand,
		Args:  cobra.MinimumNArgs(2),
	}

	return exec
}

func execCommand(cmd *cobra.Command, args []string) error {
	containerID := args[0]
	command := args[1:]

	podmanSocket, err := cmd.Flags().GetString("podman-socket")
	if err != nil {
		return err
	}

	ctx := context.Background()

	hnd, err := podman.NewHandle(ctx, podmanSocket)
	if err != nil {
		return err
	}

	return hnd.Exec(containerID, command, os.Stdout)
}
