package controllers

import (
	"testing"
	"time"

	"real-time-video-surveillance-system/models"
)

func TestLocationBroadcastDeliversNewestUpdate(t *testing.T) {
	controller := &LocationController{subscribers: make(map[string]map[chan models.TruckLocation]struct{})}
	updates, unsubscribe := controller.subscribe("truck001")
	defer unsubscribe()

	controller.broadcast(models.TruckLocation{TruckID: "truck001", Latitude: 24.1})
	controller.broadcast(models.TruckLocation{TruckID: "truck001", Latitude: 24.2})

	select {
	case location := <-updates:
		if location.Latitude != 24.2 {
			t.Fatalf("got latitude %f, want newest latitude 24.2", location.Latitude)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for location update")
	}
}

func TestLocationController_SubscribeAndUnsubscribe(t *testing.T) {
	controller := NewLocationController(nil)

	truckID := "test-truck-999"

	updates, unsubscribe := controller.subscribe(truckID)

	if updates == nil {
		t.Error("Expected a non-nil channel, but got nil")
	}

	subCount := len(controller.subscribers[truckID])

	if subCount != 1 {
		t.Errorf("Expected 1 subscriber, but got %d", subCount)
	}

	unsubscribe()

	_, exists := controller.subscribers[truckID]

	if exists {
		t.Error("Expected subscription entry for truckID to be removed after unsubscribing, but it still exists (potential memory leak))")
	}
}
