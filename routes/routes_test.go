package routes

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"real-time-video-surveillance-system/controllers"
	"real-time-video-surveillance-system/services"
)

func newRouteTestHandler(webContent fs.FS) *http.ServeMux {
	streamService := services.NewStreamService("https://stream.example", "route-test-signing-key", "publish-password")
	return New(Controllers{
		Health:     &controllers.HealthController{},
		Trucks:     controllers.NewTruckController(nil),
		Cameras:    controllers.NewCameraController(nil, streamService),
		Streams:    controllers.NewStreamController(nil, streamService),
		Recordings: controllers.NewRecordingController(nil, nil, streamService),
		MediaMTX:   controllers.NewMediaMTXController(nil, streamService),
		Auth:       services.NewAuthService(nil),
		Sessions:   controllers.NewSessionController(nil),
		Locations:  controllers.NewLocationController(nil),
	}, webContent)
}

func TestNew_RootRedirectsToLogin(t *testing.T) {
	// Arrange
	routeHandler := newRouteTestHandler(fstest.MapFS{})
	rootRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	routeHandler.ServeHTTP(responseRecorder, rootRequest)

	// Assert
	if responseRecorder.Code != http.StatusTemporaryRedirect || responseRecorder.Header().Get("Location") != "/login" {
		t.Errorf("root redirect = %d %q, want 307 /login", responseRecorder.Code, responseRecorder.Header().Get("Location"))
	}
}

func TestNew_UnknownPathReturnsNotFound(t *testing.T) {
	// Arrange
	routeHandler := newRouteTestHandler(fstest.MapFS{})
	unknownRequest := httptest.NewRequest(http.MethodGet, "/missing-page", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	routeHandler.ServeHTTP(responseRecorder, unknownRequest)

	// Assert
	if responseRecorder.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", responseRecorder.Code)
	}
}

func TestNew_WebRedirectAddsTrailingSlash(t *testing.T) {
	// Arrange
	routeHandler := newRouteTestHandler(fstest.MapFS{})
	webRequest := httptest.NewRequest(http.MethodGet, "/web", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	routeHandler.ServeHTTP(responseRecorder, webRequest)

	// Assert
	if responseRecorder.Code != http.StatusTemporaryRedirect || responseRecorder.Header().Get("Location") != "/web/" {
		t.Errorf("web redirect = %d %q, want 307 /web/", responseRecorder.Code, responseRecorder.Header().Get("Location"))
	}
}

func TestNew_LoginServesAppIndex(t *testing.T) {
	// Arrange
	indexHTML := "<!doctype html><title>Surveillance Login</title>"
	webContent := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte(indexHTML)}}
	routeHandler := newRouteTestHandler(webContent)
	loginRequest := httptest.NewRequest(http.MethodGet, "/login", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	routeHandler.ServeHTTP(responseRecorder, loginRequest)

	// Assert
	if responseRecorder.Code != http.StatusOK || responseRecorder.Body.String() != indexHTML {
		t.Errorf("login response = %d %q, want the app index", responseRecorder.Code, responseRecorder.Body.String())
	}
	if responseRecorder.Header().Get("Content-Type") != "text/html; charset=utf-8" || responseRecorder.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("unexpected login response headers: %+v", responseRecorder.Header())
	}
}

func TestNew_LoginReturnsErrorWhenIndexIsMissing(t *testing.T) {
	// Arrange
	routeHandler := newRouteTestHandler(fstest.MapFS{})
	loginRequest := httptest.NewRequest(http.MethodGet, "/login", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	routeHandler.ServeHTTP(responseRecorder, loginRequest)

	// Assert
	if responseRecorder.Code != http.StatusInternalServerError || !strings.Contains(responseRecorder.Body.String(), "Web application unavailable") {
		t.Errorf("missing-index response = %d %q, want 500", responseRecorder.Code, responseRecorder.Body.String())
	}
}

func TestNew_WebServesStaticAsset(t *testing.T) {
	// Arrange
	assetContent := "console.log('surveillance');"
	webContent := fstest.MapFS{"app.js": &fstest.MapFile{Data: []byte(assetContent)}}
	routeHandler := newRouteTestHandler(webContent)
	assetRequest := httptest.NewRequest(http.MethodGet, "/web/app.js", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	routeHandler.ServeHTTP(responseRecorder, assetRequest)

	// Assert
	if responseRecorder.Code != http.StatusOK || responseRecorder.Body.String() != assetContent {
		t.Errorf("static asset response = %d %q, want the requested asset", responseRecorder.Code, responseRecorder.Body.String())
	}
}

func TestNew_HealthRouteIsPublic(t *testing.T) {
	// Arrange
	routeHandler := newRouteTestHandler(fstest.MapFS{})
	healthRequest := httptest.NewRequest(http.MethodGet, "/health", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	routeHandler.ServeHTTP(responseRecorder, healthRequest)

	// Assert
	if responseRecorder.Code != http.StatusOK || !strings.Contains(responseRecorder.Body.String(), `"status":"ok"`) {
		t.Errorf("health response = %d %q, want 200 with status ok", responseRecorder.Code, responseRecorder.Body.String())
	}
}

func TestNew_TruckRouteRequiresAuthentication(t *testing.T) {
	// Arrange
	routeHandler := newRouteTestHandler(fstest.MapFS{})
	truckRequest := httptest.NewRequest(http.MethodGet, "/api/trucks", nil)
	responseRecorder := httptest.NewRecorder()

	// Act
	routeHandler.ServeHTTP(responseRecorder, truckRequest)

	// Assert
	if responseRecorder.Code != http.StatusUnauthorized || !strings.Contains(responseRecorder.Body.String(), "missing_bearer_token") {
		t.Errorf("protected route response = %d %q, want authentication rejection", responseRecorder.Code, responseRecorder.Body.String())
	}
}

func TestNew_LoginRouteValidatesBody(t *testing.T) {
	// Arrange
	routeHandler := newRouteTestHandler(fstest.MapFS{})
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":`))
	responseRecorder := httptest.NewRecorder()

	// Act
	routeHandler.ServeHTTP(responseRecorder, loginRequest)

	// Assert
	if responseRecorder.Code != http.StatusBadRequest || !strings.Contains(responseRecorder.Body.String(), "invalid_login") {
		t.Errorf("login route response = %d %q, want invalid login", responseRecorder.Code, responseRecorder.Body.String())
	}
}
