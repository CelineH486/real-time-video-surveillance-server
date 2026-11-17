package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExpireCacheEntry_RemovesMatchingFile(t *testing.T) {
	// Arrange
	cacheKey := "expired-recording"
	cachedFilePath := filepath.Join(t.TempDir(), "recording.mp4")
	if writeError := os.WriteFile(cachedFilePath, []byte("recording-content"), 0o600); writeError != nil {
		t.Fatalf("create cached recording: %v", writeError)
	}
	recordingService := &RecordingService{cache: map[string]cachedRecording{
		cacheKey: {path: cachedFilePath, expiresAt: time.Now().Add(-time.Minute)},
	}}

	// Act
	recordingService.expireCacheEntry(cacheKey, cachedFilePath)

	// Assert
	if _, cacheEntryExists := recordingService.cache[cacheKey]; cacheEntryExists {
		t.Error("expired recording is still registered in the cache")
	}
	if _, fileError := os.Stat(cachedFilePath); !errors.Is(fileError, os.ErrNotExist) {
		t.Errorf("expired file status = %v, want file not found", fileError)
	}
}

func TestExpireCacheEntry_PreservesReplacementFile(t *testing.T) {
	// Arrange
	cacheKey := "replaced-recording"
	cacheDirectory := t.TempDir()
	oldFilePath := filepath.Join(cacheDirectory, "old.mp4")
	replacementFilePath := filepath.Join(cacheDirectory, "replacement.mp4")
	if writeError := os.WriteFile(replacementFilePath, []byte("replacement-content"), 0o600); writeError != nil {
		t.Fatalf("create replacement recording: %v", writeError)
	}
	recordingService := &RecordingService{cache: map[string]cachedRecording{
		cacheKey: {path: replacementFilePath, expiresAt: time.Now().Add(time.Minute)},
	}}

	// Act
	// A delayed expiry callback must not remove a newer entry under the same key.
	recordingService.expireCacheEntry(cacheKey, oldFilePath)

	// Assert
	if cachedRecording, cacheEntryExists := recordingService.cache[cacheKey]; !cacheEntryExists || cachedRecording.path != replacementFilePath {
		t.Error("expiry callback removed or changed the replacement cache entry")
	}
	if replacementContent, readError := os.ReadFile(replacementFilePath); readError != nil || string(replacementContent) != "replacement-content" {
		t.Errorf("replacement recording = %q, error %v", replacementContent, readError)
	}
}

func TestExpireCacheEntry_MissingEntry(t *testing.T) {
	// Arrange
	cacheKey := "missing-recording"
	unregisteredFilePath := filepath.Join(t.TempDir(), "unregistered.mp4")
	if writeError := os.WriteFile(unregisteredFilePath, []byte("unregistered-content"), 0o600); writeError != nil {
		t.Fatalf("create unregistered recording: %v", writeError)
	}
	recordingService := &RecordingService{cache: make(map[string]cachedRecording)}

	// Act
	recordingService.expireCacheEntry(cacheKey, unregisteredFilePath)

	// Assert
	if len(recordingService.cache) != 0 {
		t.Error("expiry callback created an unexpected cache entry")
	}
	if _, fileError := os.Stat(unregisteredFilePath); fileError != nil {
		t.Errorf("expiry callback changed an unregistered file: %v", fileError)
	}
}
