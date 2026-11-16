package services

import (
	"errors"
	"testing"
	"time"

	"real-time-video-surveillance-system/models"
)

func TestValidateLocation_RejectsInvalidHeading(t *testing.T) {
	// Arrange
	validationTime := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	headingDegrees := 360.0
	locationInput := models.LocationInput{Latitude: 25, Longitude: 121, FixQuality: 1, HeadingDegrees: &headingDegrees}

	// Act
	validationError := ValidateLocation(locationInput, validationTime)

	// Assert
	if !errors.Is(validationError, ErrInvalidLocation) {
		t.Errorf("validation error = %v, want invalid location", validationError)
	}
}

func TestValidateLocation_RejectsNegativeAccuracy(t *testing.T) {
	// Arrange
	validationTime := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	accuracyMeters := -1.0
	locationInput := models.LocationInput{Latitude: 25, Longitude: 121, FixQuality: 1, AccuracyM: &accuracyMeters}

	// Act
	validationError := ValidateLocation(locationInput, validationTime)

	// Assert
	if !errors.Is(validationError, ErrInvalidLocation) {
		t.Errorf("validation error = %v, want invalid location", validationError)
	}
}

func TestValidateLocation_RejectsNegativeSatelliteCount(t *testing.T) {
	// Arrange
	validationTime := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	satelliteCount := -1
	locationInput := models.LocationInput{Latitude: 25, Longitude: 121, FixQuality: 1, Satellites: &satelliteCount}

	// Act
	validationError := ValidateLocation(locationInput, validationTime)

	// Assert
	if !errors.Is(validationError, ErrInvalidLocation) {
		t.Errorf("validation error = %v, want invalid location", validationError)
	}
}

func TestValidateLocation_RejectsTimestampBeyondFutureLimit(t *testing.T) {
	// Arrange
	validationTime := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	recordedAt := validationTime.Add(5*time.Minute + time.Second)
	locationInput := models.LocationInput{Latitude: 25, Longitude: 121, FixQuality: 1, RecordedAt: &recordedAt}

	// Act
	validationError := ValidateLocation(locationInput, validationTime)

	// Assert
	if !errors.Is(validationError, ErrInvalidLocation) {
		t.Errorf("validation error = %v, want invalid location", validationError)
	}
}

func TestValidateLocation_AcceptsTimestampAtFutureLimit(t *testing.T) {
	// Arrange
	validationTime := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	recordedAt := validationTime.Add(5 * time.Minute)
	locationInput := models.LocationInput{Latitude: 25, Longitude: 121, FixQuality: 1, RecordedAt: &recordedAt}

	// Act
	validationError := ValidateLocation(locationInput, validationTime)

	// Assert
	if validationError != nil {
		t.Errorf("timestamp at the future limit was rejected: %v", validationError)
	}
}

func TestStopState_FirstStoppedLocationCreatesAnchor(t *testing.T) {
	// Arrange
	recordedAt := time.Date(2026, time.November, 11, 8, 0, 0, 0, time.UTC)
	locationInput := models.LocationInput{Latitude: 25.033, Longitude: 121.5654, SpeedKmh: 0.5, FixQuality: 1}

	// Act
	anchorLatitude, anchorLongitude, stoppedSince := StopState(nil, locationInput, recordedAt)

	// Assert
	if anchorLatitude == nil || anchorLongitude == nil || stoppedSince == nil {
		t.Fatal("first stopped location did not initialize the stop state")
	}
	if *anchorLatitude != locationInput.Latitude || *anchorLongitude != locationInput.Longitude || !stoppedSince.Equal(recordedAt) {
		t.Errorf("unexpected initial stop state: latitude %v, longitude %v, stoppedSince %v", *anchorLatitude, *anchorLongitude, stoppedSince)
	}
}
