package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/services"
)

func TestNewStreamController(t *testing.T) {
	// Arrange
	databaseConnection, _ := newControllerTestDatabase(t)
	streamService := services.NewStreamService("https://stream.example", "stream-test-signing-key", "publish-password")

	// Act
	streamController := NewStreamController(databaseConnection, streamService)

	// Assert
	if streamController == nil {
		t.Fatal("NewStreamController returned nil")
	}
	if streamController.database != databaseConnection || streamController.streams != streamService {
		t.Error("controller did not retain the supplied database connection and stream service")
	}
}

func TestStreamController_Play_Unauthenticated(t *testing.T) {
	// Arrange
	streamController := NewStreamController(nil, nil)
	unauthenticatedRequest := httptest.NewRequest(http.MethodPost, "/api/trucks/truck001/cameras/cam01/play", nil)
	unauthenticatedRequest.SetPathValue("truckID", "truck001")
	unauthenticatedRequest.SetPathValue("cameraID", "cam01")
	responseRecorder := httptest.NewRecorder()

	// Act
	streamController.Play(responseRecorder, unauthenticatedRequest)

	// Assert
	assertControllerErrorResponse(t, responseRecorder, http.StatusUnauthorized,
		apiresponse.CodeAuthenticationRequired, apiresponse.MessageAuthenticationRequired)
}

func TestStreamController_Play_Forbidden(t *testing.T) {
	// Arrange
	requestedTruckID := "truck001"
	requestedCameraID := "cam01"
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodPost, "/api/trucks/truck001/cameras/cam01/play")
	authenticatedRequest.SetPathValue("truckID", requestedTruckID)
	authenticatedRequest.SetPathValue("cameraID", requestedCameraID)
	databaseMock.ExpectQuery(`FROM user_trucks WHERE user_id = \$1 AND truck_id = \$2`).
		WithArgs(controllerTestUserID, requestedTruckID).
		WillReturnRows(sqlmock.NewRows([]string{"allowed"}).AddRow(false))
	streamController := NewStreamController(databaseConnection, nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	streamController.Play(responseRecorder, authenticatedRequest)

	// Assert
	assertControllerErrorResponse(t, responseRecorder, http.StatusForbidden,
		apiresponse.CodeTruckAccessDenied, apiresponse.MessageTruckAccessDenied)
}

func TestStreamController_Play_AuthorizationDatabaseError(t *testing.T) {
	// Arrange
	requestedTruckID := "truck001"
	requestedCameraID := "cam01"
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodPost, "/api/trucks/truck001/cameras/cam01/play")
	authenticatedRequest.SetPathValue("truckID", requestedTruckID)
	authenticatedRequest.SetPathValue("cameraID", requestedCameraID)
	databaseMock.ExpectQuery(`FROM user_trucks WHERE user_id = \$1 AND truck_id = \$2`).
		WithArgs(controllerTestUserID, requestedTruckID).
		WillReturnError(errors.New("truck permission lookup failed"))
	streamController := NewStreamController(databaseConnection, nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	streamController.Play(responseRecorder, authenticatedRequest)

	// Assert
	assertControllerErrorResponse(t, responseRecorder, http.StatusServiceUnavailable,
		apiresponse.CodeAuthorizationUnavailable, apiresponse.MessageAuthorizationUnavailable)
}

func TestStreamController_Play_CameraDatabaseError(t *testing.T) {
	// Arrange
	requestedTruckID := "truck001"
	requestedCameraID := "cam01"
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodPost, "/api/trucks/truck001/cameras/cam01/play")
	authenticatedRequest.SetPathValue("truckID", requestedTruckID)
	authenticatedRequest.SetPathValue("cameraID", requestedCameraID)
	databaseMock.ExpectQuery(`FROM user_trucks WHERE user_id = \$1 AND truck_id = \$2`).
		WithArgs(controllerTestUserID, requestedTruckID).
		WillReturnRows(sqlmock.NewRows([]string{"allowed"}).AddRow(true))
	databaseMock.ExpectQuery(`FROM cameras WHERE truck_id = \$1 AND camera_id = \$2`).
		WithArgs(requestedTruckID, requestedCameraID).
		WillReturnError(errors.New("camera lookup failed"))
	streamController := NewStreamController(databaseConnection, nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	streamController.Play(responseRecorder, authenticatedRequest)

	// Assert
	assertControllerErrorResponse(t, responseRecorder, http.StatusInternalServerError,
		apiresponse.CodeCameraLookupFailed, apiresponse.MessageCameraLookupFailed)
}

