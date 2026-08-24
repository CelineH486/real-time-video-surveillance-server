package models

import "time"

type LocationInput struct {
	Latitude       float64    `json:"latitude"`
	Longitude      float64    `json:"longitude"`
	AltitudeM      *float64   `json:"altitudeM,omitempty"`
	SpeedKmh       float64    `json:"speedKmh"`
	HeadingDegrees *float64   `json:"headingDegrees,omitempty"`
	AccuracyM      *float64   `json:"accuracyM,omitempty"`
	Satellites     *int       `json:"satellites,omitempty"`
	FixQuality     int        `json:"fixQuality"`
	RecordedAt     *time.Time `json:"recordedAt,omitempty"`
}

type TruckLocation struct {
	TruckID             string     `json:"truckId"`
	Latitude            float64    `json:"latitude"`
	Longitude           float64    `json:"longitude"`
	AltitudeM           *float64   `json:"altitudeM,omitempty"`
	SpeedKmh            float64    `json:"speedKmh"`
	HeadingDegrees      *float64   `json:"headingDegrees,omitempty"`
	AccuracyM           *float64   `json:"accuracyM,omitempty"`
	Satellites          *int       `json:"satellites,omitempty"`
	FixQuality          int        `json:"fixQuality"`
	RecordedAt          time.Time  `json:"recordedAt"`
	ReceivedAt          time.Time  `json:"receivedAt"`
	StoppedSince        *time.Time `json:"stoppedSince,omitempty"`
	StoppedSeconds      *int64     `json:"stoppedSeconds,omitempty"`
	StopAnchorLatitude  *float64   `json:"-"`
	StopAnchorLongitude *float64   `json:"-"`
}
