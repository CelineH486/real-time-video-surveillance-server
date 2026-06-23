package controllers

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/services"
)

type recordingPlayRequest struct {
	Start    time.Time `json:"start"`
	Duration float64   `json:"durationSeconds"`
}

type RecordingController struct {
	database   *sql.DB
	recordings *services.RecordingService
	streams    *services.StreamService
}

func NewRecordingController(database *sql.DB, recordings *services.RecordingService, streams *services.StreamService) *RecordingController {
	return &RecordingController{database: database, recordings: recordings, streams: streams}
}

func (c *RecordingController) List(w http.ResponseWriter, r *http.Request) {
	truckID, cameraID := r.PathValue("truckID"), r.URL.Query().Get("cameraId")
	if cameraID == "" {
		http.Error(w, "cameraId is required", http.StatusBadRequest)
		return
	}
	if !c.cameraExists(w, truckID, cameraID) {
		return
	}
	for _, parameter := range []string{"start", "end"} {
		if value := r.URL.Query().Get(parameter); value != "" {
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				http.Error(w, parameter+" must use RFC3339 format", http.StatusBadRequest)
				return
			}
		}
	}
	recordings, err := c.recordings.List(r.Context(), truckID, cameraID, r.URL.Query().Get("start"), r.URL.Query().Get("end"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, recordings)
}

func (c *RecordingController) Play(w http.ResponseWriter, r *http.Request) {
	truckID, cameraID := r.PathValue("truckID"), r.PathValue("cameraID")
	var request recordingPlayRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil || request.Start.IsZero() {
		http.Error(w, "start and durationSeconds are required", http.StatusBadRequest)
		return
	}
	if request.Duration <= 0 || request.Duration > 24*60*60 {
		http.Error(w, "durationSeconds must be between 0 and 86400", http.StatusBadRequest)
		return
	}
	if !c.cameraExists(w, truckID, cameraID) {
		return
	}
	expires := time.Now().Add(5 * time.Minute)
	token := c.streams.SignAccess(truckID, cameraID, "main", expires)
	writeJSON(w, http.StatusOK, map[string]any{
		"url":         c.recordings.PublicURL(truckID, cameraID, request.Start, request.Duration, token),
		"accessToken": token, "expiresAt": expires,
	})
}

func (c *RecordingController) Content(w http.ResponseWriter, r *http.Request) {
	truckID, cameraID, token := r.PathValue("truckID"), r.PathValue("cameraID"), r.URL.Query().Get("token")
	claims, err := c.streams.ValidateAccess(token, time.Now())
	if err != nil || claims.TruckID != truckID || claims.CameraID != cameraID || claims.Quality != "main" {
		http.Error(w, "recording access denied", http.StatusUnauthorized)
		return
	}
	start, err := time.Parse(time.RFC3339, r.URL.Query().Get("start"))
	if err != nil {
		http.Error(w, "start must use RFC3339 format", http.StatusBadRequest)
		return
	}
	duration, err := strconv.ParseFloat(r.URL.Query().Get("duration"), 64)
	if err != nil || duration <= 0 || duration > 24*60*60 {
		http.Error(w, "invalid duration", http.StatusBadRequest)
		return
	}
	response, err := c.recordings.Open(r.Context(), truckID, cameraID, start, duration, token)
	if err != nil {
		http.Error(w, "recording service unavailable", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		http.Error(w, "recording service rejected request", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "private, no-store")
	if _, err := io.Copy(w, response.Body); err != nil {
		log.Printf("stream recording response: %v", err)
	}
}

func (c *RecordingController) cameraExists(w http.ResponseWriter, truckID, cameraID string) bool {
	exists, err := db.CameraExists(c.database, truckID, cameraID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	if !exists {
		http.Error(w, "camera not found", http.StatusNotFound)
		return false
	}
	return true
}
