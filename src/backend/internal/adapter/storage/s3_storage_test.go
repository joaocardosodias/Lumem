package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type fakeS3API struct {
	objects    map[string][]byte
	putInput   *s3.PutObjectInput
	putErr     error
	getErr     error
	deleteErr  error
	deletedKey string
}

func newFakeS3API() *fakeS3API {
	return &fakeS3API{objects: make(map[string][]byte)}
}

func (f *fakeS3API) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if f.putErr != nil {
		return nil, f.putErr
	}
	data, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	f.putInput = input
	f.objects[aws.ToString(input.Key)] = data
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeS3API) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	data, ok := f.objects[aws.ToString(input.Key)]
	if !ok {
		return nil, errors.New("objeto nao encontrado")
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(data))}, nil
}

func (f *fakeS3API) DeleteObject(_ context.Context, input *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	f.deletedKey = aws.ToString(input.Key)
	delete(f.objects, f.deletedKey)
	return &s3.DeleteObjectOutput{}, nil
}

func TestS3StorageStoreOpenAndDelete(t *testing.T) {
	client := newFakeS3API()
	store := newS3Storage(client, "ingestion-uploads", "ingestion")
	ctx := context.Background()
	content := "tag,estacao,porta\nA,FAL,1\n"

	key, hash, err := store.Store(ctx, "alarmes.csv", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Store retornou erro: %v", err)
	}
	if !strings.HasPrefix(key, "ingestion/") || !strings.HasSuffix(key, ".csv") {
		t.Fatalf("chave S3 inesperada: %s", key)
	}
	if got := aws.ToString(client.putInput.Bucket); got != "ingestion-uploads" {
		t.Fatalf("bucket inesperado: %s", got)
	}
	if got := aws.ToString(client.putInput.ContentType); got != "text/csv" {
		t.Fatalf("content-type inesperado: %s", got)
	}
	expectedHash := sha256.Sum256([]byte(content))
	if hash != hex.EncodeToString(expectedHash[:]) {
		t.Fatalf("hash inesperado: %s", hash)
	}

	reader, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open retornou erro: %v", err)
	}
	data, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || string(data) != content {
		t.Fatalf("conteudo recuperado inesperado: %q, readErr: %v, closeErr: %v", data, readErr, closeErr)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete retornou erro: %v", err)
	}
	if client.deletedKey != key {
		t.Fatalf("chave removida inesperada: %s", client.deletedKey)
	}
}

func TestS3StorageAcceptsXLSXAndRejectsOtherTypes(t *testing.T) {
	client := newFakeS3API()
	store := newS3Storage(client, "ingestion-uploads", "")

	key, _, err := store.Store(context.Background(), "planilha.XLSX", strings.NewReader("xlsx"))
	if err != nil {
		t.Fatalf("esperava aceitar XLSX: %v", err)
	}
	if !strings.HasSuffix(key, ".xlsx") {
		t.Fatalf("chave deveria preservar extensao xlsx: %s", key)
	}
	if got := aws.ToString(client.putInput.ContentType); got != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("content-type XLSX inesperado: %s", got)
	}

	_, _, err = store.Store(context.Background(), "legado.xls", strings.NewReader("xls"))
	if !errors.Is(err, ErrUnsupportedUploadType) {
		t.Fatalf("esperava ErrUnsupportedUploadType para XLS: %v", err)
	}
}

func TestS3StoragePropagatesClientErrors(t *testing.T) {
	client := newFakeS3API()
	client.putErr = errors.New("s3 indisponivel")
	store := newS3Storage(client, "ingestion-uploads", "ingestion")

	_, _, err := store.Store(context.Background(), "alarmes.csv", strings.NewReader("dados"))
	if err == nil || !strings.Contains(err.Error(), "enviar arquivo para S3") {
		t.Fatalf("esperava erro contextualizado do S3: %v", err)
	}

	_, _, err = store.Store(context.Background(), "alarmes.csv", brokenReader{})
	if err == nil || !strings.Contains(err.Error(), "preparar upload S3") {
		t.Fatalf("esperava erro ao preparar upload S3: %v", err)
	}

	client.getErr = errors.New("get falhou")
	if _, err := store.Open(context.Background(), "arquivo.csv"); err == nil || !strings.Contains(err.Error(), "baixar arquivo do S3") {
		t.Fatalf("esperava erro contextualizado ao baixar: %v", err)
	}

	client.deleteErr = errors.New("delete falhou")
	if err := store.Delete(context.Background(), "arquivo.csv"); err == nil || !strings.Contains(err.Error(), "remover arquivo do S3") {
		t.Fatalf("esperava erro contextualizado ao deletar: %v", err)
	}
}

type brokenReader struct{}

func (brokenReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("falha de leitura")
}

func TestNewS3Storage(t *testing.T) {
	_, err := NewS3Storage(context.Background(), S3Config{})
	if err == nil || !strings.Contains(err.Error(), "configuracao S3 incompleta") {
		t.Fatalf("esperava erro de configuracao incompleta, obteve: %v", err)
	}

	cfg := S3Config{
		Endpoint:        "http://localhost:9000",
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Prefix:          "uploads",
	}
	st, err := NewS3Storage(context.Background(), cfg)
	if err != nil {
		t.Fatalf("esperava sucesso ao criar S3Storage: %v", err)
	}
	if st == nil || st.bucket != "test-bucket" || st.prefix != "uploads" {
		t.Fatalf("estado incorreto do S3Storage: %+v", st)
	}
}

func TestS3StorageStoreTempFileError(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "diretorio-inexistente")
	for _, variable := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(variable, missingDir)
	}

	store := newS3Storage(newFakeS3API(), "ingestion-uploads", "ingestion")
	_, _, err := store.Store(context.Background(), "alarmes.csv", strings.NewReader("dados"))
	if err == nil || !strings.Contains(err.Error(), "criar arquivo temporario") {
		t.Fatalf("esperava erro ao criar arquivo temporario, obteve %v", err)
	}
}
