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

	"github.com/varlink/go/varlink"

	"github.com/fromanirh/pack8s/pkg/varlinkapi/virtwriter"

	"github.com/fromanirh/pack8s/iopodman"
)

const (
	DefaultSocket   string = "unix:/run/podman/io.podman"
	LabelGeneration string = "io.kubevirt/pack8s.generation"
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
	socket string
	ctx    context.Context
	conn   *varlink.Connection
}

func NewHandle(ctx context.Context, socket string) (*Handle, error) {
	if socket == "" {
		socket = DefaultSocket
	}
	log.Printf("connecting to %s", socket)
	conn, err := varlink.NewConnection(ctx, socket)
	log.Printf("connected to %s", socket)
	return &Handle{
		socket: socket,
		ctx:    ctx,
		conn:   conn,
	}, err
}

func (hnd *Handle) reconnect() (bool, error) {
	if hnd.conn == nil {
		var err error
		hnd.conn, err = varlink.NewConnection(hnd.ctx, hnd.socket)
		log.Printf("reconnected to %s (err=%v)", hnd.socket, err)
		return true, err
	}
	return false, nil
}

func (hnd *Handle) disconnect() error {
	var err error
	if hnd.conn != nil {
		err = hnd.conn.Close()
		hnd.conn = nil
		log.Printf("disconnected from %s (err=%v)", hnd.socket, err)
	}
	return err
}

