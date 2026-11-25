package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setup(t *testing.T) (worker, string, string, string) {
	t.Helper()
	root := t.TempDir()
	w := worker{recordings: filepath.Join(root, "source"), markers: filepath.Join(root, "markers"), bucket: filepath.Join(root, "bucket"), states: filepath.Join(root, "state"), verified: make(map[string]signature), probe: func(string) (float64, error) { return 12.5, nil }}
	rel := filepath.Join("truck001", "cam01", "main", "segment.mp4")
	source := filepath.Join(w.recordings, rel)
	marker := filepath.Join(w.markers, rel+".complete")
	dest := filepath.Join(w.bucket, rel)
	for _, dir := range []string{filepath.Dir(source), filepath.Dir(marker)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(source, []byte("recorded content"), 0644); err != nil {
		t.Fatal(err)
	}
	return w, source, marker, dest
}
func marker(t *testing.T, path, size string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(size), 0644); err != nil {
		t.Fatal(err)
	}
}
func status(t *testing.T, w worker) job {
	t.Helper()
	b, e := os.ReadFile(filepath.Join(w.states, "truck001", "cam01", "main", "segment.mp4.json"))
	if e != nil {
		t.Fatal(e)
	}
	var j job
	if e = json.Unmarshal(b, &j); e != nil {
		t.Fatal(e)
	}
	return j
}

func TestOnlyValidatedSegmentsUpload(t *testing.T) {
	w, src, m, dst := setup(t)
	if err := w.scan(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("unvalidated recording copied")
	}
	marker(t, m, "16")
	if err := w.scan(); err != nil {
		t.Fatal(err)
	}
	if status(t, w).Status != "uploaded" {
		t.Fatal("upload not marked complete")
	}
	info, err := os.Stat(filepath.Join(w.states, "truck001", "cam01", "main", "segment.mp4.json"))
	if err != nil || info.Mode().Perm()&0044 != 0044 {
		t.Fatalf("state is not readable by API: %v %v", info, err)
	}
	a, _ := os.ReadFile(src)
	b, err := os.ReadFile(dst)
	if err != nil || string(a) != string(b) {
		t.Fatal("copy differs")
	}
}

func TestFailureRetriesAndPreservesSource(t *testing.T) {
	w, src, m, dst := setup(t)
	marker(t, m, "16")
	if err := os.MkdirAll(w.bucket, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.bucket, "truck001"), []byte("block"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := w.process(m); err == nil {
		t.Fatal("expected failure")
	}
	if status(t, w).Status != "failed" {
		t.Fatal("failure was not saved")
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatal("source lost")
	}
	if err := os.Remove(filepath.Join(w.bucket, "truck001")); err != nil {
		t.Fatal(err)
	}
	if err := w.scan(); err != nil {
		t.Fatal(err)
	}
	if status(t, w).Status != "uploaded" {
		t.Fatal("retry failed")
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatal(err)
	}
}

func TestRestartRecoversPartialAndDetectsBadCopy(t *testing.T) {
	w, _, m, dst := setup(t)
	marker(t, m, "16")
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst+".simulated-upload", []byte("partial"), 0644); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(w.states, "truck001", "cam01", "main", "segment.mp4.json")
	if err := writeState(state, job{Status: "uploading", Size: 16}); err != nil {
		t.Fatal(err)
	}
	if err := w.scan(); err != nil {
		t.Fatal(err)
	}
	if status(t, w).Status != "uploaded" {
		t.Fatal("recovery failed")
	}
	if err := os.WriteFile(dst, []byte(strings.Repeat("x", 16)), 0644); err != nil {
		t.Fatal(err)
	}
	// A new worker models a service restart and verifies persisted state.
	w.verified = make(map[string]signature)
	if err := w.scan(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(dst)
	if string(b) != "recorded content" {
		t.Fatal("corrupt object not repaired")
	}
}

func TestWrongMarkerDoesNotUpload(t *testing.T) {
	w, _, m, dst := setup(t)
	marker(t, m, "100")
	if err := w.process(m); err == nil {
		t.Fatal("wrong marker accepted")
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("invalid file copied")
	}
}

func TestUploadedStateSurvivesSourceRemovalAndWorkerRestart(t *testing.T) {
	w, source, markerPath, dest := setup(t)
	marker(t, markerPath, "16")
	if err := w.process(markerPath); err != nil {
		t.Fatal(err)
	}
	if status(t, w).DurationSeconds != 12.5 {
		t.Fatal("duration not saved")
	}
	moved := source + ".moved"
	if err := os.Rename(source, moved); err != nil {
		t.Fatal(err)
	}
	w.verified = make(map[string]signature)
	if err := w.process(markerPath); err != nil {
		t.Fatalf("uploaded object lost after restart: %v", err)
	}
	if status(t, w).Status != "uploaded" {
		t.Fatal("uploaded state changed")
	}
	if err := os.WriteFile(dest, []byte("corrupted object"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := w.process(markerPath); err == nil {
		t.Fatal("corrupt object was accepted without source")
	}
}
