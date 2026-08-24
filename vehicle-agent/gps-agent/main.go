package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type config struct {
	TruckID         string
	SerialPort      string
	ControlPort     string
	APIBaseURL      string
	APIToken        string
	UploadEvery     time.Duration
	RequestTimeout  time.Duration
	AutoStartGPS    bool
	ConfigureSerial bool
}

type locationPayload struct {
	Latitude       float64    `json:"latitude"`
	Longitude      float64    `json:"longitude"`
	AltitudeM      *float64   `json:"altitudeM,omitempty"`
	SpeedKmh       float64    `json:"speedKmh"`
	HeadingDegrees *float64   `json:"headingDegrees,omitempty"`
	Satellites     *int       `json:"satellites,omitempty"`
	FixQuality     int        `json:"fixQuality"`
	RecordedAt     *time.Time `json:"recordedAt,omitempty"`
}

type nmeaState struct {
	speedKmh   float64
	heading    *float64
	recordedAt *time.Time
	rmcTime    string
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	runContinuously(cfg)
}

func runContinuously(cfg config) {
	const reconnectDelay = 5 * time.Second
	for {
		if cfg.ConfigureSerial {
			if err := configureSerial(cfg.SerialPort); err != nil {
				log.Printf("configure NMEA serial port: %v", err)
			}
		}
		if cfg.AutoStartGPS {
			if err := startGPS(cfg.ControlPort, cfg.ConfigureSerial); err != nil {
				log.Printf("start GPS: %v", err)
			} else {
				log.Printf("GPS start command sent on %s", cfg.ControlPort)
			}
		}
		log.Printf("reading NMEA from %s for truck %s", cfg.SerialPort, cfg.TruckID)
		if err := run(cfg); err != nil {
			log.Printf("GPS connection lost: %v", err)
		}
		log.Printf("retrying GPS connection in %s", reconnectDelay)
		time.Sleep(reconnectDelay)
	}
}

