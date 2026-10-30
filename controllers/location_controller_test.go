package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"real-time-video-surveillance-system/apiresponse"
	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/models"
	"real-time-video-surveillance-system/services"
)

var fixedLocationTime = time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)

func newLocationMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
		_ = database.Close()
	})
	return database, mock
}

func expectLocationAuthentication(mock sqlmock.Sqlmock, allowed bool) {
	mock.ExpectQuery(`FROM user_api_tokens t\s+INNER JOIN users u`).
		WithArgs(db.APITokenHash("test-token"), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "display_name"}).
			AddRow("user-1", "driver@example.com", "Test Driver"))
	mock.ExpectQuery(`SELECT EXISTS\s*\(\s*SELECT 1 FROM user_trucks`).
		WithArgs("user-1", "truck-1").
		WillReturnRows(sqlmock.NewRows([]string{"allowed"}).AddRow(allowed))
}

func locationRows(location models.TruckLocation) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"truck_id", "latitude", "longitude", "altitude_m", "speed_kmh",
		"heading_degrees", "accuracy_m", "satellites", "fix_quality",
		"recorded_at", "received_at", "stopped_since",
		"stop_anchor_latitude", "stop_anchor_longitude",
	}).AddRow(
		location.TruckID, location.Latitude, location.Longitude, location.AltitudeM,
		location.SpeedKmh, location.HeadingDegrees, location.AccuracyM,
		location.Satellites, location.FixQuality, location.RecordedAt,
		location.ReceivedAt, location.StoppedSince, location.StopAnchorLatitude,
		location.StopAnchorLongitude,
	)
}

func locationRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.SetPathValue("truckID", "truck-1")
	req.Header.Set("Authorization", "Bearer test-token")
	return req
}

func assertLocationError(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, status, recorder.Body.String())
	}
	var response apiresponse.ErrorBody
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Code != code {
		t.Errorf("error code = %q, want %q", response.Error.Code, code)
	}
}

func TestNewLocationController(t *testing.T) {
	controller := NewLocationController(nil)
	if controller == nil {
		t.Fatal("NewLocationController returned nil")
	}
	if controller.now == nil {
		t.Error("controller clock is nil")
	}
	if controller.subscribers == nil {
		t.Error("controller subscriber map is nil")
	}
}

func TestLocationControllerCurrentRequiresAuthentication(t *testing.T) {
	controller := NewLocationController(nil)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trucks/truck-1/location", nil)
	req.SetPathValue("truckID", "truck-1")

	controller.Current(recorder, req)

	assertLocationError(t, recorder, http.StatusUnauthorized, apiresponse.CodeAuthenticationRequired)
}

func TestLocationControllerCurrent(t *testing.T) {
	tests := []struct {
		name       string
		locationDB func(sqlmock.Sqlmock)
		wantStatus int
		wantCode   string
	}{
		{
			name: "returns current location",
			locationDB: func(mock sqlmock.Sqlmock) {
				stoppedSince := fixedLocationTime.Add(-90 * time.Second)
				mock.ExpectQuery(`FROM truck_location_state`).WithArgs("truck-1").
					WillReturnRows(locationRows(models.TruckLocation{
						TruckID: "truck-1", Latitude: 25.033, Longitude: 121.5654,
						SpeedKmh: 0, FixQuality: 1, RecordedAt: fixedLocationTime.Add(-time.Minute),
						ReceivedAt: fixedLocationTime.Add(-time.Minute), StoppedSince: &stoppedSince,
					}))
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "returns not found",
			locationDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`FROM truck_location_state`).WithArgs("truck-1").
					WillReturnError(sql.ErrNoRows)
			},
			wantStatus: http.StatusNotFound,
			wantCode:   apiresponse.CodeLocationNotFound,
		},
		{
			name: "reports database failure",
			locationDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`FROM truck_location_state`).WithArgs("truck-1").
					WillReturnError(errors.New("database unavailable"))
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   apiresponse.CodeLocationUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database, mock := newLocationMock(t)
			expectLocationAuthentication(mock, true)
			test.locationDB(mock)
			controller := NewLocationController(database)
			controller.now = func() time.Time { return fixedLocationTime }
			recorder := httptest.NewRecorder()

			services.NewAuthService(database).Middleware(controller.Current)(
				recorder,
				locationRequest(http.MethodGet, "/trucks/truck-1/location", ""),
			)

			if test.wantCode != "" {
				assertLocationError(t, recorder, test.wantStatus, test.wantCode)
				return
			}
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			var location models.TruckLocation
			if err := json.Unmarshal(recorder.Body.Bytes(), &location); err != nil {
				t.Fatalf("decode location response: %v", err)
			}
			if location.TruckID != "truck-1" || location.StoppedSeconds == nil || *location.StoppedSeconds != 90 {
				t.Errorf("unexpected current location: %+v", location)
			}
		})
	}
}

