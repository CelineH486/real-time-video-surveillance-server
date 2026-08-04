package controllers

import (
	"database/sql"
	"net/http"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/db"
)

type TruckController struct {
	database *sql.DB
}

func NewTruckController(database *sql.DB) *TruckController {
	return &TruckController{database: database}
}

func (c *TruckController) List(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	trucks, err := db.GetTrucksForUser(c.database, user.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, apiresponse.CodeTrucksUnavailable, apiresponse.MessageTrucksUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, trucks)
}
