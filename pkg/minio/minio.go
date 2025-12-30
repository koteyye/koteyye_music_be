package minio

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	*minio.Client
	bucket string
	logger *slog.Logger
}

// New creates a new MinIO client
func New(endpoint, accessKey, secretKey, bucket string, useSSL bool, logger *slog.Logger) (*Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Create bucket if it doesn't exist
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		logger.Info("MinIO bucket created", "bucket", bucket)
	}

	return &Client{
		Client: client,
		bucket: bucket,
		logger: logger,
	}, nil
}

// UploadFile uploads a file to MinIO
func (c *Client) UploadFile(ctx context.Context, objectName, filePath, contentType string) error {
	info, err := c.FPutObject(ctx, c.bucket, objectName, filePath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	c.logger.Info("File uploaded to MinIO",
		"object_name", objectName,
		"size", info.Size,
	)

	return nil
}

// GetObject returns a reader for the object
func (c *Client) GetObject(ctx context.Context, objectName string) (*minio.Object, error) {
	object, err := c.Client.GetObject(ctx, c.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return object, nil
}

// GetObjectInfo returns object metadata
func (c *Client) GetObjectInfo(ctx context.Context, objectName string) (minio.ObjectInfo, error) {
	info, err := c.Client.StatObject(ctx, c.bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return minio.ObjectInfo{}, fmt.Errorf("failed to get object info: %w", err)
	}
	return info, nil
}

// PutObject uploads an object to MinIO
func (c *Client) PutObject(ctx context.Context, objectName string, reader io.Reader, size int64, metadata map[string]string) (*minio.UploadInfo, error) {
	opts := minio.PutObjectOptions{}
	if contentType, ok := metadata["Content-Type"]; ok {
		opts.ContentType = contentType
	}
	if len(metadata) > 0 {
		opts.UserMetadata = metadata
	}

	info, err := c.Client.PutObject(ctx, c.bucket, objectName, reader, size, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to put object: %w", err)
	}

	c.logger.Info("Object uploaded to MinIO",
		"object_name", objectName,
		"size", info.Size,
	)

	return &info, nil
}

// DeleteObject deletes an object from MinIO
func (c *Client) DeleteObject(ctx context.Context, objectName string) error {
	err := c.Client.RemoveObject(ctx, c.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// Service provides high-level MinIO operations
type Service struct {
	client   *Client
	endpoint string
	useSSL   bool
	logger   *slog.Logger
}

// NewService creates a new MinIO service
func NewService(client *Client, endpoint string, useSSL bool, logger *slog.Logger) *Service {
	return &Service{
		client:   client,
		endpoint: endpoint,
		useSSL:   useSSL,
		logger:   logger,
	}
}

// UploadFile uploads a multipart file to MinIO
func (s *Service) UploadFile(ctx context.Context, bucket, objectName string, file multipart.File, size int64) (*minio.UploadInfo, error) {
	// Detect content type from object name
	contentType := "application/octet-stream"
	if strings.HasSuffix(strings.ToLower(objectName), ".jpg") || strings.HasSuffix(strings.ToLower(objectName), ".jpeg") {
		contentType = "image/jpeg"
	} else if strings.HasSuffix(strings.ToLower(objectName), ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(strings.ToLower(objectName), ".mp3") {
		contentType = "audio/mpeg"
	}

	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	info, err := s.client.Client.PutObject(ctx, bucket, objectName, file, size, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	s.logger.Info("File uploaded to MinIO",
		"object_name", objectName,
		"size", info.Size,
		"bucket", bucket,
	)

	return &info, nil
}

// GetFileURL generates a presigned URL for file access
func (s *Service) GetFileURL(bucket, objectName string) (string, error) {
	// For simplicity, return direct URL (in production use presigned URLs)
	protocol := "http"
	if s.useSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", protocol, s.endpoint, bucket, objectName), nil
}

// DeleteFile deletes a file from MinIO
func (s *Service) DeleteFile(ctx context.Context, bucket, objectName string) error {
	err := s.client.Client.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.logger.Info("File deleted from MinIO", "object_name", objectName, "bucket", bucket)
	return nil
}

// DeleteFolder deletes all objects with a given prefix (simulating folder deletion)
func (s *Service) DeleteFolder(ctx context.Context, bucket, prefix string) error {
	// List all objects with the prefix
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	objectCh := s.client.Client.ListObjects(ctx, bucket, opts)

	// Collect object names to delete
	var objectNames []string
	for object := range objectCh {
		if object.Err != nil {
			return fmt.Errorf("error listing objects: %w", object.Err)
		}
		objectNames = append(objectNames, object.Key)
	}

	// Delete all objects
	for _, objectName := range objectNames {
		if err := s.DeleteFile(ctx, bucket, objectName); err != nil {
			s.logger.Error("Failed to delete object in folder", "object", objectName, "error", err)
			// Continue deleting other objects even if one fails
		}
	}

	s.logger.Info("Folder deleted from MinIO", "prefix", prefix, "bucket", bucket, "objects_deleted", len(objectNames))
	return nil
}
