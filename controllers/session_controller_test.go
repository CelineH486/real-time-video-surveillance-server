package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/crypto/bcrypt"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/db"
)

const (
	testLoginEmail    = "driver@example.com"
	testLoginPassword = "Secure123"
)

func newSessionMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
		_ = database.Close()
	})
	return database, mock
}

func loginRequestForTest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(body))
}

func assertSessionError(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, status, recorder.Body.String())
	}
	var response apiresponse.ErrorBody
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Code != code {
		t.Errorf("error code = %q, want %q", response.Error.Code, code)
	}
	if response.Error.Message == "" {
		t.Error("error message is empty")
	}
}

func passwordHashForTest(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("generate password hash: %v", err)
	}
	return string(hash)
}

func expectUserByEmail(mock sqlmock.Sqlmock, passwordHash string) {
	mock.ExpectQuery(`FROM users\s+WHERE LOWER\(email\) = LOWER\(\$1\)`).
		WithArgs(testLoginEmail).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "email", "display_name", "password_hash",
		}).AddRow("user-1", testLoginEmail, "Test Driver", passwordHash))
}

func TestNewSessionController(t *testing.T) {
	// Arrange
	var database *sql.DB

	// Act
	controller := NewSessionController(database)

	// Assert
	if controller == nil {
		t.Fatal("NewSessionController returned nil")
	}
	if controller.database != nil {
		t.Error("controller database = non-nil, want nil")
	}
}

func TestValidLoginPassword_Valid(t *testing.T) {
	// Arrange
	password := "Secure123"

	// Act
	got := ValidLoginPassword(password)

	// Assert
	if !got {
		t.Errorf("ValidLoginPassword(%q) = %v, want %v", password, got, true)
	}
}

func TestValidLoginPassword_TooShort(t *testing.T) {
	// Arrange
	password := "Short1"

	// Act
	got := ValidLoginPassword(password)

	// Assert
	if got {
		t.Errorf("ValidLoginPassword(%q) = %v, want %v", password, got, false)
	}
}

func TestValidLoginPassword_TooLong(t *testing.T) {
	// Arrange
	password := strings.Repeat("A", 72) + "a1"

	// Act
	got := ValidLoginPassword(password)

	// Assert
	if got {
		t.Errorf("ValidLoginPassword(%q) = %v, want %v", password, got, false)
	}
}

func TestValidLoginPassword_MissingUppercase(t *testing.T) {
	// Arrange
	password := "secure123"

	// Act
	got := ValidLoginPassword(password)

	// Assert
	if got {
		t.Errorf("ValidLoginPassword(%q) = %v, want %v", password, got, false)
	}
}

func TestValidLoginPassword_MissingLowercase(t *testing.T) {
	// Arrange
	password := "SECURE123"

	// Act
	got := ValidLoginPassword(password)

	// Assert
	if got {
		t.Errorf("ValidLoginPassword(%q) = %v, want %v", password, got, false)
	}
}

func TestValidLoginPassword_MissingDigit(t *testing.T) {
	// Arrange
	password := "SecurePass"

	// Act
	got := ValidLoginPassword(password)

	// Assert
	if got {
		t.Errorf("ValidLoginPassword(%q) = %v, want %v", password, got, false)
	}
}

func TestValidLoginEmail_Valid(t *testing.T) {
	// Arrange
	email := "driver@example.com"

	// Act
	got := ValidLoginEmail(email)

	// Assert
	if !got {
		t.Errorf("ValidLoginEmail(%q) = %v, want %v", email, got, true)
	}
}

func TestValidLoginEmail_TrimsWhitespace(t *testing.T) {
	// Arrange
	email := "  driver@example.com  "

	// Act
	got := ValidLoginEmail(email)

	// Assert
	if !got {
		t.Errorf("ValidLoginEmail(%q) = %v, want %v", email, got, true)
	}
}

