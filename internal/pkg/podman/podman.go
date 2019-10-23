package podman

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/fromanirh/pack8s/iopodman"

	"github.com/varlink/go/varlink"
)

var NotImplemented error = fmt.Errorf("Not yet implemented")

func NewConnection() (*varlink.Connection, error) {
	return varlink.NewConnection("unix:/run/io.projectatomic.podman")
}

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

func Exec(conn *varlink.Connection, container string, args []string, out io.Writer) error {
	return iopodman.ExecContainer().Call(conn, iopodman.ExecOpts{
		Name:       container,
		Tty:        true,
		Privileged: true,
		Cmd:        args,
	})
}

func GetPrefixedContainers(conn *varlink.Connection, prefix string) ([]iopodman.Container, error) {
	ret := []iopodman.Container{}
	containers, err := iopodman.ListContainers().Call(conn)
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

func GetPrefixedVolumes(conn *varlink.Connection, prefix string) ([]string, error) {
	// TODO: how to implement this?
	return nil, fmt.Errorf("not yet implemented")
}

func FindPrefixedContainer(prefixedName string) (iopodman.Container, error) {
	containers := []iopodman.Container{}

	conn, err := NewConnection()
	if err != nil {
		return iopodman.Container{}, err
	}

	containers, err = GetPrefixedContainers(conn, prefixedName)
	if err != nil {
		return iopodman.Container{}, err
	}

	if len(containers) != 1 {
		return iopodman.Container{}, fmt.Errorf("failed to found the container with name %s", prefixedName)
	}
	return containers[0], nil
}
