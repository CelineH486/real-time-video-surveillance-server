package services_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"real-time-video-surveillance-system/services"
)

func TestAuthMiddlewareRejectsInvalidAuthorizationHeader(t *testing.T) {
	service := services.NewAuthService(nil)
	handler := service.Middleware(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler must not run for an invalid authorization header")
	})

	for _, header := range []string{"", "Basic token", "Bearer", "Bearer "} {
		t.Run(header, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Authorization", header)
			response := httptest.NewRecorder()

			handler(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
			}
		})
	}
}