func TestValidLoginEmail_CaseInsensitive(t *testing.T) {
	// Arrange
	email := "Driver@Example.COM"

	// Act
	got := ValidLoginEmail(email)

	// Assert
	if !got {
		t.Errorf("ValidLoginEmail(%q) = %v, want %v", email, got, true)
	}
}

func TestValidLoginEmail_MissingAtSign(t *testing.T) {
	// Arrange
	email := "driver.example.com"

	// Act
	got := ValidLoginEmail(email)

	// Assert
	if got {
		t.Errorf("ValidLoginEmail(%q) = %v, want %v", email, got, false)
	}
}

func TestValidLoginEmail_UnsupportedDomainSuffix(t *testing.T) {
	// Arrange
	email := "driver@example.org"

	// Act
	got := ValidLoginEmail(email)

	// Assert
	if got {
		t.Errorf("ValidLoginEmail(%q) = %v, want %v", email, got, false)
	}
}

func TestValidLoginEmail_EmbeddedWhitespace(t *testing.T) {
	// Arrange
	email := "driver @example.com"

	// Act
	got := ValidLoginEmail(email)

	// Assert
	if got {
		t.Errorf("ValidLoginEmail(%q) = %v, want %v", email, got, false)
	}
}

func TestSessionControllerLoginRejectsInvalidRequest_MalformedJSON(t *testing.T) {
	// Arrange
	controller := NewSessionController(nil)
	recorder := httptest.NewRecorder()
	req := loginRequestForTest(`{"email":`)

	// Act
	controller.Login(recorder, req)

	// Assert
	assertSessionError(t, recorder, http.StatusBadRequest, apiresponse.CodeInvalidLogin)
}

func TestSessionControllerLoginRejectsInvalidRequest_UnknownField(t *testing.T) {
	// Arrange
	controller := NewSessionController(nil)
	recorder := httptest.NewRecorder()
	req := loginRequestForTest(`{"email":"driver@example.com","password":"Secure123","extra":true}`)

	// Act
	controller.Login(recorder, req)

	// Assert
	assertSessionError(t, recorder, http.StatusBadRequest, apiresponse.CodeInvalidLogin)
}

func TestSessionControllerLoginRejectsInvalidRequest_InvalidEmail(t *testing.T) {
	// Arrange
	controller := NewSessionController(nil)
	recorder := httptest.NewRecorder()
	req := loginRequestForTest(`{"email":"driver@example.org","password":"Secure123"}`)

	// Act
	controller.Login(recorder, req)

	// Assert
	assertSessionError(t, recorder, http.StatusBadRequest, apiresponse.CodeInvalidLogin)
}

func TestSessionControllerLoginRejectsInvalidRequest_InvalidPassword(t *testing.T) {
	// Arrange
	controller := NewSessionController(nil)
	recorder := httptest.NewRecorder()
	req := loginRequestForTest(`{"email":"driver@example.com","password":"weak"}`)

	// Act
	controller.Login(recorder, req)

	// Assert
	assertSessionError(t, recorder, http.StatusBadRequest, apiresponse.CodeInvalidPasswordFormat)
}

func TestSessionControllerLoginHandlesCredentialFailures_DatabaseUnavailable(t *testing.T) {
	// Arrange
	database, mock := newSessionMock(t)

	mock.ExpectQuery(`FROM users\s+WHERE LOWER\(email\) = LOWER\(\$1\)`).
		WithArgs(testLoginEmail).
		WillReturnError(errors.New("database unavailable"))

	controller := NewSessionController(database)
	recorder := httptest.NewRecorder()
	req := loginRequestForTest(
		`{"email":"driver@example.com","password":"Secure123"}`,
	)

	// Act
	controller.Login(recorder, req)

	// Assert
	assertSessionError(t, recorder, http.StatusServiceUnavailable, apiresponse.CodeAuthenticationUnavailable)
}

