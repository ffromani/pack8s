package podman

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/fromanirh/pack8s/iopodman"

	"github.com/varlink/go/varlink"
)

func SprintError(methodname string, err error) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "Error calling %s: ", methodname)
	switch e := err.(type) {
	case *iopodman.ImageNotFound:
		//error ImageNotFound (name: string)
		fmt.Fprintf(buf, "'%v' name='%s'\n", e, e.Id)

	case *iopodman.ContainerNotFound:
		//error ContainerNotFound (name: string)
		fmt.Fprintf(buf, "'%v' name='%s'\n", e, e.Id)

	case *iopodman.NoContainerRunning:
		//error NoContainerRunning ()
		fmt.Fprintf(buf, "'%v'\n", e)

	case *iopodman.PodNotFound:
		//error PodNotFound (name: string)
		fmt.Fprintf(buf, "'%v' name='%s'\n", e, e.Name)

	case *iopodman.PodContainerError:
		//error PodContainerError (podname: string, errors: []PodContainerErrorData)
		fmt.Fprintf(buf, "'%v' podname='%s' errors='%v'\n", e, e.Podname, e.Errors)

	case *iopodman.NoContainersInPod:
		//error NoContainersInPod (name: string)
		fmt.Fprintf(buf, "'%v' name='%s'\n", e, e.Name)

	case *iopodman.ErrorOccurred:
		//error ErrorOccurred (reason: string)
		fmt.Fprintf(buf, "'%v' reason='%s'\n", e, e.Reason)

	case *iopodman.RuntimeError:
		//error RuntimeError (reason: string)
		fmt.Fprintf(buf, "'%v' reason='%s'\n", e, e.Reason)

	case *varlink.InvalidParameter:
		fmt.Fprintf(buf, "'%v' parameter='%s'\n", e, e.Parameter)

	case *varlink.MethodNotFound:
		fmt.Fprintf(buf, "'%v' method='%s'\n", e, e.Method)

	case *varlink.MethodNotImplemented:
		fmt.Fprintf(buf, "'%v' method='%s'\n", e, e.Method)

	case *varlink.InterfaceNotFound:
		fmt.Fprintf(buf, "'%v' interface='%s'\n", e, e.Interface)

	case *varlink.Error:
		fmt.Fprintf(buf, "'%v' parameters='%v'\n", e, e.Parameters)

	default:
		if err == io.EOF {
			fmt.Fprintf(buf, "Connection closed\n")
		} else if err == io.ErrUnexpectedEOF {
			fmt.Fprintf(buf, "Connection aborted\n")
		} else {
			fmt.Fprintf(buf, "%T - '%v'\n", err, err)
		}
	}
	return buf.String()
}

type Handle struct {
	ctx  context.Context
	conn *varlink.Connection
}

func NewHandle(ctx context.Context) (Handle, error) {
	conn, err := varlink.NewConnection(ctx, "unix:/run/io.projectatomic.podman")
	return Handle{
		ctx:  ctx,
		conn: conn,
	}, err
}

func (hnd Handle) Terminal(container string, args []string, file *os.File) error {
	detachKeys := ""
	start := false

	err := iopodman.Attach().Call(hnd.ctx, hnd.conn, container, detachKeys, start)
	if err != nil {
		return err
	}

	socks, err := iopodman.GetAttachSockets().Call(hnd.ctx, hnd.conn, container)
	if err != nil {
		return err
	}

	attached, err := os.OpenFile(socks.Io_socket, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer attached.Close()

	state, err := terminal.MakeRaw(int(file.Fd()))
	if err != nil {
		return err
	}
	defer terminal.Restore(int(file.Fd()), state)

	errChan := make(chan error)

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		close(errChan)
	}()

	go func() {
		_, err := io.Copy(file, attached)
		errChan <- err
	}()

	go func() {
		_, err := io.Copy(attached, file)
		errChan <- err
	}()

	go func() {
		err := iopodman.ExecContainer().Call(hnd.ctx, hnd.conn, iopodman.ExecOpts{
			Name:       container,
			Tty:        terminal.IsTerminal(int(file.Fd())),
			Privileged: true,
			Cmd:        args,
		})
		errChan <- err
	}()

	return <-errChan
}

func (hnd Handle) Exec(container string, args []string, out io.Writer) error {
	return iopodman.ExecContainer().Call(hnd.ctx, hnd.conn, iopodman.ExecOpts{
		Name:       container,
		Tty:        true,
		Privileged: true,
		Cmd:        args,
	})
}

func (hnd Handle) GetPrefixedContainers(prefix string) ([]iopodman.Container, error) {
	ret := []iopodman.Container{}
	containers, err := iopodman.ListContainers().Call(hnd.ctx, hnd.conn)
	if err != nil {
		return ret, err
	}

	for _, cont := range containers {
		// TODO: why is it Name*s*? there is a bug lurking here? docs are unclear.
		if strings.HasPrefix(cont.Names, prefix) {
			ret = append(ret, cont)
		}
	}
	return ret, nil
}

func (hnd Handle) GetPrefixedVolumes(prefix string) ([]string, error) {
	// TODO: how to implement this?
	return nil, fmt.Errorf("not yet implemented")
}

func (hnd Handle) FindPrefixedContainer(prefixedName string) (iopodman.Container, error) {
	containers := []iopodman.Container{}

	containers, err := hnd.GetPrefixedContainers(prefixedName)
	if err != nil {
		return iopodman.Container{}, err
	}

	if len(containers) != 1 {
		return iopodman.Container{}, fmt.Errorf("failed to found the container with name %s", prefixedName)
	}
	return containers[0], nil
}

func (hnd Handle) RemoveContainer(name string, force, removeVolumes bool) (string, error) {
	return iopodman.RemoveContainer().Call(hnd.ctx, hnd.conn, name, force, removeVolumes)
}

func (hnd Handle) CreateNamedVolume(name string) (string, error) {
	return iopodman.VolumeCreate().Call(hnd.ctx, hnd.conn, iopodman.VolumeCreateOpts{
		VolumeName: name,
	})
}

func (hnd Handle) CreateContainer(conf iopodman.Create) (string, error) {
	return iopodman.CreateContainer().Call(hnd.ctx, hnd.conn, conf)
}

func (hnd Handle) StartContainer(contID string) (string, error) {
	return iopodman.StartContainer().Call(hnd.ctx, hnd.conn, contID)
}

func (hnd Handle) WaitContainer(name string, interval int64) (int64, error) {
	return iopodman.WaitContainer().Call(hnd.ctx, hnd.conn, name, interval)
}

func (hnd Handle) PullImage(ref string, out io.Writer) error {
	tries := []int{0, 1, 2, 6}
	for idx, i := range tries {
		time.Sleep(time.Duration(i) * time.Second)

		log.Printf("attempt #%d to download %s\n", idx, ref)

		// TODO: print _some_ progress while this is going forward
		_, err := iopodman.PullImage().Call(hnd.ctx, hnd.conn, ref)
		if err != nil {
			log.Printf("failed to download %s: %v\n", ref, err)
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to download %s %d times, giving up.", ref, len(tries))
}
