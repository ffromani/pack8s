package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
)

func NewShowCommand() *cobra.Command {
	show := &cobra.Command{
		Use:   "show",
		Short: "show lists containers belonging to the cluster",
		RunE:  showContainers,
		Args:  cobra.NoArgs,
	}
	return show
}

func showContainers(cmd *cobra.Command, args []string) error {

	prefix, err := cmd.Flags().GetString("prefix")
	if err != nil {
		return err
	}

	conn, err := podman.NewConnection()
	if err != nil {
		return err
	}

	containers, err := podman.GetPrefixedContainers(conn, prefix+"-")
	if err != nil {
		return err
	}

	for _, cont := range containers {
		fmt.Printf("%32s\t%s\n", cont.Id, cont.Names)
	}

	return nil
}
