package controllers

import (
	"database/sql"
	"net/http"

	"real-time-video-surveillance-system/db"
)

type TruckController struct {
	database *sql.DB
}

func NewTruckController(database *sql.DB) *TruckController {
	return &TruckController{database: database}
}

func (c *TruckController) List(w http.ResponseWriter, _ *http.Request) {
	trucks, err := db.GetTrucks(c.database)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, trucks)
}
