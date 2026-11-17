package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"real-time-video-surveillance-system/services"
)

func TestMediaMTXSeparatesLiveAndRecordingTokens_LiveTokenReadsLiveStream(t *testing.T) {
	// Arrange
	streams := services.NewStreamService("http://localhost:8889", "test-signing-key", "publish-password")
	controller := NewMediaMTXController(nil, streams)
	now := time.Now()
	liveToken := streams.SignAccess("truck001", "cam01", "main", now.Add(time.Hour))

	body, err := json.Marshal(mediaMTXAuthRequest{
		Action: "read",
		Path:   "truck001/cam01/main",
		Token:  liveToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/internal/mediamtx/auth", bytes.NewReader(body))
	response := httptest.NewRecorder()

	// Act
	controller.Auth(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Fatalf("got status %d, want %d: %s", response.Code, http.StatusNoContent, response.Body.String())
	}
}

func TestMediaMTXSeparatesLiveAndRecordingTokens_RecordingTokenPlaysRecording(t *testing.T) {
	// Arrange
	streams := services.NewStreamService("http://localhost:8889", "test-signing-key", "publish-password")
	controller := NewMediaMTXController(nil, streams)
	now := time.Now()

	recordingToken := streams.SignRecordingAccess("truck001", "cam01", now, 1800, now.Add(time.Hour))

	body, err := json.Marshal(mediaMTXAuthRequest{
		Action: "playback",
		Path:   "truck001/cam01/main",
		Token:  recordingToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/internal/mediamtx/auth", bytes.NewReader(body))
	response := httptest.NewRecorder()

	// Act
	controller.Auth(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Fatalf("got status %d, want %d: %s", response.Code, http.StatusNoContent, response.Body.String())
	}
}

func TestMediaMTXSeparatesLiveAndRecordingTokens_RecordingTokenCannotReadLiveStream(t *testing.T) {
	// Arrange
	streams := services.NewStreamService("http://localhost:8889", "test-signing-key", "publish-password")
	controller := NewMediaMTXController(nil, streams)
	now := time.Now()

	recordingToken := streams.SignRecordingAccess("truck001", "cam01", now, 1800, now.Add(time.Hour))

	body, err := json.Marshal(mediaMTXAuthRequest{
		Action: "read",
		Path:   "truck001/cam01/main",
		Token:  recordingToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/internal/mediamtx/auth", bytes.NewReader(body))
	response := httptest.NewRecorder()

	// Act
	controller.Auth(response, request)

	// Assert
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d: %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
}

func TestMediaMTXSeparatesLiveAndRecordingTokens_LiveTokenCannotPlayRecording(t *testing.T) {
	// Arrange
	streams := services.NewStreamService("http://localhost:8889", "test-signing-key", "publish-password")
	controller := NewMediaMTXController(nil, streams)
	now := time.Now()
	liveToken := streams.SignAccess("truck001", "cam01", "main", now.Add(time.Hour))

	body, err := json.Marshal(mediaMTXAuthRequest{
		Action: "playback",
		Path:   "truck001/cam01/main",
		Token:  liveToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/internal/mediamtx/auth", bytes.NewReader(body))
	response := httptest.NewRecorder()

	// Act
	controller.Auth(response, request)

	// Assert
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d: %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
}
