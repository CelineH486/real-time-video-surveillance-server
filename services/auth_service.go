package services

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/db"
)

type authenticatedUserContextKey struct{}

type AuthService struct {
	database *sql.DB
}

func NewAuthService(database *sql.DB) *AuthService {
	return &AuthService{database: database}
}

func (s *AuthService) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			token, ok = websocketBearerToken(r.Header.Get("Sec-WebSocket-Protocol"))
		}
		if !ok {
			apiresponse.WriteError(w, http.StatusUnauthorized, apiresponse.CodeMissingBearerToken, apiresponse.MessageMissingBearerToken)
			return
		}
		user, found, err := db.UserByAPIToken(s.database, token, time.Now())
		if err != nil {
			apiresponse.WriteError(w, http.StatusServiceUnavailable, apiresponse.CodeAuthenticationUnavailable, apiresponse.MessageAuthenticationUnavailable)
			return
		}
		if !found {
			apiresponse.WriteError(w, http.StatusUnauthorized, apiresponse.CodeInvalidBearerToken, apiresponse.MessageInvalidBearerToken)
			return
		}
		ctx := context.WithValue(r.Context(), authenticatedUserContextKey{}, user)
		next(w, r.WithContext(ctx))
	}
}

func websocketBearerToken(header string) (string, bool) {
	protocols := strings.Split(header, ",")
	if len(protocols) < 2 || !strings.EqualFold(strings.TrimSpace(protocols[0]), "bearer") {
		return "", false
	}
	token := strings.TrimSpace(protocols[1])
	return token, token != ""
}

func CurrentUser(ctx context.Context) (db.AuthenticatedUser, bool) {
	user, ok := ctx.Value(authenticatedUserContextKey{}).(db.AuthenticatedUser)
	return user, ok
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", false
	}
	return strings.TrimSpace(token), true
}
