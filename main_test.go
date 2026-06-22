package main

import (
	"strings"
	"testing"
	"time"
)

func TestSignStreamAccessVariesByQuality(t *testing.T) {
	expires := time.Unix(1_800_000_000, 0)
	mainToken := signStreamAccess("truck001", "cam01", "main", expires)
	subToken := signStreamAccess("truck001", "cam01", "sub", expires)

	if mainToken == subToken {
		t.Fatal("main and sub streams must not share a token")
	}
	if parts := strings.Split(mainToken, "."); len(parts) != 2 {
		t.Fatalf("token should contain payload and signature, got %d parts", len(parts))
	}
}

func TestCameraStreamURL(t *testing.T) {
	url := cameraStreamURL("truck 1", "cam/01", "sub")
	if !strings.Contains(url, "truck%201/cam%2F01/sub/whep") {
		t.Fatalf("unexpected stream URL: %s", url)
	}
}

func TestValidateStreamAccess(t *testing.T) {
	expires := time.Now().Add(time.Minute)
	token := signStreamAccess("truck001", "cam01", "main", expires)
	claims, err := validateStreamAccess(token, time.Now())
	if err != nil {
		t.Fatalf("valid token was rejected: %v", err)
	}
	if claims.TruckID != "truck001" || claims.CameraID != "cam01" || claims.Quality != "main" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if _, err := validateStreamAccess(token+"x", time.Now()); err == nil {
		t.Fatal("tampered token was accepted")
	}
}

func TestValidateStreamAccessRejectsExpiredToken(t *testing.T) {
	token := signStreamAccess("truck001", "cam01", "sub", time.Now().Add(-time.Second))
	if _, err := validateStreamAccess(token, time.Now()); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestStreamPathParts(t *testing.T) {
	truckID, cameraID, quality, ok := streamPathParts("truck001/cam08/sub")
	if !ok || truckID != "truck001" || cameraID != "cam08" || quality != "sub" {
		t.Fatalf("unexpected stream path: %q %q %q %v", truckID, cameraID, quality, ok)
	}
	if _, _, _, ok := streamPathParts("truck001/cam08/unknown"); ok {
		t.Fatal("unknown stream quality was accepted")
	}
}

func TestMediaMTXRecordingURL(t *testing.T) {
	start := time.Date(2026, time.June, 22, 12, 30, 0, 123456000, time.FixedZone("UTC+8", 8*60*60))
	result := mediaMTXRecordingURL("https://video.example.com/", "truck001", "cam01", start, 60.5)
	for _, expected := range []string{"https://video.example.com/get?", "duration=60.5", "format=mp4", "path=truck001%2Fcam01%2Fmain", ".123456"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("recording URL %q does not contain %q", result, expected)
		}
	}
}

func TestRecordingAPIURL(t *testing.T) {
	start := time.Date(2026, time.June, 22, 12, 30, 0, 0, time.UTC)
	result := recordingAPIURL("truck001", "cam01", start, 30, "signed.token")
	for _, expected := range []string{"/api/trucks/truck001/cameras/cam01/recordings/content?", "duration=30", "token=signed.token"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("recording API URL %q does not contain %q", result, expected)
		}
	}
}
