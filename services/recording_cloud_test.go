package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Exercise the official client and recording service over a local HTTP fixture.
func TestCloudRecordingListAndRangeAfterSourceRemoval(t *testing.T) {
	content := []byte("local cloud test recording bytes")
	checksum := strings.Repeat("a", 64)
	var mu sync.Mutex
	var mediaRanges []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("alt") == "media" || strings.HasPrefix(r.URL.Path, "/local-test/") {
			mu.Lock()
			mediaRanges = append(mediaRanges, r.Header.Get("Range"))
			mu.Unlock()
			w.Header().Set("Content-Type", "video/mp4")
			http.ServeContent(w, r, "recording.mp4", time.Time{}, bytes.NewReader(content))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"name":"truck001/cam01/main/2026-11-20_11-00-00-000000.mp4","bucket":"local-test","size":"%d","generation":"42","metadata":{"sha256":"%s"}}`, len(content), checksum)
	}))
	defer server.Close()
	t.Setenv("STORAGE_EMULATOR_HOST", server.URL)
	t.Setenv("GCS_BUCKET", "local-test")
	t.Setenv("SIMULATED_STORAGE", "")
	root := t.TempDir()
	t.Setenv("UPLOAD_STATE_DIR", root)
	state := filepath.Join(root, "truck001", "cam01", "main", "2026-11-20_11-00-00-000000.mp4.json")
	if err := os.MkdirAll(filepath.Dir(state), 0755); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"status": "uploaded", "size": len(content), "sha256": checksum, "durationSeconds": 10, "generation": 42})
	if err := os.WriteFile(state, b, 0644); err != nil {
		t.Fatal(err)
	}
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "offline", 503) }))
	defer media.Close()
	s := NewRecordingService(media.URL, "http://localhost", NewStreamService("http://localhost", "test", "unused"))
	rows, err := s.List(context.Background(), "truck001", "cam01", "", "")
	if err != nil || len(rows) != 1 || !rows[0].PlayReady {
		t.Fatalf("list: %+v, %v", rows, err)
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Range", "bytes=2-8")
	w := httptest.NewRecorder()
	if err := s.Serve(w, r, "truck001", "cam01", time.Date(2026, 11, 20, 11, 0, 0, 0, time.UTC), 10, ""); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusPartialContent || w.Body.String() != string(content[2:9]) {
		t.Fatalf("range: %d %q", w.Code, w.Body.String())
	}
	mu.Lock()
	if len(mediaRanges) != 1 || mediaRanges[0] != "bytes=2-8" {
		t.Fatalf("storage read ranges: %v", mediaRanges)
	}
	mu.Unlock()
	w = httptest.NewRecorder()
	if err := s.Serve(w, httptest.NewRequest(http.MethodGet, "/", nil), "truck001", "cam01", rows[0].Start, 10, ""); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || w.Body.String() != string(content) {
		t.Fatalf("download: %d %q", w.Code, w.Body.String())
	}
}

func TestCloudByteRange(t *testing.T) {
	for _, tc := range []struct {
		header         string
		offset, length int64
		partial, bad   bool
	}{
		{"", 0, 100, false, false}, {"bytes=10-19", 10, 10, true, false}, {"bytes=95-", 95, 5, true, false},
		{"bytes=-7", 93, 7, true, false}, {"bytes=-999", 0, 100, true, false}, {"bytes=100-", 0, 0, false, true},
		{"bytes=-0", 0, 100, false, true}, {"bytes=1-2,5-6", 0, 100, false, false}, {"bytes=bad", 0, 100, false, false},
	} {
		o, n, p, b := cloudByteRange(tc.header, 100)
		if o != tc.offset || n != tc.length || p != tc.partial || b != tc.bad {
			t.Errorf("%q got %d %d %v %v", tc.header, o, n, p, b)
		}
	}
}
