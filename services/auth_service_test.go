package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/db"
)

func newAuthTestDatabase(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	databaseConnection, databaseMock, connectionError := sqlmock.New()
	if connectionError != nil {
		t.Fatalf("create authentication SQL mock: %v", connectionError)
	}
	t.Cleanup(func() {
		databaseMock.ExpectClose()
		if closeError := databaseConnection.Close(); closeError != nil {
			t.Errorf("close authentication SQL mock: %v", closeError)
		}
		if expectationError := databaseMock.ExpectationsWereMet(); expectationError != nil {
			t.Errorf("unmet authentication SQL expectations: %v", expectationError)
		}
	})
	return databaseConnection, databaseMock
}

func TestAuthMiddleware_DatabaseUnavailable(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newAuthTestDatabase(t)
	apiToken := "authentication-test-token"
	databaseMock.ExpectQuery(`FROM user_api_tokens`).WithArgs(db.APITokenHash(apiToken), sqlmock.AnyArg()).
		WillReturnError(errors.New("authentication store unavailable"))
	protectedHandlerCalled := false
	authenticationHandler := NewAuthService(databaseConnection).Middleware(func(http.ResponseWriter, *http.Request) {
		protectedHandlerCalled = true
	})
	protectedRequest := httptest.NewRequest(http.MethodGet, "/api/trucks", nil)
	protectedRequest.Header.Set("Authorization", "Bearer "+apiToken)
	responseRecorder := httptest.NewRecorder()

	// Act
	authenticationHandler(responseRecorder, protectedRequest)

	// Assert
	if protectedHandlerCalled || responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("authentication response = %d, handler called = %v; want 503 without forwarding", responseRecorder.Code, protectedHandlerCalled)
	}
	var errorResponse apiresponse.ErrorBody
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &errorResponse); decodeError != nil {
		t.Fatalf("decode authentication error: %v", decodeError)
	}
	if errorResponse.Error.Code != apiresponse.CodeAuthenticationUnavailable {
		t.Errorf("error code = %q, want authentication_unavailable", errorResponse.Error.Code)
	}
}

func TestAuthMiddleware_UnknownToken(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newAuthTestDatabase(t)
	apiToken := "unknown-test-token"
	databaseMock.ExpectQuery(`FROM user_api_tokens`).WithArgs(db.APITokenHash(apiToken), sqlmock.AnyArg()).WillReturnError(sql.ErrNoRows)
	protectedHandlerCalled := false
	authenticationHandler := NewAuthService(databaseConnection).Middleware(func(http.ResponseWriter, *http.Request) {
		protectedHandlerCalled = true
	})
	protectedRequest := httptest.NewRequest(http.MethodGet, "/api/trucks", nil)
	protectedRequest.Header.Set("Authorization", "Bearer "+apiToken)
	responseRecorder := httptest.NewRecorder()

	// Act
	authenticationHandler(responseRecorder, protectedRequest)

	// Assert
	if protectedHandlerCalled || responseRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("authentication response = %d, handler called = %v; want 401 without forwarding", responseRecorder.Code, protectedHandlerCalled)
	}
	var errorResponse apiresponse.ErrorBody
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &errorResponse); decodeError != nil {
		t.Fatalf("decode authentication error: %v", decodeError)
	}
	if errorResponse.Error.Code != apiresponse.CodeInvalidBearerToken {
		t.Errorf("error code = %q, want invalid_bearer_token", errorResponse.Error.Code)
	}
}

func TestAuthMiddleware_WebSocketTokenPopulatesUserContext(t *testing.T) {
	// Arrange
	databaseConnection, databaseMock := newAuthTestDatabase(t)
	apiToken := "websocket-test-token"
	expectedUser := db.AuthenticatedUser{UserID: "user001", Email: "driver@example.com", DisplayName: "Test Driver"}
	databaseMock.ExpectQuery(`FROM user_api_tokens`).WithArgs(db.APITokenHash(apiToken), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "display_name"}).
			AddRow(expectedUser.UserID, expectedUser.Email, expectedUser.DisplayName))
	var authenticatedUser db.AuthenticatedUser
	userFoundInContext := false
	authenticationHandler := NewAuthService(databaseConnection).Middleware(func(response http.ResponseWriter, request *http.Request) {
		authenticatedUser, userFoundInContext = CurrentUser(request.Context())
		response.WriteHeader(http.StatusNoContent)
	})
	webSocketRequest := httptest.NewRequest(http.MethodGet, "/api/trucks/truck001/locations/ws", nil)
	webSocketRequest.Header.Set("Sec-WebSocket-Protocol", "bearer, "+apiToken)
	responseRecorder := httptest.NewRecorder()

	// Act
	authenticationHandler(responseRecorder, webSocketRequest)

	// Assert
	if responseRecorder.Code != http.StatusNoContent || !userFoundInContext || authenticatedUser != expectedUser {
		t.Errorf("WebSocket authentication = status %d, user %+v, found %v", responseRecorder.Code, authenticatedUser, userFoundInContext)
	}
}
