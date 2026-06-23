package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
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

	offlineTimeout, err := time.ParseDuration(env("CAMERA_OFFLINE_TIMEOUT", "15s"))
	if err != nil {
		log.Fatalf("invalid CAMERA_OFFLINE_TIMEOUT: %v", err)
	}
	go statusService.ListenUDP(env("UDP_ADDRESS", ":5000"))
	go statusService.MonitorOffline(offlineTimeout)
	go statusService.MonitorMediaMTX(os.Getenv("MEDIAMTX_API_URL"))

	address := env("HTTP_ADDRESS", ":8080")
	log.Printf("HTTP server listening on %s", address)
	log.Fatal(http.ListenAndServe(address, handler))
}
