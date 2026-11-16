package apiresponse

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSON_Success(t *testing.T) {
	// Arrange
	responseRecorder := httptest.NewRecorder()
	responsePayload := map[string]string{"status": "ok"}

	// Act
	WriteJSON(responseRecorder, http.StatusAccepted, responsePayload)

	// Assert
	if responseRecorder.Code != http.StatusAccepted || responseRecorder.Header().Get("Content-Type") != "application/json" {
		t.Errorf("unexpected JSON response status or headers: %d %+v", responseRecorder.Code, responseRecorder.Header())
	}
	var decodedPayload map[string]string
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &decodedPayload); decodeError != nil || decodedPayload["status"] != "ok" {
		t.Errorf("decoded response = %+v, error %v", decodedPayload, decodeError)
	}
}

func TestWriteJSON_LogsEncodingFailure(t *testing.T) {
	// Arrange
	responseRecorder := httptest.NewRecorder()
	unsupportedPayload := make(chan int)
	logOutput := &bytes.Buffer{}
	originalLogWriter := log.Writer()
	log.SetOutput(logOutput)
	t.Cleanup(func() { log.SetOutput(originalLogWriter) })

	// Act
	WriteJSON(responseRecorder, http.StatusOK, unsupportedPayload)

	// Assert
	if !strings.Contains(logOutput.String(), "encode response:") || !strings.Contains(logOutput.String(), "unsupported type") {
		t.Errorf("encoding failure was not logged: %q", logOutput.String())
	}
	if responseRecorder.Code != http.StatusOK || responseRecorder.Body.Len() != 0 {
		t.Errorf("encoding failure response = %d %q, want 200 with an empty body", responseRecorder.Code, responseRecorder.Body.String())
	}
}

func TestWriteError_EncodesErrorEnvelope(t *testing.T) {
	// Arrange
	responseRecorder := httptest.NewRecorder()
	expectedErrorCode := CodeAuthenticationRequired
	expectedErrorMessage := MessageAuthenticationRequired

	// Act
	WriteError(responseRecorder, http.StatusUnauthorized, expectedErrorCode, expectedErrorMessage)

	// Assert
	if responseRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", responseRecorder.Code)
	}
	var errorResponse ErrorBody
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &errorResponse); decodeError != nil {
		t.Fatalf("decode error envelope: %v", decodeError)
	}
	if errorResponse.Error.Code != expectedErrorCode || errorResponse.Error.Message != expectedErrorMessage {
		t.Errorf("unexpected error envelope: %+v", errorResponse)
	}
}
