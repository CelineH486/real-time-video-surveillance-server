package storage

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectsInvalidBucket(t *testing.T) {
	for _, bucket := range []string{"", "bucket/path"} {
		if _, err := NewCloud(context.Background(), bucket); err == nil {
			t.Fatalf("accepted invalid bucket %q", bucket)
		}
	}
}

func TestCloudClientUploadsToLocalFixture(t *testing.T) {
	content := []byte("video test data")
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Query().Get("uploadType") == "multipart":
			_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil {
				t.Errorf("multipart: %v", err)
				http.Error(w, "multipart", 400)
				return
			}
			reader := multipart.NewReader(r.Body, params["boundary"])
			_, err = reader.NextPart()
			if err != nil {
				t.Errorf("metadata: %v", err)
				return
			}
			part, err := reader.NextPart()
			if err != nil {
				t.Errorf("content: %v", err)
				return
			}
			got, _ := io.ReadAll(part)
			if string(got) != string(content) {
				t.Errorf("received unexpected bytes %q", got)
			}
			crc := crc32.Checksum(got, crc32.MakeTable(crc32.Castagnoli))
			raw := make([]byte, 4)
			binary.BigEndian.PutUint32(raw, crc)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"name":"fixture.mp4","bucket":"local-test","size":"%d","generation":"7","crc32c":"%s"}`, len(got), base64.StdEncoding.EncodeToString(raw))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/upload/storage/"):
			w.Header().Set("Location", server.URL+"/session")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == "/session":
			got, _ := io.ReadAll(r.Body)
			if string(got) != string(content) {
				t.Errorf("received unexpected bytes %q", got)
			}
			crc := crc32.Checksum(got, crc32.MakeTable(crc32.Castagnoli))
			raw := make([]byte, 4)
			binary.BigEndian.PutUint32(raw, crc)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"name":"fixture.mp4","bucket":"local-test","size":"%d","generation":"7","crc32c":"%s"}`, len(got), base64.StdEncoding.EncodeToString(raw))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	defer server.Close()
	t.Setenv("STORAGE_EMULATOR_HOST", server.URL)
	file := filepath.Join(t.TempDir(), "fixture.mp4")
	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}
	c, err := NewCloud(context.Background(), "local-test")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	attrs, err := c.Upload(context.Background(), "fixture.mp4", file, map[string]string{"sha256": "test"})
	if err != nil {
		t.Fatal(err)
	}
	if attrs.Generation != 7 || attrs.Size != int64(len(content)) {
		t.Fatalf("unexpected attrs: %+v", attrs)
	}
}

func TestRejectsInvalidRangeBeforeAccess(t *testing.T) {
	c := &Cloud{}
	for _, v := range [][3]int64{{0, 0, -1}, {1, -1, -1}, {1, 0, -2}} {
		if _, err := c.OpenRange(context.Background(), "x", v[0], v[1], v[2]); err == nil {
			t.Fatalf("accepted invalid range %v", v)
		}
	}
}
