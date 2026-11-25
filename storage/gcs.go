// Package storage provides private Cloud Storage access for recording objects.
package storage

import (
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"

	gcs "cloud.google.com/go/storage"
)

type Cloud struct {
	client *gcs.Client
	bucket *gcs.BucketHandle
}

// NewCloud uses Application Default Credentials or STORAGE_EMULATOR_HOST.
func NewCloud(ctx context.Context, bucket string) (*Cloud, error) {
	if strings.TrimSpace(bucket) == "" || strings.Contains(bucket, "/") {
		return nil, fmt.Errorf("invalid bucket name")
	}
	client, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Cloud{client: client, bucket: client.Bucket(bucket)}, nil
}

func (c *Cloud) Close() error { return c.client.Close() }

// Upload creates a new object only; an existing object is never overwritten.
// GCS verifies the explicit checksum before accepting the object.
func (c *Cloud) Upload(ctx context.Context, key, filename string, metadata map[string]string) (*gcs.ObjectAttrs, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() == 0 {
		return nil, fmt.Errorf("invalid source file")
	}
	h := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	if _, err = io.Copy(h, file); err != nil {
		return nil, err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	writer := c.bucket.Object(key).If(gcs.Conditions{DoesNotExist: true}).NewWriter(ctx)
	writer.ContentType = "video/mp4"
	writer.Metadata = metadata
	writer.CRC32C = h.Sum32()
	writer.SendCRC32C = true
	if _, err = io.Copy(writer, file); err != nil {
		_ = writer.CloseWithError(err)
		return nil, err
	}
	after, err := file.Stat()
	if err != nil || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		_ = writer.CloseWithError(fmt.Errorf("source changed during upload"))
		return nil, fmt.Errorf("source changed during upload")
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return writer.Attrs(), nil
}

func (c *Cloud) Attrs(ctx context.Context, key string) (*gcs.ObjectAttrs, error) {
	return c.bucket.Object(key).Attrs(ctx)
}

// OpenRange pins the object generation to avoid serving a replacement object.
func (c *Cloud) OpenRange(ctx context.Context, key string, generation, offset, length int64) (*gcs.Reader, error) {
	if generation <= 0 || offset < 0 || length < -1 {
		return nil, fmt.Errorf("invalid range")
	}
	return c.bucket.Object(key).Generation(generation).ReadCompressed(true).NewRangeReader(ctx, offset, length)
}
