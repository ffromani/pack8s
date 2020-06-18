package okd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/fromanirh/pack8s/iopodman"

	"github.com/fromanirh/pack8s/internal/pkg/ledger"
	"github.com/fromanirh/pack8s/internal/pkg/podman"
	"github.com/fromanirh/pack8s/internal/pkg/ports"

	"github.com/fromanirh/pack8s/cmd/cmdutil"
)

type runOptions struct {
	privileged     bool
	masterMemory   string
	masterCpu      string
	workers        string
	workersMemory  string
	workersCpu     string
	secondaryNics  uint
	registryVolume string
	nfsData        string
	registryPort   uint
	ocpConsolePort uint
	k8sPort        uint
	sshMasterPort  uint
	sshWorkerPort  uint
	background     bool
	randomPorts    bool
	volume         string
	downloadOnly   bool
}

func (ro runOptions) WantsNFS() bool {
	return ro.nfsData != ""
}

func (ro runOptions) WantsCeph() bool {
	return false
}

func (ro runOptions) WantsFluentd() bool {
	return false
}

// NewRunCommand returns command that runs OKD cluster
func NewRunCommand() *cobra.Command {
	flags := &runOptions{}

	run := &cobra.Command{
		Use:   "okd",
		Short: "run OKD cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, flags, args)
		},
		Args: cobra.ExactArgs(1),
	}

	flags.privileged = true // always
	run.Flags().StringVar(&flags.masterMemory, "master-memory", "12288", "amount of RAM in MB on the master")
	run.Flags().StringVar(&flags.masterCpu, "master-cpu", "4", "number of CPU cores on the master")
	run.Flags().StringVar(&flags.workers, "workers", "1", "number of cluster worker nodes to start")
	run.Flags().StringVar(&flags.workersMemory, "workers-memory", "6144", "amount of RAM in MB per worker")
	run.Flags().StringVar(&flags.workersCpu, "workers-cpu", "2", "number of CPU per worker")
	run.Flags().UintVar(&flags.secondaryNics, "secondary-nics", 0, "number of secondary nics to add")
	run.Flags().StringVar(&flags.registryVolume, "registry-volume", "", "cache docker registry content in the specified volume")
	run.Flags().StringVar(&flags.nfsData, "nfs-data", "", "path to data which should be exposed via nfs to the nodes")
	run.Flags().UintVar(&flags.registryPort, "registry-port", 0, "port on localhost for the docker registry")
	run.Flags().UintVar(&flags.ocpConsolePort, "ocp-console-port", 0, "port on localhost for the ocp console")
	run.Flags().UintVar(&flags.k8sPort, "k8s-port", 0, "port on localhost for the k8s cluster")
	run.Flags().UintVar(&flags.sshMasterPort, "ssh-master-port", 0, "port on localhost to ssh to master node")
	run.Flags().UintVar(&flags.sshWorkerPort, "ssh-worker-port", 0, "port on localhost to ssh to worker node")
	run.Flags().BoolVar(&flags.background, "background", false, "go to background after nodes are up")
	run.Flags().BoolVar(&flags.randomPorts, "random-ports", true, "expose all ports on random localhost ports")
	run.Flags().StringVar(&flags.volume, "volume", "", "Bind mount a volume into the container")
	return run
}

func run(cmd *cobra.Command, okdRunOpts *runOptions, args []string) (err error) {
	cOpts, err := cmdutil.GetCommonOpts(cmd)
	if err != nil {
		return err
	}

	envs := []string{}
	envs = append(envs, fmt.Sprintf("WORKERS=%s", okdRunOpts.workers))
	envs = append(envs, fmt.Sprintf("MASTER_MEMORY=%s", okdRunOpts.masterMemory))
	envs = append(envs, fmt.Sprintf("MASTER_CPU=%s", okdRunOpts.masterCpu))
	envs = append(envs, fmt.Sprintf("WORKERS_MEMORY=%s", okdRunOpts.workersMemory))
	envs = append(envs, fmt.Sprintf("WORKERS_CPU=%s", okdRunOpts.workersCpu))
	envs = append(envs, fmt.Sprintf("NUM_SECONDARY_NICS=%d", okdRunOpts.secondaryNics))

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

	log := cOpts.GetLogger()
	hnd, err := podman.NewHandle(ctx, cOpts.PodmanSocket, log)
	if err != nil {
		return err
	}

	log.Noticef("downloading all the images needed for %s (from %s)", cluster, cOpts.Registry)
	err = hnd.PullClusterImages(okdRunOpts, cOpts.Registry, cluster)
	if err != nil || okdRunOpts.downloadOnly {
		return err
	}
	log.Infof("downloaded all the images needed for %s, bringing cluster up", cluster)

	ldgr := ledger.NewLedger(hnd, cmd.OutOrStderr(), log)

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

	clusterContainerName := cOpts.Prefix + "-cluster"
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
		Volume:     &[]string{okdRunOpts.volume},
	})
	if err != nil {
		return err
	}

	clusterNetwork := fmt.Sprintf("container:%s", clusterID)

	err = cmdutil.SetupRegistry(ldgr, cOpts.Prefix, clusterNetwork, okdRunOpts.registryVolume, okdRunOpts.privileged)
	if err != nil {
		return err
	}
	if okdRunOpts.nfsData != "" {
		err = cmdutil.SetupNFS(ldgr, cOpts.Prefix, clusterNetwork, okdRunOpts.nfsData, okdRunOpts.privileged)
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