func TestLocationControllerCurrentRejectsUnassignedTruck(t *testing.T) {
	database, mock := newLocationMock(t)
	expectLocationAuthentication(mock, false)
	controller := NewLocationController(database)
	recorder := httptest.NewRecorder()

	services.NewAuthService(database).Middleware(controller.Current)(
		recorder,
		locationRequest(http.MethodGet, "/trucks/truck-1/location", ""),
	)

	assertLocationError(t, recorder, http.StatusForbidden, apiresponse.CodeTruckAccessDenied)
}

func TestLocationControllerCreateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"latitude":`},
		{name: "unknown field", body: `{"latitude":25,"longitude":121,"speedKmh":5,"fixQuality":1,"extra":true}`},
		{name: "invalid coordinates", body: `{"latitude":91,"longitude":121,"speedKmh":5,"fixQuality":1}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database, mock := newLocationMock(t)
			expectLocationAuthentication(mock, true)
			controller := NewLocationController(database)
			controller.now = func() time.Time { return fixedLocationTime }
			recorder := httptest.NewRecorder()

			services.NewAuthService(database).Middleware(controller.Create)(
				recorder,
				locationRequest(http.MethodPost, "/trucks/truck-1/location", test.body),
			)

			assertLocationError(t, recorder, http.StatusBadRequest, apiresponse.CodeInvalidLocation)
		})
	}
}

func TestLocationControllerCreateReportsReadAndWriteFailures(t *testing.T) {
	tests := []struct {
		name      string
		prepareDB func(sqlmock.Sqlmock)
	}{
		{
			name: "current location read fails",
			prepareDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`FROM truck_location_state`).WithArgs("truck-1").
					WillReturnError(errors.New("read failed"))
			},
		},
		{
			name: "location write fails",
			prepareDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`FROM truck_location_state`).WithArgs("truck-1").
					WillReturnError(sql.ErrNoRows)
				mock.ExpectBegin().WillReturnError(errors.New("begin failed"))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database, mock := newLocationMock(t)
			expectLocationAuthentication(mock, true)
			test.prepareDB(mock)
			controller := NewLocationController(database)
			controller.now = func() time.Time { return fixedLocationTime }
			recorder := httptest.NewRecorder()

			services.NewAuthService(database).Middleware(controller.Create)(
				recorder,
				locationRequest(http.MethodPost, "/trucks/truck-1/location", `{"latitude":25.033,"longitude":121.5654,"speedKmh":8,"fixQuality":1}`),
			)

			assertLocationError(t, recorder, http.StatusServiceUnavailable, apiresponse.CodeLocationUnavailable)
		})
	}
}

