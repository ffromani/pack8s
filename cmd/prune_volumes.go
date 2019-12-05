package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/cmd/cmdutil"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
)

// NewPruneVolumesCommand returns command to prune unused volumes
func NewPruneVolumesCommand() *cobra.Command {
	prune := &cobra.Command{
		Use:   "prune",
		Short: "prune removes unused volumes on the host",
		RunE:  pruneVolumes,
		Args:  cobra.NoArgs,
	}
	return prune
}

func pruneVolumes(cmd *cobra.Command, _ []string) error {
	cOpts, err := cmdutil.GetCommonOpts(cmd)
	if err != nil {
		return err
	}

	ctx := context.Background()
	hnd, err := podman.NewHandle(ctx, cOpts.PodmanSocket, cmdutil.NewLogger(cOpts.Verbose))
	if err != nil {
		return err
	}

	return hnd.PruneVolumes()
}
