package models

type TruckStatus struct {
	TruckID string         `json:"truckId"`
	Status  string         `json:"status"`
	Cameras []CameraStatus `json:"cameras,omitempty"`
}

type CameraStatus struct {
	CameraID string `json:"cameraId"`
	Status   string `json:"status"`
}
