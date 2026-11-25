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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	objectstorage "real-time-video-surveillance-system/storage"
)

type RecordingSpan struct {
	Start        time.Time `json:"start"`
	Duration     float64   `json:"durationSeconds"`
	Complete     bool      `json:"complete"`
	PlayReady    bool      `json:"playReady"`
	UploadStatus string    `json:"uploadStatus,omitempty"`
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
	cloudMu          sync.Mutex
	cloud            *objectstorage.Cloud
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
	if !s.SimulationEnabled() {
		return s.listMedia(ctx, truckID, cameraID, start, end)
	}
	return s.listSimulated(ctx, truckID, cameraID, start, end)
}

func (s *RecordingService) listMedia(ctx context.Context, truckID, cameraID, start, end string) ([]RecordingSpan, error) {
	token := s.streams.SignRecordingAccess(truckID, cameraID, time.Unix(0, 0), 0, time.Now().Add(5*time.Minute))
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
		complete := s.segmentComplete(truckID, cameraID, span.Start)
		status := ""
		playReady := true
		if s.SimulationEnabled() {
			status = "pending"
			if complete {
				status = s.simulatedUploadStatus(truckID, cameraID, span.Start)
			}
			complete = complete && status == "uploaded"
			playReady = complete
		}
		result = append(result, RecordingSpan{Start: span.Start, Duration: span.Duration, Complete: complete, PlayReady: playReady, UploadStatus: status})
	}
	return result, nil
}

// Uploaded recordings are indexed by durable state, independently of local files.
// MediaMTX is consulted only to include still-recording or pending segments.
func (s *RecordingService) listSimulated(ctx context.Context, truckID, cameraID, start, end string) ([]RecordingSpan, error) {
	valid := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	if !valid.MatchString(truckID) || !valid.MatchString(cameraID) {
		return nil, fmt.Errorf("invalid recording path")
	}
	var from, until time.Time
	var err error
	if start != "" {
		from, err = time.Parse(time.RFC3339, start)
		if err != nil {
			return nil, err
		}
	}
	if end != "" {
		until, err = time.Parse(time.RFC3339, end)
		if err != nil {
			return nil, err
		}
	}
	root := os.Getenv("UPLOAD_STATE_DIR")
	if root == "" || (os.Getenv("SIMULATED_BUCKET_DIR") == "" && os.Getenv("GCS_BUCKET") == "") {
		return nil, fmt.Errorf("simulated storage not configured")
	}
	dir := filepath.Join(root, truckID, cameraID, "main")
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	result := make(map[time.Time]RecordingSpan)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".mp4.json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".mp4.json")
		if len(name) != 26 || name[19] != '-' {
			continue
		}
		t, e := time.Parse("2006-01-02_15-04-05", name[:19])
		if e != nil {
			continue
		}
		us, e := strconv.Atoi(name[20:])
		if e != nil || us < 0 || us > 999999 {
			continue
		}
		t = t.Add(time.Duration(us) * time.Microsecond)
		b, e := os.ReadFile(filepath.Join(dir, entry.Name()))
		if e != nil {
			continue
		}
		var j struct {
			Status   string  `json:"status"`
			Duration float64 `json:"durationSeconds"`
		}
		if json.Unmarshal(b, &j) != nil || j.Status != "uploaded" || j.Duration <= 0 || j.Duration > 86400 || s.simulatedUploadStatus(truckID, cameraID, t) != "uploaded" {
			continue
		}
		if !from.IsZero() && !t.Add(time.Duration(j.Duration*float64(time.Second))).After(from) {
			continue
		}
		if !until.IsZero() && !t.Before(until) {
			continue
		}
		result[t] = RecordingSpan{Start: t, Duration: j.Duration, Complete: true, PlayReady: true, UploadStatus: "uploaded"}
	}
	// Pending entries can still be shown while a source is recording or uploading.
	media, mediaErr := s.listMedia(ctx, truckID, cameraID, start, end)
	if mediaErr == nil {
		for _, row := range media {
			if _, exists := result[row.Start.UTC()]; !exists {
				row.Complete = false
				row.PlayReady = false
				result[row.Start.UTC()] = row
			}
		}
	}
	if mediaErr != nil && len(result) == 0 {
		return nil, mediaErr
	}
	rows := make([]RecordingSpan, 0, len(result))
	for _, row := range result {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Start.Before(rows[j].Start) })
	return rows, nil
}

