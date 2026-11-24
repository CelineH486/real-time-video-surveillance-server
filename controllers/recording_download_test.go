package controllers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"real-time-video-surveillance-system/services"
)

func TestRecordingDownloadReturnsCompleteSelectedSource(t *testing.T) {
	for _, source := range []string{"main"} {
		t.Run(source, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("RECORDINGS_DIR", root)
			statusRoot := t.TempDir()
			t.Setenv("RECORDING_STATUS_DIR", statusRoot)
			path := filepath.Join(root, "truck001", "cam01", "main", "2026-11-10_01-02-03-000000.mp4")
			marker := filepath.Join(statusRoot, "truck001", "cam01", "main", "2026-11-10_01-02-03-000000.mp4.complete")
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("abc"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(marker), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(marker, []byte("3\n"), 0644); err != nil {
				t.Fatal(err)
			}
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/list" {
					fmt.Fprint(w, `[{"start":"2026-11-10T01:02:03Z","duration":1800}]`)
					return
				}
				if r.URL.Query().Get("path") != "truck001/cam01/"+source || r.URL.Query().Get("duration") != "1800" || r.URL.Query().Get("format") != "mp4" {
					t.Errorf("unexpected upstream request: %s", r.URL.String())
				}
				fmt.Fprint(w, "complete mp4")
			}))
			defer upstream.Close()
			streams := services.NewStreamService("http://localhost", "test-signing-key", "test-password")
			recordings := services.NewRecordingService(upstream.URL, "http://localhost", streams)
			controller := NewRecordingController(nil, recordings, streams)
			start := time.Date(2026, 11, 10, 1, 2, 3, 0, time.UTC)
			token := streams.SignRecordingAccess("truck001", "cam01", start, 1800, time.Now().Add(time.Hour))
			query := url.Values{"start": {start.Format(time.RFC3339)}, "duration": {"1800"}, "source": {source}, "token": {token}, "download": {"1"}}
			request := httptest.NewRequest(http.MethodGet, "/?"+query.Encode(), nil)
			request.SetPathValue("truckID", "truck001")
			request.SetPathValue("cameraID", "cam01")
			response := httptest.NewRecorder()
			controller.Content(response, request)
			if response.Code != 200 || response.Body.String() != "complete mp4" || response.Header().Get("Content-Type") != "video/mp4" || !strings.Contains(response.Header().Get("Content-Disposition"), "attachment;") {
				t.Fatalf("unexpected download: status=%d headers=%v body=%q", response.Code, response.Header(), response.Body.String())
			}
			if err := os.Remove(marker); err != nil {
				t.Fatal(err)
			}
			response = httptest.NewRecorder()
			controller.Content(response, request)
			if response.Code != http.StatusConflict {
				t.Fatalf("unfinished segment accepted: %d", response.Code)
			}
		})
	}
}
