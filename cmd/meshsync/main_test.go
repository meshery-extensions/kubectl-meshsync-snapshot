package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meshery/meshkit/logger"
)

func TestGenerateDefaultFileName(t *testing.T) {
	name := GenerateDefaultFileName()
	if !strings.HasPrefix(name, "meshery-cluster-snapshot-") {
		t.Errorf("expected filename to start with prefix, got %s", name)
	}
	if !strings.HasSuffix(name, ".yaml") {
		t.Errorf("expected filename to end with .yaml, got %s", name)
	}
}

func TestBuildRootCommandFlags(t *testing.T) {
	cmd := buildRootCommand()

	flags := []string{"file", "duration", "resources", "namespace", "upload", "meshery-url", "token", "design-name"}
	for _, f := range flags {
		flag := cmd.Flags().Lookup(f)
		if flag == nil {
			t.Errorf("missing expected flag --%s", f)
		}
	}

	durationFlag := cmd.Flags().Lookup("duration")
	if durationFlag.DefValue != "8s" {
		t.Errorf("expected default duration 8s, got %s", durationFlag.DefValue)
	}
}

func TestUploadSnapshotToMeshery(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pattern/import" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		resp := []map[string]interface{}{
			{
				"id":   "mock-cluster-design-id-1234",
				"name": "cluster-snapshot",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	snapshotFile := filepath.Join(tmpDir, "snapshot.yaml")
	if err := os.WriteFile(snapshotFile, []byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: default"), 0644); err != nil {
		t.Fatalf("failed to write tmp file: %v", err)
	}

	testLogger, err := logger.New("test", logger.Options{Format: logger.SyslogLogFormat})
	if err != nil {
		t.Fatalf("failed to init logger: %v", err)
	}

	id, err := UploadSnapshotToMeshery(mockServer.URL, "mock-token", snapshotFile, "my-cluster", testLogger)
	if err != nil {
		t.Fatalf("UploadSnapshotToMeshery failed: %v", err)
	}
	if id != "mock-cluster-design-id-1234" {
		t.Errorf("unexpected design id: got %s, want mock-cluster-design-id-1234", id)
	}
}

func TestUploadSnapshotToMeshery_SingleObject(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "mock-single-id-5678",
		})
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	snapshotFile := filepath.Join(tmpDir, "snapshot.yaml")
	_ = os.WriteFile(snapshotFile, []byte("test: data"), 0644)

	testLogger, _ := logger.New("test", logger.Options{Format: logger.SyslogLogFormat})
	id, err := UploadSnapshotToMeshery(mockServer.URL, "mock-token", snapshotFile, "my-cluster", testLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "mock-single-id-5678" {
		t.Errorf("expected mock-single-id-5678, got %s", id)
	}
}

func TestUploadSnapshotToMeshery_MalformedResponse(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>Bad Gateway</body></html>"))
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	snapshotFile := filepath.Join(tmpDir, "snapshot.yaml")
	_ = os.WriteFile(snapshotFile, []byte("test: data"), 0644)

	testLogger, _ := logger.New("test", logger.Options{Format: logger.SyslogLogFormat})
	_, err := UploadSnapshotToMeshery(mockServer.URL, "mock-token", snapshotFile, "my-cluster", testLogger)
	if err == nil {
		t.Fatalf("expected error for malformed response, got nil")
	}
}

func TestUploadSnapshotToMeshery_ServerError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	snapshotFile := filepath.Join(tmpDir, "snapshot.yaml")
	_ = os.WriteFile(snapshotFile, []byte("test: data"), 0644)

	testLogger, _ := logger.New("test", logger.Options{Format: logger.SyslogLogFormat})
	_, err := UploadSnapshotToMeshery(mockServer.URL, "mock-token", snapshotFile, "my-cluster", testLogger)
	if err == nil {
		t.Fatalf("expected error for 401 response, got nil")
	}
}

func TestUploadSnapshotToMeshery_ValidResponseWithoutID(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	snapshotFile := filepath.Join(tmpDir, "snapshot.yaml")
	_ = os.WriteFile(snapshotFile, []byte("test: data"), 0644)

	testLogger, _ := logger.New("test", logger.Options{Format: logger.SyslogLogFormat})
	id, err := UploadSnapshotToMeshery(mockServer.URL, "mock-token", snapshotFile, "my-cluster", testLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "" {
		t.Errorf("expected empty id, got %s", id)
	}
}
