package controllers

import (
	"net/http"

	"real-time-video-surveillance-system/apiresponse"
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	apiresponse.WriteJSON(w, status, value)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	apiresponse.WriteError(w, status, code, message)
}
