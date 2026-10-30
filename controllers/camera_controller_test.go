package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/services"
)

func TestNewCameraController(t *testing.T) {
	controller := NewCameraController(nil, nil)

	if controller == nil {
		t.Fatal("NewCameraController returned nil")
	}
}

func TestCameraController_List_Unauthorized(t *testing.T) {
	controller := NewCameraController(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/cameras", nil)
	rec := httptest.NewRecorder()

	controller.List(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestCameraController_ListByTruck_Unauthorized(t *testing.T) {
	controller := NewCameraController(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/trucks/truck001/cameras", nil)
	req.SetPathValue("truckID", "truck001")
	rec := httptest.NewRecorder()

	controller.ListByTruck(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf(
			"status = %d, want %d",
			rec.Code,
			http.StatusUnauthorized,
		)
	}
}

func TestCameraController_List_Success(t *testing.T) {

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	defer database.Close()

	userRows := sqlmock.NewRows(
		[]string{"user_id", "email", "display_name"},
	).AddRow(
		"user001", "test@example.com", "Test User",
	)

	mock.ExpectQuery(
		`FROM user_api_tokens t\s+INNER JOIN users u`,
	).
		WithArgs(
			db.APITokenHash("test-token"),
			sqlmock.AnyArg(),
		).
		WillReturnRows(userRows)

	cameraRows := sqlmock.NewRows(
		[]string{
			"camera_id", "truck_id", "name", "status",
		},
	).AddRow(
		"cam01", "truck001", "Front Camera", "online",
	)

	mock.ExpectQuery(
		`FROM cameras c\s+INNER JOIN user_trucks ut`,
	).
		WithArgs("user001").
		WillReturnRows(cameraRows)

	controller := NewCameraController(database, nil)
	auth := services.NewAuthService(database)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/cameras",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	handler := auth.Middleware(controller.List)
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	if !strings.Contains(rec.Body.String(), "cam01") {
		t.Error("response does not contain camera cam01")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet SQL expectations: %v", err)
	}
}

func TestCameraController_ListByTruck_Success(t *testing.T) {

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	defer database.Close()

	userRows := sqlmock.NewRows(
		[]string{"user_id", "email", "display_name"},
	).AddRow(
		"user001", "test@example.com", "Test User",
	)

	mock.ExpectQuery(
		`FROM user_api_tokens t\s+INNER JOIN users u`,
	).
		WithArgs(
			db.APITokenHash("test-token"),
			sqlmock.AnyArg(),
		).
		WillReturnRows(userRows)

	accessRows := sqlmock.NewRows(
		[]string{"allowed"},
	).AddRow(true)

	mock.ExpectQuery(
		`SELECT EXISTS\s*\(\s*SELECT 1 FROM user_trucks`,
	).
		WithArgs("user001", "truck001").
		WillReturnRows(accessRows)

	cameraRows := sqlmock.NewRows(
		[]string{
			"camera_id", "truck_id", "name", "status",
		},
	).AddRow(
		"cam01", "truck001", "Front Camera", "online",
	)

	mock.ExpectQuery(
		`FROM cameras\s+WHERE truck_id = \$1`,
	).
		WithArgs("truck001").
		WillReturnRows(cameraRows)

	streams := services.NewStreamService(
		"https://stream.example",
		"test-signing-key",
		"test-publish-password",
	)

	controller := NewCameraController(database, streams)
	auth := services.NewAuthService(database)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/trucks/truck001/cameras",
		nil,
	)
	req.SetPathValue("truckID", "truck001")
	req.Header.Set("Authorization", "Bearer test-token")

	rec := httptest.NewRecorder()

	handler := auth.Middleware(controller.ListByTruck)
	before := time.Now()
	handler(rec, req)
	after := time.Now()

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d; body = %s",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}

	var cameras []map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &cameras); err != nil {
		t.Fatalf("decode camera response: %v", err)
	}

	if len(cameras) != 1 {
		t.Fatalf("camera count = %d, want 1", len(cameras))
	}

	camera := cameras[0]

	if camera["cameraId"] != "cam01" {
		t.Errorf("cameraId = %q, want %q",
			camera["cameraId"], "cam01")
	}

	expectedURL := "https://stream.example/truck001/cam01/sub/whep"

	if camera["subUrl"] != expectedURL {
		t.Errorf("subUrl = %q, want %q",
			camera["subUrl"], expectedURL)
	}

	if camera["subToken"] == "" {
		t.Error("subToken is empty")
	} else {
		// Validate the signed token instead of checking only its presence.
		claims, err := streams.ValidateAccess(
			camera["subToken"],
			before,
		)
		if err != nil {
			t.Errorf("validate subToken: %v", err)
		} else if claims.TruckID != "truck001" ||
			claims.CameraID != "cam01" ||
			claims.Quality != "sub" {
			t.Errorf("unexpected subToken claims: %+v", claims)
		}
	}

	expiresAt, err := time.Parse(
		time.RFC3339,
		camera["expiresAt"],
	)
	if err != nil {
		t.Fatalf("parse expiresAt: %v", err)
	}

	expectedMin := before.Add(5*time.Minute - time.Second)
	expectedMax := after.Add(5 * time.Minute)

	// Bound the expiry by timestamps captured around the request.
	if expiresAt.Before(expectedMin) ||
		expiresAt.After(expectedMax) {
		t.Errorf("expiresAt = %v, want between %v and %v", expiresAt, expectedMin, expectedMax)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet SQL expectations: %v", err)
	}
}

func TestCameraController_List_DatabaseError(t *testing.T) {

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	defer database.Close()

	userRows := sqlmock.NewRows(
		[]string{"user_id", "email", "display_name"},
	).AddRow("user001", "test@example.com", "Test User")

	mock.ExpectQuery(
		`FROM user_api_tokens t\s+INNER JOIN users u`,
	).
		WithArgs(
			db.APITokenHash("test-token"),
			sqlmock.AnyArg(),
		).
		WillReturnRows(userRows)

	mock.ExpectQuery(
		`FROM cameras c\s+INNER JOIN user_trucks ut`,
	).
		WithArgs("user001").
		WillReturnError(errors.New("camera database query failed"))

	controller := NewCameraController(database, nil)
	auth := services.NewAuthService(database)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/cameras",
		nil,
	)

	req.Header.Set("Authorization", "Bearer test-token")

	rec := httptest.NewRecorder()

	handler := auth.Middleware(controller.List)
	handler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(
			"status = %d, want %d; body = %s",
			rec.Code,
			http.StatusInternalServerError,
			rec.Body.String(),
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet SQL expectations: %v", err)
	}
}

func TestCameraController_ListByTruck_DatabaseError(t *testing.T) {

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	defer database.Close()

	userRows := sqlmock.NewRows(
		[]string{"user_id", "email", "display_name"},
	).AddRow("user001", "test@example.com", "Test User")

	mock.ExpectQuery(
		`FROM user_api_tokens t\s+INNER JOIN users u`,
	).
		WithArgs(
			db.APITokenHash("test-token"),
			sqlmock.AnyArg(),
		).
		WillReturnRows(userRows)

	accessRows := sqlmock.NewRows(
		[]string{"allowed"},
	).AddRow(true)

	mock.ExpectQuery(
		`SELECT EXISTS\s*\(\s*SELECT 1 FROM user_trucks`,
	).
		WithArgs("user001", "truck001").
		WillReturnRows(accessRows)

	mock.ExpectQuery(
		`FROM cameras\s+WHERE truck_id = \$1`,
	).
		WithArgs("truck001").
		WillReturnError(
			errors.New("truck camera database query failed"),
		)

	controller := NewCameraController(database, nil)
	auth := services.NewAuthService(database)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/trucks/truck001/cameras",
		nil,
	)
	req.SetPathValue("truckID", "truck001")
	req.Header.Set("Authorization", "Bearer test-token")

	rec := httptest.NewRecorder()

	handler := auth.Middleware(controller.ListByTruck)
	handler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(
			"status = %d, want %d; body = %s",
			rec.Code,
			http.StatusInternalServerError,
			rec.Body.String(),
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet SQL expectations: %v", err)
	}
}
