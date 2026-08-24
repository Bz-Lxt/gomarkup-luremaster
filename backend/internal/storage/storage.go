package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"luremaster/internal/config"
	"luremaster/internal/logger"
)

type Storage interface {
	Put(ctx context.Context, key, contentType string, r io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	URL(key string) string
}

func New(cfg config.Config) (Storage, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.StorageDriver)) {
	case "s3", "minio":
		return NewMinIO(cfg)
	default:
		return NewLocal(cfg.UploadDir)
	}
}

type Local struct {
	Dir    string
	Public string
}

func NewLocal(dir string) (*Local, error) {
	if dir == "" {
		dir = "/tmp/lure-uploads"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir upload dir: %w", err)
	}
	return &Local{Dir: dir, Public: "/api/v1/files"}, nil
}

func (l *Local) abs(key string) (string, error) {
	clean := filepath.Clean("/" + strings.TrimPrefix(key, "/"))
	full := filepath.Join(l.Dir, clean)
	if !strings.HasPrefix(full, filepath.Clean(l.Dir)+string(os.PathSeparator)) && full != filepath.Clean(l.Dir) {
		return "", fmt.Errorf("invalid key")
	}
	return full, nil
}

func (l *Local) Put(_ context.Context, key, _ string, r io.Reader, _ int64) error {
	full, err := l.abs(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	f, err := os.Create(full)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (l *Local) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	full, err := l.abs(key)
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, "", err
	}
	return f, sniffType(key), nil
}

func (l *Local) URL(key string) string {
	if key == "" {
		return ""
	}
	return path.Join(l.Public, key)
}

type MinIO struct {
	client   *minio.Client
	bucket   string
	endpoint string
	secure   bool
}

func NewMinIO(cfg config.Config) (*MinIO, error) {
	raw := strings.TrimSpace(cfg.S3Endpoint)
	if raw == "" {
		return nil, fmt.Errorf("s3 endpoint required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("s3 endpoint: %w", err)
	}
	host := u.Host
	if host == "" {
		host = strings.TrimPrefix(strings.TrimPrefix(raw, "http://"), "https://")
	}
	secure := u.Scheme == "https"
	cli, err := minio.New(host, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure:       secure,
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	m := &MinIO{client: cli, bucket: cfg.S3Bucket, endpoint: strings.TrimRight(raw, "/"), secure: secure}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	ok, err := cli.BucketExists(ctx, cfg.S3Bucket)
	if err != nil {
		logger.From().Warn("minio bucket check", "err", err)
		return m, nil
	}
	if !ok {
		if err := cli.MakeBucket(ctx, cfg.S3Bucket, minio.MakeBucketOptions{}); err != nil {
			logger.From().Warn("minio make bucket", "err", err)
		}
	}
	return m, nil
}

func (m *MinIO) Put(ctx context.Context, key, contentType string, r io.Reader, size int64) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := m.client.PutObject(ctx, m.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (m *MinIO) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	st, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, "", err
	}
	ct := st.ContentType
	if ct == "" {
		ct = sniffType(key)
	}
	return obj, ct, nil
}

func (m *MinIO) URL(key string) string {
	if key == "" {
		return ""
	}
	return "/api/v1/files/" + strings.TrimPrefix(key, "/")
}

func sniffType(key string) string {
	switch strings.ToLower(filepath.Ext(key)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}
