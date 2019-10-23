package cmd

import (
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
)

// NewSSHCommand returns command to SSH to the cluster node
func NewSSHCommand() *cobra.Command {

	ssh := &cobra.Command{
		Use:   "ssh",
		Short: "ssh into a node",
		RunE:  ssh,
		Args:  cobra.MinimumNArgs(1),
	}
	return ssh
}

func ssh(cmd *cobra.Command, args []string) error {
	prefix, err := cmd.Flags().GetString("prefix")
	if err != nil {
		return err
	}

	node := args[0]

	conn, err := podman.NewConnection()
	if err != nil {
		return err
	}

	container := prefix + "-" + node
	sshCommand := append([]string{"ssh.sh"}, args[1:]...)

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		err = podman.Terminal(conn, container, sshCommand, os.Stdout)
	} else {
		err = podman.Exec(conn, container, sshCommand, os.Stdout)
	}
	return err
}
