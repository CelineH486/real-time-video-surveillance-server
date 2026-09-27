package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RecordingSpan struct {
	Start    time.Time `json:"start"`
	Duration float64   `json:"durationSeconds"`
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
	cacheDirectory   string
	cacheLifetime    time.Duration
	cacheMu          sync.Mutex
	cache            map[string]cachedRecording
	inflight         map[string]*recordingDownload
}

type cachedRecording struct {
	path      string
	expiresAt time.Time
}

type recordingDownload struct {
	done chan struct{}
	path string
	err  error
}

func NewRecordingService(internalBaseURL, apiPublicBaseURL string, streams *StreamService) *RecordingService {
	return &RecordingService{
		internalBaseURL:  strings.TrimRight(internalBaseURL, "/"),
		apiPublicBaseURL: strings.TrimRight(apiPublicBaseURL, "/"),
		streams:          streams,
		client:           &http.Client{Timeout: 5 * time.Minute},
		cacheDirectory:   filepath.Join(os.TempDir(), "recording-playback-cache"),
		cacheLifetime:    10 * time.Minute,
		cache:            make(map[string]cachedRecording),
		inflight:         make(map[string]*recordingDownload),
	}
}

func (s *RecordingService) List(ctx context.Context, truckID, cameraID, start, end string) ([]RecordingSpan, error) {
	token := s.streams.SignAccess(truckID, cameraID, "main", time.Now().Add(5*time.Minute))
	query := url.Values{"path": {strings.Join([]string{truckID, cameraID, "main"}, "/")}}
	// Our pinned MediaMTX build exposes physical segments instead of merging
	// adjacent files. Keep the recorder's true start time after every reconnect.
	query.Set("segments", "true")
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

// Open returns a complete, seekable MP4 file. MediaMTX creates playback
// responses on demand and therefore cannot honor HTTP byte ranges itself.
// Materializing the response allows http.ServeContent to provide the 206
// responses required by Safari and avoids regenerating a clip for retries.
func (s *RecordingService) Open(ctx context.Context, truckID, cameraID string, start time.Time, duration float64, token string) (*os.File, error) {
	key := recordingCacheKey(truckID, cameraID, start, duration)
	download := s.downloadFor(key, truckID, cameraID, start, duration, token)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-download.done:
		if download.err != nil {
			return nil, download.err
		}
		return os.Open(download.path)
	}
}

// Serve writes a materialized recording with standard HTTP range support.
func (s *RecordingService) Serve(w http.ResponseWriter, r *http.Request, truckID, cameraID string, start time.Time, duration float64, token string) error {
	file, err := s.Open(r.Context(), truckID, cameraID, start, duration, token)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "video/mp4")
	http.ServeContent(w, r, "recording.mp4", info.ModTime(), file)
	return nil
}

func (s *RecordingService) downloadFor(key, truckID, cameraID string, start time.Time, duration float64, token string) *recordingDownload {
	now := time.Now()
	s.cacheMu.Lock()
	if cached, ok := s.cache[key]; ok {
		if now.Before(cached.expiresAt) {
			s.cacheMu.Unlock()
			return completedDownload(cached.path, nil)
		}
		delete(s.cache, key)
		_ = os.Remove(cached.path)
	}
	if download, ok := s.inflight[key]; ok {
		s.cacheMu.Unlock()
		return download
	}
	download := &recordingDownload{done: make(chan struct{})}
	s.inflight[key] = download
	s.cacheMu.Unlock()

	go func() {
		download.path, download.err = s.downloadRecording(truckID, cameraID, start, duration, token)
		s.cacheMu.Lock()
		delete(s.inflight, key)
		if download.err == nil {
			expiresAt := time.Now().Add(s.cacheLifetime)
			s.cache[key] = cachedRecording{path: download.path, expiresAt: expiresAt}
			time.AfterFunc(s.cacheLifetime, func() { s.expireCacheEntry(key, download.path) })
		}
		close(download.done)
		s.cacheMu.Unlock()
	}()
	return download
}

func completedDownload(path string, err error) *recordingDownload {
	download := &recordingDownload{done: make(chan struct{}), path: path, err: err}
	close(download.done)
	return download
}

func (s *RecordingService) downloadRecording(truckID, cameraID string, start time.Time, duration float64, token string) (string, error) {
	if err := os.MkdirAll(s.cacheDirectory, 0o700); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	query := mediaMTXRecordingQuery(truckID, cameraID, start, duration)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.internalBaseURL+"/get?"+query.Encode(), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := s.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return "", fmt.Errorf("recording service returned %d", response.StatusCode)
	}
	file, err := os.CreateTemp(s.cacheDirectory, "recording-*.mp4")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if _, err = io.Copy(file, response.Body); err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func (s *RecordingService) expireCacheEntry(key, path string) {
	s.cacheMu.Lock()
	if cached, ok := s.cache[key]; ok && cached.path == path {
		delete(s.cache, key)
		_ = os.Remove(path)
	}
	s.cacheMu.Unlock()
}

func recordingCacheKey(truckID, cameraID string, start time.Time, duration float64) string {
	value := strings.Join([]string{
		truckID,
		cameraID,
		start.UTC().Format(time.RFC3339Nano),
		strconv.FormatFloat(duration, 'f', -1, 64),
	}, "|")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

func mediaMTXRecordingQuery(truckID, cameraID string, start time.Time, duration float64) url.Values {
	query := url.Values{}
	query.Set("path", strings.Join([]string{truckID, cameraID, "main"}, "/"))
	query.Set("start", start.Format(time.RFC3339Nano))
	query.Set("duration", strconv.FormatFloat(duration, 'f', -1, 64))
	query.Set("format", "mp4")
	return query
}
