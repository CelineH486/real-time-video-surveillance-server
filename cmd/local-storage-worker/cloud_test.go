package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	gcs "cloud.google.com/go/storage"
)

type fakeCloud struct {
	objects     map[string]*gcs.ObjectAttrs
	fail        bool
	failGet     bool
	afterUpload func()
}

func (f *fakeCloud) Attrs(_ context.Context, key string) (*gcs.ObjectAttrs, error) {
	if f.failGet {
		return nil, errors.New("cloud unavailable")
	}
	if x := f.objects[key]; x != nil {
		return x, nil
	}
	return nil, gcs.ErrObjectNotExist
}

func TestCloudCleanupAfterConfirmedUpload(t *testing.T) {
	w, source, markerPath, _ := setup(t)
	w.cloud = &fakeCloud{objects: map[string]*gcs.ObjectAttrs{}}
	w.cleanupUploaded = true
	marker(t, markerPath, "16")
	if err := w.process(markerPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("confirmed source not removed: %v", err)
	}
	if state := status(t, w); state.Status != "uploaded" || state.Generation != 42 {
		t.Fatalf("state lost: %+v", state)
	}
	if err := w.process(markerPath); err != nil {
		t.Fatalf("restart after cleanup: %v", err)
	}
}

func TestCloudCleanupRetainsFailedUpload(t *testing.T) {
	w, source, markerPath, _ := setup(t)
	w.cloud = &fakeCloud{objects: map[string]*gcs.ObjectAttrs{}, fail: true}
	w.cleanupUploaded = true
	marker(t, markerPath, "16")
	if err := w.process(markerPath); err == nil {
		t.Fatal("expected upload failure")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source was removed: %v", err)
	}
}

func TestCloudCleanupWaitsForSavedState(t *testing.T) {
	w, source, markerPath, _ := setup(t)
	w.cloud = &fakeCloud{objects: map[string]*gcs.ObjectAttrs{}}
	w.cleanupUploaded = true
	marker(t, markerPath, "16")
	if err := os.MkdirAll(w.states, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.states, "truck001"), []byte("blocked"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := w.process(markerPath); err == nil {
		t.Fatal("expected state write failure")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source removed despite state failure: %v", err)
	}
}

func TestCloudDoesNotProcessWithoutCompletionMarker(t *testing.T) {
	w, source, _, _ := setup(t)
	f := &fakeCloud{objects: map[string]*gcs.ObjectAttrs{}}
	w.cloud = f
	w.cleanupUploaded = true
	if err := w.scan(); err != nil {
		t.Fatal(err)
	}
	if len(f.objects) != 0 {
		t.Fatal("uploaded unfinished recording")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("unfinished recording removed: %v", err)
	}
}

func TestCloudCleanupRetainsChangedOrUnverifiableSource(t *testing.T) {
	for _, scenario := range []string{"source", "marker", "cloud_generation", "cloud_crc", "cloud_unavailable"} {
		t.Run(scenario, func(t *testing.T) {
			w, source, markerPath, _ := setup(t)
			f := &fakeCloud{objects: map[string]*gcs.ObjectAttrs{}}
			w.cloud = f
			marker(t, markerPath, "16")
			if err := w.process(markerPath); err != nil {
				t.Fatal(err)
			}
			j := status(t, w)
			key := "truck001/cam01/main/segment.mp4"
			switch scenario {
			case "source":
				if err := os.WriteFile(source, []byte("modified content"), 0644); err != nil {
					t.Fatal(err)
				}
			case "marker":
				marker(t, markerPath, "15")
			case "cloud_generation":
				f.objects[key].Generation++
			case "cloud_crc":
				f.objects[key].CRC32C++
			case "cloud_unavailable":
				f.failGet = true
			}
			w.cleanupUploaded = true
			if err := w.cleanupCloudSource(context.Background(), source, markerPath, key, j); err == nil {
				t.Fatal("expected cleanup refusal")
			}
			if _, err := os.Stat(source); err != nil {
				t.Fatalf("source was removed: %v", err)
			}
		})
	}
}
func (f *fakeCloud) Upload(_ context.Context, key, path string, metadata map[string]string) (*gcs.ObjectAttrs, error) {
	if f.fail {
		return nil, errors.New("offline")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	crc, err := checksumCRC32C(path)
	if err != nil {
		return nil, err
	}
	x := &gcs.ObjectAttrs{Size: info.Size(), Generation: 42, Metadata: metadata, CRC32C: crc}
	f.objects[key] = x
	if f.afterUpload != nil {
		f.afterUpload()
	}
	return x, nil
}

func TestCloudCleanupRetriesThenRemovesSource(t *testing.T) {
	w, source, m, _ := setup(t)
	f := &fakeCloud{objects: map[string]*gcs.ObjectAttrs{}, fail: true}
	w.cloud = f
	w.cleanupUploaded = true
	marker(t, m, "16")
	if err := w.process(m); err == nil {
		t.Fatal("expected failure")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("failed upload lost source")
	}
	f.fail = false
	if err := w.process(m); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("confirmed upload not cleaned: %v", err)
	}
}

func TestCloudCleanupRefusesMarkerRemovedDuringUpload(t *testing.T) {
	w, source, m, _ := setup(t)
	f := &fakeCloud{objects: map[string]*gcs.ObjectAttrs{}, afterUpload: func() { _ = os.Remove(m) }}
	w.cloud = f
	w.cleanupUploaded = true
	marker(t, m, "16")
	if err := w.process(m); err == nil {
		t.Fatal("cleanup must reject missing marker")
	}
	if state := status(t, w); state.Status != "uploaded" {
		t.Fatalf("uploaded status should survive cleanup failure: %+v", state)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source removed after marker change: %v", err)
	}
	marker(t, m, "16")
	if err := w.process(m); err != nil {
		t.Fatalf("cleanup retry failed: %v", err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source not removed after cleanup retry: %v", err)
	}
}

func TestCloudUploadFailureRetryAndRestart(t *testing.T) {
	w, source, m, _ := setup(t)
	f := &fakeCloud{objects: map[string]*gcs.ObjectAttrs{}, fail: true}
	w.cloud = f
	marker(t, m, "16")
	if err := w.process(m); err == nil || status(t, w).Status != "failed" {
		t.Fatal("failed upload must persist failure")
	}
	f.fail = false
	if err := w.process(m); err != nil {
		t.Fatal(err)
	}
	if state := status(t, w); state.Status != "uploaded" || state.Generation != 42 {
		t.Fatalf("bad state: %+v", state)
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	if err := w.process(m); err != nil {
		t.Fatalf("restart without local source: %v", err)
	}
}

func TestCloudUploadRecoversIfStateWasNotSaved(t *testing.T) {
	w, _, m, _ := setup(t)
	f := &fakeCloud{objects: map[string]*gcs.ObjectAttrs{}}
	w.cloud = f
	marker(t, m, "16")
	if err := w.process(m); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(w.states, "truck001", "cam01", "main", "segment.mp4.json")
	if err := writeState(path, job{Status: "uploading", Size: 16}); err != nil {
		t.Fatal(err)
	}
	if err := w.process(m); err != nil {
		t.Fatal(err)
	}
	if state := status(t, w); state.Status != "uploaded" || state.Generation != 42 {
		t.Fatalf("recovery failed: %+v", state)
	}
}
