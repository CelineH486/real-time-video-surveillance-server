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
	controller := NewSessionController(nil)
	if controller == nil {
		t.Fatal("NewSessionController returned nil")
	}
	if controller.database != nil {
		t.Error("controller database = non-nil, want nil")
	}
}

func TestValidLoginPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{name: "valid", password: "Secure123", want: true},
		{name: "too short", password: "Short1", want: false},
		{name: "too long", password: strings.Repeat("A", 72) + "a1", want: false},
		{name: "missing uppercase", password: "secure123", want: false},
		{name: "missing lowercase", password: "SECURE123", want: false},
		{name: "missing digit", password: "SecurePass", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidLoginPassword(test.password); got != test.want {
				t.Errorf("ValidLoginPassword(%q) = %v, want %v", test.password, got, test.want)
			}
		})
	}
}

func TestValidLoginEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{name: "valid", email: "driver@example.com", want: true},
		{name: "trims whitespace", email: "  driver@example.com  ", want: true},
		{name: "case insensitive", email: "Driver@Example.COM", want: true},
		{name: "missing at sign", email: "driver.example.com", want: false},
		{name: "unsupported domain suffix", email: "driver@example.org", want: false},
		{name: "embedded whitespace", email: "driver @example.com", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidLoginEmail(test.email); got != test.want {
				t.Errorf("ValidLoginEmail(%q) = %v, want %v", test.email, got, test.want)
			}
		})
	}
}

func TestSessionControllerLoginRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode string
	}{
		{name: "malformed JSON", body: `{"email":`, wantCode: apiresponse.CodeInvalidLogin},
		{name: "unknown field", body: `{"email":"driver@example.com","password":"Secure123","extra":true}`, wantCode: apiresponse.CodeInvalidLogin},
		{name: "invalid email", body: `{"email":"driver@example.org","password":"Secure123"}`, wantCode: apiresponse.CodeInvalidLogin},
		{name: "invalid password", body: `{"email":"driver@example.com","password":"weak"}`, wantCode: apiresponse.CodeInvalidPasswordFormat},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := NewSessionController(nil)
			recorder := httptest.NewRecorder()

			controller.Login(recorder, loginRequestForTest(test.body))

			assertSessionError(t, recorder, http.StatusBadRequest, test.wantCode)
		})
	}
}

func TestSessionControllerLoginHandlesCredentialFailures(t *testing.T) {
	tests := []struct {
		name       string
		prepareDB  func(*testing.T, sqlmock.Sqlmock)
		wantStatus int
		wantCode   string
	}{
		{
			name: "database unavailable",
			prepareDB: func(_ *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`FROM users\s+WHERE LOWER\(email\) = LOWER\(\$1\)`).
					WithArgs(testLoginEmail).
					WillReturnError(errors.New("database unavailable"))
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   apiresponse.CodeAuthenticationUnavailable,
		},
		{
			name: "user not found",
			prepareDB: func(_ *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`FROM users\s+WHERE LOWER\(email\) = LOWER\(\$1\)`).
					WithArgs(testLoginEmail).
					WillReturnError(sql.ErrNoRows)
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   apiresponse.CodeInvalidLogin,
		},
		{
			name: "wrong password",
			prepareDB: func(t *testing.T, mock sqlmock.Sqlmock) {
				expectUserByEmail(mock, passwordHashForTest(t, "Different123"))
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   apiresponse.CodeInvalidLogin,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database, mock := newSessionMock(t)
			test.prepareDB(t, mock)
			controller := NewSessionController(database)
			recorder := httptest.NewRecorder()

			controller.Login(recorder, loginRequestForTest(
				`{"email":"driver@example.com","password":"Secure123"}`,
			))

			assertSessionError(t, recorder, test.wantStatus, test.wantCode)
		})
	}
}

func TestSessionControllerLoginReportsTokenStorageFailure(t *testing.T) {
	database, mock := newSessionMock(t)
	expectUserByEmail(mock, passwordHashForTest(t, testLoginPassword))
	mock.ExpectExec(`INSERT INTO user_api_tokens`).
		WithArgs(sqlmock.AnyArg(), "user-1", "Web login", sqlmock.AnyArg()).
		WillReturnError(errors.New("token storage unavailable"))
	controller := NewSessionController(database)
	recorder := httptest.NewRecorder()

	controller.Login(recorder, loginRequestForTest(
		`{"email":"driver@example.com","password":"Secure123"}`,
	))

	assertSessionError(t, recorder, http.StatusServiceUnavailable, apiresponse.CodeAuthenticationUnavailable)
}

func TestSessionControllerLoginSuccess(t *testing.T) {
	database, mock := newSessionMock(t)
	expectUserByEmail(mock, passwordHashForTest(t, testLoginPassword))
	mock.ExpectExec(`INSERT INTO user_api_tokens`).
		WithArgs(sqlmock.AnyArg(), "user-1", "Web login", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	controller := NewSessionController(database)
	recorder := httptest.NewRecorder()
	before := time.Now()

	controller.Login(recorder, loginRequestForTest(
		`{"email":"  driver@example.com  ","password":"Secure123"}`,
	))
	after := time.Now()

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

func TestSessionControllerLogout(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		prepareDB  func(sqlmock.Sqlmock)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing bearer token",
			wantStatus: http.StatusUnauthorized,
			wantCode:   apiresponse.CodeMissingBearerToken,
		},
		{
			name:   "database unavailable",
			header: "Bearer logout-token",
			prepareDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM user_api_tokens WHERE token_hash = \$1`).
					WithArgs(db.APITokenHash("logout-token")).
					WillReturnError(errors.New("delete failed"))
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   apiresponse.CodeLogoutUnavailable,
		},
		{
			name:   "success",
			header: "Bearer logout-token",
			prepareDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM user_api_tokens WHERE token_hash = \$1`).
					WithArgs(db.APITokenHash("logout-token")).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var database *sql.DB
			if test.prepareDB != nil {
				var mock sqlmock.Sqlmock
				database, mock = newSessionMock(t)
				test.prepareDB(mock)
			}
			controller := NewSessionController(database)
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/sessions/current", nil)
			req.Header.Set("Authorization", test.header)

			controller.Logout(recorder, req)

			if test.wantCode != "" {
				assertSessionError(t, recorder, test.wantStatus, test.wantCode)
				return
			}
			if recorder.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if recorder.Body.Len() != 0 {
				t.Errorf("response body = %q, want empty body", recorder.Body.String())
			}
		})
	}
}

func TestBearerTokenFromRequest(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{name: "valid", header: "Bearer session-token", wantToken: "session-token", wantOK: true},
		{name: "case insensitive", header: "bearer session-token", wantToken: "session-token", wantOK: true},
		{name: "trims whitespace", header: "  Bearer   session-token  ", wantToken: "session-token", wantOK: true},
		{name: "missing token", header: "Bearer", wantOK: false},
		{name: "empty token", header: "Bearer   ", wantOK: false},
		{name: "wrong scheme", header: "Basic session-token", wantToken: "session-token", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/sessions/current", nil)
			req.Header.Set("Authorization", test.header)

			token, ok := bearerTokenFromRequest(req)

			if token != test.wantToken || ok != test.wantOK {
				t.Errorf("bearerTokenFromRequest() = (%q, %v), want (%q, %v)", token, ok, test.wantToken, test.wantOK)
			}
		})
	}
}
