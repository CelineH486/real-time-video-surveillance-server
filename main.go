package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/models"
)

var databaseConn *sql.DB

type streamInfo struct {
	TruckID  string    `json:"truckId"`
	CameraID string    `json:"cameraId"`
	Quality  string    `json:"quality"`
	Protocol string    `json:"protocol"`
	URL      string    `json:"url"`
	Token    string    `json:"accessToken"`
	Expires  time.Time `json:"expiresAt"`
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func startUDPServer() {
	address := env("UDP_ADDRESS", ":5000")
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
		if err := json.Unmarshal(buffer[:n], &status); err != nil {
			log.Printf("invalid truck status: %v", err)
			continue
		}
		if status.TruckID == "" || status.Status == "" {
			log.Printf("invalid truck status: truckId and status are required")
			continue
		}
		if err := db.UpdateTruckStatus(databaseConn, status.TruckID, status.Status); err != nil {
			log.Printf("update truck status: %v", err)
			continue
		}
		for _, camera := range status.Cameras {
			if err := db.UpdateCameraStatus(databaseConn, camera.CameraID, status.TruckID, camera.Status); err != nil {
				log.Printf("update camera %s status: %v", camera.CameraID, err)
			}
		}
	}
}

func startCameraStatusMonitor() {
	timeout, err := time.ParseDuration(env("CAMERA_OFFLINE_TIMEOUT", "15s"))
	if err != nil {
		log.Fatalf("invalid CAMERA_OFFLINE_TIMEOUT: %v", err)
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		count, err := db.MarkStaleCamerasOffline(databaseConn, timeout.String())
		if err != nil {
			log.Printf("mark stale cameras offline: %v", err)
			continue
		}
		if count > 0 {
			log.Printf("marked %d stale camera(s) offline", count)
		}
	}
}

func signStreamAccess(truckID, cameraID, quality string, expires time.Time) string {
	payload := strings.Join([]string{truckID, cameraID, quality, strconv.FormatInt(expires.Unix(), 10)}, "|")
	encodedPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, []byte(env("STREAM_SIGNING_KEY", "development-only-change-me")))
	_, _ = mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encodedPayload + "." + signature
}

func cameraStreamURL(truckID, cameraID, quality, token string) string {
	base := strings.TrimRight(env("STREAM_PUBLIC_BASE_URL", "http://localhost:8889"), "/")
	path := url.PathEscape(truckID) + "/" + url.PathEscape(cameraID) + "/" + quality
	return base + "/" + path + "?token=" + url.QueryEscape(token)
}

func startHTTPServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/cameras", func(w http.ResponseWriter, _ *http.Request) {
		cameras, err := db.GetCameras(databaseConn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, cameras)
	})
	mux.HandleFunc("GET /api/trucks", func(w http.ResponseWriter, _ *http.Request) {
		trucks, err := db.GetTrucks(databaseConn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, trucks)
	})
	mux.HandleFunc("GET /api/trucks/{truckID}/cameras", func(w http.ResponseWriter, r *http.Request) {
		truckID := r.PathValue("truckID")
		cameras, err := db.GetCamerasByTruckID(databaseConn, truckID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		expires := time.Now().Add(5 * time.Minute)
		for _, camera := range cameras {
			cameraID := camera["cameraId"]
			token := signStreamAccess(truckID, cameraID, "sub", expires)
			camera["subUrl"] = cameraStreamURL(truckID, cameraID, "sub", token)
			camera["subToken"] = token
			camera["expiresAt"] = expires.Format(time.RFC3339)
		}
		writeJSON(w, http.StatusOK, cameras)
	})
	mux.HandleFunc("POST /api/trucks/{truckID}/cameras/{cameraID}/play", func(w http.ResponseWriter, r *http.Request) {
		truckID, cameraID := r.PathValue("truckID"), r.PathValue("cameraID")
		exists, err := db.CameraExists(databaseConn, truckID, cameraID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "camera not found", http.StatusNotFound)
			return
		}
		expires := time.Now().Add(5 * time.Minute)
		token := signStreamAccess(truckID, cameraID, "main", expires)
		writeJSON(w, http.StatusOK, streamInfo{
			TruckID: truckID, CameraID: cameraID, Quality: "main", Protocol: "WebRTC",
			URL: cameraStreamURL(truckID, cameraID, "main", token), Token: token, Expires: expires,
		})
	})
	mux.HandleFunc("GET /api/trucks/{truckID}/recordings", func(w http.ResponseWriter, r *http.Request) {
		recordings, err := db.GetRecordings(databaseConn, r.PathValue("truckID"), r.URL.Query().Get("cameraId"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, recordings)
	})

	address := env("HTTP_ADDRESS", ":8080")
	log.Printf("HTTP server listening on %s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func main() {
	var err error
	databaseConn, err = db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer databaseConn.Close()
	go startUDPServer()
	go startCameraStatusMonitor()
	startHTTPServer()
}
