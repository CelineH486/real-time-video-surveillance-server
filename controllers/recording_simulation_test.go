package controllers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"real-time-video-surveillance-system/services"
	"strings"
	"testing"
	"time"
)

func TestSimulationGatesPlaybackAndDownloadAndReadsBucket(t *testing.T) {
	root := t.TempDir()
	markerRoot := t.TempDir()
	bucket := t.TempDir()
	states := t.TempDir()
	t.Setenv("SIMULATED_STORAGE", "true")
	t.Setenv("RECORDINGS_DIR", root)
	t.Setenv("RECORDING_STATUS_DIR", markerRoot)
	t.Setenv("SIMULATED_BUCKET_DIR", bucket)
	t.Setenv("UPLOAD_STATE_DIR", states)
	rel := filepath.Join("truck001", "cam01", "main", "2026-11-10_01-02-03-000000.mp4")
	put := func(path string, b []byte) {
		t.Helper()
		if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(path, b, 0644); e != nil {
			t.Fatal(e)
		}
	}
	put(filepath.Join(root, rel), []byte("source-video"))
	put(filepath.Join(markerRoot, rel+".complete"), []byte("12"))
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/list" {
			t.Errorf("unexpected media request: %s", r.URL.Path)
			w.WriteHeader(500)
			return
		}
		fmt.Fprint(w, `[{"start":"2026-11-10T01:02:03Z","duration":5}]`)
	}))
	defer upstream.Close()
	streams := services.NewStreamService("http://localhost", "test-key", "test-password")
	recordings := services.NewRecordingService(upstream.URL, "http://localhost", streams)
	controller := NewRecordingController(nil, recordings, streams)
	start := time.Date(2026, 11, 10, 1, 2, 3, 0, time.UTC)
	token := streams.SignRecordingAccess("truck001", "cam01", start, 5, time.Now().Add(time.Hour))
	call := func(download bool, rangeHeader string) *httptest.ResponseRecorder {
		t.Helper()
		q := url.Values{"start": {start.Format(time.RFC3339)}, "duration": {"5"}, "token": {token}}
		if download {
			q.Set("download", "1")
		}
		req := httptest.NewRequest("GET", "/?"+q.Encode(), nil)
		req.SetPathValue("truckID", "truck001")
		req.SetPathValue("cameraID", "cam01")
		if rangeHeader != "" {
			req.Header.Set("Range", rangeHeader)
		}
		out := httptest.NewRecorder()
		controller.Content(out, req)
		return out
	}
	rows, err := recordings.List(t.Context(), "truck001", "cam01", "", "")
	if err != nil || len(rows) != 1 || rows[0].Complete || rows[0].PlayReady {
		t.Fatalf("pending state: %+v %v", rows, err)
	}
	if call(false, "").Code != http.StatusConflict || call(true, "").Code != http.StatusConflict {
		t.Fatal("pending segment was accessible")
	}
	put(filepath.Join(states, rel+".json"), []byte(`{"status":"failed","size":12}`))
	rows, _ = recordings.List(t.Context(), "truck001", "cam01", "", "")
	if rows[0].UploadStatus != "failed" || rows[0].PlayReady {
		t.Fatal("failed state accessible")
	}
	put(filepath.Join(bucket, rel), []byte("bucket-video"))
	put(filepath.Join(states, rel+".json"), []byte(`{"status":"uploaded","size":12,"durationSeconds":5,"sha256":"`+strings.Repeat("a", 64)+`"}`))
	rows, err = recordings.List(t.Context(), "truck001", "cam01", "", "")
	if err != nil || !rows[0].Complete || !rows[0].PlayReady {
		t.Fatalf("uploaded state: %+v %v", rows, err)
	}
	out := call(false, "")
	if out.Code != 200 || out.Body.String() != "bucket-video" {
		t.Fatalf("playback did not use bucket: %d %q", out.Code, out.Body.String())
	}
	out = call(true, "bytes=0-5")
	if out.Code != 206 || out.Body.String() != "bucket" || !strings.Contains(out.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("download range: %d %q", out.Code, out.Body.String())
	}
	if e := os.Remove(filepath.Join(bucket, rel)); e != nil {
		t.Fatal(e)
	}
	if call(false, "").Code != http.StatusConflict {
		t.Fatal("missing object accessible")
	}
}
