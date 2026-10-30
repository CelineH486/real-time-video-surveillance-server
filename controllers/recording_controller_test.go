package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"real-time-video-surveillance-system/db"
	"real-time-video-surveillance-system/services"
)

const (
	recordingTruckID  = "truck001"
	recordingCameraID = "cam01"
	recordingToken    = "recording-test-token"
	recordingStart    = "2026-10-20T08:00:00Z"
)

var recordingStartTime = time.Date(2026, time.October, 20, 8, 0, 0, 0, time.UTC)

func newRecordingMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}

	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}

		mock.ExpectClose()

		if err := database.Close(); err != nil {
			t.Errorf("close SQL mock: %v", err)
		}
	})

	return database, mock
}

func expectRecordingUser(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`FROM user_api_tokens t\s+INNER JOIN users u`).
		WithArgs(db.APITokenHash(recordingToken), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "display_name"}).
			AddRow("user001", "test@example.com", "Test User"))
}

func expectRecordingAccess(mock sqlmock.Sqlmock, allowed bool, queryErr error) {
	expected := mock.ExpectQuery(`FROM user_trucks WHERE user_id = \$1 AND truck_id = \$2`).
		WithArgs("user001", recordingTruckID)
	if queryErr != nil {
		expected.WillReturnError(queryErr)
		return
	}
	expected.WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(allowed))
}

func expectRecordingCamera(mock sqlmock.Sqlmock, exists bool, queryErr error) {
	expected := mock.ExpectQuery(`FROM cameras WHERE truck_id = \$1 AND camera_id = \$2`).
		WithArgs(recordingTruckID, recordingCameraID)
	if queryErr != nil {
		expected.WillReturnError(queryErr)
		return
	}
	expected.WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(exists))
}

func recordingStreamService() *services.StreamService {
	return services.NewStreamService("https://stream.example", "recording-test-signing-key", "publish-password")
}

func recordingServices(baseURL string) (*services.RecordingService, *services.StreamService) {
	streams := recordingStreamService()
	return services.NewRecordingService(baseURL, "https://api.example", streams), streams
}

func recordingRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.SetPathValue("truckID", recordingTruckID)
	req.SetPathValue("cameraID", recordingCameraID)
	req.Header.Set("Authorization", "Bearer "+recordingToken)
	return req
}

func recordingAuthorizedRequest(database *sql.DB, next http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	services.NewAuthService(database).Middleware(next)(rec, req)
	return rec
}

func expectRecordingStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, want, rec.Body.String())
	}
}

func TestNewRecordingController(t *testing.T) {
	controller := NewRecordingController(nil, nil, nil)
	if controller == nil {
		t.Fatal("NewRecordingController returned nil")
	}
}

func TestRecordingController_List_Unauthorized(t *testing.T) {
	controller := NewRecordingController(nil, nil, nil)
	req := recordingRequest(http.MethodGet, "/api/trucks/truck001/recordings?cameraId=cam01", "")
	rec := httptest.NewRecorder()
	controller.List(rec, req)
	expectRecordingStatus(t, rec, http.StatusUnauthorized)
}

func TestRecordingController_Play_Unauthorized(t *testing.T) {
	controller := NewRecordingController(nil, nil, nil)
	req := recordingRequest(http.MethodPost, "/api/trucks/truck001/cameras/cam01/play", "")
	rec := httptest.NewRecorder()
	controller.Play(rec, req)
	expectRecordingStatus(t, rec, http.StatusUnauthorized)
}

func TestRecordingController_List_AccessErrors(t *testing.T) {
	cases := []struct {
		name     string
		allowed  bool
		queryErr error
		wantCode int
	}{
		{"Forbidden", false, nil, http.StatusForbidden},
		{"AccessDatabaseError", false, errors.New("truck access query failed"), http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			database, mock := newRecordingMock(t)
			expectRecordingUser(mock)
			expectRecordingAccess(mock, tc.allowed, tc.queryErr)
			controller := NewRecordingController(database, nil, nil)
			req := recordingRequest(http.MethodGet, "/api/trucks/truck001/recordings?cameraId=cam01", "")
			rec := recordingAuthorizedRequest(database, controller.List, req)
			expectRecordingStatus(t, rec, tc.wantCode)
		})
	}
}

