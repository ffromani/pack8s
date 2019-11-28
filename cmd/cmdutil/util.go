package cmdutil

import (
	"os"
	"strconv"

	logger "github.com/apsdehal/go-logger"

	"github.com/spf13/cobra"
)

type CommonOpts struct {
	Prefix       string
	PodmanSocket string
	Verbose      int
}

func GetCommonOpts(cmd *cobra.Command) (CommonOpts, error) {
	prefix, err := cmd.Flags().GetString("prefix")
	if err != nil {
		return CommonOpts{}, err
	}
	podmanSocket, err := cmd.Flags().GetString("podman-socket")
	if err != nil {
		return CommonOpts{}, err
	}
	verbose, err := cmd.Flags().GetInt("verbose")
	if err != nil {
		return CommonOpts{}, err
	}
	if val, ok := os.LookupEnv("PACK8S_VERBOSE"); ok {
		if v, err := strconv.Atoi(val); err == nil {
			verbose = v
		}
	}
	return CommonOpts{
		Prefix:       prefix,
		PodmanSocket: podmanSocket,
		Verbose:      verbose,
	}, nil
}

func NewLogger(lev, color int) *logger.Logger {
	log, err := logger.New("pack8s", color, toLogLevel(lev))
	if err != nil {
		panic(err)
	}
	log.Debugf("logger level: %v", lev)
	return log
}

func toLogLevel(lev int) logger.LogLevel {
	switch {
	// ALWAYS emit crit messages
	case lev <= 1:
		return logger.ErrorLevel
	case lev == 2:
		return logger.WarningLevel
	case lev == 3:
		return logger.NoticeLevel
	case lev == 4:
		return logger.InfoLevel
	case lev >= 5:
		return logger.DebugLevel
	}
	// should never be reached
	return logger.InfoLevel
}
