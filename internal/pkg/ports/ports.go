package ports

import (
	"fmt"
	"strconv"

	"github.com/spf13/pflag"

	"github.com/fromanirh/pack8s/iopodman"
)

const (
	// PortSSH contains SSH port for the master node
	PortSSH = 2201
	// PortSSHWorker contains SSH port for the worker node
	PortSSHWorker = 2202
	// PortRegistry contains private image registry port
	PortRegistry = 5000
	// PortOCP contains OCP API server port
	PortOCP = 8443
	// PortAPI contains API server port
	PortAPI = 6443
	// PortVNC contains first VM VNC port
	PortVNC = 5901
	//PortOCPConsole contains OCP console port
	PortOCPConsole = 443

	// PortNameSSH contains master node SSH port name
	PortNameSSH = "ssh"
	// PortNameSSHWorker contains worker node SSH port name
	PortNameSSHWorker = "ssh-worker"
	// PortNameOCP contains OCP port name
	PortNameOCP = "ocp"
	// PortNameRegistry contains registry port name
	PortNameRegistry = "registry"
	// PortNameAPI contains API port name
	// TODO: change the name to API
	PortNameAPI = "k8s"
	// PortNameVNC contains VNC port name
	PortNameVNC = "vnc"
	// PortNameOCPConsole contains OCP console port
	PortNameOCPConsole = "console"
)

func ToStrings(ports ...int) []string {
	res := []string{}
	for _, port := range ports {
		res = append(res, fmt.Sprintf("%d", port))
	}
	return res
}

type PortInfo struct {
	ExposedPort int
	Name        string
}

type PortMapping struct {
	data []iopodman.ContainerPortMappings
}

func NewMappingFromFlags(flagSet *pflag.FlagSet, portInfos []PortInfo) (PortMapping, error) {
	pm := PortMapping{}
	var err error
	for _, info := range portInfos {
		err = pm.appendPort(info.ExposedPort, flagSet, info.Name)
		if err != nil {
			break
		}
	}
	return pm, err
}

func (pm PortMapping) ToStrings() []string {
	res := []string{}
	for _, pmItem := range pm.data {
		res = append(res, fmt.Sprintf("%s:%s", pmItem.Host_port, pmItem.Container_port))
	}
	return res
}

func (pm PortMapping) appendPort(exposedPort int, flagSet *pflag.FlagSet, flagName string) error {
	flag := flagSet.Lookup(flagName)
	if flag != nil && flag.Changed {
		publicPort, err := flagSet.GetUint(flagName)
		if err != nil {
			return err
		}

		pm.data = append(pm.data, iopodman.ContainerPortMappings{
			Host_port:      strconv.Itoa(int(publicPort)),
			Host_ip:        "127.0.0.1",
			Protocol:       "tcp",
			Container_port: strconv.Itoa(exposedPort),
		})
	}
	return nil
}
