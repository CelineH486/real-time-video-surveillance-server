package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/services"
)

type mediaMTXAuthRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Token    string `json:"token"`
	Action   string `json:"action"`
	Path     string `json:"path"`
	Protocol string `json:"protocol"`
}

type MediaMTXController struct {
	database *sql.DB
	streams  *services.StreamService
}

func NewMediaMTXController(database *sql.DB, streams *services.StreamService) *MediaMTXController {
	return &MediaMTXController{database: database, streams: streams}
}

func (c *MediaMTXController) Auth(w http.ResponseWriter, r *http.Request) {
	var request mediaMTXAuthRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
		http.Error(w, "invalid authentication request", http.StatusBadRequest)
		return
	}
	truckID, cameraID, quality, ok := services.StreamPathParts(request.Path)
	if !ok {
		http.Error(w, "invalid stream path", http.StatusForbidden)
		return
	}
	switch request.Action {
	case "publish":
		if !c.streams.PublishCredentialsValid(request.User, request.Password, truckID) {
			http.Error(w, "publish denied", http.StatusUnauthorized)
			return
		}
		exists, err := db.CameraExists(c.database, truckID, cameraID)
		if err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		if !exists {
			http.Error(w, "camera not found", http.StatusForbidden)
			return
		}
	case "read", "playback":
		claims, err := c.streams.ValidateAccess(request.Token, time.Now())
		if err != nil || claims.TruckID != truckID || claims.CameraID != cameraID || claims.Quality != quality {
			http.Error(w, "stream access denied", http.StatusUnauthorized)
			return
		}
	default:
		http.Error(w, "action denied", http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
