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

func TestMediaMTXSeparatesLiveAndRecordingTokens(t *testing.T) {
	streams := services.NewStreamService("http://localhost:8889", "test-signing-key", "publish-password")
	controller := NewMediaMTXController(nil, streams)
	now := time.Now()
	liveToken := streams.SignAccess("truck001", "cam01", "main", now.Add(time.Hour))
	recordingToken := streams.SignRecordingAccess("truck001", "cam01", now, 1800, now.Add(time.Hour))

	tests := []struct {
		name   string
		action string
		path   string
		token  string
		status int
	}{
		{name: "live token reads live stream", action: "read", path: "truck001/cam01/main", token: liveToken, status: http.StatusNoContent},
		{name: "recording token plays old recording", action: "playback", path: "truck001/cam01/main", token: recordingToken, status: http.StatusNoContent},
		{name: "recording token plays archive", action: "playback", path: "truck001/cam01/archive", token: recordingToken, status: http.StatusNoContent},
		{name: "recording token cannot read live stream", action: "read", path: "truck001/cam01/main", token: recordingToken, status: http.StatusUnauthorized},
		{name: "live token cannot play recording", action: "playback", path: "truck001/cam01/archive", token: liveToken, status: http.StatusUnauthorized},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := json.Marshal(mediaMTXAuthRequest{
				Action: test.action,
				Path:   test.path,
				Token:  test.token,
			})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/internal/mediamtx/auth", bytes.NewReader(body))
			response := httptest.NewRecorder()
			controller.Auth(response, request)
			if response.Code != test.status {
				t.Fatalf("got status %d, want %d: %s", response.Code, test.status, response.Body.String())
			}
		})
	}
}
