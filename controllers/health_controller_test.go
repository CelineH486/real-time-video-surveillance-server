package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthController_Get_Success(t *testing.T) {
	// Arrange
	healthController := HealthController{}
	healthRequest := httptest.NewRequest(http.MethodGet, "/health", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	healthController.Get(responseRecorder, healthRequest)

	// Assert
	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", responseRecorder.Code, http.StatusOK)
	}
	if responseContentType := responseRecorder.Header().Get("Content-Type"); responseContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", responseContentType)
	}
	var healthResponse map[string]string
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &healthResponse); decodeError != nil {
		t.Fatalf("decode health response: %v", decodeError)
	}
	if len(healthResponse) != 1 || healthResponse["status"] != "ok" {
		t.Errorf("health response = %+v, want status ok", healthResponse)
	}
}
