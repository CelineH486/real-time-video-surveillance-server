package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

type streamTokenClaims struct {
	TruckID  string
	CameraID string
	Quality  string
	Expires  time.Time
}

type mediaMTXAuthRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Token    string `json:"token"`
	Action   string `json:"action"`
	Path     string `json:"path"`
	Protocol string `json:"protocol"`
}

type recordingPlayRequest struct {
	Start    time.Time `json:"start"`
	Duration float64   `json:"durationSeconds"`
}

type mediaMTXRecordingSpan struct {
	Start    time.Time `json:"start"`
	Duration float64   `json:"duration"`
}

type recordingSpan struct {
	Start    time.Time `json:"start"`
	Duration float64   `json:"durationSeconds"`
	URL      string    `json:"url"`
	Expires  time.Time `json:"expiresAt"`
}

type mediaMTXPathList struct {
	Items []struct {
		Name   string `json:"name"`
		Online bool   `json:"online"`
	} `json:"items"`
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

func startMediaMTXPathMonitor() {
	base := strings.TrimRight(os.Getenv("MEDIAMTX_API_URL"), "/")
	if base == "" {
		return
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	client := &http.Client{Timeout: 3 * time.Second}
	for {
		response, err := client.Get(base + "/v3/paths/list?itemsPerPage=1000")
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
					truckID, cameraID, _, ok := streamPathParts(path.Name)
					if !ok || !path.Online {
						continue
					}
					if err := db.UpdateCameraStatus(databaseConn, cameraID, truckID, "online"); err != nil {
						log.Printf("sync camera %s stream status: %v", cameraID, err)
					}
				}
			}
		}
		<-ticker.C
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

func validateStreamAccess(token string, now time.Time) (streamTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return streamTokenClaims{}, fmt.Errorf("invalid token format")
	}
	mac := hmac.New(sha256.New, []byte(env("STREAM_SIGNING_KEY", "development-only-change-me")))
	_, _ = mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(actual, expected) {
		return streamTokenClaims{}, fmt.Errorf("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return streamTokenClaims{}, fmt.Errorf("invalid token payload")
	}
	values := strings.Split(string(payload), "|")
	if len(values) != 4 {
		return streamTokenClaims{}, fmt.Errorf("invalid token claims")
	}
	expiresUnix, err := strconv.ParseInt(values[3], 10, 64)
	if err != nil {
		return streamTokenClaims{}, fmt.Errorf("invalid token expiry")
	}
	claims := streamTokenClaims{
		TruckID: values[0], CameraID: values[1], Quality: values[2], Expires: time.Unix(expiresUnix, 0),
	}
	if !claims.Expires.After(now) {
		return streamTokenClaims{}, fmt.Errorf("token expired")
	}
	return claims, nil
}

func streamPathParts(path string) (truckID, cameraID, quality string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || (parts[2] != "main" && parts[2] != "sub") {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func cameraStreamURL(truckID, cameraID, quality string) string {
	base := strings.TrimRight(env("STREAM_PUBLIC_BASE_URL", "http://localhost:8889"), "/")
	path := url.PathEscape(truckID) + "/" + url.PathEscape(cameraID) + "/" + quality
	return base + "/" + path + "/whep"
}

func mediaMTXRecordingURL(base, truckID, cameraID string, start time.Time, duration float64) string {
	query := url.Values{}
	query.Set("path", strings.Join([]string{truckID, cameraID, "main"}, "/"))
	query.Set("start", start.Format(time.RFC3339Nano))
	query.Set("duration", strconv.FormatFloat(duration, 'f', -1, 64))
	query.Set("format", "mp4")
	return strings.TrimRight(base, "/") + "/get?" + query.Encode()
}

func recordingAPIURL(truckID, cameraID string, start time.Time, duration float64, token string) string {
	query := url.Values{}
	query.Set("start", start.Format(time.RFC3339Nano))
	query.Set("duration", strconv.FormatFloat(duration, 'f', -1, 64))
	query.Set("token", token)
	base := strings.TrimRight(env("API_PUBLIC_BASE_URL", "http://localhost:8080"), "/")
	return base + "/api/trucks/" + url.PathEscape(truckID) + "/cameras/" + url.PathEscape(cameraID) +
		"/recordings/content?" + query.Encode()
}

func startHTTPServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /internal/mediamtx/auth", func(w http.ResponseWriter, r *http.Request) {
		var request mediaMTXAuthRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			http.Error(w, "invalid authentication request", http.StatusBadRequest)
			return
		}
		truckID, cameraID, quality, ok := streamPathParts(request.Path)
		if !ok {
			http.Error(w, "invalid stream path", http.StatusForbidden)
			return
		}
		switch request.Action {
		case "publish":
			if request.User != truckID || !hmac.Equal(
				[]byte(request.Password), []byte(env("STREAM_PUBLISH_PASSWORD", "development-publish-password")),
			) {
				http.Error(w, "publish denied", http.StatusUnauthorized)
				return
			}
			exists, err := db.CameraExists(databaseConn, truckID, cameraID)
			if err != nil {
				http.Error(w, "database unavailable", http.StatusServiceUnavailable)
				return
			}
			if !exists {
				http.Error(w, "camera not found", http.StatusForbidden)
				return
			}
		case "read", "playback":
			claims, err := validateStreamAccess(request.Token, time.Now())
			if err != nil || claims.TruckID != truckID || claims.CameraID != cameraID || claims.Quality != quality {
				http.Error(w, "stream access denied", http.StatusUnauthorized)
				return
			}
		default:
			http.Error(w, "action denied", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNoContent)
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
			camera["subUrl"] = cameraStreamURL(truckID, cameraID, "sub")
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
			URL: cameraStreamURL(truckID, cameraID, "main"), Token: token, Expires: expires,
		})
	})
	mux.HandleFunc("GET /api/trucks/{truckID}/recordings", func(w http.ResponseWriter, r *http.Request) {
		truckID, cameraID := r.PathValue("truckID"), r.URL.Query().Get("cameraId")
		if cameraID == "" {
			http.Error(w, "cameraId is required", http.StatusBadRequest)
			return
		}
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
		query := url.Values{}
		query.Set("path", strings.Join([]string{truckID, cameraID, "main"}, "/"))
		query.Set("token", token)
		for _, parameter := range []string{"start", "end"} {
			if value := r.URL.Query().Get(parameter); value != "" {
				if _, err := time.Parse(time.RFC3339, value); err != nil {
					http.Error(w, parameter+" must use RFC3339 format", http.StatusBadRequest)
					return
				}
				query.Set(parameter, value)
			}
		}
		internalBase := strings.TrimRight(env("PLAYBACK_INTERNAL_BASE_URL", "http://localhost:9996"), "/")
		request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, internalBase+"/list?"+query.Encode(), nil)
		if err != nil {
			http.Error(w, "create playback request", http.StatusInternalServerError)
			return
		}
		request.Header.Set("Authorization", "Bearer "+token)
		client := &http.Client{Timeout: 5 * time.Second}
		response, err := client.Do(request)
		if err != nil {
			http.Error(w, "recording service unavailable", http.StatusBadGateway)
			return
		}
		defer response.Body.Close()
		if response.StatusCode/100 != 2 {
			http.Error(w, "recording service rejected request", http.StatusBadGateway)
			return
		}
		var source []mediaMTXRecordingSpan
		if err := json.NewDecoder(response.Body).Decode(&source); err != nil {
			http.Error(w, "invalid recording service response", http.StatusBadGateway)
			return
		}
		result := make([]recordingSpan, 0, len(source))
		for _, span := range source {
			result = append(result, recordingSpan{
				Start: span.Start, Duration: span.Duration,
				URL: recordingAPIURL(truckID, cameraID, span.Start, span.Duration, token), Expires: expires,
			})
		}
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("POST /api/trucks/{truckID}/cameras/{cameraID}/recordings/play", func(w http.ResponseWriter, r *http.Request) {
		truckID, cameraID := r.PathValue("truckID"), r.PathValue("cameraID")
		var request recordingPlayRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil || request.Start.IsZero() {
			http.Error(w, "start and durationSeconds are required", http.StatusBadRequest)
			return
		}
		if request.Duration <= 0 || request.Duration > 24*60*60 {
			http.Error(w, "durationSeconds must be between 0 and 86400", http.StatusBadRequest)
			return
		}
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
		writeJSON(w, http.StatusOK, map[string]any{
			"url":         recordingAPIURL(truckID, cameraID, request.Start, request.Duration, token),
			"accessToken": token, "expiresAt": expires,
		})
	})
	mux.HandleFunc("GET /api/trucks/{truckID}/cameras/{cameraID}/recordings/content", func(w http.ResponseWriter, r *http.Request) {
		truckID, cameraID := r.PathValue("truckID"), r.PathValue("cameraID")
		token := r.URL.Query().Get("token")
		claims, err := validateStreamAccess(token, time.Now())
		if err != nil || claims.TruckID != truckID || claims.CameraID != cameraID || claims.Quality != "main" {
			http.Error(w, "recording access denied", http.StatusUnauthorized)
			return
		}
		start, err := time.Parse(time.RFC3339, r.URL.Query().Get("start"))
		if err != nil {
			http.Error(w, "start must use RFC3339 format", http.StatusBadRequest)
			return
		}
		duration, err := strconv.ParseFloat(r.URL.Query().Get("duration"), 64)
		if err != nil || duration <= 0 || duration > 24*60*60 {
			http.Error(w, "invalid duration", http.StatusBadRequest)
			return
		}
		playbackURL := mediaMTXRecordingURL(
			env("PLAYBACK_INTERNAL_BASE_URL", "http://localhost:9996"), truckID, cameraID, start, duration,
		)
		request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, playbackURL, nil)
		if err != nil {
			http.Error(w, "create playback request", http.StatusInternalServerError)
			return
		}
		request.Header.Set("Authorization", "Bearer "+token)
		response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
		if err != nil {
			http.Error(w, "recording service unavailable", http.StatusBadGateway)
			return
		}
		defer response.Body.Close()
		if response.StatusCode/100 != 2 {
			http.Error(w, "recording service rejected request", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
		w.Header().Set("Cache-Control", "private, no-store")
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, response.Body); err != nil {
			log.Printf("stream recording response: %v", err)
		}
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
	go startMediaMTXPathMonitor()
	startHTTPServer()
}
