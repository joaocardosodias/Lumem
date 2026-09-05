package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/g04/backend/internal/usecase/port"
)

var _ port.IngestionStorage = (*LocalTempStorage)(nil)

type LocalTempStorage struct{}

func NewLocalTempStorage() *LocalTempStorage {
	return &LocalTempStorage{}
}

func (s *LocalTempStorage) Store(ctx context.Context, name string, r io.Reader) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		ext = ".csv"
	}

	tmpFile, err := os.CreateTemp("", "ingestion_*"+ext)
	if err != nil {
		return "", "", err
	}
	defer tmpFile.Close()

	hasher := sha256.New()
	tee := io.TeeReader(r, hasher)

	if _, err := io.Copy(tmpFile, tee); err != nil {
		os.Remove(tmpFile.Name())
		return "", "", err
	}

	hashStr := hex.EncodeToString(hasher.Sum(nil))
	return tmpFile.Name(), hashStr, nil
}

func (s *LocalTempStorage) Open(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	return os.Open(storageKey)
}

func (s *LocalTempStorage) Delete(ctx context.Context, storageKey string) error {
	return os.Remove(storageKey)
}
