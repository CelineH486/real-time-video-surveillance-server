package models

type Camera struct {
	CameraID string `json:"cameraId"`
	TruckID  string `json:"truckId"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	MainPath string `json:"mainPath"`
	SubPath  string `json:"subPath"`
}
