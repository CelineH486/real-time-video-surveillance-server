package models

import "time"

type Recording struct {
	RecordingID int64     `json:"recordingId"`
	TruckID     string    `json:"truckId"`
	CameraID    string    `json:"cameraId"`
	StartedAt   time.Time `json:"startedAt"`
	EndedAt     time.Time `json:"endedAt"`
	FilePath    string    `json:"filePath"`
	FileSize    int64     `json:"fileSize"`
	Status      string    `json:"status"`
}
