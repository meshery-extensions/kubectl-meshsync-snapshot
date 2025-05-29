package main

import (
	"fmt"
	"os"
	"time"

	"github.com/layer5io/meshkit/logger"
	libmeshsync "github.com/n2h9/fork-meshery-meshsync/pkg/lib/meshsync"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	// var namespace string
	// var outputFile string
	var duration time.Duration

	// Root command for the plugin
	var rootCmd = &cobra.Command{
		Use:   "kubectl-meshery-cluster-snapshot",
		Short: "kubectl krew plugin to capture a cluster snapshot",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize Logger instance
			log, errLoggerNew := logger.New("kubectl-meshsync-snapshot", logger.Options{
				Format:   logger.SyslogLogFormat,
				LogLevel: int(logrus.InfoLevel),
			})
			if errLoggerNew != nil {
				// TODO meshkit errors
				return fmt.Errorf("could not instantiate logger: %v", errLoggerNew)
			}
			log.Info("startings meshsync lib...")
			if err := libmeshsync.Run(
				log,
				// TODO
				// add option to libmeshsync to specify
				// (the functionality is in place, need to add corresponding WithXXX function):
				// - output file name
				// - k8s namespace
				// - k8s resources list
				libmeshsync.WithOutputMode("file"),
				libmeshsync.WithStopAfterDuration(duration),
			); err != nil {
				return fmt.Errorf("error running meshsync lib %v", err)
			}
			log.Info("done")

			return nil
		},
	}

	// rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace to make a snapshot of (if not specified collects events from all namespaces)")
	// rootCmd.Flags().StringVarP(&outputFile, "outputFile", "o", "", "name of a result output file (if not specified produces file with name meshery-cluster-snapshot-YYYYMMDD-NN.yaml)")
	rootCmd.Flags().DurationVarP(&duration, "duration", "d", 8*time.Second, "duration of event collection")

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
