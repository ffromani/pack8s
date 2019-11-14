package cmd

import (
	"context"
	"log"
	"os"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
	"github.com/spf13/cobra"
)

type copyFromOptions struct {
	source      string
	destination string
}

var copyOpts copyFromOptions

// NewCopyFromCommand returns new command to copy files from container to host
func NewCopyFromCommand() *cobra.Command {
	copy := &cobra.Command{
		Use:   "copy-from",
		Short: "copy-from copies file from container to host",
		Long:  "copy-from copies file from container to host",
		RunE:  copyFrom,
		Args:  cobra.ExactArgs(1),
	}

	copy.Flags().StringVarP(&copyOpts.source, "source", "x", "", "location of file to copy")
	copy.Flags().StringVarP(&copyOpts.destination, "destination", "w", "", "destination of file to copy")
	return copy
}

func copyFrom(cmd *cobra.Command, args []string) error {
	containerID := args[0]

	podmanSocket, err := cmd.Flags().GetString("podman-socket")
	if err != nil {
		return err
	}

	if copyOpts.source == "" {
		log.Println("source can't be empty")
		return nil
	}

	if copyOpts.destination == "" {
		copyOpts.destination = copyOpts.source
		log.Println("source can't be empty")
		return nil
	}

	ctx := context.Background()

	hnd, err := podman.NewHandle(ctx, podmanSocket)
	if err != nil {
		return err
	}
	copyCommand := []string{"/bin/cat", copyOpts.source}

	f, err := os.Create(copyOpts.destination)
	if err != nil {
		return err
	}

	return hnd.Exec(containerID, copyCommand, f)
}