func TestRecordingController_List_Validation(t *testing.T) {
	cases := []struct {
		name         string
		query        string
		cameraCheck  bool
		cameraExists bool
		cameraError  error
		wantCode     int
	}{
		{"MissingCameraID", "", false, false, nil, http.StatusBadRequest},
		{"CameraNotFound", "cameraId=cam01", true, false, nil, http.StatusNotFound},
		{"CameraLookupError", "cameraId=cam01", true, false, errors.New("camera lookup failed"), http.StatusInternalServerError},
		{"InvalidStart", "cameraId=cam01&start=bad-time", true, true, nil, http.StatusBadRequest},
		{"InvalidEnd", "cameraId=cam01&end=bad-time", true, true, nil, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			database, mock := newRecordingMock(t)
			expectRecordingUser(mock)
			expectRecordingAccess(mock, true, nil)
			if tc.cameraCheck {
				expectRecordingCamera(mock, tc.cameraExists, tc.cameraError)
			}
			controller := NewRecordingController(database, nil, nil)
			target := "/api/trucks/truck001/recordings"
			if tc.query != "" {
				target += "?" + tc.query
			}
			rec := recordingAuthorizedRequest(database, controller.List, recordingRequest(http.MethodGet, target, ""))
			expectRecordingStatus(t, rec, tc.wantCode)
		})
	}
}

func TestRecordingController_List_Success(t *testing.T) {
	var upstreamCalled bool
	// The local server verifies the exact request forwarded to MediaMTX.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		if r.URL.Path != "/list" || r.Method != http.MethodGet {
			t.Errorf("MediaMTX request = %s %s, want GET /list", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("path"); got != "truck001/cam01/main" {
			t.Errorf("MediaMTX path = %q, want %q", got, "truck001/cam01/main")
		}
		if got := r.URL.Query().Get("segments"); got != "true" {
			t.Errorf("segments = %q, want %q", got, "true")
		}
		if got := r.URL.Query().Get("start"); got != recordingStart {
			t.Errorf("start = %q, want %q", got, recordingStart)
		}
		if got := r.URL.Query().Get("end"); got != "2026-10-20T09:00:00Z" {
			t.Errorf("end = %q, want %q", got, "2026-10-20T09:00:00Z")
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("MediaMTX request is missing a Bearer token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"start":"2026-10-20T08:00:00Z","duration":30.5}]`))
	}))
	defer server.Close()

	database, mock := newRecordingMock(t)
	expectRecordingUser(mock)
	expectRecordingAccess(mock, true, nil)
	expectRecordingCamera(mock, true, nil)
	recordings, streams := recordingServices(server.URL)
	controller := NewRecordingController(database, recordings, streams)
	target := "/api/trucks/truck001/recordings?cameraId=cam01&start=" +
		url.QueryEscape(recordingStart) + "&end=" + url.QueryEscape("2026-10-20T09:00:00Z")
	rec := recordingAuthorizedRequest(database, controller.List, recordingRequest(http.MethodGet, target, ""))
	expectRecordingStatus(t, rec, http.StatusOK)
	if !upstreamCalled {
		t.Fatal("RecordingService did not call the MediaMTX test server")
	}
	var spans []services.RecordingSpan
	if err := json.Unmarshal(rec.Body.Bytes(), &spans); err != nil {
		t.Fatalf("decode recording list response: %v", err)
	}
	if len(spans) != 1 || spans[0].Duration != 30.5 || spans[0].Start.Format(time.RFC3339) != recordingStart {
		t.Fatalf("unexpected recording spans: %+v", spans)
	}
}