func loadConfig() (config, error) {
	uploadEvery, err := time.ParseDuration(env("GPS_UPLOAD_INTERVAL", "1s"))
	if err != nil || uploadEvery < time.Second {
		return config{}, errors.New("GPS_UPLOAD_INTERVAL must be a duration of at least 1s")
	}
	requestTimeout, err := time.ParseDuration(env("GPS_REQUEST_TIMEOUT", "10s"))
	if err != nil || requestTimeout <= 0 {
		return config{}, errors.New("GPS_REQUEST_TIMEOUT must be a positive duration")
	}
	cfg := config{
		TruckID:         strings.TrimSpace(os.Getenv("GPS_TRUCK_ID")),
		SerialPort:      env("GPS_SERIAL_PORT", "/dev/ttyUSB1"),
		ControlPort:     env("GPS_CONTROL_PORT", "/dev/ttyUSB2"),
		APIBaseURL:      strings.TrimRight(strings.TrimSpace(os.Getenv("GPS_API_BASE_URL")), "/"),
		APIToken:        strings.TrimSpace(os.Getenv("GPS_API_TOKEN")),
		UploadEvery:     uploadEvery,
		RequestTimeout:  requestTimeout,
		AutoStartGPS:    !strings.EqualFold(env("GPS_AUTO_START", "true"), "false"),
		ConfigureSerial: !strings.EqualFold(env("GPS_CONFIGURE_SERIAL", "true"), "false"),
	}
	if cfg.TruckID == "" || cfg.APIBaseURL == "" || cfg.APIToken == "" {
		return config{}, errors.New("GPS_TRUCK_ID, GPS_API_BASE_URL, and GPS_API_TOKEN are required")
	}
	return cfg, nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func configureSerial(port string) error {
	return exec.Command("stty", "-F", port, "115200", "raw", "-echo", "-ixon", "-ixoff").Run()
}

func startGPS(port string, configure bool) error {
	if configure {
		if err := configureSerial(port); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(port, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString("AT+QGPS=1\r\n")
	return err
}

func run(cfg config) error {
	serial, err := os.Open(cfg.SerialPort)
	if err != nil {
		return fmt.Errorf("open NMEA serial port: %w", err)
	}
	defer serial.Close()

	client := &http.Client{Timeout: cfg.RequestTimeout}
	scanner := bufio.NewScanner(serial)
	scanner.Buffer(make([]byte, 4096), 64*1024)
	state := nmeaState{}
	lastUpload := time.Time{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := nmeaFields(line)
		if len(fields) == 0 {
			continue
		}
		sentence := fields[0]
		switch {
		case strings.HasSuffix(sentence, "RMC"):
			updateRMC(&state, fields)
		case strings.HasSuffix(sentence, "GGA"):
			payload, ok := parseGGA(fields, state)
			if !ok || time.Since(lastUpload) < cfg.UploadEvery {
				continue
			}
			if err := upload(client, cfg, payload); err != nil {
				log.Printf("upload GPS location: %v", err)
				continue
			}
			lastUpload = time.Now()
			log.Printf("location uploaded: %.6f, %.6f fix=%d", payload.Latitude, payload.Longitude, payload.FixQuality)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read NMEA serial port: %w", err)
	}
	return errors.New("NMEA serial port closed")
}

func nmeaFields(line string) []string {
	start := strings.IndexByte(line, '$')
	if start < 0 {
		return nil
	}
	line = line[start:]
	if checksum := strings.IndexByte(line, '*'); checksum >= 0 {
		line = line[:checksum]
	}
	return strings.Split(line, ",")
}

func updateRMC(state *nmeaState, fields []string) {
	if len(fields) < 10 || fields[2] != "A" {
		return
	}
	state.rmcTime = fields[1]
	if knots, err := strconv.ParseFloat(fields[7], 64); err == nil && knots >= 0 {
		state.speedKmh = knots * 1.852
	}
	if heading, err := strconv.ParseFloat(fields[8], 64); err == nil && heading >= 0 && heading < 360 {
		state.heading = &heading
	} else {
		state.heading = nil
	}
	if recordedAt, err := parseNMEATime(fields[9], fields[1]); err == nil {
		state.recordedAt = &recordedAt
	}
}

func parseGGA(fields []string, state nmeaState) (locationPayload, bool) {
	if len(fields) < 10 {
		return locationPayload{}, false
	}
	fixQuality, err := strconv.Atoi(fields[6])
	if err != nil || fixQuality <= 0 {
		return locationPayload{}, false
	}
	latitude, err := nmeaCoordinate(fields[2], fields[3], true)
	if err != nil {
		return locationPayload{}, false
	}
	longitude, err := nmeaCoordinate(fields[4], fields[5], false)
	if err != nil {
		return locationPayload{}, false
	}
	payload := locationPayload{
		Latitude: latitude, Longitude: longitude, SpeedKmh: state.speedKmh,
		HeadingDegrees: state.heading, FixQuality: fixQuality,
	}
	if satellites, err := strconv.Atoi(fields[7]); err == nil && satellites >= 0 {
		payload.Satellites = &satellites
	}
	if altitude, err := strconv.ParseFloat(fields[9], 64); err == nil {
		payload.AltitudeM = &altitude
	}
	if state.recordedAt != nil && sameNMEATime(fields[1], state.rmcTime) {
		payload.RecordedAt = state.recordedAt
	}
	return payload, true
}

func nmeaCoordinate(raw, hemisphere string, latitude bool) (float64, error) {
	degreeDigits := 3
	if latitude {
		degreeDigits = 2
	}
	if len(raw) <= degreeDigits {
		return 0, errors.New("coordinate is too short")
	}
	degrees, err := strconv.ParseFloat(raw[:degreeDigits], 64)
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.ParseFloat(raw[degreeDigits:], 64)
	if err != nil || minutes < 0 || minutes >= 60 {
		return 0, errors.New("invalid coordinate minutes")
	}
	value := degrees + minutes/60
	if hemisphere == "S" || hemisphere == "W" {
		value = -value
	} else if hemisphere != "N" && hemisphere != "E" {
		return 0, errors.New("invalid coordinate hemisphere")
	}
	if latitude && math.Abs(value) > 90 || !latitude && math.Abs(value) > 180 {
		return 0, errors.New("coordinate is out of range")
	}
	return value, nil
}

func parseNMEATime(date, clock string) (time.Time, error) {
	if len(date) != 6 || len(clock) < 6 {
		return time.Time{}, errors.New("invalid NMEA date or time")
	}
	base := date + clock[:6]
	return time.ParseInLocation("020106150405", base, time.UTC)
}

func sameNMEATime(left, right string) bool {
	return len(left) >= 6 && len(right) >= 6 && left[:6] == right[:6]
}

func upload(client *http.Client, cfg config, payload locationPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/trucks/%s/locations", cfg.APIBaseURL, cfg.TruckID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIToken)
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("API returned %s", response.Status)
	}
	return nil
}
