package controllers

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/services"
)

const (
	controllerTestAPIToken    = "controller-test-token"
	controllerTestUserID      = "user001"
	controllerTestUserEmail   = "driver@example.com"
	controllerTestDisplayName = "Test Driver"
)

func newControllerTestDatabase(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	databaseConnection, databaseMock, createDatabaseError := sqlmock.New()
	if createDatabaseError != nil {
		t.Fatalf("create SQL mock: %v", createDatabaseError)
	}
	t.Cleanup(func() {
		databaseMock.ExpectClose()
		if closeDatabaseError := databaseConnection.Close(); closeDatabaseError != nil {
			t.Errorf("close SQL mock: %v", closeDatabaseError)
		}
		if expectationError := databaseMock.ExpectationsWereMet(); expectationError != nil {
			t.Errorf("unmet SQL expectations: %v", expectationError)
		}
	})
	return databaseConnection, databaseMock
}

func newAuthenticatedControllerTestRequest(t *testing.T, databaseConnection *sql.DB, databaseMock sqlmock.Sqlmock, requestMethod, requestURL string) *http.Request {
	t.Helper()
	databaseMock.ExpectQuery(`FROM user_api_tokens t\s+INNER JOIN users u`).
		WithArgs(db.APITokenHash(controllerTestAPIToken), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "display_name"}).
			AddRow(controllerTestUserID, controllerTestUserEmail, controllerTestDisplayName))
	unauthenticatedRequest := httptest.NewRequest(requestMethod, requestURL, nil)
	unauthenticatedRequest.Header.Set("Authorization", "Bearer "+controllerTestAPIToken)
	var authenticatedRequest *http.Request
	// Use the real middleware to populate the private authentication context.
	authenticationHandler := services.NewAuthService(databaseConnection).Middleware(func(_ http.ResponseWriter, request *http.Request) {
		authenticatedRequest = request
	})
	authenticationResponse := httptest.NewRecorder()
	authenticationHandler(authenticationResponse, unauthenticatedRequest)
	if authenticatedRequest == nil {
		t.Fatalf("authenticate test request: status = %d; body = %s", authenticationResponse.Code, authenticationResponse.Body.String())
	}
	return authenticatedRequest
}

func TestCurrentUser_Unauthenticated(t *testing.T) {
	// Arrange
	unauthenticatedRequest := httptest.NewRequest(http.MethodGet, "/api/trucks", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	returnedUser, userFound := currentUser(responseRecorder, unauthenticatedRequest)

	// Assert
	if userFound || returnedUser != (db.AuthenticatedUser{}) {
		t.Errorf("currentUser = (%+v, %v), want an empty user and false", returnedUser, userFound)
	}
	assertControllerErrorResponse(t, responseRecorder, http.StatusUnauthorized,
		apiresponse.CodeAuthenticationRequired, apiresponse.MessageAuthenticationRequired)
}

func TestCurrentUser_Authenticated(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodGet, "/api/trucks")
	responseRecorder := httptest.NewRecorder()
	expectedUser := db.AuthenticatedUser{
		UserID: controllerTestUserID, Email: controllerTestUserEmail, DisplayName: controllerTestDisplayName,
	}

	// Act
	returnedUser, userFound := currentUser(responseRecorder, authenticatedRequest)

	// Assert
	if !userFound || returnedUser != expectedUser {
		t.Errorf("currentUser = (%+v, %v), want (%+v, true)", returnedUser, userFound, expectedUser)
	}
	if responseRecorder.Body.Len() != 0 {
		t.Errorf("response body = %q, want empty", responseRecorder.Body.String())
	}
}

