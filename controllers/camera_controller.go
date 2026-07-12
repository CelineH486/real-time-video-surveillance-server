package controllers

import (
	"database/sql"
	"net/http"
	"time"

	"real-time-video-surveillance-system/apiresponse"
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

func (c *CameraController) List(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	cameras, err := db.GetCamerasForUser(c.database, user.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, apiresponse.CodeCamerasUnavailable, apiresponse.MessageCamerasUnavailableForUser)
		return
	}
	writeJSON(w, http.StatusOK, cameras)
}

func (c *CameraController) ListByTruck(w http.ResponseWriter, r *http.Request) {
	truckID := r.PathValue("truckID")
	if _, ok := requireTruckAccess(c.database, w, r, truckID); !ok {
		return
	}
	cameras, err := db.GetCamerasByTruckID(c.database, truckID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, apiresponse.CodeCamerasUnavailable, apiresponse.MessageCamerasUnavailableForTruck)
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
