package apiresponse

import (
	"encoding/json"
	"log"
	"net/http"
)

type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	CodeMissingBearerToken          = "missing_bearer_token"
	CodeAuthenticationUnavailable   = "authentication_unavailable"
	CodeInvalidBearerToken          = "invalid_bearer_token"
	CodeAuthenticationRequired      = "authentication_required"
	CodeAuthorizationUnavailable    = "authorization_unavailable"
	CodeTruckAccessDenied           = "truck_access_denied"
	CodeTrucksUnavailable           = "trucks_unavailable"
	CodeCamerasUnavailable          = "cameras_unavailable"
	CodeCameraLookupFailed          = "camera_lookup_failed"
	CodeCameraNotFound              = "camera_not_found"
	CodeCameraIDRequired            = "camera_id_required"
	CodeInvalidTimeFormat           = "invalid_time_format"
	CodeRecordingServiceUnavailable = "recording_service_unavailable"
	CodeInvalidRecordingPlayRequest = "invalid_recording_play_request"
	CodeInvalidRecordingDuration    = "invalid_recording_duration"
	CodeRecordingAccessDenied       = "recording_access_denied"
	CodeRecordingServiceRejected    = "recording_service_rejected_request"
	CodeInvalidAuthentication       = "invalid_authentication_request"
	CodeInvalidStreamPath           = "invalid_stream_path"
	CodePublishDenied               = "publish_denied"
	CodeDatabaseUnavailable         = "database_unavailable"
	CodeStreamAccessDenied          = "stream_access_denied"
	CodeActionDenied                = "action_denied"
	CodeInvalidLogin                = "invalid_login"
	CodeInvalidPasswordFormat       = "invalid_password_format"
	CodeLogoutUnavailable           = "logout_unavailable"
)

const (
	MessageMissingBearerToken          = "Authorization header must be provided as Bearer token"
	MessageAuthenticationUnavailable   = "The authentication store could not be reached"
	MessageInvalidBearerToken          = "The supplied API token is invalid or expired"
	MessageAuthenticationRequired      = "This endpoint requires an authenticated API user"
	MessageAuthorizationUnavailable    = "Truck permissions could not be verified"
	MessageTruckAccessDenied           = "The authenticated user is not assigned to this truck"
	MessageTrucksUnavailable           = "Unable to load trucks for the authenticated user"
	MessageCamerasUnavailableForUser   = "Unable to load cameras for the authenticated user"
	MessageCamerasUnavailableForTruck  = "Unable to load cameras for the requested truck"
	MessageCameraLookupFailed          = "Unable to verify the requested camera"
	MessageCameraNotFound              = "No camera exists for the requested truck and camera ID"
	MessageCameraIDRequired            = "Query parameter cameraId is required"
	MessageQueryTimeFormat             = "Query parameter %s must use RFC3339 format"
	MessageQueryDurationRange          = "Query parameter duration must be greater than 0 and no more than 86400"
	MessageRecordingListUnavailable    = "Unable to fetch recording spans from the playback service"
	MessageInvalidRecordingPlayRequest = "Request body must include start and durationSeconds"
	MessageInvalidRecordingDuration    = "durationSeconds must be greater than 0 and no more than 86400"
	MessageRecordingAccessDenied       = "The recording playback token is invalid for this truck and camera"
	MessageRecordingOpenUnavailable    = "Unable to open the requested recording from the playback service"
	MessageRecordingServiceRejected    = "The playback service rejected the recording request"
	MessageInvalidMediaMTXAuthRequest  = "MediaMTX auth request body must be valid JSON"
	MessageInvalidStreamPath           = "Stream path must match truck/camera quality"
	MessagePublishDenied               = "Publisher credentials do not match the stream truck"
	MessageDatabaseUnavailable         = "Unable to verify stream path against registered cameras"
	MessageMediaMTXCameraNotFound      = "No registered camera matches the requested stream path"
	MessageStreamAccessDenied          = "Stream token is expired or does not match the requested path"
	MessageActionDenied                = "MediaMTX action is not allowed by this API"
	MessageInvalidLogin                = "Email or password is incorrect"
	MessageInvalidPasswordFormat       = "Password must be 8-72 characters and include uppercase, lowercase, and a number"
	MessageLogoutUnavailable           = "Unable to revoke the current session"
)

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func WriteError(w http.ResponseWriter, status int, code string, message string) {
	WriteJSON(w, status, ErrorBody{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