func TestLocationControllerCreateSavesAndBroadcastsLocation(t *testing.T) {
	database, mock := newLocationMock(t)
	expectLocationAuthentication(mock, true)
	recordedAt := fixedLocationTime.Add(-30 * time.Second)
	stoppedSince := fixedLocationTime.Add(-2 * time.Minute)
	anchorLat, anchorLon := 25.033, 121.5654
	previous := models.TruckLocation{
		TruckID: "truck-1", Latitude: anchorLat, Longitude: anchorLon,
		SpeedKmh: 0, FixQuality: 1, RecordedAt: fixedLocationTime.Add(-time.Minute),
		ReceivedAt: fixedLocationTime.Add(-time.Minute), StoppedSince: &stoppedSince,
		StopAnchorLatitude: &anchorLat, StopAnchorLongitude: &anchorLon,
	}
	saved := previous
	saved.Latitude = 25.03301
	saved.Longitude = 121.56541
	saved.RecordedAt = recordedAt
	saved.ReceivedAt = fixedLocationTime

	mock.ExpectQuery(`FROM truck_location_state`).WithArgs("truck-1").WillReturnRows(locationRows(previous))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO truck_locations`).WithArgs(
		"truck-1", saved.Latitude, saved.Longitude, nil, float64(0), nil, nil, nil, 1,
		recordedAt, fixedLocationTime,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	// Verify that the stop anchor calculated from the previous location reaches persistence.
	mock.ExpectExec(`INSERT INTO truck_location_state`).WithArgs(
		"truck-1", saved.Latitude, saved.Longitude, nil, float64(0), nil, nil, nil, 1,
		recordedAt, fixedLocationTime, anchorLat, anchorLon, stoppedSince,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`FROM truck_location_state`).WithArgs("truck-1").WillReturnRows(locationRows(saved))

	controller := NewLocationController(database)
	controller.now = func() time.Time { return fixedLocationTime }
	updates, unsubscribe := controller.subscribe("truck-1")
	defer unsubscribe()
	recorder := httptest.NewRecorder()
	body := `{"latitude":25.03301,"longitude":121.56541,"speedKmh":0,"fixQuality":1,"recordedAt":"` + recordedAt.Format(time.RFC3339) + `"}`

	services.NewAuthService(database).Middleware(controller.Create)(
		recorder,
		locationRequest(http.MethodPost, "/trucks/truck-1/location", body),
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	var response models.TruckLocation
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode created location: %v", err)
	}
	if response.TruckID != "truck-1" || response.Latitude != saved.Latitude {
		t.Errorf("unexpected created location: %+v", response)
	}
	select {
	case update := <-updates:
		if update.RecordedAt != recordedAt {
			t.Errorf("broadcast recordedAt = %v, want %v", update.RecordedAt, recordedAt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for created location broadcast")
	}
}

func TestLocationBroadcastKeepsNewestUpdate(t *testing.T) {
	controller := NewLocationController(nil)
	updates, unsubscribe := controller.subscribe("truck-1")
	defer unsubscribe()

	controller.broadcast(models.TruckLocation{TruckID: "other-truck", Latitude: 1})
	controller.broadcast(models.TruckLocation{TruckID: "truck-1", Latitude: 24.1})
	controller.broadcast(models.TruckLocation{TruckID: "truck-1", Latitude: 24.2})

	select {
	case location := <-updates:
		if location.Latitude != 24.2 {
			t.Fatalf("latitude = %f, want newest latitude 24.2", location.Latitude)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for location update")
	}
}

func TestLocationSubscribeUnsubscribeRemovesEmptyTruckEntry(t *testing.T) {
	controller := NewLocationController(nil)
	_, unsubscribeFirst := controller.subscribe("truck-1")
	_, unsubscribeSecond := controller.subscribe("truck-1")

	unsubscribeFirst()
	if got := len(controller.subscribers["truck-1"]); got != 1 {
		t.Fatalf("subscriber count = %d, want 1", got)
	}
	unsubscribeSecond()
	if _, exists := controller.subscribers["truck-1"]; exists {
		t.Error("truck subscriber entry still exists after the last unsubscribe")
	}
}

func TestLocationControllerStreamSendsBroadcast(t *testing.T) {
	database, mock := newLocationMock(t)
	expectLocationAuthentication(mock, true)
	controller := NewLocationController(database)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /trucks/{truckID}/stream", services.NewAuthService(database).Middleware(controller.Stream))
	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	streamURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/trucks/truck-1/stream"
	conn, _, err := websocket.Dial(ctx, streamURL, &websocket.DialOptions{
		Subprotocols: []string{"bearer", "test-token"},
	})
	if err != nil {
		t.Fatalf("connect to location stream: %v", err)
	}
	defer conn.CloseNow()

	// Wait until the handler has registered the subscriber before broadcasting.
	deadline := time.Now().Add(time.Second)
	for {
		controller.mu.Lock()
		count := len(controller.subscribers["truck-1"])
		controller.mu.Unlock()
		if count == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("stream subscriber was not registered")
		}
		time.Sleep(5 * time.Millisecond)
	}

	want := models.TruckLocation{
		TruckID: "truck-1", Latitude: 25.033, Longitude: 121.5654,
		SpeedKmh: 12, FixQuality: 1, RecordedAt: fixedLocationTime, ReceivedAt: fixedLocationTime,
	}
	controller.broadcast(want)
	var got models.TruckLocation
	if err := wsjson.Read(ctx, conn, &got); err != nil {
		t.Fatalf("read streamed location: %v", err)
	}
	if got.TruckID != want.TruckID || got.Latitude != want.Latitude {
		t.Errorf("streamed location = %+v, want %+v", got, want)
	}
	if err := conn.Close(websocket.StatusNormalClosure, "test complete"); err != nil {
		t.Fatalf("close location stream: %v", err)
	}
}
