package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/models"
	"real-time-video-surveillance-system/services"
)

type LocationController struct {
	database    *sql.DB
	now         func() time.Time
	subscribers map[string]map[chan models.TruckLocation]struct{}
	mu          sync.Mutex
}

func NewLocationController(database *sql.DB) *LocationController {
	return &LocationController{
		database: database, now: time.Now,
		subscribers: make(map[string]map[chan models.TruckLocation]struct{}),
	}
}

func (c *LocationController) Current(w http.ResponseWriter, r *http.Request) {
	truckID := r.PathValue("truckID")
	if _, ok := requireTruckAccess(c.database, w, r, truckID); !ok {
		return
	}
	location, found, err := db.CurrentTruckLocation(c.database, truckID, c.now())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, apiresponse.CodeLocationUnavailable, apiresponse.MessageLocationUnavailable)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, apiresponse.CodeLocationNotFound, apiresponse.MessageLocationNotFound)
		return
	}
	writeJSON(w, http.StatusOK, location)
}

func (c *LocationController) Stream(w http.ResponseWriter, r *http.Request) {
	truckID := r.PathValue("truckID")
	if _, ok := requireTruckAccess(c.database, w, r, truckID); !ok {
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"bearer"}})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	ctx := conn.CloseRead(context.Background())
	updates, unsubscribe := c.subscribe(truckID)
	defer unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case location := <-updates:
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := wsjson.Write(writeCtx, conn, location)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *LocationController) subscribe(truckID string) (<-chan models.TruckLocation, func()) {
	updates := make(chan models.TruckLocation, 1)
	c.mu.Lock()
	if c.subscribers[truckID] == nil {
		c.subscribers[truckID] = make(map[chan models.TruckLocation]struct{})
	}
	c.subscribers[truckID][updates] = struct{}{}
	c.mu.Unlock()
	return updates, func() {
		c.mu.Lock()
		delete(c.subscribers[truckID], updates)
		if len(c.subscribers[truckID]) == 0 {
			delete(c.subscribers, truckID)
		}
		c.mu.Unlock()
	}
}

func (c *LocationController) broadcast(location models.TruckLocation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for updates := range c.subscribers[location.TruckID] {
		select {
		case updates <- location:
		default:
			select {
			case <-updates:
			default:
			}
			select {
			case updates <- location:
			default:
			}
		}
	}
}

func (c *LocationController) Create(w http.ResponseWriter, r *http.Request) {
	truckID := r.PathValue("truckID")
	if _, ok := requireTruckAccess(c.database, w, r, truckID); !ok {
		return
	}
	var input models.LocationInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidLocation, apiresponse.MessageInvalidLocation)
		return
	}
	now := c.now().UTC()
	if err := services.ValidateLocation(input, now); err != nil {
		writeError(w, http.StatusBadRequest, apiresponse.CodeInvalidLocation, apiresponse.MessageInvalidLocation)
		return
	}
	recordedAt := now
	if input.RecordedAt != nil {
		recordedAt = input.RecordedAt.UTC()
	}
	previous, found, err := db.CurrentTruckLocation(c.database, truckID, now)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, apiresponse.CodeLocationUnavailable, apiresponse.MessageLocationUnavailable)
		return
	}
	var previousPtr *models.TruckLocation
	if found && !recordedAt.Before(previous.RecordedAt) {
		previousPtr = &previous
	}
	anchorLat, anchorLon, stoppedSince := services.StopState(previousPtr, input, recordedAt)
	location, err := db.SaveTruckLocation(c.database, truckID, input, recordedAt, now, anchorLat, anchorLon, stoppedSince)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, apiresponse.CodeLocationUnavailable, apiresponse.MessageLocationUnavailable)
		return
	}
	c.broadcast(location)
	writeJSON(w, http.StatusCreated, location)
}
