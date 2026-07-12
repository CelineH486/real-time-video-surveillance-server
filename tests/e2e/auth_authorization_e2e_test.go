//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type truckResponse struct {
	TruckID string `json:"truckId"`
}

type cameraResponse struct {
	CameraID string `json:"cameraId"`
	SubToken string `json:"subToken"`
}

type streamResponse struct {
	TruckID     string `json:"truckId"`
	CameraID    string `json:"cameraId"`
	Quality     string `json:"quality"`
	URL         string `json:"url"`
	AccessToken string `json:"accessToken"`
}

const missingBearerTokenException = `{
	"error": {
		"code": "missing_bearer_token",
		"message": "Authorization header must be provided as Bearer token"
	}
}`

const truckAccessDeniedException = `{
	"error": {
		"code": "truck_access_denied",
		"message": "The authenticated user is not assigned to this truck"
	}
}`

func TestAuthorizationRequiresBearerToken(t *testing.T) {
	// Arrange
	client := testClient()
	baseURL := requiredE2EEnv(t, "E2E_API_BASE_URL")

	// Act
	status, body := request(t, client, http.MethodGet, baseURL+"/api/trucks", "", nil)

	// Assert
	if status != http.StatusUnauthorized {
		t.Fatalf("expected missing bearer token to return 401, got %d", status)
	}
	exception := missingBearerTokenException
	assertException(t, body, exception)
}

func TestAuthorizedUserCanOnlyUseAssignedTruck(t *testing.T) {
	// Arrange
	client := testClient()
	baseURL := requiredE2EEnv(t, "E2E_API_BASE_URL")
	apiToken := requiredE2EEnv(t, "E2E_API_TOKEN")
	truckID := e2eEnv("E2E_TRUCK_ID", "truck001")
	unassignedTruckID := e2eEnv("E2E_UNASSIGNED_TRUCK_ID", "__not_assigned__")

	// Act
	status, body := request(t, client, http.MethodGet, baseURL+"/api/trucks", apiToken, nil)
	forbiddenStatus, forbiddenBody := request(t, client, http.MethodGet, baseURL+"/api/trucks/"+unassignedTruckID+"/cameras", apiToken, nil)

	// Assert
	if status != http.StatusOK {
		t.Fatalf("expected authorized truck list to return 200, got %d: %s", status, body)
	}
	var trucks []truckResponse
	decodeJSON(t, body, &trucks)
	if !containsTruck(trucks, truckID) {
		t.Fatalf("expected authorized truck list to include %q, got %+v", truckID, trucks)
	}
	if forbiddenStatus != http.StatusForbidden {
		t.Fatalf("expected unassigned truck access to return 403, got %d", forbiddenStatus)
	}
	exception := truckAccessDeniedException
	assertException(t, forbiddenBody, exception)
}

func TestAuthorizedUserCanIssueStreamTokenForAssignedTruck(t *testing.T) {
	// Arrange
	client := testClient()
	baseURL := requiredE2EEnv(t, "E2E_API_BASE_URL")
	apiToken := requiredE2EEnv(t, "E2E_API_TOKEN")
	truckID := e2eEnv("E2E_TRUCK_ID", "truck001")
	cameraID := e2eEnv("E2E_CAMERA_ID", "cam01")

	// Act
	cameraStatus, cameraBody := request(t, client, http.MethodGet, baseURL+"/api/trucks/"+truckID+"/cameras", apiToken, nil)
	playStatus, playBody := request(t, client, http.MethodPost, baseURL+"/api/trucks/"+truckID+"/cameras/"+cameraID+"/play", apiToken, nil)

	// Assert
	if cameraStatus != http.StatusOK {
		t.Fatalf("expected camera list to return 200, got %d: %s", cameraStatus, cameraBody)
	}
	var cameras []cameraResponse
	decodeJSON(t, cameraBody, &cameras)
	camera, ok := findCamera(cameras, cameraID)
	if !ok {
		t.Fatalf("expected camera list to include %q, got %+v", cameraID, cameras)
	}
	if camera.SubToken == "" {
		t.Fatalf("expected camera %q to include a sub stream token", cameraID)
	}

	if playStatus != http.StatusOK {
		t.Fatalf("expected play endpoint to return 200, got %d: %s", playStatus, playBody)
	}
	var stream streamResponse
	decodeJSON(t, playBody, &stream)
	if stream.TruckID != truckID || stream.CameraID != cameraID || stream.Quality != "main" {
		t.Fatalf("unexpected stream response: %+v", stream)
	}
	if stream.URL == "" || stream.AccessToken == "" {
		t.Fatalf("expected stream response to include URL and access token: %+v", stream)
	}
}

func TestIssuedStreamTokenAuthorizesMediaMTXRead(t *testing.T) {
	// Arrange
	client := testClient()
	baseURL := requiredE2EEnv(t, "E2E_API_BASE_URL")
	apiToken := requiredE2EEnv(t, "E2E_API_TOKEN")
	truckID := e2eEnv("E2E_TRUCK_ID", "truck001")
	cameraID := e2eEnv("E2E_CAMERA_ID", "cam01")
	_, playBody := request(t, client, http.MethodPost, baseURL+"/api/trucks/"+truckID+"/cameras/"+cameraID+"/play", apiToken, nil)
	var stream streamResponse
	decodeJSON(t, playBody, &stream)

	// Act
	status, body := request(t, client, http.MethodPost, baseURL+"/internal/mediamtx/auth", "", map[string]string{
		"action": "read",
		"path":   truckID + "/" + cameraID + "/main",
		"token":  stream.AccessToken,
	})

	// Assert
	if status != http.StatusNoContent {
		t.Fatalf("expected MediaMTX read auth to return 204, got %d: %s", status, body)
	}
}

func testClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

func request(t *testing.T, client *http.Client, method, url, apiToken string, payload any) (int, string) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("call %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return resp.StatusCode, string(raw)
}

func decodeJSON(t *testing.T, body string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), target); err != nil {
		t.Fatalf("decode JSON response %q: %v", body, err)
	}
}

func assertException(t *testing.T, body string, exceptionJSON string) {
	t.Helper()

	actual := normalizeJSON(t, body)
	expected := normalizeJSON(t, exceptionJSON)
	if actual != expected {
		t.Fatalf("expected exception JSON %s, got %s", expected, actual)
	}
}

func normalizeJSON(t *testing.T, value string) string {
	t.Helper()
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		t.Fatalf("decode JSON %q: %v", value, err)
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("normalize JSON %q: %v", value, err)
	}
	return string(normalized)
}

func containsTruck(trucks []truckResponse, truckID string) bool {
	for _, truck := range trucks {
		if truck.TruckID == truckID {
			return true
		}
	}
	return false
}

func findCamera(cameras []cameraResponse, cameraID string) (cameraResponse, bool) {
	for _, camera := range cameras {
		if camera.CameraID == cameraID {
			return camera, true
		}
	}
	return cameraResponse{}, false
}

func e2eEnv(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return strings.TrimRight(value, "/")
	}
	return strings.TrimRight(fallback, "/")
}

func requiredE2EEnv(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Skipf("%s must be set to run authorization E2E tests", name)
	}
	return strings.TrimRight(value, "/")
}
