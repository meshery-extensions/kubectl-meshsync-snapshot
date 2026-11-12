package errors

import (
	"fmt"

	"github.com/meshery/meshkit/errors"
)

var (
	ErrInstantiateLoggerCode      = "meshsync-snapshot-1001"
	ErrRunningMeshSyncCode        = "meshsync-snapshot-1002"
	ErrReadingSnapshotFileCode    = "meshsync-snapshot-1003"
	ErrUploadingSnapshotCode      = "meshsync-snapshot-1004"
	ErrHTTPPostRequestCode        = "meshsync-snapshot-1005"
	ErrDecodingAPICode            = "meshsync-snapshot-1006"
	ErrUnexpectedResponseCodeCode = "meshsync-snapshot-1007"
)

func ErrInstantiateLogger(err error) error {
	return errors.New(ErrInstantiateLoggerCode, errors.Alert,
		[]string{"Could not instantiate logger."},
		[]string{err.Error()},
		[]string{"Logger creation failed during command startup."},
		[]string{"Check logging configuration and environment options."},
	)
}

func ErrRunningMeshSync(err error) error {
	return errors.New(ErrRunningMeshSyncCode, errors.Alert,
		[]string{"Failed to run MeshSync cluster snapshot collection."},
		[]string{err.Error()},
		[]string{"MeshSync library encountered an error while discovering cluster resources."},
		[]string{"Verify Kubernetes cluster connectivity via kubeconfig and check user RBAC permissions."},
	)
}

func ErrReadingSnapshotFile(err error) error {
	return errors.New(ErrReadingSnapshotFileCode, errors.Alert,
		[]string{"Failed to read generated cluster snapshot file."},
		[]string{err.Error()},
		[]string{"The snapshot YAML file could not be read for uploading."},
		[]string{"Ensure write permissions in current directory and verify file existence."},
	)
}

func ErrUploadingSnapshot(err error) error {
	return errors.New(ErrUploadingSnapshotCode, errors.Alert,
		[]string{"Failed to upload snapshot to Meshery Server."},
		[]string{err.Error()},
		[]string{"HTTP request to Meshery API pattern import endpoint failed."},
		[]string{"Check Meshery Server URL, provider authentication token, and network connectivity."},
	)
}

func ErrHTTPPostRequest(err error) error {
	return errors.New(ErrHTTPPostRequestCode, errors.Alert,
		[]string{"Failed to perform HTTP POST request."},
		[]string{err.Error()},
		[]string{"HTTP request to Meshery API failed during communication."},
		[]string{"Verify Meshery API URL and server availability."},
	)
}

func ErrDecodingAPI(err error) error {
	return errors.New(ErrDecodingAPICode, errors.Alert,
		[]string{"Failed to decode API response."},
		[]string{err.Error()},
		[]string{"Response from Meshery API could not be parsed into expected format."},
		[]string{"Ensure Meshery Server version is compatible with design import API."},
	)
}

func ErrUnexpectedResponseCode(statusCode int, body string) error {
	return errors.New(ErrUnexpectedResponseCodeCode, errors.Alert,
		[]string{"Received unexpected response code from Meshery Server."},
		[]string{fmt.Sprintf("Status Code: %d, Response: %s", statusCode, body)},
		[]string{"The server returned an unexpected HTTP status code."},
		[]string{"Verify your provider authentication token and ensure valid design data."},
	)
}
