package podman

import (
	"bufio"
	"context"

	"github.com/fromanirh/pack8s/iopodman"

	"github.com/fromanirh/varlink-go/varlink"
)

// ExecContainer executes a command in the given container.
type ExecContainer_methods struct{}

func ExecContainer() ExecContainer_methods { return ExecContainer_methods{} }

func (m ExecContainer_methods) Call(ctx context.Context, c *varlink.Connection, opts_in_ iopodman.ExecOpts) (r_ *bufio.Reader, err_ error) {
	receive, err_ := m.Send(ctx, c, varlink.Upgrade, opts_in_)
	if err_ != nil {
		return
	}
	r_, err_ = receive(ctx)
	return
}

func (m ExecContainer_methods) Send(ctx context.Context, c *varlink.Connection, flags uint64, opts_in_ iopodman.ExecOpts) (func(ctx context.Context) (*bufio.Reader, error), error) {
	var in struct {
		Opts iopodman.ExecOpts `json:"opts"`
	}
	in.Opts = opts_in_
	receive, err := c.Send(ctx, "io.podman.ExecContainer", in, flags)
	if err != nil {
		return nil, err
	}
	return func(context.Context) (rd *bufio.Reader, err error) {
		_, err = receive(ctx, nil)
		if err != nil {
			err = iopodman.Dispatch_Error(err)
			return
		}
		rd = c.GetReader()
		return
	}, nil
}
