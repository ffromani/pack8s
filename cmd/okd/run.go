package okd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/iopodman"

	"github.com/fromanirh/pack8s/internal/pkg/images"
	"github.com/fromanirh/pack8s/internal/pkg/ledger"
	"github.com/fromanirh/pack8s/internal/pkg/mounts"
	"github.com/fromanirh/pack8s/internal/pkg/podman"
	"github.com/fromanirh/pack8s/internal/pkg/ports"
)

type options struct {
	privileged     bool
	masterMemory   string
	masterCpu      string
	workers        string
	workersMemory  string
	workersCpu     string
	registryVolume string
	nfsData        string
	registryPort   uint
	ocpConsolePort uint
	k8sPort        uint
	sshMasterPort  uint
	sshWorkerPort  uint
	background     bool
	randomPorts    bool
}

var opts options

// NewRunCommand returns command that runs OKD cluster
func NewRunCommand() *cobra.Command {
	run := &cobra.Command{
		Use:   "okd",
		Short: "run OKD cluster",
		RunE:  run,
		Args:  cobra.ExactArgs(1),
	}

	opts.privileged = true // always
	run.Flags().StringVar(&opts.masterMemory, "master-memory", "12288", "amount of RAM in MB on the master")
	run.Flags().StringVar(&opts.masterCpu, "master-cpu", "4", "number of CPU cores on the master")
	run.Flags().StringVar(&opts.workers, "workers", "1", "number of cluster worker nodes to start")
	run.Flags().StringVar(&opts.workersMemory, "workers-memory", "6144", "amount of RAM in MB per worker")
	run.Flags().StringVar(&opts.workersCpu, "workers-cpu", "2", "number of CPU per worker")
	run.Flags().StringVar(&opts.registryVolume, "registry-volume", "", "cache docker registry content in the specified volume")
	run.Flags().StringVar(&opts.nfsData, "nfs-data", "", "path to data which should be exposed via nfs to the nodes")
	run.Flags().UintVar(&opts.registryPort, "registry-port", 0, "port on localhost for the docker registry")
	run.Flags().UintVar(&opts.ocpConsolePort, "ocp-console-port", 0, "port on localhost for the ocp console")
	run.Flags().UintVar(&opts.k8sPort, "k8s-port", 0, "port on localhost for the k8s cluster")
	run.Flags().UintVar(&opts.sshMasterPort, "ssh-master-port", 0, "port on localhost to ssh to master node")
	run.Flags().UintVar(&opts.sshWorkerPort, "ssh-worker-port", 0, "port on localhost to ssh to worker node")
	run.Flags().BoolVar(&opts.background, "background", false, "go to background after nodes are up")
	run.Flags().BoolVar(&opts.randomPorts, "random-ports", true, "expose all ports on random localhost ports")
	return run
}

func run(cmd *cobra.Command, args []string) (err error) {

	prefix, err := cmd.Flags().GetString("prefix")
	if err != nil {
		return err
	}

	envs := []string{}
	envs = append(envs, fmt.Sprintf("WORKERS=%s", opts.workers))
	envs = append(envs, fmt.Sprintf("MASTER_MEMORY=%s", opts.masterMemory))
	envs = append(envs, fmt.Sprintf("MASTER_CPU=%s", opts.masterCpu))
	envs = append(envs, fmt.Sprintf("WORKERS_MEMORY=%s", opts.workersMemory))
	envs = append(envs, fmt.Sprintf("WORKERS_CPU=%s", opts.workersCpu))

	portMap, err := ports.NewMappingFromFlags(cmd.Flags(), []ports.PortInfo{
		ports.PortInfo{
			ExposedPort: ports.PortSSH,
			Name:        "ssh-master-port",
		},
		ports.PortInfo{
			ExposedPort: ports.PortSSHWorker,
			Name:        "ssh-worker-port",
		},
		ports.PortInfo{
			ExposedPort: ports.PortAPI,
			Name:        "k8s-port",
		},
		ports.PortInfo{
			ExposedPort: ports.PortOCPConsole,
			Name:        "ocp-console-port",
		},
		ports.PortInfo{
			ExposedPort: ports.PortRegistry,
			Name:        "registry-port",
		},
	})
	if err != nil {
		return err
	}

	cluster := args[0]

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

	clusterContainerName := prefix + "-cluster"
	clusterExpose := ports.ToStrings(
		ports.PortSSH, ports.PortSSHWorker, ports.PortRegistry,
		ports.PortOCPConsole, ports.PortAPI,
	)
	clusterPorts := portMap.ToStrings()
	clusterID, err := ldgr.RunContainer(iopodman.Create{
		Args:       []string{cluster},
		Env:        &envs,
		Expose:     &clusterExpose,
		Name:       &clusterContainerName,
		Privileged: &opts.privileged,
		Publish:    &clusterPorts,
		PublishAll: &opts.randomPorts,
	})
	if err != nil {
		return err
	}

	clusterNetwork := fmt.Sprintf("container:%s", clusterID)

	// Pull the registry image
	err = images.PullImage(conn, images.DockerRegistryImage, os.Stdout)
	if err != nil {
		return err
	}

	// TODO: how to use the user-supplied name?
	var registryMounts mounts.MountMapping
	if opts.registryVolume != "" {
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
		Privileged: &opts.privileged,
		Network:    &clusterNetwork,
	})
	if err != nil {
		return err
	}

	if opts.nfsData != "" {
		nfsData, err := filepath.Abs(opts.nfsData)
		if err != nil {
			return err
		}

		err = images.PullImage(conn, images.NFSGaneshaImage, os.Stdout)
		if err != nil {
			return err
		}

		nfsName := fmt.Sprintf("%s-nfs-ganesha", prefix)
		nfsMounts := []string{fmt.Sprintf("type=bind,source=%s,destination=/data/nfs", nfsData)}
		_, err = ldgr.RunContainer(iopodman.Create{
			Args:       []string{images.NFSGaneshaImage},
			Name:       &nfsName,
			Mount:      &nfsMounts,
			Privileged: &opts.privileged,
			Network:    &clusterNetwork,
		})
		if err != nil {
			return err
		}
	}

	// Run the cluster
	fmt.Printf("Run the cluster\n")
	err = podman.Exec(conn, clusterContainerName, []string{"/bin/bash", "-c", "/scripts/run.sh"}, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to run the OKD cluster under the container %s: %s", clusterContainerName, err)
	}

	// If background flag was specified, we don't want to clean up if we reach that state
	if !opts.background {
		ldgr.Done <- fmt.Errorf("Done. please clean up")
	}

	return nil
}
