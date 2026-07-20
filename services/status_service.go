package services

import (
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/models"
)

type mediaMTXPathList struct {
	Items []struct {
		Name   string `json:"name"`
		Online bool   `json:"online"`
	} `json:"items"`
}

type StatusService struct {
	database *sql.DB
}

func NewStatusService(database *sql.DB) *StatusService {
	return &StatusService{database: database}
}

func (s *StatusService) ListenUDP(address string) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Printf("UDP server listening on %s", address)
	buffer := make([]byte, 65535)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("read UDP: %v", err)
			continue
		}
		var status models.TruckStatus
		if err := json.Unmarshal(buffer[:n], &status); err != nil || status.TruckID == "" || status.Status == "" {
			log.Printf("invalid truck status: %v", err)
			continue
		}
		if err := db.UpdateTruckStatus(s.database, status.TruckID, status.Status); err != nil {
			log.Printf("update truck status: %v", err)
			continue
		}
		for _, camera := range status.Cameras {
			if err := db.UpdateCameraStatus(s.database, camera.CameraID, status.TruckID, camera.Status); err != nil {
				log.Printf("update camera %s status: %v", camera.CameraID, err)
			}
		}
	}
}

func (s *StatusService) MonitorOffline(timeout time.Duration) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		count, err := db.MarkStaleCamerasOffline(s.database, timeout.String())
		if err != nil {
			log.Printf("mark stale cameras offline: %v", err)
		} else if count > 0 {
			log.Printf("marked %d stale camera(s) offline", count)
		}
		if err := db.RefreshTruckStatusesFromCameras(s.database); err != nil {
			log.Printf("refresh truck statuses: %v", err)
		}
	}
}

func (s *StatusService) MonitorMediaMTX(baseURL string) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return
	}
	client := &http.Client{Timeout: 3 * time.Second}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		response, err := client.Get(baseURL + "/v3/paths/list?itemsPerPage=1000")
		if err != nil {
			log.Printf("query MediaMTX paths: %v", err)
		} else {
			var paths mediaMTXPathList
			decodeErr := json.NewDecoder(response.Body).Decode(&paths)
			response.Body.Close()
			if response.StatusCode/100 != 2 || decodeErr != nil {
				log.Printf("query MediaMTX paths: status=%d decode=%v", response.StatusCode, decodeErr)
			} else {
				for _, path := range paths.Items {
					truckID, cameraID, _, ok := StreamPathParts(path.Name)
					if ok && path.Online {
						if err := db.UpdateCameraStatus(s.database, cameraID, truckID, "online"); err != nil {
							log.Printf("sync camera %s stream status: %v", cameraID, err)
						}
					}
				}
				if err := db.RefreshTruckStatusesFromCameras(s.database); err != nil {
					log.Printf("refresh truck statuses: %v", err)
				}
			}
		}
		<-ticker.C
	}
}
