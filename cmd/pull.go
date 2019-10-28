package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
)

func NewPullCommand() *cobra.Command {
	show := &cobra.Command{
		Use:   "pull",
		Short: "pull downloads an image from a registry",
		RunE:  pullImage,
		Args:  cobra.ExactArgs(1),
	}
	return show
}

func pullImage(cmd *cobra.Command, args []string) error {
	podmanSocket, err := cmd.Flags().GetString("podman-socket")
	if err != nil {
		return err
	}

	ctx := context.Background()
	hnd, err := podman.NewHandle(ctx, podmanSocket)
	if err != nil {
		return err
	}

	ref := args[0]
	log.Printf("pulling image: %s", ref)
	return hnd.PullImage(ref)
}
