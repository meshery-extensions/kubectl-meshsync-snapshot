package main

import (
	"fmt"
	"os"
	"time"

	"github.com/layer5io/meshkit/logger"
	libmeshsync "github.com/meshery/meshsync/pkg/lib/meshsync"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	var namespaces []string
	var resources []string
	var outputFile string
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
				libmeshsync.WithOutputMode("file"),
				libmeshsync.WithOutputFileName(outputFile),
				libmeshsync.WithStopAfterDuration(duration),
				libmeshsync.WithOnlyK8sResources(resources),
				libmeshsync.WithOnlyK8sNamespaces(namespaces...),
			); err != nil {
				return fmt.Errorf("error running meshsync lib %v", err)
			}
			log.Info("done")

			return nil
		},
	}

	rootCmd.Flags().StringVarP(
		&outputFile, "file", "f", "",
		"Name of the output file.\nIf not specified, the file will be named: meshery-cluster-snapshot-YYYYMMDD-NN.yaml",
	)

	rootCmd.Flags().DurationVarP(
		&duration, "duration", "d", 8*time.Second,
		"Duration of event collection (e.g., 8ss, 12s, 1m)",
	)

	rootCmd.Flags().StringSliceVarP(
		&resources, "resources", "r", []string{},
		"Comma-separated list of Kubernetes resources to snapshot (case-insensitive).\nFor example: \"pod,deployment,service\"",
	)

	rootCmd.Flags().StringSliceVarP(
		&namespaces, "namespace", "n", []string{},
		"One or more namespaces to restrict the snapshot to.\nFor example: \"default,agile-otter\"\nIf not specified, events will be collected from all namespaces",
	)

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
