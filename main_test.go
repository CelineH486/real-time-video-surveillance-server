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
	url := cameraStreamURL("truck 1", "cam/01", "sub", "signed.token")
	if !strings.Contains(url, "truck%201/cam%2F01/sub?token=signed.token") {
		t.Fatalf("unexpected stream URL: %s", url)
	}
}
