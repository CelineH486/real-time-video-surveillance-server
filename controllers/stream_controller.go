package controllers

import (
	"database/sql"
	"net/http"
	"time"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/services"
)

type streamInfo struct {
	TruckID  string    `json:"truckId"`
	CameraID string    `json:"cameraId"`
	Quality  string    `json:"quality"`
	Protocol string    `json:"protocol"`
	URL      string    `json:"url"`
	Token    string    `json:"accessToken"`
	Expires  time.Time `json:"expiresAt"`
}

type StreamController struct {
	database *sql.DB
	streams  *services.StreamService
}

func NewStreamController(database *sql.DB, streams *services.StreamService) *StreamController {
	return &StreamController{database: database, streams: streams}
}

func (c *StreamController) Play(w http.ResponseWriter, r *http.Request) {
	truckID, cameraID := r.PathValue("truckID"), r.PathValue("cameraID")
	if _, ok := requireTruckAccess(c.database, w, r, truckID); !ok {
		return
	}
	exists, err := db.CameraExists(c.database, truckID, cameraID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, apiresponse.CodeCameraLookupFailed, apiresponse.MessageCameraLookupFailed)
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, apiresponse.CodeCameraNotFound, apiresponse.MessageCameraNotFound)
		return
	}
	expires := time.Now().Add(5 * time.Minute)
	token := c.streams.SignAccess(truckID, cameraID, "main", expires)
	writeJSON(w, http.StatusOK, streamInfo{
		TruckID: truckID, CameraID: cameraID, Quality: "main", Protocol: "WebRTC",
		URL: c.streams.CameraURL(truckID, cameraID, "main"), Token: token, Expires: expires,
	})
}
