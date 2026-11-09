package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/services"
)

type recordingPlayRequest struct {
	Start    time.Time `json:"start"`
	Duration float64   `json:"durationSeconds"`
	Source   string    `json:"source"`
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
	if _, ok := requireTruckAccess(c.database, w, r, truckID); !ok {
		return
	}
	if cameraID == "" {
		writeError(w, http.StatusBadRequest, apiresponse.CodeCameraIDRequired, apiresponse.MessageCameraIDRequired)
		return
	}
	if !c.cameraExists(w, truckID, cameraID) {
		return
	}
	for _, parameter := range []string{"start", "end"} {
		if value := r.URL.Query().Get(parameter); value != "" {
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidTimeFormat, fmt.Sprintf(apiresponse.MessageQueryTimeFormat, parameter))
				return
			}
		}
	}
	recordings, err := c.recordings.List(r.Context(), truckID, cameraID, r.URL.Query().Get("start"), r.URL.Query().Get("end"))
	if err != nil {
		writeError(w, http.StatusBadGateway, apiresponse.CodeRecordingServiceUnavailable, apiresponse.MessageRecordingListUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, recordings)
}

func (c *RecordingController) Play(w http.ResponseWriter, r *http.Request) {
	truckID, cameraID := r.PathValue("truckID"), r.PathValue("cameraID")
	if _, ok := requireTruckAccess(c.database, w, r, truckID); !ok {
		return
	}
	var request recordingPlayRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil || request.Start.IsZero() {
		writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidRecordingPlayRequest, apiresponse.MessageInvalidRecordingPlayRequest)
		return
	}
	if request.Duration <= 0 || request.Duration > 24*60*60 {
		writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidRecordingDuration, apiresponse.MessageInvalidRecordingDuration)
		return
	}
	if request.Source == "" {
		request.Source = "main"
	}
	if !validRecordingSource(request.Source) {
		writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidRecordingPlayRequest, apiresponse.MessageInvalidRecordingPlayRequest)
		return
	}
	if !c.cameraExists(w, truckID, cameraID) {
		return
	}
	expires := services.RecordingAccessExpiry(time.Now(), request.Duration)
	token := c.streams.SignRecordingAccess(truckID, cameraID, request.Start, request.Duration, expires)
	writeJSON(w, http.StatusOK, map[string]any{
		"url":         c.recordings.PublicURLForSource(truckID, cameraID, request.Source, request.Start, request.Duration, token),
		"accessToken": token, "expiresAt": expires,
	})
}

func (c *RecordingController) Content(w http.ResponseWriter, r *http.Request) {
	truckID, cameraID, token := r.PathValue("truckID"), r.PathValue("cameraID"), r.URL.Query().Get("token")
	source := r.URL.Query().Get("source")
	if source == "" {
		source = "main"
	}
	if !validRecordingSource(source) {
		writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidRecordingPlayRequest, apiresponse.MessageInvalidRecordingPlayRequest)
		return
	}
	claims, err := c.streams.ValidateRecordingAccess(token, time.Now())
	if err != nil || claims.TruckID != truckID || claims.CameraID != cameraID {
		writeError(w, http.StatusUnauthorized, apiresponse.CodeRecordingAccessDenied, apiresponse.MessageRecordingAccessDenied)
		return
	}
	start, err := time.Parse(time.RFC3339, r.URL.Query().Get("start"))
	if err != nil {
		writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidTimeFormat, fmt.Sprintf(apiresponse.MessageQueryTimeFormat, "start"))
		return
	}
	duration, err := strconv.ParseFloat(r.URL.Query().Get("duration"), 64)
	if err != nil || duration <= 0 || duration > 24*60*60 {
		writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidRecordingDuration, apiresponse.MessageQueryDurationRange)
		return
	}
	if !claims.Allows(truckID, cameraID, start, duration) {
		writeError(w, http.StatusUnauthorized, apiresponse.CodeRecordingAccessDenied, apiresponse.MessageRecordingAccessDenied)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if err := c.recordings.ServeSource(w, r, truckID, cameraID, source, start, duration, token); err != nil {
		writeError(w, http.StatusBadGateway, apiresponse.CodeRecordingServiceUnavailable, apiresponse.MessageRecordingOpenUnavailable)
	}
}

func validRecordingSource(source string) bool {
	return source == "main" || source == "archive"
}

func (c *RecordingController) cameraExists(w http.ResponseWriter, truckID, cameraID string) bool {
	exists, err := db.CameraExists(c.database, truckID, cameraID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, apiresponse.CodeCameraLookupFailed, apiresponse.MessageCameraLookupFailed)
		return false
	}
	if !exists {
		writeError(w, http.StatusNotFound, apiresponse.CodeCameraNotFound, apiresponse.MessageCameraNotFound)
		return false
	}
	return true
}
