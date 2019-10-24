package cmd

import (
	"context"
	"log"
	"sort"

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

type containerList []iopodman.Container

func (cl containerList) Len() int {
	return len(cl)
}

func (cl containerList) Less(i, j int) bool {
	contA := cl[i]
	contB := cl[j]
	genA := contA.Labels[podman.LabelGeneration]
	genB := contB.Labels[podman.LabelGeneration]
	if genA != "" && genB != "" {
		// CAVEAT! we want the latest generation first, so we swap the condition
		return genA > genB
	}
	return false // do not change the ordering
}

func (cl containerList) Swap(i, j int) {
	cl[i], cl[j] = cl[j], cl[i]
}

func remove(cmd *cobra.Command, _ []string) error {
	prefix, err := cmd.Flags().GetString("prefix")
	if err != nil {
		return err
	}

	ctx := context.Background()

	hnd, err := podman.NewHandle(ctx)
	if err != nil {
		return err
	}

	containers, err := hnd.GetPrefixedContainers(prefix)
	if err != nil {
		return err
	}

	sort.Sort(containerList(containers))

	for _, cont := range containers {
		log.Printf("container to remove: %s (%s)\n", cont.Names, cont.Id)
	}

	force := true
	removeVolumes := true

	for _, cont := range containers {
		_, err := hnd.RemoveContainer(cont.Id, force, removeVolumes)
		if err != nil {
			return err
		}
	}

	// TODO: needed?
	/*
		_, err = podman.GetPrefixedVolumes(hnd, prefix)
		if err != nil {
			return err
		}

			for _, v := range volumes {
				err := cli.VolumeRemove(context.Background(), v.Name, true)
				if err != nil {
					return err
				}
			}
	*/
	return nil
}
