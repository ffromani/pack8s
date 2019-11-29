package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/cmd/cmdutil"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
)

type pullOptions struct {
	auxImages bool
}

func (po pullOptions) WantsNFS() bool {
	return po.auxImages
}

func (po pullOptions) WantsCeph() bool {
	return po.auxImages
}

func (po pullOptions) WantsFluentd() bool {
	return po.auxImages
}

var pullOpts pullOptions

func NewPullCommand() *cobra.Command {
	show := &cobra.Command{
		Use:   "pull",
		Short: "pull downloads an image from a registry",
		RunE:  pullImage,
		Args:  cobra.ExactArgs(1),
	}
	show.Flags().BoolVarP(&pullOpts.auxImages, "aux-images", "a", false, "pull the cluster auxiliary images")
	return show
}

func pullImage(cmd *cobra.Command, args []string) error {
	cOpts, err := cmdutil.GetCommonOpts(cmd)
	if err != nil {
		return err
	}

	ctx := context.Background()

	hnd, err := podman.NewHandle(ctx, cOpts.PodmanSocket, cmdutil.NewLogger(cOpts.Verbose, 0))
	if err != nil {
		return err
	}

	ref := args[0]
	if pullOpts.auxImages {
		// if we always do PullClusterImages, we bring the docker registry, which is something
		// we may actually don't want to do here (wasted work)
		return hnd.PullClusterImages(pullOpts, ref)
	}
	return hnd.PullImage(ref)
}
