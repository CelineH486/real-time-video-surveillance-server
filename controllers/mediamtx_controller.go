package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"real-time-video-surveillance-system/apiresponse"
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
		writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidAuthentication, apiresponse.MessageInvalidMediaMTXAuthRequest)
		return
	}
	truckID, cameraID, quality, ok := services.StreamPathParts(request.Path)
	if !ok {
		writeError(w, http.StatusForbidden, apiresponse.CodeInvalidStreamPath, apiresponse.MessageInvalidStreamPath)
		return
	}
	switch request.Action {
	case "publish":
		if !c.streams.PublishCredentialsValid(request.User, request.Password, truckID) {
			writeError(w, http.StatusUnauthorized, apiresponse.CodePublishDenied, apiresponse.MessagePublishDenied)
			return
		}
		exists, err := db.CameraExists(c.database, truckID, cameraID)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, apiresponse.CodeDatabaseUnavailable, apiresponse.MessageDatabaseUnavailable)
			return
		}
		if !exists {
			writeError(w, http.StatusForbidden, apiresponse.CodeCameraNotFound, apiresponse.MessageMediaMTXCameraNotFound)
			return
		}
	case "read", "playback":
		claims, err := c.streams.ValidateAccess(request.Token, time.Now())
		if err != nil || claims.TruckID != truckID || claims.CameraID != cameraID || claims.Quality != quality {
			writeError(w, http.StatusUnauthorized, apiresponse.CodeStreamAccessDenied, apiresponse.MessageStreamAccessDenied)
			return
		}
	default:
		writeError(w, http.StatusForbidden, apiresponse.CodeActionDenied, apiresponse.MessageActionDenied)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
