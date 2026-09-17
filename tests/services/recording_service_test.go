package services_test

import (
	"context"
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
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := services.NewRecordingService(server.URL, "http://localhost:8080", testStreamService())
	start := time.Date(2026, time.June, 22, 12, 30, 0, 123456000, time.FixedZone("UTC+8", 8*60*60))
	response, err := service.Open(context.Background(), "truck001", "cam01", start, 60.5, "signed.token")
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
