// local-storage-worker simulates object storage using a local directory.
// It consumes only MediaMTX segment validation markers and retains every source.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gcs "cloud.google.com/go/storage"
	objectstorage "real-time-video-surveillance-system/storage"
)

type job struct {
	Status          string  `json:"status"`
	Size            int64   `json:"size"`
	SHA256          string  `json:"sha256,omitempty"`
	Error           string  `json:"error,omitempty"`
	DurationSeconds float64 `json:"durationSeconds,omitempty"`
	Generation      int64   `json:"generation,omitempty"`
}

type cloudStore interface {
	Upload(context.Context, string, string, map[string]string) (*gcs.ObjectAttrs, error)
	Attrs(context.Context, string) (*gcs.ObjectAttrs, error)
}

type worker struct {
	recordings, markers, bucket, states string
	verified                            map[string]signature
	probe                               func(string) (float64, error)
	cloud                               cloudStore
	cleanupUploaded                     bool
}
type signature struct {
	size                         int64
	sourceModified, destModified time.Time
}

func writeState(path string, j job) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := json.Marshal(j)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".state-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	// The API runs under a different unprivileged user and mounts this directory
	// read-only, so state files must be readable across the two containers.
	if err = f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (w worker) duration(path string) (float64, error) {
	if w.probe != nil {
		return w.probe(path)
	}
	b, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path).Output()
	if err != nil {
		return 0, err
	}
	d, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
	if err != nil || d <= 0 || d > 86400 {
		return 0, fmt.Errorf("invalid recording duration")
	}
	return d, nil
}

func (w worker) process(marker string) (err error) {
	if w.cloud != nil {
		return w.processCloud(marker)
	}
	rel, err := filepath.Rel(w.markers, marker)
	if err != nil || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || !strings.HasSuffix(rel, ".mp4.complete") {
		return fmt.Errorf("invalid marker path")
	}
	rel = strings.TrimSuffix(rel, ".complete")
	source := filepath.Join(w.recordings, rel)
	dest := filepath.Join(w.bucket, rel)
	state := filepath.Join(w.states, rel+".json")
	if err = os.MkdirAll(filepath.Dir(state), 0755); err != nil {
		return err
	}
	previous, _ := os.ReadFile(state)
	if len(previous) > 0 {
		if e := os.Chmod(state, 0644); e != nil {
			return e
		}
	}
	var old job
	_ = json.Unmarshal(previous, &old)
	st, err := os.Lstat(source)
	if os.IsNotExist(err) && old.Status == "uploaded" && old.Size > 0 && old.SHA256 != "" {
		ds, e := os.Stat(dest)
		if e != nil || !ds.Mode().IsRegular() || ds.Size() != old.Size {
			return fmt.Errorf("uploaded copy missing or changed")
		}
		dh, e := hashFile(dest)
		if e != nil || dh != old.SHA256 {
			return fmt.Errorf("uploaded copy checksum mismatch")
		}
		if old.DurationSeconds <= 0 {
			old.DurationSeconds, e = w.duration(dest)
			if e != nil {
				return e
			}
			return writeState(state, old)
		}
		return nil
	}
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file")
	}
	b, err := os.ReadFile(marker)
	if err != nil {
		return err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil || size <= 0 || st.Size() != size {
		return fmt.Errorf("marker does not match source")
	}
	if cached, ok := w.verified[rel]; ok {
		if ds, e := os.Stat(dest); e == nil && cached.size == st.Size() && ds.Size() == cached.size && cached.sourceModified.Equal(st.ModTime()) && cached.destModified.Equal(ds.ModTime()) {
			return nil
		}
	}
	if old.Status == "uploaded" && old.Size == size && old.SHA256 != "" {
		// Verify both sides before trusting saved state after a restart.
		sh, se := hashFile(source)
		dh, de := hashFile(dest)
		if se == nil && de == nil && sh == old.SHA256 && dh == sh {
			if old.DurationSeconds <= 0 {
				old.DurationSeconds, err = w.duration(dest)
				if err != nil {
					return err
				}
				if err = writeState(state, old); err != nil {
					return err
				}
			}
			if ds, e := os.Stat(dest); e == nil {
				w.verified[rel] = signature{size, st.ModTime(), ds.ModTime()}
			}
			return nil
		}
	}
	j := job{Status: "uploading", Size: size}
	if err = writeState(state, j); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			j.Status = "failed"
			j.Error = err.Error()
			if e := writeState(state, j); e != nil {
				log.Printf("cannot persist failure: %v", e)
			}
		}
	}()
	if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dest + ".simulated-upload"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(out, h), in)
	if err == nil {
		err = out.Sync()
	}
	if e := out.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	checksum := hex.EncodeToString(h.Sum(nil))
	newStat, err := os.Stat(source)
	if err != nil {
		return err
	}
	if st.Size() != newStat.Size() || !st.ModTime().Equal(newStat.ModTime()) {
		return fmt.Errorf("source changed during upload")
	}
	copyHash, err := hashFile(tmp)
	if err != nil {
		return err
	}
	if copyHash != checksum {
		return fmt.Errorf("copy checksum mismatch")
	}
	if err = os.Rename(tmp, dest); err != nil {
		return err
	}
	duration, err := w.duration(dest)
	if err != nil {
		return err
	}
	j = job{Status: "uploaded", Size: size, SHA256: checksum, DurationSeconds: duration}
	if err = writeState(state, j); err != nil {
		return err
	}
	if ds, e := os.Stat(dest); e == nil {
		w.verified[rel] = signature{size, newStat.ModTime(), ds.ModTime()}
	}
	return nil
}

func (w worker) scan() error {
	return filepath.WalkDir(w.markers, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !d.Type().IsRegular() || !strings.HasSuffix(path, ".mp4.complete") {
			return nil
		}
		if err := w.process(path); err != nil {
			log.Printf("upload %s: %v", path, err)
		}
		return nil
	})
}

func main() {
	w := worker{recordings: env("RECORDINGS_DIR", "/recordings"), markers: env("RECORDING_STATUS_DIR", "/recording-status"), bucket: env("SIMULATED_BUCKET_DIR", "/simulated-bucket"), states: env("UPLOAD_STATE_DIR", "/upload-state"), verified: make(map[string]signature)}
	w.cleanupUploaded = os.Getenv("CLEANUP_UPLOADED_RECORDINGS") == "true"
	if bucket := os.Getenv("GCS_BUCKET"); bucket != "" {
		c, err := objectstorage.NewCloud(context.Background(), bucket)
		if err != nil {
			log.Fatalf("cloud storage: %v", err)
		}
		defer c.Close()
		w.cloud = c
	}
	for {
		if err := w.scan(); err != nil {
			log.Printf("scan: %v", err)
		}
		time.Sleep(10 * time.Second)
	}
}
func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
