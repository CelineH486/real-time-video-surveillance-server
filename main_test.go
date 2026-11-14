package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestEnv_UsesConfiguredValue(t *testing.T) {
	// Arrange
	settingName := "SURVEILLANCE_TEST_SETTING"
	t.Setenv(settingName, "  configured-value  ")

	// Act
	settingValue := env(settingName, "fallback-value")

	// Assert
	if settingValue != "configured-value" {
		t.Errorf("setting value = %q, want configured-value", settingValue)
	}
}

func TestEnv_UsesFallbackForEmptyValue(t *testing.T) {
	// Arrange
	settingName := "SURVEILLANCE_TEST_SETTING"
	t.Setenv(settingName, "")

	// Act
	settingValue := env(settingName, "fallback-value")

	// Assert
	if settingValue != "fallback-value" {
		t.Errorf("setting value = %q, want fallback-value", settingValue)
	}
}

func TestEnv_UsesFallbackForWhitespace(t *testing.T) {
	// Arrange
	settingName := "SURVEILLANCE_TEST_SETTING"
	t.Setenv(settingName, " \t ")

	// Act
	settingValue := env(settingName, "fallback-value")

	// Assert
	if settingValue != "fallback-value" {
		t.Errorf("setting value = %q, want fallback-value", settingValue)
	}
}

func TestSplitCSV_TrimsAndDeduplicatesValues(t *testing.T) {
	// Arrange
	configuredOrigins := " https://first.example,https://second.example,https://first.example "
	expectedOrigins := map[string]bool{"https://first.example": true, "https://second.example": true}

	// Act
	parsedOrigins := splitCSV(configuredOrigins)

	// Assert
	if !reflect.DeepEqual(parsedOrigins, expectedOrigins) {
		t.Errorf("parsed origins = %+v, want %+v", parsedOrigins, expectedOrigins)
	}
}

func TestSplitCSV_IgnoresEmptyValues(t *testing.T) {
	// Arrange
	configuredOrigins := ", ,https://app.example,,"
	expectedOrigins := map[string]bool{"https://app.example": true}

	// Act
	parsedOrigins := splitCSV(configuredOrigins)

	// Assert
	if !reflect.DeepEqual(parsedOrigins, expectedOrigins) {
		t.Errorf("parsed origins = %+v, want %+v", parsedOrigins, expectedOrigins)
	}
}

func TestSplitCSV_EmptyInput(t *testing.T) {
	// Arrange
	configuredOrigins := ""

	// Act
	parsedOrigins := splitCSV(configuredOrigins)

	// Assert
	if parsedOrigins == nil || len(parsedOrigins) != 0 {
		t.Errorf("parsed origins = %+v, want an initialized empty map", parsedOrigins)
	}
}

func TestIsLocalDevelopmentOrigin_Localhost(t *testing.T) {
	// Arrange
	requestOrigin := "http://localhost:5173"

	// Act
	isDevelopmentOrigin := isLocalDevelopmentOrigin(requestOrigin)

	// Assert
	if !isDevelopmentOrigin {
		t.Error("localhost origin was rejected")
	}
}

func TestIsLocalDevelopmentOrigin_IPv4Loopback(t *testing.T) {
	// Arrange
	requestOrigin := "https://127.0.0.1:3000"

	// Act
	isDevelopmentOrigin := isLocalDevelopmentOrigin(requestOrigin)

	// Assert
	if !isDevelopmentOrigin {
		t.Error("IPv4 loopback origin was rejected")
	}
}

func TestIsLocalDevelopmentOrigin_IPv6Loopback(t *testing.T) {
	// Arrange
	requestOrigin := "http://[::1]:8080"

	// Act
	isDevelopmentOrigin := isLocalDevelopmentOrigin(requestOrigin)

	// Assert
	if !isDevelopmentOrigin {
		t.Error("IPv6 loopback origin was rejected")
	}
}

func TestIsLocalDevelopmentOrigin_RejectsLookalikeHost(t *testing.T) {
	// Arrange
	requestOrigin := "https://localhost.attacker.example"

	// Act
	isDevelopmentOrigin := isLocalDevelopmentOrigin(requestOrigin)

	// Assert
	if isDevelopmentOrigin {
		t.Error("a non-loopback host was accepted as a development origin")
	}
}

func TestIsLocalDevelopmentOrigin_RejectsUnsupportedScheme(t *testing.T) {
	// Arrange
	requestOrigin := "ftp://localhost"

	// Act
	isDevelopmentOrigin := isLocalDevelopmentOrigin(requestOrigin)

	// Assert
	if isDevelopmentOrigin {
		t.Error("an unsupported origin scheme was accepted")
	}
}

func TestIsLocalDevelopmentOrigin_RejectsMalformedURL(t *testing.T) {
	// Arrange
	requestOrigin := "://invalid-origin"

	// Act
	isDevelopmentOrigin := isLocalDevelopmentOrigin(requestOrigin)

	// Assert
	if isDevelopmentOrigin {
		t.Error("a malformed origin URL was accepted")
	}
}

