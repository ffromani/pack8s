package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/cmd/cmdutil"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
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

	cOpts, err := cmdutil.GetCommonOpts(cmd)
	if err != nil {
		return err
	}

	ctx := context.Background()

	hnd, err := podman.NewHandle(ctx, cOpts.PodmanSocket, cmdutil.NewLogger(cOpts.Verbose, 0))
	if err != nil {
		return err
	}

	return hnd.Exec(containerID, command, os.Stdout)
}
