package services_test

import (
	"testing"
	"time"

	"real-time-video-surveillance-system/models"
	"real-time-video-surveillance-system/services"
)

func TestValidateLocation(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	valid := models.LocationInput{Latitude: 25.033, Longitude: 121.5654, SpeedKmh: 0.5, FixQuality: 1}
	if err := services.ValidateLocation(valid, now); err != nil {
		t.Fatalf("expected valid location: %v", err)
	}
	for _, invalid := range []models.LocationInput{
		{Latitude: 91, Longitude: 121, FixQuality: 1},
		{Latitude: 25, Longitude: 181, FixQuality: 1},
		{Latitude: 25, Longitude: 121, SpeedKmh: -1, FixQuality: 1},
		{Latitude: 25, Longitude: 121, FixQuality: 0},
	} {
		if err := services.ValidateLocation(invalid, now); err == nil {
			t.Fatalf("expected invalid location to fail: %+v", invalid)
		}
	}
}

func TestStopStateKeepsOriginalAnchorWithinRadius(t *testing.T) {
	since := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	anchorLat, anchorLon := 25.033, 121.5654
	previous := &models.TruckLocation{
		Latitude: 25.0331, Longitude: 121.5655, StoppedSince: &since,
		StopAnchorLatitude: &anchorLat, StopAnchorLongitude: &anchorLon,
	}
	_, _, gotSince := services.StopState(previous, models.LocationInput{
		Latitude: 25.03315, Longitude: 121.56555, SpeedKmh: 0.4,
	}, since.Add(2*time.Minute))
	if gotSince == nil || !gotSince.Equal(since) {
		t.Fatalf("expected the original stop start, got %v", gotSince)
	}
}

func TestStopStateResetsAfterMovingOutsideRadius(t *testing.T) {
	since := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	anchorLat, anchorLon := 25.033, 121.5654
	previous := &models.TruckLocation{
		Latitude: anchorLat, Longitude: anchorLon, StoppedSince: &since,
		StopAnchorLatitude: &anchorLat, StopAnchorLongitude: &anchorLon,
	}
	nextAt := since.Add(5 * time.Minute)
	_, _, gotSince := services.StopState(previous, models.LocationInput{
		Latitude: 25.034, Longitude: 121.5654, SpeedKmh: 0.5,
	}, nextAt)
	if gotSince == nil || !gotSince.Equal(nextAt) {
		t.Fatalf("expected a new stop start, got %v", gotSince)
	}
}

func TestStopStateClearsWhileMoving(t *testing.T) {
	lat, lon, since := services.StopState(nil, models.LocationInput{Latitude: 25, Longitude: 121, SpeedKmh: 10}, time.Now())
	if lat != nil || lon != nil || since != nil {
		t.Fatal("expected moving vehicle to have no stop state")
	}
}