func TestWithCORS_ConfiguredOrigin(t *testing.T) {
	// Arrange
	allowedOrigin := "https://app.example"
	protectedHandlerCalled := false
	protectedHandler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		protectedHandlerCalled = true
		response.WriteHeader(http.StatusAccepted)
	})
	corsHandler := withCORS(protectedHandler, allowedOrigin)
	apiRequest := httptest.NewRequest(http.MethodGet, "/api/trucks", nil)
	apiRequest.Header.Set("Origin", allowedOrigin)
	responseRecorder := httptest.NewRecorder()

	// Act
	corsHandler.ServeHTTP(responseRecorder, apiRequest)

	// Assert
	if !protectedHandlerCalled || responseRecorder.Code != http.StatusAccepted {
		t.Fatal("CORS middleware did not forward the request")
	}
	if responseRecorder.Header().Get("Access-Control-Allow-Origin") != allowedOrigin {
		t.Errorf("unexpected allowed origin: %q", responseRecorder.Header().Get("Access-Control-Allow-Origin"))
	}
	if responseRecorder.Header().Get("Vary") != "Origin" {
		t.Errorf("Vary = %q, want Origin", responseRecorder.Header().Get("Vary"))
	}
}

func TestWithCORS_WildcardOrigin(t *testing.T) {
	// Arrange
	requestOrigin := "https://external.example"
	corsHandler := withCORS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), "*")
	apiRequest := httptest.NewRequest(http.MethodGet, "/api/trucks", nil)
	apiRequest.Header.Set("Origin", requestOrigin)
	responseRecorder := httptest.NewRecorder()

	// Act
	corsHandler.ServeHTTP(responseRecorder, apiRequest)

	// Assert
	if responseRecorder.Header().Get("Access-Control-Allow-Origin") != requestOrigin {
		t.Errorf("allowed origin = %q, want %q", responseRecorder.Header().Get("Access-Control-Allow-Origin"), requestOrigin)
	}
}

func TestWithCORS_LocalDevelopmentOrigin(t *testing.T) {
	// Arrange
	requestOrigin := "http://localhost:5173"
	corsHandler := withCORS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), "https://app.example")
	apiRequest := httptest.NewRequest(http.MethodGet, "/api/trucks", nil)
	apiRequest.Header.Set("Origin", requestOrigin)
	responseRecorder := httptest.NewRecorder()

	// Act
	corsHandler.ServeHTTP(responseRecorder, apiRequest)

	// Assert
	if responseRecorder.Header().Get("Access-Control-Allow-Origin") != requestOrigin {
		t.Errorf("allowed origin = %q, want %q", responseRecorder.Header().Get("Access-Control-Allow-Origin"), requestOrigin)
	}
}

func TestWithCORS_DisallowedOrigin(t *testing.T) {
	// Arrange
	protectedHandlerCalled := false
	corsHandler := withCORS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		protectedHandlerCalled = true
	}), "https://app.example")
	apiRequest := httptest.NewRequest(http.MethodGet, "/api/trucks", nil)
	apiRequest.Header.Set("Origin", "https://untrusted.example")
	responseRecorder := httptest.NewRecorder()

	// Act
	corsHandler.ServeHTTP(responseRecorder, apiRequest)

	// Assert
	if !protectedHandlerCalled {
		t.Error("ordinary requests must still reach the protected handler")
	}
	if responseRecorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS middleware authorized an untrusted origin")
	}
}

func TestWithCORS_NoOriginHeader(t *testing.T) {
	// Arrange
	protectedHandlerCalled := false
	corsHandler := withCORS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		protectedHandlerCalled = true
	}), "*")
	apiRequest := httptest.NewRequest(http.MethodGet, "/api/trucks", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	corsHandler.ServeHTTP(responseRecorder, apiRequest)

	// Assert
	if !protectedHandlerCalled {
		t.Error("request without Origin was not forwarded")
	}
	if responseRecorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("response contains CORS authorization without an Origin request header")
	}
}

func TestWithCORS_PreflightRequest(t *testing.T) {
	// Arrange
	allowedOrigin := "https://app.example"
	protectedHandlerCalled := false
	corsHandler := withCORS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		protectedHandlerCalled = true
	}), allowedOrigin)
	preflightRequest := httptest.NewRequest(http.MethodOptions, "/api/trucks", nil)
	preflightRequest.Header.Set("Origin", allowedOrigin)
	responseRecorder := httptest.NewRecorder()

	// Act
	corsHandler.ServeHTTP(responseRecorder, preflightRequest)

	// Assert
	if protectedHandlerCalled {
		t.Error("preflight request reached the protected handler")
	}
	if responseRecorder.Code != http.StatusNoContent || responseRecorder.Body.Len() != 0 {
		t.Errorf("preflight response = %d %q, want 204 with an empty body", responseRecorder.Code, responseRecorder.Body.String())
	}
	if responseRecorder.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PATCH, DELETE, OPTIONS" {
		t.Errorf("unexpected allowed methods: %q", responseRecorder.Header().Get("Access-Control-Allow-Methods"))
	}
	if responseRecorder.Header().Get("Access-Control-Allow-Headers") != "Authorization, Content-Type" {
		t.Errorf("unexpected allowed headers: %q", responseRecorder.Header().Get("Access-Control-Allow-Headers"))
	}
}
