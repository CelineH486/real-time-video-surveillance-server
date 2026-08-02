package controllers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/db"
)

var (
	passwordUpper = regexp.MustCompile(`[A-Z]`)
	passwordLower = regexp.MustCompile(`[a-z]`)
	passwordDigit = regexp.MustCompile(`[0-9]`)
	emailPattern  = regexp.MustCompile(`(?i)^[^@\s]+@[^@\s]+\.com$`)
)

type SessionController struct{ database *sql.DB }

func NewSessionController(database *sql.DB) *SessionController {
	return &SessionController{database: database}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ValidLoginPassword reports whether a password satisfies the login password policy.
func ValidLoginPassword(password string) bool {
	return len(password) >= 8 && len(password) <= 72 &&
		passwordUpper.MatchString(password) && passwordLower.MatchString(password) && passwordDigit.MatchString(password)
}

// ValidLoginEmail reports whether an email satisfies the current login email policy.
func ValidLoginEmail(email string) bool {
	return emailPattern.MatchString(strings.TrimSpace(email))
}

func (c *SessionController) Login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || !ValidLoginEmail(request.Email) {
		writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidLogin, apiresponse.MessageInvalidLogin)
		return
	}
	if !ValidLoginPassword(request.Password) {
		writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidPasswordFormat, apiresponse.MessageInvalidPasswordFormat)
		return
	}
	user, found, err := db.UserByEmail(c.database, strings.TrimSpace(request.Email))
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, apiresponse.CodeAuthenticationUnavailable, apiresponse.MessageAuthenticationUnavailable)
		return
	}
	if !found || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)) != nil {
		writeError(w, http.StatusUnauthorized, apiresponse.CodeInvalidLogin, apiresponse.MessageInvalidLogin)
		return
	}
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		writeError(w, http.StatusServiceUnavailable, apiresponse.CodeAuthenticationUnavailable, apiresponse.MessageAuthenticationUnavailable)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(rawToken)
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := db.CreateAPIToken(c.database, user.UserID, token, "Web login", expiresAt); err != nil {
		writeError(w, http.StatusServiceUnavailable, apiresponse.CodeAuthenticationUnavailable, apiresponse.MessageAuthenticationUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "expiresAt": expiresAt, "user": user.AuthenticatedUser})
}

func (c *SessionController) Logout(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerTokenFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, apiresponse.CodeMissingBearerToken, apiresponse.MessageMissingBearerToken)
		return
	}
	if err := db.DeleteAPIToken(c.database, token); err != nil {
		writeError(w, http.StatusServiceUnavailable, apiresponse.CodeLogoutUnavailable, apiresponse.MessageLogoutUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func bearerTokenFromRequest(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(r.Header.Get("Authorization")), " ")
	return strings.TrimSpace(token), ok && strings.EqualFold(scheme, "Bearer") && strings.TrimSpace(token) != ""
}
