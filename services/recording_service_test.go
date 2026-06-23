package services

import (
	"strings"
	"testing"
	"time"
)

func TestMediaMTXRecordingURL(t *testing.T) {
	service := NewRecordingService("https://video.example.com/", "http://localhost:8080", testStreamService())
	start := time.Date(2026, time.June, 22, 12, 30, 0, 123456000, time.FixedZone("UTC+8", 8*60*60))
	result := service.internalBaseURL + "/get?"
	queryResult := mediaMTXTestURL(service, "truck001", "cam01", start, 60.5)
	for _, expected := range []string{result, "duration=60.5", "format=mp4", "path=truck001%2Fcam01%2Fmain", ".123456"} {
		if !strings.Contains(queryResult, expected) {
			t.Fatalf("recording URL %q does not contain %q", queryResult, expected)
		}
	}
}

func TestRecordingPublicURL(t *testing.T) {
	service := NewRecordingService("http://localhost:9996", "http://localhost:8080", testStreamService())
	start := time.Date(2026, time.June, 22, 12, 30, 0, 0, time.UTC)
	result := service.PublicURL("truck001", "cam01", start, 30, "signed.token")
	for _, expected := range []string{"/api/trucks/truck001/cameras/cam01/recordings/content?", "duration=30", "token=signed.token"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("recording API URL %q does not contain %q", result, expected)
		}
	}
}

// mediaMTXTestURL mirrors the URL built by Open without issuing an HTTP request.
func mediaMTXTestURL(service *RecordingService, truckID, cameraID string, start time.Time, duration float64) string {
	query := mediaMTXRecordingQuery(truckID, cameraID, start, duration)
	return service.internalBaseURL + "/get?" + query.Encode()
}
