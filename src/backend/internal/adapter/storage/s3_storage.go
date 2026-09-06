package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
)

var (
	_ port.IngestionStorage = (*S3Storage)(nil)

	ErrUnsupportedUploadType = errors.New("tipo de arquivo nao suportado; use .csv ou .xlsx")
)

type S3Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Prefix          string
}

type s3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type S3Storage struct {
	client s3API
	bucket string
	prefix string
}

func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	region := strings.TrimSpace(cfg.Region)
	bucket := strings.TrimSpace(cfg.Bucket)
	accessKeyID := strings.TrimSpace(cfg.AccessKeyID)
	secretAccessKey := strings.TrimSpace(cfg.SecretAccessKey)
	if endpoint == "" || region == "" || bucket == "" || accessKeyID == "" || secretAccessKey == "" {
		return nil, errors.New("configuracao S3 incompleta")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("carregar configuracao S3: %w", err)
	}
	awsCfg.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	awsCfg.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(strings.TrimRight(endpoint, "/"))
		options.UsePathStyle = true
	})

	return newS3Storage(client, bucket, cfg.Prefix), nil
}

func newS3Storage(client s3API, bucket, prefix string) *S3Storage {
	return &S3Storage{
		client: client,
		bucket: strings.TrimSpace(bucket),
		prefix: strings.Trim(strings.TrimSpace(prefix), "/"),
	}
}

func uploadMetadata(name string) (extension, contentType string, err error) {
	extension = strings.ToLower(filepath.Ext(strings.TrimSpace(name)))
	switch extension {
	case ".csv":
		return extension, "text/csv", nil
	case ".xlsx":
		return extension, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	default:
		return "", "", ErrUnsupportedUploadType
	}
}

func (s *S3Storage) Store(ctx context.Context, name string, reader io.Reader) (string, string, error) {
	extension, contentType, err := uploadMetadata(name)
	if err != nil {
		return "", "", err
	}

	tmpFile, err := os.CreateTemp("", "s3-ingestion-*"+extension)
	if err != nil {
		return "", "", fmt.Errorf("criar arquivo temporario para upload S3: %w", err)
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	hasher := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmpFile, hasher), reader)
	if err != nil {
		return "", "", fmt.Errorf("preparar upload S3: %w", err)
	}
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return "", "", fmt.Errorf("reposicionar arquivo para upload S3: %w", err)
	}

	objectName := uuid.NewString() + extension
	storageKey := objectName
	if s.prefix != "" {
		storageKey = path.Join(s.prefix, objectName)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(storageKey),
		Body:          tmpFile,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return "", "", fmt.Errorf("enviar arquivo para S3: %w", err)
	}

	return storageKey, hex.EncodeToString(hasher.Sum(nil)), nil
}

func (s *S3Storage) Open(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return nil, fmt.Errorf("baixar arquivo do S3: %w", err)
	}
	return output.Body, nil
}

func (s *S3Storage) Delete(ctx context.Context, storageKey string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return fmt.Errorf("remover arquivo do S3: %w", err)
	}
	return nil
}