func (hnd *Handle) Terminal(container string, args []string, file *os.File) error {
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

type ReaderContext interface {
	Read(context.Context, []byte) (int, error)
}

type readerCtx struct {
	ReaderContext
	ctx context.Context
}

func (r readerCtx) Read(in []byte) (int, error) {
	return r.ReaderContext.Read(r.ctx, in)
}

func (hnd *Handle) Exec(container string, args []string, out io.Writer) error {
	log.Printf("exec start")
	_, err := hnd.reconnect()
	if err != nil {
		return err
	}
	defer hnd.disconnect()

	rwc, err := ExecContainer().Call(hnd.ctx, hnd.conn, iopodman.ExecOpts{
		Name:       container,
		Tty:        true,
		Privileged: true,
		Cmd:        args,
	})
	log.Printf("exec call done")

	if err != nil {
		return err
	}

	rd := readerCtx{
		ReaderContext: rwc,
		ctx:           context.TODO(),
	}

	ecChan := make(chan int, 1)
	errChan := make(chan error, 1)
	go func() {
		// Read from the wire and direct to stdout or stderr
		err := virtwriter.Reader(rd, out, os.Stderr, nil, ecChan)
		errChan <- err
	}()

	err = <-errChan
	if err != nil {
		return err
	}
	rc := <-ecChan
	log.Printf("exec done -> %d", rc)
	if rc != 0 {
		return fmt.Errorf("exec failed: rc=%d", rc)
	}
	return nil
}

func (hnd *Handle) GetPrefixedContainers(prefix string) ([]iopodman.Container, error) {
	ret := []iopodman.Container{}
	containers, err := iopodman.ListContainers().Call(hnd.ctx, hnd.conn)
	if err != nil {
		return ret, err
	}

	log.Printf("found %d containers in the system", len(containers))
	for _, cont := range containers {
		// TODO: why is it Name*s*? there is a bug lurking here? docs are unclear.
		if strings.HasPrefix(cont.Names, prefix) {
			log.Printf("matching container: %s (%s)\n", cont.Names, cont.Id)
			ret = append(ret, cont)
		}
	}
	log.Printf("found %d containers matching the prefix", len(ret))
	return ret, nil
}

func (hnd *Handle) GetPrefixedVolumes(prefix string) ([]iopodman.Volume, error) {
	ret := []iopodman.Volume{}
	args := []string{}
	all := true
	volumes, err := iopodman.GetVolumes().Call(hnd.ctx, hnd.conn, args, all)
	if err != nil {
		return ret, err
	}

	log.Printf("found %d volumess in the system", len(volumes))
	for _, vol := range volumes {
		if strings.HasPrefix(vol.Name, prefix) {
			log.Printf("matching volume: %s @(%s)\n", vol.Name, vol.MountPoint)
			ret = append(ret, vol)
		}
	}
	log.Printf("found %d volumes matching the prefix", len(ret))
	return ret, err
}

func (hnd *Handle) FindPrefixedContainer(prefixedName string) (iopodman.Container, error) {
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

//PruneVolumes removes all unused volumes on the host.
func (hnd *Handle) PruneVolumes() error {
	_, _, err := iopodman.VolumesPrune().Call(hnd.ctx, hnd.conn)
	return err
}

//GetAllVolumes returns all volumes
func (hnd *Handle) GetAllVolumes() ([]iopodman.Volume, error) {
	return iopodman.GetVolumes().Call(hnd.ctx, hnd.conn, []string{}, true)
}

func (hnd *Handle) RemoveVolumes(volumes []iopodman.Volume) error {
	volumeNames := []string{}
	for _, vol := range volumes {
		log.Printf("removing volume %s @%s", vol.Name, vol.MountPoint)
		volumeNames = append(volumeNames, vol.Name)
	}
	_, _, err := iopodman.VolumeRemove().Call(hnd.ctx, hnd.conn, iopodman.VolumeRemoveOpts{
		Volumes: volumeNames,
		Force:   true,
	})
	return err
}

func (hnd *Handle) RemoveContainer(cont iopodman.Container, force, removeVolumes bool) (string, error) {
	log.Printf("trying to remove: %s (%s) force=%v removeVolumes=%v\n", cont.Names, cont.Id, force, removeVolumes)
	return iopodman.RemoveContainer().Call(hnd.ctx, hnd.conn, cont.Id, force, removeVolumes)
}

func (hnd *Handle) CreateNamedVolume(name string) (string, error) {
	return iopodman.VolumeCreate().Call(hnd.ctx, hnd.conn, iopodman.VolumeCreateOpts{
		VolumeName: name,
	})
}

func (hnd *Handle) CreateContainer(conf iopodman.Create) (string, error) {
	return iopodman.CreateContainer().Call(hnd.ctx, hnd.conn, conf)
}

func (hnd *Handle) StopContainer(name string, timeout int64) (string, error) {
	return iopodman.StopContainer().Call(hnd.ctx, hnd.conn, name, timeout)
}

func (hnd *Handle) StartContainer(contID string) (string, error) {
	return iopodman.StartContainer().Call(hnd.ctx, hnd.conn, contID)
}

func (hnd *Handle) WaitContainer(name string, interval int64) (int64, error) {
	return iopodman.WaitContainer().Call(hnd.ctx, hnd.conn, name, interval)
}

//ListImages returns all images on host
func (hnd *Handle) ListImages() ([]iopodman.Image, error) {
	return iopodman.ListImages().Call(hnd.ctx, hnd.conn)
}

func (hnd *Handle) PullImage(ref string) error {
	tries := []int{0, 1, 2, 6}
	interval := 3 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for idx, i := range tries {
		time.Sleep(time.Duration(i) * time.Second)

		prefix := fmt.Sprintf("attempt #%d", idx)
		log.Printf("%s to download '%s' - progress every %v\n", prefix, ref, interval)
		err := hnd.pullImage(ticker, prefix, ref)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("failed to download %s %d times, giving up.", ref, len(tries))
}

func (hnd *Handle) pullImage(ticker *time.Ticker, prefix, ref string) error {
	var err error
	errChan := make(chan error)
	go func() {
		_, err := iopodman.PullImage().Call(hnd.ctx, hnd.conn, ref)
		errChan <- err
	}()

	for {
		select {
		case <-ticker.C:
			log.Printf("%s: downloading...", prefix)
		case err = <-errChan:
			if err != nil {
				log.Printf("%s: failed to download %s: %v\n", prefix, ref, err)
			} else {
				log.Printf("%s: downloaded: %s", prefix, ref)
			}
			return err
		}
	}

	return fmt.Errorf("pull failed - internal error") // can't be reached
}
