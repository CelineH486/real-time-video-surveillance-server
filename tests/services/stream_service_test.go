package services_test

import (
	"strings"
	"testing"
	"time"

	"real-time-video-surveillance-system/services"
)

func testStreamService() *services.StreamService {
	return services.NewStreamService("http://localhost:8889", "test-signing-key", "test-publish-password")
}

func TestSignAccessVariesByQuality(t *testing.T) {
	service := testStreamService()
	expires := time.Unix(1_800_000_000, 0)
	mainToken := service.SignAccess("truck001", "cam01", "main", expires)
	subToken := service.SignAccess("truck001", "cam01", "sub", expires)
	if mainToken == subToken {
		t.Fatal("main and sub streams must not share a token")
	}
	if parts := strings.Split(mainToken, "."); len(parts) != 2 {
		t.Fatalf("token should contain payload and signature, got %d parts", len(parts))
	}
}

func TestCameraURL(t *testing.T) {
	result := testStreamService().CameraURL("truck 1", "cam/01", "sub")
	if !strings.Contains(result, "truck%201/cam%2F01/sub/whep") {
		t.Fatalf("unexpected stream URL: %s", result)
	}
}

func TestValidateAccess(t *testing.T) {
	service := testStreamService()
	expires := time.Now().Add(time.Minute)
	token := service.SignAccess("truck001", "cam01", "main", expires)
	claims, err := service.ValidateAccess(token, time.Now())
	if err != nil {
		t.Fatalf("valid token was rejected: %v", err)
	}
	if claims.TruckID != "truck001" || claims.CameraID != "cam01" || claims.Quality != "main" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if _, err := service.ValidateAccess(token+"x", time.Now()); err == nil {
		t.Fatal("tampered token was accepted")
	}
}

func TestValidateAccessRejectsExpiredToken(t *testing.T) {
	service := testStreamService()
	token := service.SignAccess("truck001", "cam01", "sub", time.Now().Add(-time.Second))
	if _, err := service.ValidateAccess(token, time.Now()); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestRecordingAccessIsSeparateAndBoundToTimeRange(t *testing.T) {
	service := testStreamService()
	start := time.Date(2026, time.September, 30, 2, 0, 0, 123456000, time.UTC)
	expires := services.RecordingAccessExpiry(start, 1800)
	token := service.SignRecordingAccess("truck001", "cam01", start, 1800, expires)

	claims, err := service.ValidateRecordingAccess(token, start)
	if err != nil {
		t.Fatalf("valid recording token was rejected: %v", err)
	}
	if !claims.Allows("truck001", "cam01", start, 1800) {
		t.Fatal("recording token did not allow its complete recording range")
	}
	if !claims.Allows("truck001", "cam01", start.Add(10*time.Minute), 5*60) {
		t.Fatal("recording token did not allow a contained seek range")
	}
	if claims.Allows("truck001", "cam02", start, 1800) {
		t.Fatal("recording token allowed another camera")
	}
	if claims.Allows("truck001", "cam01", start.Add(-time.Second), 1800) {
		t.Fatal("recording token allowed a range before the recording")
	}
	if claims.Allows("truck001", "cam01", start.Add(10*time.Minute), 21*60) {
		t.Fatal("recording token allowed a range after the recording")
	}
	if _, err := service.ValidateAccess(token, start); err == nil {
		t.Fatal("recording token was accepted as a live stream token")
	}
}

func TestRecordingAccessRejectsLiveAndExpiredTokens(t *testing.T) {
	service := testStreamService()
	now := time.Now()
	liveToken := service.SignAccess("truck001", "cam01", "main", now.Add(time.Hour))
	if _, err := service.ValidateRecordingAccess(liveToken, now); err == nil {
		t.Fatal("live stream token was accepted as a recording token")
	}
	recordingToken := service.SignRecordingAccess("truck001", "cam01", now, 1800, now.Add(-time.Second))
	if _, err := service.ValidateRecordingAccess(recordingToken, now); err == nil {
		t.Fatal("expired recording token was accepted")
	}
}

func TestRecordingAccessExpiryCoversPlaybackAndBuffer(t *testing.T) {
	now := time.Date(2026, time.September, 30, 2, 0, 0, 0, time.UTC)
	if got, want := services.RecordingAccessExpiry(now, 1800), now.Add(40*time.Minute); !got.Equal(want) {
		t.Fatalf("unexpected recording token expiry: got %s want %s", got, want)
	}
}

func TestStreamPathParts(t *testing.T) {
	truckID, cameraID, quality, ok := services.StreamPathParts("truck001/cam08/sub")
	if !ok || truckID != "truck001" || cameraID != "cam08" || quality != "sub" {
		t.Fatalf("unexpected stream path: %q %q %q %v", truckID, cameraID, quality, ok)
	}
	if _, _, _, ok := services.StreamPathParts("truck001/cam08/unknown"); ok {
		t.Fatal("unknown stream quality was accepted")
	}
}

func TestPublishCredentialsValid(t *testing.T) {
	service := testStreamService()
	if !service.PublishCredentialsValid("truck001", "test-publish-password", "truck001") {
		t.Fatal("valid publish credentials were rejected")
	}
	if service.PublishCredentialsValid("truck002", "test-publish-password", "truck001") {
		t.Fatal("publisher was allowed to publish for another truck")
	}
}
