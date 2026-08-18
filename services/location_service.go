package services

import (
	"errors"
	"math"
	"time"

	"real-time-video-surveillance-system/models"
)

const (
	stopSpeedKmh = 3.0
	stopRadiusM  = 30.0
)

var ErrInvalidLocation = errors.New("invalid GPS location")

func ValidateLocation(location models.LocationInput, now time.Time) error {
	if math.IsNaN(location.Latitude) || math.IsInf(location.Latitude, 0) || location.Latitude < -90 || location.Latitude > 90 ||
		math.IsNaN(location.Longitude) || math.IsInf(location.Longitude, 0) || location.Longitude < -180 || location.Longitude > 180 ||
		math.IsNaN(location.SpeedKmh) || math.IsInf(location.SpeedKmh, 0) || location.SpeedKmh < 0 || location.FixQuality <= 0 {
		return ErrInvalidLocation
	}
	if location.HeadingDegrees != nil && (*location.HeadingDegrees < 0 || *location.HeadingDegrees >= 360) {
		return ErrInvalidLocation
	}
	if location.AccuracyM != nil && *location.AccuracyM < 0 {
		return ErrInvalidLocation
	}
	if location.Satellites != nil && *location.Satellites < 0 {
		return ErrInvalidLocation
	}
	if location.RecordedAt != nil && location.RecordedAt.After(now.Add(5*time.Minute)) {
		return ErrInvalidLocation
	}
	return nil
}

func StopState(previous *models.TruckLocation, next models.LocationInput, recordedAt time.Time) (*float64, *float64, *time.Time) {
	if next.SpeedKmh >= stopSpeedKmh {
		return nil, nil, nil
	}
	if previous == nil || previous.StoppedSince == nil {
		lat, lon, since := next.Latitude, next.Longitude, recordedAt
		return &lat, &lon, &since
	}
	anchorLat, anchorLon := previous.Latitude, previous.Longitude
	if previous.StopAnchorLatitude != nil && previous.StopAnchorLongitude != nil {
		anchorLat, anchorLon = *previous.StopAnchorLatitude, *previous.StopAnchorLongitude
	}
	if distanceMeters(anchorLat, anchorLon, next.Latitude, next.Longitude) > stopRadiusM {
		lat, lon, since := next.Latitude, next.Longitude, recordedAt
		return &lat, &lon, &since
	}
	lat, lon := anchorLat, anchorLon
	return &lat, &lon, previous.StoppedSince
}

func distanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusM = 6371000.0
	toRadians := math.Pi / 180
	dLat := (lat2 - lat1) * toRadians
	dLon := (lon2 - lon1) * toRadians
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*toRadians)*math.Cos(lat2*toRadians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
