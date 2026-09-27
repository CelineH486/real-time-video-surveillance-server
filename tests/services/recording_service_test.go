package services_test

import (
	"context"
	"encoding/json"
	"io"
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
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("complete mp4"))
	}))
	defer server.Close()

	service := services.NewRecordingService(server.URL, "http://localhost:8080", testStreamService())
	start := time.Date(2026, time.June, 22, 12, 30, 0, 123456000, time.FixedZone("UTC+8", 8*60*60))
	file, err := service.Open(context.Background(), "truck001", "cam01", start, 60.5, "signed.token")
	if err != nil {
		t.Fatalf("Open returned an error: %v", err)
	}
	defer file.Close()

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
	content, err := io.ReadAll(file)
	if err != nil || string(content) != "complete mp4" {
		t.Fatalf("unexpected cached recording: content=%q err=%v", content, err)
	}
}

func TestRecordingOpenReusesCachedMP4(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests++
		_, _ = w.Write([]byte("cached mp4"))
	}))
	defer server.Close()

	service := services.NewRecordingService(server.URL, "http://localhost:8080", testStreamService())
	start := time.Date(2026, time.September, 27, 8, 19, 11, 0, time.UTC)
	for range 2 {
		file, err := service.Open(context.Background(), "truck001", "cam01", start, 1800, "signed.token")
		if err != nil {
			t.Fatal(err)
		}
		file.Close()
	}
	if requests != 1 {
		t.Fatalf("expected one MediaMTX request, got %d", requests)
	}
}

func TestRecordingServeSupportsByteRanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer server.Close()

	service := services.NewRecordingService(server.URL, "http://localhost:8080", testStreamService())
	request := httptest.NewRequest(http.MethodGet, "/recording.mp4", nil)
	request.Header.Set("Range", "bytes=2-5")
	response := httptest.NewRecorder()
	err := service.Serve(
		response,
		request,
		"truck001",
		"cam01",
		time.Date(2026, time.September, 27, 8, 19, 11, 0, time.UTC),
		1800,
		"signed.token",
	)
	if err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusPartialContent || response.Header().Get("Accept-Ranges") != "bytes" ||
		response.Header().Get("Content-Range") != "bytes 2-5/10" || response.Body.String() != "2345" {
		t.Fatalf("unexpected range response: status=%d headers=%v body=%q", response.Code, response.Header(), response.Body.String())
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
