package controllers

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"real-time-video-surveillance-system/services"
)

// Run only in a purpose-built isolated container with disposable recordings.
func TestLocalStorageSourceRemovalDiagnostic(t *testing.T) {
	endpoint := os.Getenv("TEST_MEDIAMTX_ENDPOINT")
	if endpoint == "" {
		t.Skip("requires isolated MediaMTX fixture")
	}
	streams := services.NewStreamService("http://localhost", "test-signing", "unused")
	svc := services.NewRecordingService(endpoint, "http://localhost", streams)
	rows, err := svc.List(t.Context(), "truck001", "cam01", "", "")
	if err != nil || len(rows) == 0 {
		t.Fatalf("list before move: %v %v", rows, err)
	}
	var row services.RecordingSpan
	for _, candidate := range rows {
		if candidate.Complete {
			row = candidate
			break
		}
	}
	if row.Start.IsZero() {
		t.Fatal("no uploaded recording found")
	}
	token := streams.SignRecordingAccess("truck001", "cam01", row.Start, row.Duration, time.Now().Add(time.Hour))
	invoke := func(download bool, ranges string) *httptest.ResponseRecorder {
		q := url.Values{"start": {row.Start.Format(time.RFC3339Nano)}, "duration": {strconv.FormatFloat(row.Duration, 'f', -1, 64)}, "token": {token}}
		if download {
			q.Set("download", "1")
		}
		r := httptest.NewRequest("GET", "/?"+q.Encode(), nil)
		r.SetPathValue("truckID", "truck001")
		r.SetPathValue("cameraID", "cam01")
		if ranges != "" {
			r.Header.Set("Range", ranges)
		}
		w := httptest.NewRecorder()
		NewRecordingController(nil, svc, streams).Content(w, r)
		return w
	}
	if w := invoke(false, "bytes=0-1023"); w.Code != 206 || w.Body.Len() != 1024 {
		t.Fatalf("range before move: %d %d", w.Code, w.Body.Len())
	}
	if w := invoke(true, ""); w.Code != 200 || w.Body.Len() == 0 {
		t.Fatalf("download before move: %d %d", w.Code, w.Body.Len())
	}
	timeName := row.Start.UTC().Format("2006-01-02_15-04-05")
	name := timeName + "-" + fmt.Sprintf("%06d", row.Start.Nanosecond()/1000) + ".mp4"
	source := filepath.Join(os.Getenv("RECORDINGS_DIR"), "truck001", "cam01", "main", name)
	moved := source + ".moved-for-diagnostic"
	if err := os.Rename(source, moved); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Rename(moved, source); err != nil {
			t.Errorf("restore test source: %v", err)
		}
	})
	rows, err = svc.List(t.Context(), "truck001", "cam01", "", "")
	if err != nil {
		t.Fatalf("list after move: %v", err)
	}
	found := false
	for _, r := range rows {
		if r.Start.Equal(row.Start) && r.Complete {
			found = true
		}
	}
	if !found {
		t.Fatal("uploaded segment missing after move")
	}
	if w := invoke(false, "bytes=0-1023"); w.Code != 206 || w.Body.Len() != 1024 {
		t.Fatalf("range after move: %d %d", w.Code, w.Body.Len())
	}
	if w := invoke(true, ""); w.Code != 200 || w.Body.Len() == 0 {
		t.Fatalf("download after move: %d %d", w.Code, w.Body.Len())
	}
	t.Log("after moving isolated local source: listing, range playback and download succeeded")
}
