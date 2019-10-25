package cmd

import (
	"context"
	"log"
	"sort"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/iopodman"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
)

type rmOptions struct {
	prune bool
}

var rmOpts rmOptions

// NewRemoveCommand returns command to remove the cluster
func NewRemoveCommand() *cobra.Command {
	rm := &cobra.Command{
		Use:   "rm",
		Short: "rm deletes all traces of a cluster",
		RunE:  remove,
		Args:  cobra.NoArgs,
	}

	rm.Flags().BoolVarP(&rmOpts.prune, "prune", "p", false, "prune removes unused volumes on the host")

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

	podmanSocket, err := cmd.Flags().GetString("podman-socket")
	if err != nil {
		return err
	}

	ctx := context.Background()

	hnd, err := podman.NewHandle(ctx, podmanSocket)
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
		_, err = hnd.StopContainer(cont.Id, 5) // TODO
		if err != nil {
			return err
		}
		_, err = hnd.RemoveContainer(cont, force, removeVolumes)
		if err != nil {
			return err
		}
	}

	volumes, err := hnd.GetPrefixedVolumes(prefix)
	if err != nil {
		return err
	}

	if len(volumes) > 0 {
		err = hnd.RemoveVolumes(volumes)
		if err != nil {
			return err
		}
	}

	if rmOpts.prune {
		err = hnd.PruneVolumes()
	}
	return err
}