func TestSessionControllerLoginHandlesCredentialFailures_UserNotFound(t *testing.T) {
	// Arrange
	database, mock := newSessionMock(t)

	mock.ExpectQuery(`FROM users\s+WHERE LOWER\(email\) = LOWER\(\$1\)`).
		WithArgs(testLoginEmail).
		WillReturnError(sql.ErrNoRows)

	controller := NewSessionController(database)
	recorder := httptest.NewRecorder()
	req := loginRequestForTest(
		`{"email":"driver@example.com","password":"Secure123"}`,
	)

	// Act
	controller.Login(recorder, req)

	// Assert
	assertSessionError(t, recorder, http.StatusUnauthorized, apiresponse.CodeInvalidLogin)
}

func TestSessionControllerLoginHandlesCredentialFailures_WrongPassword(t *testing.T) {
	// Arrange
	database, mock := newSessionMock(t)

	expectUserByEmail(mock, passwordHashForTest(t, "Different123"))

	controller := NewSessionController(database)
	recorder := httptest.NewRecorder()
	req := loginRequestForTest(
		`{"email":"driver@example.com","password":"Secure123"}`,
	)

	// Act
	controller.Login(recorder, req)

	// Assert
	assertSessionError(t, recorder, http.StatusUnauthorized, apiresponse.CodeInvalidLogin)
}

func TestSessionControllerLoginReportsTokenStorageFailure(t *testing.T) {
	// Arrange
	database, mock := newSessionMock(t)
	expectUserByEmail(mock, passwordHashForTest(t, testLoginPassword))
	mock.ExpectExec(`INSERT INTO user_api_tokens`).
		WithArgs(sqlmock.AnyArg(), "user-1", "Web login", sqlmock.AnyArg()).
		WillReturnError(errors.New("token storage unavailable"))
	controller := NewSessionController(database)
	recorder := httptest.NewRecorder()
	req := loginRequestForTest(
		`{"email":"driver@example.com","password":"Secure123"}`,
	)

	// Act
	controller.Login(recorder, req)

	// Assert
	assertSessionError(t, recorder, http.StatusServiceUnavailable, apiresponse.CodeAuthenticationUnavailable)
}