func TestRequireTruckAccess_Unauthenticated(t *testing.T) {
	// Arrange
	requestedTruckID := "truck001"
	unauthenticatedRequest := httptest.NewRequest(http.MethodGet, "/api/trucks/truck001", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	returnedUser, accessGranted := requireTruckAccess(nil, responseRecorder, unauthenticatedRequest, requestedTruckID)

	// Assert
	if accessGranted || returnedUser != (db.AuthenticatedUser{}) {
		t.Errorf("requireTruckAccess = (%+v, %v), want an empty user and false", returnedUser, accessGranted)
	}
	assertControllerErrorResponse(t, responseRecorder, http.StatusUnauthorized,
		apiresponse.CodeAuthenticationRequired, apiresponse.MessageAuthenticationRequired)
}

func TestRequireTruckAccess_DatabaseError(t *testing.T) {
	// Arrange
	requestedTruckID := "truck001"
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodGet, "/api/trucks/truck001")
	databaseMock.ExpectQuery(`FROM user_trucks WHERE user_id = \$1 AND truck_id = \$2`).
		WithArgs(controllerTestUserID, requestedTruckID).
		WillReturnError(errors.New("truck permission lookup failed"))
	responseRecorder := httptest.NewRecorder()

	// Act
	returnedUser, accessGranted := requireTruckAccess(databaseConnection, responseRecorder, authenticatedRequest, requestedTruckID)

	// Assert
	if accessGranted || returnedUser != (db.AuthenticatedUser{}) {
		t.Errorf("requireTruckAccess = (%+v, %v), want an empty user and false", returnedUser, accessGranted)
	}
	assertControllerErrorResponse(t, responseRecorder, http.StatusServiceUnavailable,
		apiresponse.CodeAuthorizationUnavailable, apiresponse.MessageAuthorizationUnavailable)
}

func TestRequireTruckAccess_Forbidden(t *testing.T) {
	// Arrange
	requestedTruckID := "truck001"
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodGet, "/api/trucks/truck001")
	databaseMock.ExpectQuery(`FROM user_trucks WHERE user_id = \$1 AND truck_id = \$2`).
		WithArgs(controllerTestUserID, requestedTruckID).
		WillReturnRows(sqlmock.NewRows([]string{"allowed"}).AddRow(false))
	responseRecorder := httptest.NewRecorder()

	// Act
	returnedUser, accessGranted := requireTruckAccess(databaseConnection, responseRecorder, authenticatedRequest, requestedTruckID)

	// Assert
	if accessGranted || returnedUser != (db.AuthenticatedUser{}) {
		t.Errorf("requireTruckAccess = (%+v, %v), want an empty user and false", returnedUser, accessGranted)
	}
	assertControllerErrorResponse(t, responseRecorder, http.StatusForbidden,
		apiresponse.CodeTruckAccessDenied, apiresponse.MessageTruckAccessDenied)
}

func TestRequireTruckAccess_Allowed(t *testing.T) {
	// Arrange
	requestedTruckID := "truck001"
	databaseConnection, databaseMock := newControllerTestDatabase(t)
	authenticatedRequest := newAuthenticatedControllerTestRequest(t, databaseConnection, databaseMock, http.MethodGet, "/api/trucks/truck001")
	databaseMock.ExpectQuery(`FROM user_trucks WHERE user_id = \$1 AND truck_id = \$2`).
		WithArgs(controllerTestUserID, requestedTruckID).
		WillReturnRows(sqlmock.NewRows([]string{"allowed"}).AddRow(true))
	responseRecorder := httptest.NewRecorder()
	expectedUser := db.AuthenticatedUser{
		UserID: controllerTestUserID, Email: controllerTestUserEmail, DisplayName: controllerTestDisplayName,
	}

	// Act
	returnedUser, accessGranted := requireTruckAccess(databaseConnection, responseRecorder, authenticatedRequest, requestedTruckID)

	// Assert
	if !accessGranted || returnedUser != expectedUser {
		t.Errorf("requireTruckAccess = (%+v, %v), want (%+v, true)", returnedUser, accessGranted, expectedUser)
	}
	if responseRecorder.Body.Len() != 0 {
		t.Errorf("response body = %q, want empty", responseRecorder.Body.String())
	}
}