func TestRecordingController_List_UpstreamError(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"MediaMTXUnavailable", http.StatusServiceUnavailable, `{"error":"unavailable"}`},
		{"InvalidMediaMTXJSON", http.StatusOK, `not-json`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			database, mock := newRecordingMock(t)
			expectRecordingUser(mock)
			expectRecordingAccess(mock, true, nil)
			expectRecordingCamera(mock, true, nil)
			recordings, streams := recordingServices(server.URL)
			controller := NewRecordingController(database, recordings, streams)
			req := recordingRequest(http.MethodGet, "/api/trucks/truck001/recordings?cameraId=cam01", "")
			rec := recordingAuthorizedRequest(database, controller.List, req)
			expectRecordingStatus(t, rec, http.StatusBadGateway)
		})
	}
}

func TestRecordingController_Play_Validation(t *testing.T) {
	cases := []struct {
		name         string
		body         string
		cameraCheck  bool
		cameraExists bool
		cameraError  error
		wantCode     int
	}{
		{"InvalidJSON", `not-json`, false, false, nil, http.StatusBadRequest},
		{"MissingStart", `{"durationSeconds":30}`, false, false, nil, http.StatusBadRequest},
		{"InvalidStart", `{"start":"wrong","durationSeconds":30}`, false, false, nil, http.StatusBadRequest},
		{"ZeroDuration", `{"start":"` + recordingStart + `","durationSeconds":0}`, false, false, nil, http.StatusBadRequest},
		{"NegativeDuration", `{"start":"` + recordingStart + `","durationSeconds":-1}`, false, false, nil, http.StatusBadRequest},
		{"TooLongDuration", `{"start":"` + recordingStart + `","durationSeconds":86401}`, false, false, nil, http.StatusBadRequest},
		{"CameraNotFound", `{"start":"` + recordingStart + `","durationSeconds":30}`, true, false, nil, http.StatusNotFound},
		{"CameraLookupError", `{"start":"` + recordingStart + `","durationSeconds":30}`, true, false, errors.New("camera lookup failed"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			database, mock := newRecordingMock(t)
			expectRecordingUser(mock)
			expectRecordingAccess(mock, true, nil)
			if tc.cameraCheck {
				expectRecordingCamera(mock, tc.cameraExists, tc.cameraError)
			}
			controller := NewRecordingController(database, nil, nil)
			req := recordingRequest(http.MethodPost, "/api/trucks/truck001/cameras/cam01/play", tc.body)
			rec := recordingAuthorizedRequest(database, controller.Play, req)
			expectRecordingStatus(t, rec, tc.wantCode)
		})
	}
}

func TestRecordingController_Play_BodyTooLarge(t *testing.T) {
	database, mock := newRecordingMock(t)
	expectRecordingUser(mock)
	expectRecordingAccess(mock, true, nil)
	controller := NewRecordingController(database, nil, nil)
	body := strings.Repeat("x", 65*1024)
	rec := recordingAuthorizedRequest(database, controller.Play,
		recordingRequest(http.MethodPost, "/api/trucks/truck001/cameras/cam01/play", body))
	expectRecordingStatus(t, rec, http.StatusBadRequest)
}

