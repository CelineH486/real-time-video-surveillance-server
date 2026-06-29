package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"real-time-video-surveillance-system/controllers"
	"real-time-video-surveillance-system/db"
	appRoutes "real-time-video-surveillance-system/routes"
	"real-time-video-surveillance-system/services"
)

//go:embed web/*
var webAssets embed.FS

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) map[string]bool {
	result := map[string]bool{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result[item] = true
		}
	}
	return result
}

func withCORS(next http.Handler, allowedOrigins string) http.Handler {
	origins := splitCSV(allowedOrigins)
	allowAny := origins["*"]

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (allowAny || origins[origin] || isLocalDevelopmentOrigin(origin)) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isLocalDevelopmentOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	hostname := parsed.Hostname()
	return hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
}

func main() {
	database, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	streamService := services.NewStreamService(
		env("STREAM_PUBLIC_BASE_URL", "http://localhost:8889"),
		env("STREAM_SIGNING_KEY", "development-only-change-me"),
		env("STREAM_PUBLISH_PASSWORD", "development-publish-password"),
	)
	recordingService := services.NewRecordingService(
		env("PLAYBACK_INTERNAL_BASE_URL", "http://localhost:9996"),
		env("API_PUBLIC_BASE_URL", "http://localhost:8080"),
		streamService,
	)
	statusService := services.NewStatusService(database)

	healthController := &controllers.HealthController{}
	truckController := controllers.NewTruckController(database)
	cameraController := controllers.NewCameraController(database, streamService)
	streamController := controllers.NewStreamController(database, streamService)
	recordingController := controllers.NewRecordingController(database, recordingService, streamService)
	mediaMTXController := controllers.NewMediaMTXController(database, streamService)

	webContent, err := fs.Sub(webAssets, "web")
	if err != nil {
		log.Fatalf("load web assets: %v", err)
	}
	handler := appRoutes.New(appRoutes.Controllers{
		Health: healthController, Trucks: truckController, Cameras: cameraController,
		Streams: streamController, Recordings: recordingController, MediaMTX: mediaMTXController,
	}, webContent)
	handlerWithCORS := withCORS(handler, env("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173,http://localhost:8080"))

	offlineTimeout, err := time.ParseDuration(env("CAMERA_OFFLINE_TIMEOUT", "15s"))
	if err != nil {
		log.Fatalf("invalid CAMERA_OFFLINE_TIMEOUT: %v", err)
	}
	go statusService.ListenUDP(env("UDP_ADDRESS", ":5000"))
	go statusService.MonitorOffline(offlineTimeout)
	go statusService.MonitorMediaMTX(os.Getenv("MEDIAMTX_API_URL"))

	address := env("HTTP_ADDRESS", ":8080")
	log.Printf("HTTP server listening on %s", address)
	log.Fatal(http.ListenAndServe(address, handlerWithCORS))
}
