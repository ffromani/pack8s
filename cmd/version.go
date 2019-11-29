package cmd

import (
	"fmt"

	"github.com/fromanirh/pack8s/internal/pkg/version"
	"github.com/spf13/cobra"
)

// NewVersionCommand runs given command inside container
func NewVersionCommand() *cobra.Command {
	exec := &cobra.Command{
		Use:   "version",
		Short: "dump the version and exits",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Printf("pack8s %s %s\n", version.VERSION, version.REVISION)
			return nil
		},
		Args: cobra.NoArgs,
	}

	return exec
}
