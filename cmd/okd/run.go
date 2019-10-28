package okd

import (
	"context"
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

type okdRunOptions struct {
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

var okdRunOpts okdRunOptions

// NewRunCommand returns command that runs OKD cluster
func NewRunCommand() *cobra.Command {
	run := &cobra.Command{
		Use:   "okd",
		Short: "run OKD cluster",
		RunE:  run,
		Args:  cobra.ExactArgs(1),
	}

	okdRunOpts.privileged = true // always
	run.Flags().StringVar(&okdRunOpts.masterMemory, "master-memory", "12288", "amount of RAM in MB on the master")
	run.Flags().StringVar(&okdRunOpts.masterCpu, "master-cpu", "4", "number of CPU cores on the master")
	run.Flags().StringVar(&okdRunOpts.workers, "workers", "1", "number of cluster worker nodes to start")
	run.Flags().StringVar(&okdRunOpts.workersMemory, "workers-memory", "6144", "amount of RAM in MB per worker")
	run.Flags().StringVar(&okdRunOpts.workersCpu, "workers-cpu", "2", "number of CPU per worker")
	run.Flags().StringVar(&okdRunOpts.registryVolume, "registry-volume", "", "cache docker registry content in the specified volume")
	run.Flags().StringVar(&okdRunOpts.nfsData, "nfs-data", "", "path to data which should be exposed via nfs to the nodes")
	run.Flags().UintVar(&okdRunOpts.registryPort, "registry-port", 0, "port on localhost for the docker registry")
	run.Flags().UintVar(&okdRunOpts.ocpConsolePort, "ocp-console-port", 0, "port on localhost for the ocp console")
	run.Flags().UintVar(&okdRunOpts.k8sPort, "k8s-port", 0, "port on localhost for the k8s cluster")
	run.Flags().UintVar(&okdRunOpts.sshMasterPort, "ssh-master-port", 0, "port on localhost to ssh to master node")
	run.Flags().UintVar(&okdRunOpts.sshWorkerPort, "ssh-worker-port", 0, "port on localhost to ssh to worker node")
	run.Flags().BoolVar(&okdRunOpts.background, "background", false, "go to background after nodes are up")
	run.Flags().BoolVar(&okdRunOpts.randomPorts, "random-ports", true, "expose all ports on random localhost ports")
	return run
}

func run(cmd *cobra.Command, args []string) (err error) {

	prefix, err := cmd.Flags().GetString("prefix")
	if err != nil {
		return err
	}

	podmanSocket, err := cmd.Flags().GetString("podman-socket")
	if err != nil {
		return err
	}

	envs := []string{}
	envs = append(envs, fmt.Sprintf("WORKERS=%s", okdRunOpts.workers))
	envs = append(envs, fmt.Sprintf("MASTER_MEMORY=%s", okdRunOpts.masterMemory))
	envs = append(envs, fmt.Sprintf("MASTER_CPU=%s", okdRunOpts.masterCpu))
	envs = append(envs, fmt.Sprintf("WORKERS_MEMORY=%s", okdRunOpts.workersMemory))
	envs = append(envs, fmt.Sprintf("WORKERS_CPU=%s", okdRunOpts.workersCpu))

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

	ctx, cancel := context.WithCancel(context.Background())

	hnd, err := podman.NewHandle(ctx, podmanSocket)
	if err != nil {
		return err
	}

	ldgr := ledger.NewLedger(hnd, cmd.OutOrStderr())

	defer func() {
		ldgr.Done <- err
	}()

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		cancel()
		ldgr.Done <- fmt.Errorf("Interrupt received, clean up")
	}()
	// Pull the cluster image
	err = hnd.PullImage("docker.io/" + cluster)
	if err != nil {
		return err
	}

	clusterContainerName := prefix + "-cluster"
	clusterExpose := ports.ToStrings(
		ports.PortSSH, ports.PortSSHWorker, ports.PortRegistry,
		ports.PortOCPConsole, ports.PortAPI,
	)
	clusterPorts := portMap.ToStrings()
	clusterLabels := []string{fmt.Sprintf("%s=000", podman.LabelGeneration)}
	clusterID, err := ldgr.RunContainer(iopodman.Create{
		Args:       []string{cluster},
		Env:        &envs,
		Expose:     &clusterExpose,
		Label:      &clusterLabels,
		Name:       &clusterContainerName,
		Privileged: &okdRunOpts.privileged,
		Publish:    &clusterPorts,
		PublishAll: &okdRunOpts.randomPorts,
	})
	if err != nil {
		return err
	}

	clusterNetwork := fmt.Sprintf("container:%s", clusterID)

	// Pull the registry image
	err = hnd.PullImage(images.DockerRegistryImage)
	if err != nil {
		return err
	}

	// TODO: how to use the user-supplied name?
	var registryMounts mounts.MountMapping
	if okdRunOpts.registryVolume != "" {
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
	registryLabels := []string{fmt.Sprintf("%s=001", podman.LabelGeneration)}
	_, err = ldgr.RunContainer(iopodman.Create{
		Args:       []string{images.DockerRegistryImage},
		Label:      &registryLabels,
		Mount:      &registryMountsStrings,
		Name:       &registryName,
		Network:    &clusterNetwork,
		Privileged: &okdRunOpts.privileged,
	})
	if err != nil {
		return err
	}

	if okdRunOpts.nfsData != "" {
		nfsData, err := filepath.Abs(okdRunOpts.nfsData)
		if err != nil {
			return err
		}

		err = hnd.PullImage(images.NFSGaneshaImage)
		if err != nil {
			return err
		}

		nfsName := fmt.Sprintf("%s-nfs-ganesha", prefix)
		nfsMounts := []string{fmt.Sprintf("type=bind,source=%s,destination=/data/nfs", nfsData)}
		nfsLabels := []string{fmt.Sprintf("%s=010", podman.LabelGeneration)}
		_, err = ldgr.RunContainer(iopodman.Create{
			Args:       []string{images.NFSGaneshaImage},
			Label:      &nfsLabels,
			Mount:      &nfsMounts,
			Name:       &nfsName,
			Network:    &clusterNetwork,
			Privileged: &okdRunOpts.privileged,
		})
		if err != nil {
			return err
		}
	}

	// Run the cluster
	fmt.Printf("Run the cluster\n")
	err = hnd.Exec(clusterContainerName, []string{"/bin/bash", "-c", "/scripts/run.sh"}, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to run the OKD cluster under the container %s: %s", clusterContainerName, err)
	}

	// If background flag was specified, we don't want to clean up if we reach that state
	if !okdRunOpts.background {
		ldgr.Done <- fmt.Errorf("Done. please clean up")
	}

	return nil
}
