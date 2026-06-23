package controllers

import (
	"database/sql"
	"net/http"
	"time"

	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/services"
)

type CameraController struct {
	database *sql.DB
	streams  *services.StreamService
}

func NewCameraController(database *sql.DB, streams *services.StreamService) *CameraController {
	return &CameraController{database: database, streams: streams}
}

func (c *CameraController) List(w http.ResponseWriter, _ *http.Request) {
	cameras, err := db.GetCameras(c.database)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, cameras)
}

func (c *CameraController) ListByTruck(w http.ResponseWriter, r *http.Request) {
	truckID := r.PathValue("truckID")
	cameras, err := db.GetCamerasByTruckID(c.database, truckID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	expires := time.Now().Add(5 * time.Minute)
	for _, camera := range cameras {
		cameraID := camera["cameraId"]
		camera["subUrl"] = c.streams.CameraURL(truckID, cameraID, "sub")
		camera["subToken"] = c.streams.SignAccess(truckID, cameraID, "sub", expires)
		camera["expiresAt"] = expires.Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, cameras)
}
