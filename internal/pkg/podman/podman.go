package podman

import (
	"bytes"
	"fmt"
	"io"

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

func GetPrefixedContainers(conn *varlink.Connection, prefix string) ([]string, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func GetPrefixedVolumes(conn *varlink.Connection, prefix string) ([]string, error) {
	return nil, fmt.Errorf("not yet implemented")
}
