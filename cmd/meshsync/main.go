package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/meshery-extensions/kubectl-meshsync-snapshot/internal/errors"
	"github.com/meshery/meshkit/logger"
	libmeshsync "github.com/meshery/meshsync/pkg/lib/meshsync"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	namespaces  []string
	resources   []string
	outputFile  string
	duration    time.Duration
	upload      bool
	mesheryURL  string
	token       string
	designName  string
)

// GenerateDefaultFileName returns a timestamped filename when not provided
func GenerateDefaultFileName() string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("meshery-cluster-snapshot-%s.yaml", timestamp)
}

// UploadSnapshotToMeshery uploads the generated snapshot to Meshery Server
func UploadSnapshotToMeshery(apiURL, authToken, filePath, name string, log logger.Handler) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", errors.ErrReadingSnapshotFile(err)
	}

	if name == "" {
		base := filepath.Base(filePath)
		ext := filepath.Ext(base)
		name = strings.TrimSuffix(base, ext)
	}

	payload := map[string]interface{}{
		"save":     true,
		"file":     content,
		"fileName": filepath.Base(filePath),
		"name":     name,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", errors.ErrDecodingAPI(err)
	}

	endpoint := fmt.Sprintf("%s/api/pattern/import", strings.TrimRight(apiURL, "/"))
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", errors.ErrHTTPPostRequest(err)
	}

	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Cookie", fmt.Sprintf("token=%s;meshery-provider=Layer5", authToken))
	}
	req.Header.Set("Origin", apiURL)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.ErrUploadingSnapshot(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.ErrDecodingAPI(err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", errors.ErrUnexpectedResponseCode(resp.StatusCode, string(body))
	}

	var sliceResult []map[string]interface{}
	if jsonErr := json.Unmarshal(body, &sliceResult); jsonErr == nil && len(sliceResult) > 0 {
		if id, ok := sliceResult[0]["id"].(string); ok && id != "" {
			return id, nil
		}
	}

	var mapResult map[string]interface{}
	if jsonErr := json.Unmarshal(body, &mapResult); jsonErr == nil {
		if id, ok := mapResult["id"].(string); ok && id != "" {
			return id, nil
		}
	}

	return "", nil
}

func buildRootCommand() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "kubectl-meshsync-snapshot",
		Short: "Capture a cluster snapshot using MeshSync",
		Long: `A kubectl plugin via Krew that captures an ad-hoc snapshot of Kubernetes cluster resources
using the MeshSync library, and optionally imports it into Meshery Server for visualization.

Examples:
  # Capture all resources across all namespaces into a default timestamped file
  kubectl meshsync-snapshot

  # Capture resources in a specific namespace for 15 seconds
  kubectl meshsync-snapshot -n default -d 15s -f cluster-snapshot.yaml

  # Capture selective resource types
  kubectl meshsync-snapshot -r pod,deployment,service -n default

  # Capture snapshot and automatically import into Meshery
  kubectl meshsync-snapshot --upload --meshery-url https://playground.meshery.io --token <token>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log, errLoggerNew := logger.New("kubectl-meshsync-snapshot", logger.Options{
				Format:   logger.TerminalLogFormat,
				LogLevel: int(logrus.InfoLevel),
			})
			if errLoggerNew != nil {
				return errors.ErrInstantiateLogger(errLoggerNew)
			}

			if outputFile == "" {
				outputFile = GenerateDefaultFileName()
			}

			if mesheryURL == "" {
				mesheryURL = os.Getenv("MESHERY_API_BASE_URL")
				if mesheryURL == "" {
					mesheryURL = "https://playground.meshery.io"
				}
			}
			if token == "" {
				token = os.Getenv("PROVIDER_TOKEN")
			}

			log.Infof("Starting MeshSync snapshot collection for duration: %v...", duration)
			if len(namespaces) > 0 {
				log.Infof("Restricted to namespace(s): %s", strings.Join(namespaces, ", "))
			}
			if len(resources) > 0 {
				log.Infof("Restricted to resource(s): %s", strings.Join(resources, ", "))
			}

			if err := libmeshsync.Run(
				log,
				libmeshsync.WithOutputMode("file"),
				libmeshsync.WithOutputFileName(outputFile),
				libmeshsync.WithStopAfterDuration(duration),
				libmeshsync.WithOnlyK8sResources(resources),
				libmeshsync.WithOnlyK8sNamespaces(namespaces...),
			); err != nil {
				return errors.ErrRunningMeshSync(err)
			}

			log.Infof("✓ Cluster snapshot successfully written to: %s", outputFile)

			if upload {
				log.Infof("Uploading snapshot to Meshery Server at %s...", mesheryURL)
				designID, err := UploadSnapshotToMeshery(mesheryURL, token, outputFile, designName, log)
				if err != nil {
					log.Warnf("Warning: Could not upload snapshot to Meshery: %v", err)
				} else if designID != "" {
					log.Infof("✓ Snapshot imported into Meshery successfully! Design ID: %s", designID)
					log.Infof("View in Meshery: %s/extension/meshmap?design=%s", mesheryURL, designID)
				} else {
					log.Infof("✓ Snapshot imported into Meshery successfully.")
				}
			}

			return nil
		},
	}

	rootCmd.Flags().StringVarP(
		&outputFile, "file", "f", "",
		"Name of the output file (defaults to meshery-cluster-snapshot-YYYYMMDD-HHMMSS.yaml)",
	)

	rootCmd.Flags().DurationVarP(
		&duration, "duration", "d", 8*time.Second,
		"Duration of event collection (e.g., 8s, 15s, 1m)",
	)

	rootCmd.Flags().StringSliceVarP(
		&resources, "resources", "r", []string{},
		"Comma-separated list of Kubernetes resources to snapshot (e.g., pod,deployment,service)",
	)

	rootCmd.Flags().StringSliceVarP(
		&namespaces, "namespace", "n", []string{},
		"One or more namespaces to restrict the snapshot to (defaults to all namespaces)",
	)

	rootCmd.Flags().BoolVarP(
		&upload, "upload", "u", false,
		"Automatically upload the snapshot to Meshery Server after capture",
	)

	rootCmd.Flags().StringVar(
		&mesheryURL, "meshery-url", "",
		"Meshery Server API endpoint (default: https://playground.meshery.io or MESHERY_API_BASE_URL)",
	)

	rootCmd.Flags().StringVar(
		&token, "token", "",
		"Meshery provider token for authentication (or PROVIDER_TOKEN env var)",
	)

	rootCmd.Flags().StringVar(
		&designName, "design-name", "",
		"Name for the imported design in Meshery (optional)",
	)

	return rootCmd
}

func main() {
	cmd := buildRootCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