func (s *RecordingService) SimulationEnabled() bool {
	return os.Getenv("SIMULATED_STORAGE") == "true" || os.Getenv("GCS_BUCKET") != ""
}

func simulatedRelativePath(truckID, cameraID string, start time.Time) (string, bool) {
	valid := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	if !valid.MatchString(truckID) || !valid.MatchString(cameraID) {
		return "", false
	}
	t := start.UTC()
	name := fmt.Sprintf("%s-%06d.mp4", t.Format("2006-01-02_15-04-05"), t.Nanosecond()/1000)
	return filepath.Join(truckID, cameraID, "main", name), true
}

func (s *RecordingService) simulatedUploadStatus(truckID, cameraID string, start time.Time) string {
	rel, ok := simulatedRelativePath(truckID, cameraID, start)
	if !ok {
		return "pending"
	}
	root, bucket := os.Getenv("UPLOAD_STATE_DIR"), os.Getenv("SIMULATED_BUCKET_DIR")
	if root == "" || (bucket == "" && os.Getenv("GCS_BUCKET") == "") {
		return "pending"
	}
	b, err := os.ReadFile(filepath.Join(root, rel+".json"))
	if err != nil {
		return "pending"
	}
	var j struct {
		Status     string `json:"status"`
		Size       int64  `json:"size"`
		SHA256     string `json:"sha256"`
		Generation int64  `json:"generation"`
	}
	if json.Unmarshal(b, &j) != nil {
		return "pending"
	}
	switch j.Status {
	case "failed":
		return "failed"
	case "uploading":
		return "uploading"
	case "uploaded":
		if os.Getenv("GCS_BUCKET") != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cloud, err := s.cloudClient(ctx)
			if err != nil {
				return "pending"
			}
			attrs, err := cloud.Attrs(ctx, filepath.ToSlash(rel))
			if err == nil && attrs.Size == j.Size && j.Size > 0 && attrs.Generation == j.Generation && j.Generation > 0 && attrs.Metadata["sha256"] == j.SHA256 && len(j.SHA256) == 64 {
				return "uploaded"
			}
			return "pending"
		}
		info, err := os.Stat(filepath.Join(bucket, rel))
		if err == nil && info.Mode().IsRegular() && info.Size() > 0 && info.Size() == j.Size && len(j.SHA256) == 64 {
			return "uploaded"
		}
	}
	return "pending"
}

// A marker is written only after the recorder closes a file and validation succeeds.
func (s *RecordingService) segmentComplete(truckID, cameraID string, start time.Time) bool {
	if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(truckID) || !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(cameraID) {
		return false
	}
	root := os.Getenv("RECORDINGS_DIR")
	if root == "" {
		return false
	}
	t := start.UTC()
	name := fmt.Sprintf("%s-%06d.mp4", t.Format("2006-01-02_15-04-05"), t.Nanosecond()/1000)
	path := filepath.Join(root, truckID, cameraID, "main", name)
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return false
	}
	statusRoot := os.Getenv("RECORDING_STATUS_DIR")
	if statusRoot == "" {
		return false
	}
	marker, err := os.ReadFile(filepath.Join(statusRoot, truckID, cameraID, "main", name+".complete"))
	if err != nil {
		return false
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(marker)), 10, 64)
	return err == nil && size == info.Size()
}

func (s *RecordingService) DownloadReady(ctx context.Context, truckID, cameraID string, start time.Time, duration float64) (bool, error) {
	rows, err := s.List(ctx, truckID, cameraID, "", "")
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.Start.Equal(start) && row.Complete && duration <= row.Duration {
			return true, nil
		}
	}
	return false, nil
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
	if os.Getenv("GCS_BUCKET") != "" {
		return s.serveCloud(w, r, truckID, cameraID, start)
	}
	if s.SimulationEnabled() {
		rel, ok := simulatedRelativePath(truckID, cameraID, start)
		if !ok {
			return fmt.Errorf("invalid recording path")
		}
		file, err := os.Open(filepath.Join(os.Getenv("SIMULATED_BUCKET_DIR"), rel))
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
