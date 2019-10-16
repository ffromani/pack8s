package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/iopodman"

	"github.com/fromanirh/pack8s/internal/pkg/images"
	"github.com/fromanirh/pack8s/internal/pkg/ledger"
	"github.com/fromanirh/pack8s/internal/pkg/mounts"
	"github.com/fromanirh/pack8s/internal/pkg/podman"
	"github.com/fromanirh/pack8s/internal/pkg/ports"
)

// NewRunCommand returns command that runs given cluster
func NewRunCommand() *cobra.Command {
	run := &cobra.Command{
		Use:   "run",
		Short: "run a given cluster",
		RunE:  run,
		Args:  cobra.ExactArgs(1),
	}
	run.Flags().UintP("nodes", "n", 1, "number of cluster nodes to start")
	run.Flags().StringP("memory", "m", "3096M", "amount of ram per node")
	run.Flags().UintP("cpu", "c", 2, "number of cpu cores per node")
	run.Flags().UintP("secondary-nics", "", 0, "number of secondary nics to add")
	run.Flags().String("qemu-args", "", "additional qemu args to pass through to the nodes")
	run.Flags().BoolP("background", "b", false, "go to background after nodes are up")
	run.Flags().BoolP("reverse", "r", false, "revert node startup order")
	run.Flags().Bool("random-ports", true, "expose all ports on random localhost ports")
	run.Flags().String("registry-volume", "", "cache docker registry content in the specified volume")
	run.Flags().Uint("vnc-port", 0, "port on localhost for vnc")
	run.Flags().Uint("registry-port", 0, "port on localhost for the docker registry")
	run.Flags().Uint("ocp-port", 0, "port on localhost for the ocp cluster")
	run.Flags().Uint("k8s-port", 0, "port on localhost for the k8s cluster")
	run.Flags().Uint("ssh-port", 0, "port on localhost for ssh server")
	run.Flags().String("nfs-data", "", "path to data which should be exposed via nfs to the nodes")
	run.Flags().String("log-to-dir", "", "enables aggregated cluster logging to the folder")
	run.Flags().Bool("enable-ceph", false, "enables dynamic storage provisioning using Ceph")

	//	run.AddCommand(
	//		okd.NewRunCommand(),
	//	)
	return run
}

