package cmd

import (
	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/iopodman"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
)

// NewRemoveCommand returns command to remove the cluster
func NewRemoveCommand() *cobra.Command {
	rm := &cobra.Command{
		Use:   "rm",
		Short: "rm deletes all traces of a cluster",
		RunE:  remove,
		Args:  cobra.NoArgs,
	}
	return rm
}

func remove(cmd *cobra.Command, _ []string) error {
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

	force := true
	removeVolumes := true

	for _, cont := range containers {
		_, err := iopodman.RemoveContainer().Call(conn, cont, force, removeVolumes)
		if err != nil {
			return err
		}
	}

	/*volumes*/
	_, err = podman.GetPrefixedVolumes(conn, prefix)
	if err != nil {
		return err
	}

	// TODO
	/*
		for _, v := range volumes {
			err := cli.VolumeRemove(context.Background(), v.Name, true)
			if err != nil {
				return err
			}
		}
	*/
	return nil
}
