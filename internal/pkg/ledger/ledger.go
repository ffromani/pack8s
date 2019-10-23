package ledger

import (
	"context"
	"fmt"
	"io"

	"github.com/varlink/go/varlink"

	"github.com/fromanirh/pack8s/iopodman"
)

type Ledger struct {
	conn       *varlink.Connection
	ctx        context.Context
	containers chan string
	volumes    chan string
	Done       chan error
}

func NewLedger(ctx context.Context, conn *varlink.Connection, errWriter io.Writer) Ledger {
	containers := make(chan string)
	volumes := make(chan string)
	done := make(chan error)

	go func() {
		createdContainers := []string{}
		createdVolumes := []string{}

		for {
			select {
			case container := <-containers:
				createdContainers = append(createdContainers, container)
			case volume := <-volumes:
				createdVolumes = append(createdVolumes, volume)
			case err := <-done:
				if err != nil {
					for _, c := range createdContainers {
						name, err := iopodman.RemoveContainer().Call(ctx, conn, c, true, false)
						if err == nil {
							fmt.Printf("removed container: %v (%v)\n", name, c)
						} else {
							fmt.Fprintf(errWriter, "error removing container %v: %v\n", c, err)
						}
					}

					for _, v := range createdVolumes {
						//err := conn.VolumeRemove(ctx, v, true)
						fmt.Printf("volume: %v - can't do it yet", v)
						if err != nil {
							fmt.Fprintf(errWriter, "%v\n", err)
						}
					}
				}
			}
		}
	}()

	return Ledger{
		conn:       conn,
		ctx:        ctx,
		containers: containers,
		volumes:    volumes,
		Done:       done,
	}
}

func (ld Ledger) MakeVolume(name string) (string, error) {
	volName, err := iopodman.VolumeCreate().Call(ld.ctx, ld.conn, iopodman.VolumeCreateOpts{
		VolumeName: name,
	})
	if err != nil {
		return volName, err
	}

	ld.volumes <- volName
	return volName, err
}

func (ld Ledger) RunContainer(conf iopodman.Create) (string, error) {
	contID, err := iopodman.CreateContainer().Call(ld.ctx, ld.conn, conf)
	if err != nil {
		return contID, err
	}

	ld.containers <- contID
	if _, err := iopodman.StartContainer().Call(ld.ctx, ld.conn, contID); err != nil {
		return contID, err
	}
	return contID, nil
}
