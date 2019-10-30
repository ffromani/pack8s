package podman_test

import (
	"context"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("podman", func() {
	Context("create new handler", func() {
		It("Should create with default socket without any error", func() {
			ctx := context.Background()

			_, err := podman.NewHandle(ctx, "")
			Expect(err).To(BeNil())
		})

		It("Should create with user defined socket without any error", func() {
			ctx := context.Background()

			_, err := podman.NewHandle(ctx, "unix:/run/podman/io.podman")
			Expect(err).To(BeNil())
		})
	})

})
