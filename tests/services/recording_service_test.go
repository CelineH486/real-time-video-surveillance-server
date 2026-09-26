package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"real-time-video-surveillance-system/services"
)

func TestRecordingOpenBuildsMediaMTXRequest(t *testing.T) {
	var received *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		received = request
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", "bytes 100-199/1000")
		w.WriteHeader(http.StatusPartialContent)
	}))
	defer server.Close()

	service := services.NewRecordingService(server.URL, "http://localhost:8080", testStreamService())
	start := time.Date(2026, time.June, 22, 12, 30, 0, 123456000, time.FixedZone("UTC+8", 8*60*60))
	response, err := service.Open(context.Background(), "truck001", "cam01", start, 60.5, "signed.token", "bytes=100-199")
	if err != nil {
		t.Fatalf("Open returned an error: %v", err)
	}
	response.Body.Close()

	if received == nil {
		t.Fatal("MediaMTX did not receive a request")
	}
	query := received.URL.Query()
	if received.URL.Path != "/get" ||
		query.Get("path") != "truck001/cam01/main" ||
		query.Get("duration") != "60.5" ||
		query.Get("format") != "mp4" ||
		!strings.Contains(query.Get("start"), ".123456") {
		t.Fatalf("unexpected recording request URL: %s", received.URL.String())
	}
	if authorization := received.Header.Get("Authorization"); authorization != "Bearer signed.token" {
		t.Fatalf("unexpected authorization header: %q", authorization)
	}
	if byteRange := received.Header.Get("Range"); byteRange != "bytes=100-199" {
		t.Fatalf("unexpected range header: %q", byteRange)
	}
	if response.StatusCode != http.StatusPartialContent ||
		response.Header.Get("Content-Range") != "bytes 100-199/1000" {
		t.Fatalf("unexpected partial response: status=%d headers=%v", response.StatusCode, response.Header)
	}
}

func TestRecordingListRequestsPhysicalSegments(t *testing.T) {
	var received *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		received = request
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"start":"2026-09-16T03:24:00Z","duration":1800}]`))
	}))
	defer server.Close()

	service := services.NewRecordingService(server.URL, "http://localhost:8080", testStreamService())
	rows, err := service.List(context.Background(), "truck001", "cam01", "", "")
	if err != nil {
		t.Fatalf("List returned an error: %v", err)
	}
	if received == nil || received.URL.Query().Get("segments") != "true" {
		t.Fatalf("physical segments were not requested: %v", received)
	}
	if len(rows) != 1 || !rows[0].Start.Equal(time.Date(2026, 9, 16, 3, 24, 0, 0, time.UTC)) {
		t.Fatalf("unexpected recording rows: %#v", rows)
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "token") || strings.Contains(string(encoded), "url") {
		t.Fatalf("recording list exposed playback credentials: %s", encoded)
	}
}

func TestRecordingPublicURL(t *testing.T) {
	service := services.NewRecordingService("http://localhost:9996", "http://localhost:8080", testStreamService())
	start := time.Date(2026, time.June, 22, 12, 30, 0, 0, time.UTC)
	result := service.PublicURL("truck001", "cam01", start, 30, "signed.token")
	for _, expected := range []string{"/api/trucks/truck001/cameras/cam01/recordings/content?", "duration=30", "token=signed.token"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("recording API URL %q does not contain %q", result, expected)
		}
	}
}
