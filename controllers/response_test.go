package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"real-time-video-surveillance-system/apiresponse"
)

func assertControllerErrorResponse(t *testing.T, responseRecorder *httptest.ResponseRecorder, expectedStatusCode int, expectedErrorCode, expectedErrorMessage string) {
	t.Helper()
	if responseRecorder.Code != expectedStatusCode {
		t.Fatalf("status = %d, want %d; body = %s", responseRecorder.Code, expectedStatusCode, responseRecorder.Body.String())
	}
	if responseContentType := responseRecorder.Header().Get("Content-Type"); responseContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", responseContentType)
	}
	var errorResponse apiresponse.ErrorBody
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &errorResponse); decodeError != nil {
		t.Fatalf("decode error response: %v", decodeError)
	}
	if errorResponse.Error.Code != expectedErrorCode {
		t.Errorf("error code = %q, want %q", errorResponse.Error.Code, expectedErrorCode)
	}
	if errorResponse.Error.Message != expectedErrorMessage {
		t.Errorf("error message = %q, want %q", errorResponse.Error.Message, expectedErrorMessage)
	}
}

func TestWriteJSON_StatusAndPayload(t *testing.T) {
	// Arrange
	responseRecorder := httptest.NewRecorder()
	expectedStatusCode := http.StatusCreated
	expectedPayload := map[string]string{"truckId": "truck001", "status": "online"}

	// Act
	writeJSON(responseRecorder, expectedStatusCode, expectedPayload)

	// Assert
	if responseRecorder.Code != expectedStatusCode {
		t.Fatalf("status = %d, want %d", responseRecorder.Code, expectedStatusCode)
	}
	if responseContentType := responseRecorder.Header().Get("Content-Type"); responseContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", responseContentType)
	}
	var responsePayload map[string]string
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &responsePayload); decodeError != nil {
		t.Fatalf("decode JSON response: %v", decodeError)
	}
	if len(responsePayload) != len(expectedPayload) || responsePayload["truckId"] != expectedPayload["truckId"] || responsePayload["status"] != expectedPayload["status"] {
		t.Errorf("response payload = %+v, want %+v", responsePayload, expectedPayload)
	}
}

func TestWriteError_StatusAndErrorBody(t *testing.T) {
	// Arrange
	responseRecorder := httptest.NewRecorder()
	expectedStatusCode := http.StatusServiceUnavailable
	expectedErrorCode := apiresponse.CodeAuthorizationUnavailable
	expectedErrorMessage := apiresponse.MessageAuthorizationUnavailable

	// Act
	writeError(responseRecorder, expectedStatusCode, expectedErrorCode, expectedErrorMessage)

	// Assert
	assertControllerErrorResponse(t, responseRecorder, expectedStatusCode, expectedErrorCode, expectedErrorMessage)
}
