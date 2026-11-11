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

var ErrInvalidLocationHistoryDate = errors.New("location history date must be within the last 7 Taiwan calendar days")

const LocationHistoryRetentionDays = 7

var taiwanLocation = time.FixedZone("Asia/Taipei", 8*60*60)

func LocationHistoryDayRange(date string, now time.Time) (time.Time, time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", date, taiwanLocation)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidLocationHistoryDate
	}
	today := now.In(taiwanLocation)
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, taiwanLocation)
	oldest := todayStart.AddDate(0, 0, -(LocationHistoryRetentionDays - 1))
	if day.Before(oldest) || day.After(todayStart) {
		return time.Time{}, time.Time{}, ErrInvalidLocationHistoryDate
	}
	return day.UTC(), day.AddDate(0, 0, 1).UTC(), nil
}

func LocationHistoryCutoff(now time.Time) time.Time {
	today := now.In(taiwanLocation)
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, taiwanLocation)
	return todayStart.AddDate(0, 0, -(LocationHistoryRetentionDays - 1)).UTC()
}

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