func TestStreamController_Play_CameraNotFound(t *testing.T) {
	// Arrange
	requestedTruckID := "truck001"
	requestedCameraID := "cam01"
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodPost, "/api/trucks/truck001/cameras/cam01/play")
	authenticatedRequest.SetPathValue("truckID", requestedTruckID)
	authenticatedRequest.SetPathValue("cameraID", requestedCameraID)
	databaseMock.ExpectQuery(`FROM user_trucks WHERE user_id = \$1 AND truck_id = \$2`).
		WithArgs(controllerTestUserID, requestedTruckID).
		WillReturnRows(sqlmock.NewRows([]string{"allowed"}).AddRow(true))
	databaseMock.ExpectQuery(`FROM cameras WHERE truck_id = \$1 AND camera_id = \$2`).
		WithArgs(requestedTruckID, requestedCameraID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	streamController := NewStreamController(databaseConnection, nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	streamController.Play(responseRecorder, authenticatedRequest)

	// Assert
	assertControllerErrorResponse(t, responseRecorder, http.StatusNotFound,
		apiresponse.CodeCameraNotFound, apiresponse.MessageCameraNotFound)
}

func TestStreamController_Play_Success(t *testing.T) {
	// Arrange
	requestedTruckID := "truck001"
	requestedCameraID := "cam01"
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodPost, "/api/trucks/truck001/cameras/cam01/play")
	authenticatedRequest.SetPathValue("truckID", requestedTruckID)
	authenticatedRequest.SetPathValue("cameraID", requestedCameraID)
	databaseMock.ExpectQuery(`FROM user_trucks WHERE user_id = \$1 AND truck_id = \$2`).
		WithArgs(controllerTestUserID, requestedTruckID).
		WillReturnRows(sqlmock.NewRows([]string{"allowed"}).AddRow(true))
	databaseMock.ExpectQuery(`FROM cameras WHERE truck_id = \$1 AND camera_id = \$2`).
		WithArgs(requestedTruckID, requestedCameraID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	streamService := services.NewStreamService("https://stream.example", "stream-test-signing-key", "publish-password")
	streamController := NewStreamController(databaseConnection, streamService)
	responseRecorder := httptest.NewRecorder()
	requestStartedAt := time.Now()

	// Act
	streamController.Play(responseRecorder, authenticatedRequest)
	requestCompletedAt := time.Now()

	// Assert
	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", responseRecorder.Code, http.StatusOK, responseRecorder.Body.String())
	}
	var playbackResponse streamInfo
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &playbackResponse); decodeError != nil {
		t.Fatalf("decode stream playback response: %v", decodeError)
	}
	if playbackResponse.TruckID != requestedTruckID || playbackResponse.CameraID != requestedCameraID {
		t.Errorf("unexpected playback truck or camera: %+v", playbackResponse)
	}
	if playbackResponse.Quality != "main" || playbackResponse.Protocol != "WebRTC" {
		t.Errorf("unexpected playback quality or protocol: %+v", playbackResponse)
	}
	expectedStreamURL := "https://stream.example/truck001/cam01/main/whep"
	if playbackResponse.URL != expectedStreamURL {
		t.Errorf("stream URL = %q, want %q", playbackResponse.URL, expectedStreamURL)
	}
	streamTokenClaims, tokenValidationError := streamService.ValidateAccess(playbackResponse.Token, requestCompletedAt)
	if tokenValidationError != nil {
		t.Fatalf("validate stream access token: %v", tokenValidationError)
	}
	if streamTokenClaims.TruckID != requestedTruckID || streamTokenClaims.CameraID != requestedCameraID || streamTokenClaims.Quality != "main" {
		t.Errorf("unexpected stream token claims: %+v", streamTokenClaims)
	}
	minimumExpiry := requestStartedAt.Add(5 * time.Minute)
	maximumExpiry := requestCompletedAt.Add(5 * time.Minute)
	if playbackResponse.Expires.Before(minimumExpiry) || playbackResponse.Expires.After(maximumExpiry) {
		t.Errorf("expiresAt = %v, want between %v and %v", playbackResponse.Expires, minimumExpiry, maximumExpiry)
	}
	// Signed tokens encode their expiry as whole Unix seconds.
	if streamTokenClaims.Expires.Unix() != playbackResponse.Expires.Unix() {
		t.Errorf("token expiry = %v, want the response expiry %v", streamTokenClaims.Expires, playbackResponse.Expires)
	}
}
