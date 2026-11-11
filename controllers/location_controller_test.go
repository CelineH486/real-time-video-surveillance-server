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

func TestLocationHistoryCleanupSchedulesInitialCleanup(t *testing.T) {
	// Arrange
	controller := &LocationController{}
	now := time.Date(2026, 10, 14, 1, 0, 0, 0, time.UTC)

	// Act
	cutoff := controller.locationHistoryCleanupCutoff(now)

	// Assert
	if cutoff.IsZero() {
		t.Fatal("expected first location update to schedule cleanup")
	}
}

func TestLocationHistoryCleanupSkipsSecondCleanupOnSameTaiwanDay(t *testing.T) {
	// Arrange
	controller := &LocationController{
		lastCleanup: time.Date(2026, 10, 7, 16, 0, 0, 0, time.UTC),
	}
	now := time.Date(2026, 10, 14, 2, 0, 0, 0, time.UTC)

	// Act
	cutoff := controller.locationHistoryCleanupCutoff(now)

	// Assert
	if !cutoff.IsZero() {
		t.Fatal("expected same-day cleanup to be skipped")
	}
}

func TestLocationHistoryCleanupSchedulesCleanupOnNextTaiwanDay(t *testing.T) {
	// Arrange
	controller := &LocationController{
		lastCleanup: time.Date(2026, 10, 7, 16, 0, 0, 0, time.UTC),
	}
	now := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)

	// Act
	cutoff := controller.locationHistoryCleanupCutoff(now)

	// Assert
	if cutoff.IsZero() {
		t.Fatal("expected next-day location update to schedule cleanup")
	}
}
