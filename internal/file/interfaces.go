package file

import (
	"context"
	"io"
)

// FileRepository defines the interface for file metadata persistence
type FileRepository interface {
	// Create creates a new file record
	Create(ctx context.Context, file *File) error

	// GetByFileID retrieves a file by its file_id
	GetByFileID(ctx context.Context, fileID string) (*File, error)

	// Delete soft-deletes a file record
	Delete(ctx context.Context, fileID string) error

	// ListByUploader retrieves files uploaded by a specific user
	ListByUploader(ctx context.Context, userID int64, limit, offset int32) ([]*File, error)
}

// StorageClient defines the interface for object storage operations
type StorageClient interface {
	// Upload uploads a file to object storage
	Upload(ctx context.Context, key string, body io.Reader, contentType string) error

	// Download downloads a file from object storage
	Download(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete deletes a file from object storage
	Delete(ctx context.Context, key string) error

	// GetPresignedURL generates a presigned URL for file download
	GetPresignedURL(ctx context.Context, key string) (string, error)
}
