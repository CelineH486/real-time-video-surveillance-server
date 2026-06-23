package controllers

import "net/http"

type HealthController struct{}

func (HealthController) Get(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
