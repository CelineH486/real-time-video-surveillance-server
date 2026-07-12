package controllers

import (
	"database/sql"
	"net/http"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/services"
)

func currentUser(w http.ResponseWriter, r *http.Request) (db.AuthenticatedUser, bool) {
	user, ok := services.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, apiresponse.CodeAuthenticationRequired, apiresponse.MessageAuthenticationRequired)
		return db.AuthenticatedUser{}, false
	}
	return user, true
}

func requireTruckAccess(database *sql.DB, w http.ResponseWriter, r *http.Request, truckID string) (db.AuthenticatedUser, bool) {
	user, ok := currentUser(w, r)
	if !ok {
		return db.AuthenticatedUser{}, false
	}
	allowed, err := db.UserCanAccessTruck(database, user.UserID, truckID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, apiresponse.CodeAuthorizationUnavailable, apiresponse.MessageAuthorizationUnavailable)
		return db.AuthenticatedUser{}, false
	}
	if !allowed {
		writeError(w, http.StatusForbidden, apiresponse.CodeTruckAccessDenied, apiresponse.MessageTruckAccessDenied)
		return db.AuthenticatedUser{}, false
	}
	return user, true
}