func TestRecordingController_Play_Success(t *testing.T) {
	database, mock := newRecordingMock(t)
	expectRecordingUser(mock)
	expectRecordingAccess(mock, true, nil)
	expectRecordingCamera(mock, true, nil)
	recordings, streams := recordingServices("http://127.0.0.1:1")
	controller := NewRecordingController(database, recordings, streams)
	body := `{"start":"` + recordingStart + `","durationSeconds":30.5}`
	before := time.Now()
	rec := recordingAuthorizedRequest(database, controller.Play,
		recordingRequest(http.MethodPost, "/api/trucks/truck001/cameras/cam01/play", body))
	after := time.Now()
	expectRecordingStatus(t, rec, http.StatusOK)

	var result struct {
		URL         string    `json:"url"`
		AccessToken string    `json:"accessToken"`
		ExpiresAt   time.Time `json:"expiresAt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode play response: %v", err)
	}
	if result.URL == "" || result.AccessToken == "" {
		t.Fatalf("play URL or access token is missing: %+v", result)
	}
	parsed, err := url.Parse(result.URL)
	if err != nil {
		t.Fatalf("parse play URL: %v", err)
	}
	wantPath := "/api/trucks/truck001/cameras/cam01/recordings/content"
	if parsed.Scheme != "https" || parsed.Host != "api.example" || parsed.Path != wantPath {
		t.Errorf("play URL = %q, want HTTPS api.example%s", result.URL, wantPath)
	}
	if parsed.Query().Get("token") != result.AccessToken ||
		parsed.Query().Get("start") != recordingStart || parsed.Query().Get("duration") != "30.5" {
		t.Errorf("unexpected play URL query parameters: %s", result.URL)
	}
	claims, err := streams.ValidateRecordingAccess(result.AccessToken, before)
	if err != nil {
		t.Fatalf("validate playback token: %v", err)
	}
	if claims.TruckID != recordingTruckID || claims.CameraID != recordingCameraID ||
		!claims.Start.Equal(recordingStartTime) || claims.Duration != 30.5 {
		t.Errorf("unexpected playback token claims: %+v", claims)
	}
	minimumExpiry := services.RecordingAccessExpiry(before, 30.5)
	maximumExpiry := services.RecordingAccessExpiry(after, 30.5)
	if result.ExpiresAt.Before(minimumExpiry) || result.ExpiresAt.After(maximumExpiry) {
		t.Errorf("expiresAt = %v, want between %v and %v", result.ExpiresAt, minimumExpiry, maximumExpiry)
	}
}

func recordingContentRequest(token, start, duration string) *http.Request {
	query := url.Values{}
	query.Set("token", token)
	query.Set("start", start)
	query.Set("duration", duration)
	return recordingRequest(http.MethodGet,
		"/api/trucks/truck001/cameras/cam01/recordings/content?"+query.Encode(), "")
}

func TestRecordingController_Content_Unauthorized(t *testing.T) {
	streams := recordingStreamService()
	wrongTruck := streams.SignRecordingAccess("other-truck", recordingCameraID, recordingStartTime, 30, time.Now().Add(time.Hour))
	wrongCamera := streams.SignRecordingAccess(recordingTruckID, "other-camera", recordingStartTime, 30, time.Now().Add(time.Hour))
	wrongWindow := streams.SignRecordingAccess(recordingTruckID, recordingCameraID, recordingStartTime.Add(-time.Hour), 30, time.Now().Add(time.Hour))
	expired := streams.SignRecordingAccess(recordingTruckID, recordingCameraID, recordingStartTime, 30, time.Now().Add(-time.Minute))
	cases := []struct {
		name  string
		token string
	}{
		{"MissingToken", ""},
		{"InvalidSignature", "invalid-token"},
		{"WrongTruck", wrongTruck},
		{"WrongCamera", wrongCamera},
		{"WrongRecordingWindow", wrongWindow},
		{"ExpiredToken", expired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			controller := NewRecordingController(nil, nil, streams)
			rec := httptest.NewRecorder()
			controller.Content(rec, recordingContentRequest(tc.token, recordingStart, "30"))
			expectRecordingStatus(t, rec, http.StatusUnauthorized)
		})
	}
}

func TestRecordingController_Content_Validation(t *testing.T) {
	streams := recordingStreamService()
	token := streams.SignRecordingAccess(recordingTruckID, recordingCameraID, recordingStartTime, 30, time.Now().Add(time.Hour))
	cases := []struct {
		name     string
		start    string
		duration string
	}{
		{"MissingStart", "", "30"},
		{"InvalidStart", "wrong", "30"},
		{"MissingDuration", recordingStart, ""},
		{"InvalidDuration", recordingStart, "abc"},
		{"ZeroDuration", recordingStart, "0"},
		{"NegativeDuration", recordingStart, "-10"},
		{"TooLongDuration", recordingStart, "86401"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			controller := NewRecordingController(nil, nil, streams)
			rec := httptest.NewRecorder()
			controller.Content(rec, recordingContentRequest(token, tc.start, tc.duration))
			expectRecordingStatus(t, rec, http.StatusBadRequest)
		})
	}
}

func TestRecordingController_Content_UpstreamErrors(t *testing.T) {
	cases := []struct {
		name    string
		makeURL func(t *testing.T) string
	}{
		{"ConnectionFailed", func(t *testing.T) string { return "http://127.0.0.1:1" }},
		{"MediaMTXRejected", func(t *testing.T) string {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
			t.Cleanup(server.Close)
			return server.URL
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recordings, streams := recordingServices(tc.makeURL(t))
			token := streams.SignRecordingAccess(recordingTruckID, recordingCameraID, recordingStartTime, 30, time.Now().Add(time.Hour))
			controller := NewRecordingController(nil, recordings, streams)
			rec := httptest.NewRecorder()
			controller.Content(rec, recordingContentRequest(token, recordingStart, "30"))
			expectRecordingStatus(t, rec, http.StatusBadGateway)
		})
	}
}

func TestRecordingController_Content_Success(t *testing.T) {
	cases := []struct {
		name      string
		upstream  int
		byteRange string
	}{
		{"FullFile200", http.StatusOK, ""},
		{"PartialFile206", http.StatusPartialContent, "bytes=0-3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var upstreamCalled bool
			// Exercise full and ranged responses from materialized recording content.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upstreamCalled = true
				if r.Method != http.MethodGet || r.URL.Path != "/get" || r.URL.Query().Get("path") != "truck001/cam01/main" {
					t.Errorf("unexpected MediaMTX /get request: %s", r.URL.String())
				}
				if r.URL.Query().Get("start") != recordingStart || r.URL.Query().Get("duration") != "30" || r.URL.Query().Get("format") != "mp4" {
					t.Errorf("unexpected MediaMTX recording parameters: %s", r.URL.RawQuery)
				}
				if r.Header.Get("Range") != "" {
					t.Errorf("upstream Range header = %q, want empty", r.Header.Get("Range"))
				}
				if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
					t.Error("MediaMTX /get request is missing a Bearer token")
				}
				w.Header().Set("Content-Type", "video/mp4")
				w.Header().Set("Accept-Ranges", "bytes")
				w.Header().Set("Content-Length", "4")
				if tc.upstream == http.StatusPartialContent {
					w.Header().Set("Content-Range", "bytes 0-3/100")
				}
				w.WriteHeader(tc.upstream)
				_, _ = w.Write([]byte("ABCD"))
			}))
			defer server.Close()
			recordings, streams := recordingServices(server.URL)
			token := streams.SignRecordingAccess(recordingTruckID, recordingCameraID, recordingStartTime, 30, time.Now().Add(time.Hour))
			controller := NewRecordingController(nil, recordings, streams)
			req := recordingContentRequest(token, recordingStart, "30")
			if tc.byteRange != "" {
				req.Header.Set("Range", tc.byteRange)
			}
			rec := httptest.NewRecorder()
			controller.Content(rec, req)
			expectRecordingStatus(t, rec, tc.upstream)
			if !upstreamCalled || rec.Body.String() != "ABCD" {
				t.Errorf("response body = %q, want %q", rec.Body.String(), "ABCD")
			}
			if rec.Header().Get("Content-Type") != "video/mp4" || rec.Header().Get("Accept-Ranges") != "bytes" ||
				rec.Header().Get("Content-Length") != "4" || rec.Header().Get("Cache-Control") != "private, no-store" {
				t.Errorf("recording response headers were not forwarded: %+v", rec.Header())
			}
			if tc.upstream == http.StatusPartialContent && rec.Header().Get("Content-Range") != "bytes 0-3/4" {
				t.Errorf("Content-Range = %q, want %q", rec.Header().Get("Content-Range"), "bytes 0-3/4")
			}
		})
	}
}

func TestRecordingController_Content_ErrorResponseJSON(t *testing.T) {
	controller := NewRecordingController(nil, nil, recordingStreamService())
	rec := httptest.NewRecorder()
	controller.Content(rec, recordingContentRequest("invalid", recordingStart, "30"))
	expectRecordingStatus(t, rec, http.StatusUnauthorized)
	if !json.Valid(rec.Body.Bytes()) {
		t.Errorf("error response is not valid JSON: %q", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
}