func run(cmd *cobra.Command, args []string) (err error) {
	privileged := true

	prefix, err := cmd.Flags().GetString("prefix")
	if err != nil {
		return err
	}

	nodes, err := cmd.Flags().GetUint("nodes")
	if err != nil {
		return err
	}
	memory, err := cmd.Flags().GetString("memory")
	if err != nil {
		return err
	}

	reverse, err := cmd.Flags().GetBool("reverse")
	if err != nil {
		return err
	}
	randomPorts, err := cmd.Flags().GetBool("random-ports")
	if err != nil {
		return err
	}

	portMap, err := ports.NewMappingFromFlags(cmd.Flags(), []ports.PortInfo{
		ports.PortInfo{
			ExposedPort: ports.PortSSH,
			Name:        "ssh-port",
		},
		ports.PortInfo{
			ExposedPort: ports.PortVNC,
			Name:        "vnc-port",
		},
		ports.PortInfo{
			ExposedPort: ports.PortAPI,
			Name:        "k8s-port",
		},
		ports.PortInfo{
			ExposedPort: ports.PortOCP,
			Name:        "ocp-port",
		},
		ports.PortInfo{
			ExposedPort: ports.PortRegistry,
			Name:        "registry-port",
		},
	})
	if err != nil {
		return err
	}

	qemuArgs, err := cmd.Flags().GetString("qemu-args")
	if err != nil {
		return err
	}

	cpu, err := cmd.Flags().GetUint("cpu")
	if err != nil {
		return err
	}

	secondaryNics, err := cmd.Flags().GetUint("secondary-nics")
	if err != nil {
		return err
	}
	registryVol, err := cmd.Flags().GetString("registry-volume")
	if err != nil {
		return err
	}

	nfsData, err := cmd.Flags().GetString("nfs-data")
	if err != nil {
		return err
	}

	logDir, err := cmd.Flags().GetString("log-to-dir")
	if err != nil {
		return err
	}

	cephEnabled, err := cmd.Flags().GetBool("enable-ceph")
	if err != nil {
		return err
	}

	cluster := args[0]

	background, err := cmd.Flags().GetBool("background")
	if err != nil {
		return err
	}

	conn, err := podman.NewConnection()
	if err != nil {
		return err
	}

	ldgr := ledger.NewLedger(conn, cmd.OutOrStderr())

	defer func() {
		ldgr.Done <- err
	}()

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		ldgr.Done <- fmt.Errorf("Interrupt received, clean up")
	}()
	// Pull the cluster image
	err = images.PullImage(conn, "docker.io/"+cluster, os.Stdout)
	if err != nil {
		return err
	}

	dnsmasqName := fmt.Sprintf("%s-dnsmasq", prefix)
	dnsmasqExpose := ports.ToStrings(
		ports.PortSSH, ports.PortRegistry, ports.PortOCP,
		ports.PortAPI, ports.PortVNC,
	)
	dnsmasqPorts := portMap.ToStrings()
	dnsmasqID, err := ldgr.RunContainer(iopodman.Create{
		AddHost: &[]string{
			"nfs:192.168.66.2",
			"registry:192.168.66.2",
			"ceph:192.168.66.2",
		},
		Args: []string{cluster, "/bin/bash", "-c", "/dnsmasq.sh"},
		Env: &[]string{
			fmt.Sprintf("NUM_NODES=%d", nodes),
			fmt.Sprintf("NUM_SECONDARY_NICS=%d", secondaryNics),
		},
		Expose:     &dnsmasqExpose,
		Name:       &dnsmasqName,
		Privileged: &privileged,
		Publish:    &dnsmasqPorts,
		PublishAll: &randomPorts,
	})
	if err != nil {
		return err
	}

	dnsmasqNetwork := fmt.Sprintf("container:%s", dnsmasqID)

	// Pull the registry image
	err = images.PullImage(conn, images.DockerRegistryImage, os.Stdout)
	if err != nil {
		return err
	}

	// Create registry volume
	var registryMounts mounts.MountMapping
	if registryVol != "" {
		registryMounts, err = mounts.NewVolumeMappings(ldgr, []mounts.MountInfo{
			mounts.MountInfo{
				Name: fmt.Sprintf("%s-registry", prefix),
				Path: "/var/lib/registry",
				Type: "volume",
			},
		})
		if err != nil {
			return err
		}
	}

	registryName := fmt.Sprintf("%s-registry", prefix)
	registryMountsStrings := registryMounts.ToStrings()
	_, err = ldgr.RunContainer(iopodman.Create{
		Args:       []string{images.DockerRegistryImage},
		Name:       &registryName,
		Mount:      &registryMountsStrings,
		Privileged: &privileged,
		Network:    &dnsmasqNetwork,
	})
	if err != nil {
		return err
	}

	if nfsData != "" {
		nfsData, err := filepath.Abs(nfsData)
		if err != nil {
			return err
		}

		err = images.PullImage(conn, images.NFSGaneshaImage, os.Stdout)
		if err != nil {
			return err
		}

		nfsName := fmt.Sprintf("%s-nfs", prefix)
		nfsMounts := []string{fmt.Sprintf("type=bind,source=%s,destination=/data/nfs", nfsData)}
		_, err = ldgr.RunContainer(iopodman.Create{
			Args:       []string{images.NFSGaneshaImage},
			Name:       &nfsName,
			Mount:      &nfsMounts,
			Privileged: &privileged,
			Network:    &dnsmasqNetwork,
		})
		if err != nil {
			return err
		}
	}

	if cephEnabled {
		err = images.PullImage(conn, images.CephImage, os.Stdout)
		if err != nil {
			return err
		}

		cephName := fmt.Sprintf("%s-ceph", prefix)
		_, err = ldgr.RunContainer(iopodman.Create{
			Args: []string{images.CephImage, "demo"},
			Name: &cephName,
			Env: &[]string{
				"MON_IP=192.168.66.2",
				"CEPH_PUBLIC_NETWORK=0.0.0.0/0",
				"DEMO_DAEMONS=osd,mds",
				"CEPH_DEMO_UID=demo",
			},
			Privileged: &privileged,
			Network:    &dnsmasqNetwork,
		})
		if err != nil {
			return err
		}
	}

	if logDir != "" {
		logDir, err := filepath.Abs(logDir)
		if err != nil {
			return err
		}

		if _, err = os.Stat(logDir); os.IsNotExist(err) {
			os.Mkdir(logDir, 0755)
		}

		err = images.PullImage(conn, images.FluentdImage, os.Stdout)
		if err != nil {
			return err
		}

		fluentdMounts := []string{fmt.Sprintf("type=bind,source=%s,destination=/fluentd/log/collected", logDir)}
		fluentdName := fmt.Sprintf("%s-fluentd", prefix)
		_, err = ldgr.RunContainer(iopodman.Create{
			Args: []string{
				images.FluentdImage,
				"exec", "fluentd",
				"-i", "\"<system>\n log_level debug\n</system>\n<source>\n@type  forward\n@log_level error\nport  24224\n</source>\n<match **>\n@type file\npath /fluentd/log/collected\n</match>\"",
				"-p", "/fluentd/plugins", "$FLUENTD_OPT", "-v",
			},
			Name:       &fluentdName,
			Mount:      &fluentdMounts,
			Privileged: &privileged,
			Network:    &dnsmasqNetwork,
		})
		if err != nil {
			return err
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(int(nodes))
	macCounter := 0

	for x := 0; x < int(nodes); x++ {
		nodeQemuArgs := qemuArgs

		for i := 0; i < int(secondaryNics); i++ {
			netSuffix := fmt.Sprintf("%d-%d", x, i)
			macSuffix := fmt.Sprintf("%02x", macCounter)
			macCounter++
			nodeQemuArgs = fmt.Sprintf("%s -device virtio-net-pci,netdev=secondarynet%s,mac=52:55:00:d1:56:%s -netdev tap,id=secondarynet%s,ifname=stap%s,script=no,downscript=no", nodeQemuArgs, netSuffix, macSuffix, netSuffix, netSuffix)
		}

		if len(nodeQemuArgs) > 0 {
			nodeQemuArgs = "--qemu-args \"" + nodeQemuArgs + "\""
		}

		nodeName := nodeNameFromIndex(x + 1)
		nodeNum := fmt.Sprintf("%02d", x+1)
		if reverse {
			nodeName = nodeNameFromIndex((int(nodes) - x))
			nodeNum = fmt.Sprintf("%02d", (int(nodes) - x))
		}

		nodeMounts, err := mounts.NewVolumeMappings(ldgr, []mounts.MountInfo{
			mounts.MountInfo{
				Name: fmt.Sprintf("%s-%s", prefix, nodeName),
				Path: "/var/run/disk",
				Type: "volume",
			},
		})
		if err != nil {
			return err
		}

		contNodeName := nodeContainer(prefix, nodeName)
		contNodeMountsStrings := nodeMounts.ToStrings()
		contNodeID, err := ldgr.RunContainer(iopodman.Create{
			Args: []string{cluster, "/bin/bash", "-c", fmt.Sprintf("/vm.sh -n /var/run/disk/disk.qcow2 --memory %s --cpu %s %s", memory, strconv.Itoa(int(cpu)), nodeQemuArgs)},
			Env: &[]string{
				fmt.Sprintf("NODE_NUM=%s", nodeNum),
			},
			Name:       &contNodeName,
			Mount:      &contNodeMountsStrings,
			Privileged: &privileged,
			Network:    &dnsmasqNetwork,
		})
		if err != nil {
			return err
		}

		err = podman.Exec(conn, contNodeName, []string{"/bin/bash", "-c", "while [ ! -f /ssh_ready ] ; do sleep 1; done"}, os.Stdout)
		if err != nil {
			return fmt.Errorf("checking for ssh.sh script for node %s failed: %s", nodeName, err)
		}

		//check if we have a special provision script
		err = podman.Exec(conn, contNodeName, []string{"/bin/bash", "-c", fmt.Sprintf("test -f /scripts/%s.sh", nodeName)}, os.Stdout)
		if err == nil {
			err = podman.Exec(conn, contNodeName, []string{"/bin/bash", "-c", fmt.Sprintf("ssh.sh sudo /bin/bash < /scripts/%s.sh", nodeName)}, os.Stdout)
		} else {
			err = podman.Exec(conn, contNodeName, []string{"/bin/bash", "-c", "ssh.sh sudo /bin/bash < /scripts/nodes.sh"}, os.Stdout)
		}

		if err != nil {
			return fmt.Errorf("provisioning node %s failed: %s", nodeName, err)
		}

		go func(id string) {
			iopodman.WaitContainer().Call(conn, id, int64(1*time.Second))
			wg.Done()
		}(contNodeID)
	}

	if cephEnabled {
		// XXX begin
		keyRing := new(bytes.Buffer)
		err := podman.Exec(conn, nodeContainer(prefix, "ceph"), []string{
			"/bin/bash",
			"-c",
			"ceph auth print-key connent.admin | base64",
		}, keyRing)
		// XXX end
		if err != nil {
			return err
		}
		nodeName := nodeNameFromIndex(1)
		key := bytes.TrimSpace(keyRing.Bytes())
		err = podman.Exec(conn, nodeContainer(prefix, nodeName), []string{
			"/bin/bash",
			"-c",
			fmt.Sprintf("ssh.sh sudo sed -i \"s/replace-me/%s/g\" /tmp/ceph/ceph-secret.yaml", key),
		}, os.Stdout)
		if err != nil {
			return err
		}
		err = podman.Exec(conn, nodeContainer(prefix, nodeName), []string{
			"/bin/bash",
			"-c",
			"ssh.sh sudo /bin/bash < /scripts/ceph-csi.sh",
		}, os.Stdout)
		if err != nil {
			return fmt.Errorf("provisioning Ceph CSI failed: %s", err)
		}
	}

	// If logging is enabled, deploy the default fluent logging
	if logDir != "" {
		nodeName := nodeNameFromIndex(1)
		err := podman.Exec(conn, nodeContainer(prefix, nodeName), []string{
			"/bin/bash",
			"-c",
			"ssh.sh sudo /bin/bash < /scripts/logging.sh",
		}, os.Stdout)
		if err != nil {
			return fmt.Errorf("provisioning logging failed: %s", err)
		}
	}

	// If background flag was specified, we don't want to clean up if we reach that state
	if !background {
		wg.Wait()
		ldgr.Done <- fmt.Errorf("Done. please clean up")
	}
	return nil
}

func nodeNameFromIndex(x int) string {
	return fmt.Sprintf("node%02d", x)
}

func nodeContainer(prefix string, node string) string {
	return prefix + "-" + node
}
