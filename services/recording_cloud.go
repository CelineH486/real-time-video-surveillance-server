package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	objectstorage "real-time-video-surveillance-system/storage"
)

func (s *RecordingService) cloudClient(ctx context.Context) (*objectstorage.Cloud, error) {
	s.cloudMu.Lock()
	defer s.cloudMu.Unlock()
	if s.cloud != nil {
		return s.cloud, nil
	}
	c, err := objectstorage.NewCloud(ctx, os.Getenv("GCS_BUCKET"))
	if err != nil {
		return nil, err
	}
	s.cloud = c
	return c, nil
}

func (s *RecordingService) serveCloud(w http.ResponseWriter, r *http.Request, truckID, cameraID string, start time.Time) error {
	rel, ok := simulatedRelativePath(truckID, cameraID, start)
	if !ok {
		return fmt.Errorf("invalid recording path")
	}
	b, err := os.ReadFile(filepath.Join(os.Getenv("UPLOAD_STATE_DIR"), rel+".json"))
	if err != nil {
		return err
	}
	var state struct {
		Status     string `json:"status"`
		Generation int64  `json:"generation"`
		Size       int64  `json:"size"`
	}
	if json.Unmarshal(b, &state) != nil || state.Status != "uploaded" || state.Generation <= 0 || state.Size <= 0 {
		return fmt.Errorf("recording unavailable")
	}
	etag := fmt.Sprintf("\"gcs-%d\"", state.Generation)
	w.Header().Set("ETag", etag)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", "video/mp4")
	offset, length, partial := int64(0), state.Size, false
	if r.Method == http.MethodGet && (r.Header.Get("If-Range") == "" || r.Header.Get("If-Range") == etag) {
		var unsatisfiable bool
		offset, length, partial, unsatisfiable = cloudByteRange(r.Header.Get("Range"), state.Size)
		if unsatisfiable {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", state.Size))
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return nil
		}
	}
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
		return nil
	}
	cloud, err := s.cloudClient(r.Context())
	if err != nil {
		return err
	}
	readLength := length
	if !partial {
		readLength = -1
	}
	reader, err := cloud.OpenRange(r.Context(), filepath.ToSlash(rel), state.Generation, offset, readLength)
	if err != nil {
		return err
	}
	defer reader.Close()
	if reader.Attrs.Decompressed || (reader.Remain() >= 0 && reader.Remain() != length) {
		return fmt.Errorf("unexpected recording range")
	}
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	if partial {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, offset+length-1, state.Size))
	}
	if partial {
		w.WriteHeader(http.StatusPartialContent)
	}
	if _, err = io.CopyN(w, reader, length); err != nil {
		// Headers may already be sent; don't append a JSON error to video bytes.
		if r.Context().Err() == nil {
			log.Printf("recording stream interrupted: %v", err)
		}
	}
	return nil
}

// A single byte range is supported. Unsupported or malformed ranges are ignored;
// a valid range outside the object returns 416. Multiple ranges are served as 200.
func cloudByteRange(header string, size int64) (offset, length int64, partial, unsatisfiable bool) {
	length = size
	if !strings.HasPrefix(header, "bytes=") || strings.Contains(header, ",") {
		return
	}
	value := strings.TrimSpace(strings.TrimPrefix(header, "bytes="))
	first, last, ok := strings.Cut(value, "-")
	if !ok || (first == "" && last == "") {
		return
	}
	if first == "" {
		suffix, err := strconv.ParseInt(last, 10, 64)
		if err != nil || suffix < 0 {
			return
		}
		if suffix == 0 {
			unsatisfiable = true
			return
		}
		if suffix > size {
			suffix = size
		}
		return size - suffix, suffix, true, false
	}
	start, err := strconv.ParseInt(first, 10, 64)
	if err != nil || start < 0 {
		return
	}
	end := size - 1
	if last != "" {
		end, err = strconv.ParseInt(last, 10, 64)
		if err != nil || end < start {
			return 0, size, false, false
		}
		if end >= size {
			end = size - 1
		}
	}
	if start >= size {
		return 0, 0, false, true
	}
	return start, end - start + 1, true, false
}
