package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/internal/pkg/podman"
	"github.com/fromanirh/pack8s/internal/pkg/ports"
)

// NewPortCommand returns new command to expose public ports for the cluster
func NewPortCommand() *cobra.Command {
	port := &cobra.Command{
		Use:   "ports",
		Short: "ports shows exposed ports of the cluster",
		Long: `ports shows exposed ports of the cluster

If no port name is specified, all exposed ports are printed.
If an extra port name is specified, only the exposed port is printed.

Known port names are 'ssh', 'registry', 'ocp' and 'k8s'.
`,
		RunE: showPorts,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("only one port name can be specified at once")
			}

			if len(args) == 1 {
				switch args[0] {
				case ports.PortNameSSH, ports.PortNameSSHWorker, ports.PortNameAPI, ports.PortNameOCP, ports.PortNameOCPConsole, ports.PortNameRegistry, ports.PortNameVNC:
					return nil
				default:
					return fmt.Errorf("unknown port name %s", args[0])
				}
			}
			return nil
		},
	}

	port.Flags().String("container-name", "dnsmasq", "the container name to SSH copy from")

	return port
}

func showPorts(cmd *cobra.Command, args []string) error {

	prefix, err := cmd.Flags().GetString("prefix")
	if err != nil {
		return err
	}

	containerName, err := cmd.Flags().GetString("container-name")
	if err != nil {
		return err
	}

	conn, err := podman.NewConnection()
	if err != nil {
		return err
	}

	containers, err := podman.GetPrefixedContainers(conn, prefix+"-"+containerName)
	if err != nil {
		return err
	}

	if len(containers) != 1 {
		return fmt.Errorf("failed to found the container with name %s", prefix+"-"+containerName)
	}

	portName := ""
	if len(args) > 0 {
		portName = args[0]
	}

	if portName != "" {
		err = nil
		switch portName {
		case ports.PortNameSSH:
			err = ports.PrintPublicPort(ports.PortSSH, containers[0].Ports)
		case ports.PortNameSSHWorker:
			err = ports.PrintPublicPort(ports.PortSSHWorker, containers[0].Ports)
		case ports.PortNameAPI:
			err = ports.PrintPublicPort(ports.PortAPI, containers[0].Ports)
		case ports.PortNameRegistry:
			err = ports.PrintPublicPort(ports.PortRegistry, containers[0].Ports)
		case ports.PortNameOCP:
			err = ports.PrintPublicPort(ports.PortOCP, containers[0].Ports)
		case ports.PortNameOCPConsole:
			err = ports.PrintPublicPort(ports.PortOCPConsole, containers[0].Ports)
		case ports.PortNameVNC:
			err = ports.PrintPublicPort(ports.PortVNC, containers[0].Ports)
		}

		if err != nil {
			return err
		}

	} else {
		for _, p := range containers[0].Ports {
			fmt.Printf("%s/%s -> %s:%s\n", p.Container_port, p.Protocol, p.Host_ip, p.Host_port)
		}
	}

	return nil
}
