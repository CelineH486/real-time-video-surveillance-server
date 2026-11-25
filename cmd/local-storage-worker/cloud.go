package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gcs "cloud.google.com/go/storage"
)

func (w worker) processCloud(marker string) (err error) {
	rel, err := filepath.Rel(w.markers, marker)
	if err != nil || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || !strings.HasSuffix(rel, ".mp4.complete") {
		return fmt.Errorf("invalid marker path")
	}
	rel = strings.TrimSuffix(rel, ".complete")
	key := filepath.ToSlash(rel)
	parts := strings.Split(key, "/")
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] != "main" {
		return fmt.Errorf("recording is outside the main stream")
	}
	source := filepath.Join(w.recordings, rel)
	state := filepath.Join(w.states, rel+".json")
	var old job
	if b, e := os.ReadFile(state); e == nil {
		_ = json.Unmarshal(b, &old)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	attrs, lookupErr := w.cloud.Attrs(ctx, key)
	if lookupErr != nil && !errors.Is(lookupErr, gcs.ErrObjectNotExist) {
		return lookupErr
	}
	if lookupErr == nil {
		if old.Status == "uploaded" && attrs.Size == old.Size && attrs.Generation == old.Generation && attrs.Metadata["sha256"] == old.SHA256 && old.DurationSeconds > 0 {
			return w.cleanupCloudSource(ctx, source, marker, key, old)
		}
		info, e := os.Lstat(source)
		if e != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("cannot verify existing cloud object")
		}
		b, e := os.ReadFile(marker)
		if e != nil {
			return e
		}
		size, e := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
		if e != nil || size <= 0 || size != info.Size() || size != attrs.Size {
			return fmt.Errorf("existing cloud object size mismatch")
		}
		hash, e := hashFile(source)
		if e != nil || hash != attrs.Metadata["sha256"] {
			return fmt.Errorf("existing cloud object checksum metadata mismatch")
		}
		crc, e := checksumCRC32C(source)
		if e != nil || crc != attrs.CRC32C {
			return fmt.Errorf("existing cloud object checksum mismatch")
		}
		duration, e := w.duration(source)
		if e != nil {
			return e
		}
		if attrs.Generation <= 0 {
			return fmt.Errorf("invalid object generation")
		}
		recovered := job{Status: "uploaded", Size: size, SHA256: hash, DurationSeconds: duration, Generation: attrs.Generation}
		if err := writeState(state, recovered); err != nil {
			return err
		}
		return w.cleanupCloudSource(ctx, source, marker, key, recovered)
	}
	info, err := os.Lstat(source)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("valid source unavailable")
	}
	b, err := os.ReadFile(marker)
	if err != nil {
		return err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil || size <= 0 || info.Size() != size {
		return fmt.Errorf("marker does not match source")
	}
	duration, err := w.duration(source)
	if err != nil {
		return err
	}
	hash, err := hashFile(source)
	if err != nil {
		return err
	}
	j := job{Status: "uploading", Size: size}
	if err = writeState(state, j); err != nil {
		return err
	}
	defer func() {
		if err != nil && j.Status != "uploaded" {
			j.Status = "failed"
			j.Error = err.Error()
			_ = writeState(state, j)
		}
	}()
	attrs, err = w.cloud.Upload(ctx, key, source, map[string]string{"sha256": hash, "durationSeconds": strconv.FormatFloat(duration, 'f', -1, 64)})
	if err != nil {
		return err
	}
	if attrs == nil || attrs.Size != size || attrs.Generation <= 0 {
		return fmt.Errorf("invalid uploaded object metadata")
	}
	after, e := os.Stat(source)
	if e != nil || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
		return fmt.Errorf("source changed during upload")
	}
	crc, e := checksumCRC32C(source)
	if e != nil || attrs.CRC32C != crc {
		return fmt.Errorf("uploaded object checksum mismatch")
	}
	j = job{Status: "uploaded", Size: size, SHA256: hash, DurationSeconds: duration, Generation: attrs.Generation}
	if err = writeState(state, j); err != nil {
		return err
	}
	return w.cleanupCloudSource(ctx, source, marker, key, j)
}

// Cleanup is opt-in and only happens after an uploaded state is durably saved.
// The source, completion marker and exact cloud generation are rechecked before removal.
func (w worker) cleanupCloudSource(ctx context.Context, source, marker, key string, state job) error {
	if !w.cleanupUploaded {
		return nil
	}
	before, err := os.Lstat(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !before.Mode().IsRegular() || before.Size() != state.Size || state.Size <= 0 || state.Status != "uploaded" || state.Generation <= 0 || len(state.SHA256) != 64 {
		return fmt.Errorf("cleanup source or state invalid")
	}
	b, err := os.ReadFile(marker)
	if err != nil {
		return err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil || size != state.Size {
		return fmt.Errorf("cleanup completion marker changed")
	}
	hash, err := hashFile(source)
	if err != nil || hash != state.SHA256 {
		return fmt.Errorf("cleanup source hash mismatch")
	}
	crc, err := checksumCRC32C(source)
	if err != nil {
		return err
	}
	attrs, err := w.cloud.Attrs(ctx, key)
	if err != nil {
		return err
	}
	if attrs == nil || attrs.Generation != state.Generation || attrs.Size != state.Size || attrs.CRC32C != crc || attrs.Metadata["sha256"] != state.SHA256 {
		return fmt.Errorf("cleanup cloud object mismatch")
	}
	after, err := os.Lstat(source)
	if err != nil || !os.SameFile(before, after) || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return fmt.Errorf("cleanup source changed")
	}
	markerNow, err := os.ReadFile(marker)
	if err != nil || strings.TrimSpace(string(markerNow)) != strconv.FormatInt(state.Size, 10) {
		return fmt.Errorf("cleanup marker changed")
	}
	rel, err := filepath.Rel(w.recordings, source)
	if err != nil || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("cleanup path outside recording root")
	}
	root, err := os.OpenRoot(w.recordings)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.Remove(rel)
}

func checksumCRC32C(path string) (uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	h := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	if _, err := io.Copy(h, f); err != nil {
		return 0, err
	}
	return h.Sum32(), nil
}
