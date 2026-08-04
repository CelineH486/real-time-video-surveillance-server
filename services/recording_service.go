package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type RecordingSpan struct {
	Start    time.Time `json:"start"`
	Duration float64   `json:"durationSeconds"`
	URL      string    `json:"url"`
	Expires  time.Time `json:"expiresAt"`
}

type mediaMTXRecordingSpan struct {
	Start    time.Time `json:"start"`
	Duration float64   `json:"duration"`
}

type RecordingService struct {
	internalBaseURL  string
	apiPublicBaseURL string
	streams          *StreamService
	client           *http.Client
}

func NewRecordingService(internalBaseURL, apiPublicBaseURL string, streams *StreamService) *RecordingService {
	return &RecordingService{
		internalBaseURL:  strings.TrimRight(internalBaseURL, "/"),
		apiPublicBaseURL: strings.TrimRight(apiPublicBaseURL, "/"),
		streams:          streams,
		client:           &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *RecordingService) List(ctx context.Context, truckID, cameraID, start, end string) ([]RecordingSpan, error) {
	expires := time.Now().Add(5 * time.Minute)
	token := s.streams.SignAccess(truckID, cameraID, "main", expires)
	query := url.Values{"path": {strings.Join([]string{truckID, cameraID, "main"}, "/")}}
	for name, value := range map[string]string{"start": start, "end": end} {
		if value == "" {
			continue
		}
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return nil, fmt.Errorf("%s must use RFC3339 format", name)
		}
		query.Set(name, value)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.internalBaseURL+"/list?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("recording service returned %d", response.StatusCode)
	}
	var source []mediaMTXRecordingSpan
	if err := json.NewDecoder(response.Body).Decode(&source); err != nil {
		return nil, err
	}
	result := make([]RecordingSpan, 0, len(source))
	for _, span := range source {
		result = append(result, RecordingSpan{
			Start: span.Start, Duration: span.Duration,
			URL: s.PublicURL(truckID, cameraID, span.Start, span.Duration, token), Expires: expires,
		})
	}
	return result, nil
}

func (s *RecordingService) PublicURL(truckID, cameraID string, start time.Time, duration float64, token string) string {
	query := url.Values{}
	query.Set("start", start.Format(time.RFC3339Nano))
	query.Set("duration", strconv.FormatFloat(duration, 'f', -1, 64))
	query.Set("token", token)
	return s.apiPublicBaseURL + "/api/trucks/" + url.PathEscape(truckID) + "/cameras/" + url.PathEscape(cameraID) +
		"/recordings/content?" + query.Encode()
}

func (s *RecordingService) Open(ctx context.Context, truckID, cameraID string, start time.Time, duration float64, token string) (*http.Response, error) {
	query := mediaMTXRecordingQuery(truckID, cameraID, start, duration)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.internalBaseURL+"/get?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	return s.client.Do(request)
}

func mediaMTXRecordingQuery(truckID, cameraID string, start time.Time, duration float64) url.Values {
	query := url.Values{}
	query.Set("path", strings.Join([]string{truckID, cameraID, "main"}, "/"))
	query.Set("start", start.Format(time.RFC3339Nano))
	query.Set("duration", strconv.FormatFloat(duration, 'f', -1, 64))
	query.Set("format", "mp4")
	return query
}
