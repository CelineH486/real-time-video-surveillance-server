package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"real-time-video-surveillance-system/services"
	objectstorage "real-time-video-surveillance-system/storage"
)

// The official client, worker, durable index and playback service use an isolated
// local HTTP fixture; no Google account, cloud bucket or real recording is used.
func TestCloudUploadCleanupAndPlaybackIntegration(t *testing.T) {
	const key = "truck001/cam01/main/2026-11-20_11-00-00-000000.mp4"
	var mu sync.Mutex
	var body []byte
	var metadata map[string]string
	failFirstUpload := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		respond := func() {
			crc := crc32.Checksum(body, crc32.MakeTable(crc32.Castagnoli))
			c := make([]byte, 4)
			binary.BigEndian.PutUint32(c, crc)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"name": key, "bucket": "local-test", "size": fmt.Sprint(len(body)), "generation": "42", "crc32c": base64.StdEncoding.EncodeToString(c), "metadata": metadata})
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Query().Get("uploadType") == "multipart":
			if failFirstUpload {
				failFirstUpload = false
				http.Error(w, "simulated upload denial", http.StatusForbidden)
				return
			}
			_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil {
				http.Error(w, "invalid multipart", 400)
				return
			}
			reader := multipart.NewReader(r.Body, params["boundary"])
			p, err := reader.NextPart()
			if err != nil {
				http.Error(w, "missing metadata", 400)
				return
			}
			var m struct {
				Metadata map[string]string `json:"metadata"`
			}
			if json.NewDecoder(p).Decode(&m) != nil {
				http.Error(w, "bad metadata", 400)
				return
			}
			metadata = m.Metadata
			p, err = reader.NextPart()
			if err != nil {
				http.Error(w, "missing data", 400)
				return
			}
			body, err = io.ReadAll(p)
			if err != nil {
				http.Error(w, "bad data", 400)
				return
			}
			respond()
		case strings.HasPrefix(r.URL.Path, "/storage/v1/"):
			if body == nil {
				http.NotFound(w, r)
				return
			}
			respond()
		case strings.HasPrefix(r.URL.Path, "/local-test/"):
			if body == nil {
				http.NotFound(w, r)
				return
			}
			http.ServeContent(w, r, "recording.mp4", time.Time{}, bytes.NewReader(body))
		default:
			http.Error(w, "unexpected request", 400)
		}
	}))
	defer server.Close()
	t.Setenv("STORAGE_EMULATOR_HOST", server.URL)
	t.Setenv("GCS_BUCKET", "local-test")
	t.Setenv("SIMULATED_STORAGE", "")
	root := t.TempDir()
	w := worker{recordings: filepath.Join(root, "recordings"), markers: filepath.Join(root, "markers"), states: filepath.Join(root, "states"), cleanupUploaded: true, probe: func(string) (float64, error) { return 12.5, nil }}
	t.Setenv("UPLOAD_STATE_DIR", w.states)
	source := filepath.Join(w.recordings, filepath.FromSlash(key))
	markerPath := filepath.Join(w.markers, filepath.FromSlash(key)+".complete")
	for _, p := range []string{filepath.Dir(source), filepath.Dir(markerPath)} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	video := []byte("test recording content")
	if fixture := os.Getenv("TEST_REAL_RECORDING"); fixture != "" {
		fixtureVideo, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatal(err)
		}
		video = fixtureVideo
		markerData, err := os.ReadFile(os.Getenv("TEST_REAL_MARKER"))
		if err != nil || strings.TrimSpace(string(markerData)) != fmt.Sprint(len(video)) {
			t.Fatalf("recording validation marker missing or incorrect: %v", err)
		}
		w.probe = nil // Run real ffprobe against the recorded MP4.
	}
	if err := os.WriteFile(source, video, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(markerPath, []byte(fmt.Sprint(len(video))), 0644); err != nil {
		t.Fatal(err)
	}
	cloud, err := objectstorage.NewCloud(context.Background(), "local-test")
	if err != nil {
		t.Fatal(err)
	}
	defer cloud.Close()
	w.cloud = cloud
	if err := w.process(markerPath); err == nil {
		t.Fatal("expected simulated upload failure")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("failed upload lost source: %v", err)
	}
	failedState, err := os.ReadFile(filepath.Join(w.states, filepath.FromSlash(key)+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var failed job
	if json.Unmarshal(failedState, &failed) != nil || failed.Status != "failed" {
		t.Fatalf("failure state: %s", failedState)
	}
	// Recreate worker and client to model a service restart before retry.
	if err := cloud.Close(); err != nil {
		t.Fatal(err)
	}
	cloud, err = objectstorage.NewCloud(context.Background(), "local-test")
	if err != nil {
		t.Fatal(err)
	}
	defer cloud.Close()
	w = worker{recordings: w.recordings, markers: w.markers, states: w.states, cleanupUploaded: true, probe: w.probe, cloud: cloud}
	if err := w.process(markerPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source not cleaned: %v", err)
	}
	if err := w.process(markerPath); err != nil {
		t.Fatalf("restart: %v", err)
	}
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "offline", 503) }))
	defer media.Close()
	s := services.NewRecordingService(media.URL, "http://localhost", services.NewStreamService("http://localhost", "test", "unused"))
	rows, err := s.List(context.Background(), "truck001", "cam01", "", "")
	if err != nil || len(rows) != 1 || !rows[0].PlayReady {
		t.Fatalf("list after cleanup: %+v %v", rows, err)
	}
	for _, tt := range []struct {
		rangeHeader string
		expected    int
		data        []byte
	}{{"", 200, video}, {"bytes=2-7", 206, video[2:8]}} {
		r := httptest.NewRequest("GET", "/", nil)
		if tt.rangeHeader != "" {
			r.Header.Set("Range", tt.rangeHeader)
		}
		out := httptest.NewRecorder()
		if err := s.Serve(out, r, "truck001", "cam01", rows[0].Start, rows[0].Duration, ""); err != nil {
			t.Fatal(err)
		}
		if out.Code != tt.expected || !bytes.Equal(out.Body.Bytes(), tt.data) {
			t.Fatalf("response: %d %q", out.Code, out.Body.String())
		}
	}
}