func TestSessionControllerLoginSuccess(t *testing.T) {
	// Arrange
	database, mock := newSessionMock(t)
	expectUserByEmail(mock, passwordHashForTest(t, testLoginPassword))
	mock.ExpectExec(`INSERT INTO user_api_tokens`).
		WithArgs(sqlmock.AnyArg(), "user-1", "Web login", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	controller := NewSessionController(database)
	recorder := httptest.NewRecorder()
	before := time.Now()
	req := loginRequestForTest(
		`{"email":"  driver@example.com  ","password":"Secure123"}`,
	)

	// Act
	controller.Login(recorder, req)
	after := time.Now()

	// Assert
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response struct {
		Token     string               `json:"token"`
		ExpiresAt time.Time            `json:"expiresAt"`
		User      db.AuthenticatedUser `json:"user"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if response.Token == "" {
		t.Error("login token is empty")
	}
	if response.User.UserID != "user-1" || response.User.Email != testLoginEmail {
		t.Errorf("unexpected authenticated user: %+v", response.User)
	}
	// Allow for the small amount of time spent hashing, storing, and encoding the token.
	minimumExpiry := before.Add(24 * time.Hour)
	maximumExpiry := after.Add(24 * time.Hour)
	if response.ExpiresAt.Before(minimumExpiry) || response.ExpiresAt.After(maximumExpiry) {
		t.Errorf("expiresAt = %v, want between %v and %v", response.ExpiresAt, minimumExpiry, maximumExpiry)
	}
}

func TestSessionControllerLogout_MissingBearerToken(t *testing.T) {
	// Arrange
	controller := NewSessionController(nil)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/sessions/current", nil)
	req.Header.Set("Authorization", "")

	// Act
	controller.Logout(recorder, req)

	// Assert
	assertSessionError(t, recorder, http.StatusUnauthorized, apiresponse.CodeMissingBearerToken)
}

func TestSessionControllerLogout_DatabaseUnavailable(t *testing.T) {
	// Arrange
	database, mock := newSessionMock(t)

	mock.ExpectExec(`DELETE FROM user_api_tokens WHERE token_hash = \$1`).
		WithArgs(db.APITokenHash("logout-token")).
		WillReturnError(errors.New("delete failed"))

	controller := NewSessionController(database)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/sessions/current", nil)
	req.Header.Set("Authorization", "Bearer logout-token")

	// Act
	controller.Logout(recorder, req)

	// Assert
	assertSessionError(t, recorder, http.StatusServiceUnavailable, apiresponse.CodeLogoutUnavailable)
}

func TestSessionControllerLogout_Success(t *testing.T) {
	// Arrange
	database, mock := newSessionMock(t)

	mock.ExpectExec(`DELETE FROM user_api_tokens WHERE token_hash = \$1`).
		WithArgs(db.APITokenHash("logout-token")).
		WillReturnResult(sqlmock.NewResult(0, 1))

	controller := NewSessionController(database)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/sessions/current", nil)
	req.Header.Set("Authorization", "Bearer logout-token")

	// Act
	controller.Logout(recorder, req)

	// Assert
	if recorder.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("response body = %q, want empty body", recorder.Body.String())
	}
}

func TestBearerTokenFromRequest_Valid(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodDelete, "/sessions/current", nil)
	req.Header.Set("Authorization", "Bearer session-token")

	// Act
	token, ok := bearerTokenFromRequest(req)

	// Assert
	if token != "session-token" || !ok {
		t.Errorf("bearerTokenFromRequest() = (%q, %v), want (%q, %v)", token, ok, "session-token", true)
	}
}

func TestBearerTokenFromRequest_CaseInsensitive(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodDelete, "/sessions/current", nil)
	req.Header.Set("Authorization", "bearer session-token")

	// Act
	token, ok := bearerTokenFromRequest(req)

	// Assert
	if token != "session-token" || !ok {
		t.Errorf("bearerTokenFromRequest() = (%q, %v), want (%q, %v)", token, ok, "session-token", true)
	}
}

func TestBearerTokenFromRequest_TrimsWhitespace(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodDelete, "/sessions/current", nil)
	req.Header.Set("Authorization", "  Bearer   session-token  ")

	// Act
	token, ok := bearerTokenFromRequest(req)

	// Assert
	if token != "session-token" || !ok {
		t.Errorf("bearerTokenFromRequest() = (%q, %v), want (%q, %v)", token, ok, "session-token", true)
	}
}

func TestBearerTokenFromRequest_MissingToken(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodDelete, "/sessions/current", nil)
	req.Header.Set("Authorization", "Bearer")

	// Act
	token, ok := bearerTokenFromRequest(req)

	// Assert
	if token != "" || ok {
		t.Errorf("bearerTokenFromRequest() = (%q, %v), want (%q, %v)", token, ok, "", false)
	}
}

func TestBearerTokenFromRequest_EmptyToken(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodDelete, "/sessions/current", nil)
	req.Header.Set("Authorization", "Bearer   ")

	// Act
	token, ok := bearerTokenFromRequest(req)

	// Assert
	if token != "" || ok {
		t.Errorf("bearerTokenFromRequest() = (%q, %v), want (%q, %v)", token, ok, "", false)
	}
}

func TestBearerTokenFromRequest_WrongScheme(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodDelete, "/sessions/current", nil)
	req.Header.Set("Authorization", "Basic session-token")

	// Act
	token, ok := bearerTokenFromRequest(req)

	// Assert
	if token != "session-token" || ok {
		t.Errorf("bearerTokenFromRequest() = (%q, %v), want (%q, %v)", token, ok, "session-token", false)
	}
}
