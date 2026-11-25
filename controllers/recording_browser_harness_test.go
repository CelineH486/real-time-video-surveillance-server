package controllers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"real-time-video-surveillance-system/services"
)

// Manual browser fixture for the real API content handler, using only a
// simulated Storage copy; the local recording source is deliberately absent.
func TestStorageBrowserHarness(t *testing.T) {
	if os.Getenv("TEST_STORAGE_BROWSER") != "true" {
		t.Skip("manual local browser fixture")
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "[]") }))
	defer upstream.Close()
	streams := services.NewStreamService("http://localhost", "browser-test-key", "unused")
	svc := services.NewRecordingService(upstream.URL, "http://localhost", streams)
	rows, err := svc.List(t.Context(), "truck001", "cam01", "", "")
	if err != nil || len(rows) == 0 {
		t.Fatalf("no uploaded recordings: %v", err)
	}
	row := rows[0]
	token := streams.SignRecordingAccess("truck001", "cam01", row.Start, row.Duration, time.Now().Add(time.Hour))
	q := url.Values{"start": {row.Start.Format(time.RFC3339Nano)}, "duration": {strconv.FormatFloat(row.Duration, 'f', -1, 64)}, "token": {token}}
	controller := NewRecordingController(nil, svc, streams)
	mux := http.NewServeMux()
	mux.HandleFunc("/video", func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("truckID", "truck001")
		r.SetPathValue("cameraID", "cam01")
		controller.Content(w, r)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><meta charset="utf-8"><title>Storage playback diagnostic</title><h1>Simulated Storage playback</h1><video id="v" controls width="640" height="480" src="/video?%s"></video><p id="state">Loading</p><button onclick="document.getElementById('v').play()">Play</button><script>let v=document.getElementById('v');setInterval(()=>{document.getElementById('state').textContent='Time '+v.currentTime.toFixed(1)+' / '+v.duration.toFixed(1)+'; '+(v.paused?'paused':'playing')},200)</script>`, q.Encode())
	})
	if err := http.ListenAndServe(":18766", mux); err != nil {
		t.Fatal(err)
	}
}
