package file

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/dollarkillerx/im-system/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = logger.Init("error", "console", []string{"stdout"})
}

// MockFileRepository is a mock implementation of FileRepository
type MockFileRepository struct {
	files          map[string]*File
	createFunc     func(ctx context.Context, file *File) error
	getByFileIDFunc func(ctx context.Context, fileID string) (*File, error)
	deleteFunc     func(ctx context.Context, fileID string) error
}

func newMockFileRepository() *MockFileRepository {
	return &MockFileRepository{
		files: make(map[string]*File),
	}
}

func (m *MockFileRepository) Create(ctx context.Context, file *File) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, file)
	}
	m.files[file.FileID] = file
	return nil
}

func (m *MockFileRepository) GetByFileID(ctx context.Context, fileID string) (*File, error) {
	if m.getByFileIDFunc != nil {
		return m.getByFileIDFunc(ctx, fileID)
	}
	file, ok := m.files[fileID]
	if !ok {
		return nil, errors.New("file not found")
	}
	return file, nil
}

func (m *MockFileRepository) Delete(ctx context.Context, fileID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, fileID)
	}
	file, ok := m.files[fileID]
	if !ok {
		return errors.New("file not found")
	}
	file.Status = "deleted"
	return nil
}

func (m *MockFileRepository) ListByUploader(ctx context.Context, userID int64, limit, offset int32) ([]*File, error) {
	var result []*File
	for _, file := range m.files {
		if file.UploaderID == userID && file.Status == "active" {
			result = append(result, file)
		}
	}
	return result, nil
}

// MockStorageClient is a mock implementation of StorageClient
type MockStorageClient struct {
	storage            map[string][]byte
	uploadFunc         func(ctx context.Context, key string, body io.Reader, contentType string) error
	downloadFunc       func(ctx context.Context, key string) (io.ReadCloser, error)
	deleteFunc         func(ctx context.Context, key string) error
	getPresignedURLFunc func(ctx context.Context, key string) (string, error)
}

func newMockStorageClient() *MockStorageClient {
	return &MockStorageClient{
		storage: make(map[string][]byte),
	}
}

func (m *MockStorageClient) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, key, body, contentType)
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.storage[key] = data
	return nil
}

func (m *MockStorageClient) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, key)
	}
	data, ok := m.storage[key]
	if !ok {
		return nil, errors.New("file not found in storage")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *MockStorageClient) Delete(ctx context.Context, key string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, key)
	}
	delete(m.storage, key)
	return nil
}

func (m *MockStorageClient) GetPresignedURL(ctx context.Context, key string) (string, error) {
	if m.getPresignedURLFunc != nil {
		return m.getPresignedURLFunc(ctx, key)
	}
	if _, ok := m.storage[key]; !ok {
		return "", errors.New("file not found in storage")
	}
	return "https://example.com/presigned/" + key, nil
}

func TestService_UploadFile(t *testing.T) {
	tests := []struct {
		name        string
		uploaderID  int64
		fileName    string
		fileSize    int64
		contentType string
		fileData    string
		maxSize     int64
		setupMock   func(*MockFileRepository, *MockStorageClient)
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful upload",
			uploaderID:  100,
			fileName:    "test.txt",
			fileSize:    1024,
			contentType: "text/plain",
			fileData:    "test file content",
			maxSize:     10 * 1024 * 1024,
			wantErr:     false,
		},
		{
			name:        "file size exceeds maximum",
			uploaderID:  100,
			fileName:    "large.txt",
			fileSize:    100 * 1024 * 1024,
			contentType: "text/plain",
			fileData:    "large file",
			maxSize:     10 * 1024 * 1024,
			wantErr:     true,
			errContains: "exceeds maximum allowed size",
		},
		{
			name:        "storage upload failure",
			uploaderID:  100,
			fileName:    "test.txt",
			fileSize:    1024,
			contentType: "text/plain",
			fileData:    "test content",
			maxSize:     10 * 1024 * 1024,
			setupMock: func(repo *MockFileRepository, storage *MockStorageClient) {
				storage.uploadFunc = func(ctx context.Context, key string, body io.Reader, contentType string) error {
					return errors.New("S3 upload failed")
				}
			},
			wantErr:     true,
			errContains: "failed to upload to storage",
		},
		{
			name:        "database create failure",
			uploaderID:  100,
			fileName:    "test.txt",
			fileSize:    1024,
			contentType: "text/plain",
			fileData:    "test content",
			maxSize:     10 * 1024 * 1024,
			setupMock: func(repo *MockFileRepository, storage *MockStorageClient) {
				repo.createFunc = func(ctx context.Context, file *File) error {
					return errors.New("database error")
				}
			},
			wantErr:     true,
			errContains: "failed to create file record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockFileRepository()
			storage := newMockStorageClient()
			if tt.setupMock != nil {
				tt.setupMock(repo, storage)
			}
			service := NewService(repo, storage, tt.maxSize)

			file, err := service.UploadFile(
				context.Background(),
				tt.uploaderID,
				tt.fileName,
				tt.fileSize,
				tt.contentType,
				strings.NewReader(tt.fileData),
			)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, file)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, file)
				assert.NotEmpty(t, file.FileID)
				assert.Equal(t, tt.uploaderID, file.UploaderID)
				assert.Equal(t, tt.fileName, file.FileName)
				assert.Equal(t, tt.fileSize, file.FileSize)
				assert.Equal(t, tt.contentType, file.ContentType)
				assert.Equal(t, "active", file.Status)
				assert.NotEmpty(t, file.StorageKey)
			}
		})
	}
}

func TestService_GetFile(t *testing.T) {
	repo := newMockFileRepository()
	storage := newMockStorageClient()
	service := NewService(repo, storage, 10*1024*1024)

	// Setup: Create a test file
	testFile := &File{
		FileID:      "file-123",
		UploaderID:  100,
		FileName:    "test.txt",
		FileSize:    1024,
		ContentType: "text/plain",
		StorageKey:  "uploads/2024/01/01/file-123.txt",
		Status:      "active",
		CreatedAt:   time.Now(),
	}
	repo.files["file-123"] = testFile

	tests := []struct {
		name    string
		fileID  string
		wantErr bool
	}{
		{
			name:    "file found",
			fileID:  "file-123",
			wantErr: false,
		},
		{
			name:    "file not found",
			fileID:  "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := service.GetFile(context.Background(), tt.fileID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, file)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, file)
				assert.Equal(t, tt.fileID, file.FileID)
			}
		})
	}
}

func TestService_DownloadFile(t *testing.T) {
	tests := []struct {
		name        string
		fileID      string
		setupData   func(*MockFileRepository, *MockStorageClient)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful download",
			fileID: "file-123",
			setupData: func(repo *MockFileRepository, storage *MockStorageClient) {
				file := &File{
					FileID:      "file-123",
					UploaderID:  100,
					FileName:    "test.txt",
					FileSize:    1024,
					ContentType: "text/plain",
					StorageKey:  "uploads/2024/01/01/file-123.txt",
					Status:      "active",
				}
				repo.files["file-123"] = file
				storage.storage["uploads/2024/01/01/file-123.txt"] = []byte("test file content")
			},
			wantErr: false,
		},
		{
			name:    "file not found in database",
			fileID:  "nonexistent",
			wantErr: true,
		},
		{
			name:   "file not found in storage",
			fileID: "file-456",
			setupData: func(repo *MockFileRepository, storage *MockStorageClient) {
				file := &File{
					FileID:     "file-456",
					StorageKey: "uploads/missing.txt",
				}
				repo.files["file-456"] = file
				// Don't add to storage
			},
			wantErr:     true,
			errContains: "failed to download from storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockFileRepository()
			storage := newMockStorageClient()
			if tt.setupData != nil {
				tt.setupData(repo, storage)
			}
			service := NewService(repo, storage, 10*1024*1024)

			body, file, err := service.DownloadFile(context.Background(), tt.fileID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, body)
				assert.Nil(t, file)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, body)
				assert.NotNil(t, file)
				defer body.Close()

				// Verify file content
				data, err := io.ReadAll(body)
				require.NoError(t, err)
				assert.Equal(t, "test file content", string(data))
			}
		})
	}
}

func TestService_GetDownloadURL(t *testing.T) {
	tests := []struct {
		name      string
		fileID    string
		setupData func(*MockFileRepository, *MockStorageClient)
		wantErr   bool
	}{
		{
			name:   "successful URL generation",
			fileID: "file-123",
			setupData: func(repo *MockFileRepository, storage *MockStorageClient) {
				file := &File{
					FileID:     "file-123",
					StorageKey: "uploads/2024/01/01/file-123.txt",
				}
				repo.files["file-123"] = file
				storage.storage["uploads/2024/01/01/file-123.txt"] = []byte("test")
			},
			wantErr: false,
		},
		{
			name:    "file not found",
			fileID:  "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockFileRepository()
			storage := newMockStorageClient()
			if tt.setupData != nil {
				tt.setupData(repo, storage)
			}
			service := NewService(repo, storage, 10*1024*1024)

			url, err := service.GetDownloadURL(context.Background(), tt.fileID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, url)
				assert.Contains(t, url, "presigned")
			}
		})
	}
}

func TestService_DeleteFile(t *testing.T) {
	tests := []struct {
		name        string
		fileID      string
		userID      int64
		setupData   func(*MockFileRepository, *MockStorageClient)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful deletion",
			fileID: "file-123",
			userID: 100,
			setupData: func(repo *MockFileRepository, storage *MockStorageClient) {
				file := &File{
					FileID:     "file-123",
					UploaderID: 100,
					StorageKey: "uploads/2024/01/01/file-123.txt",
					Status:     "active",
				}
				repo.files["file-123"] = file
				storage.storage["uploads/2024/01/01/file-123.txt"] = []byte("test")
			},
			wantErr: false,
		},
		{
			name:   "permission denied - different user",
			fileID: "file-123",
			userID: 200,
			setupData: func(repo *MockFileRepository, storage *MockStorageClient) {
				file := &File{
					FileID:     "file-123",
					UploaderID: 100,
					StorageKey: "uploads/2024/01/01/file-123.txt",
					Status:     "active",
				}
				repo.files["file-123"] = file
			},
			wantErr:     true,
			errContains: "permission denied",
		},
		{
			name:    "file not found",
			fileID:  "nonexistent",
			userID:  100,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockFileRepository()
			storage := newMockStorageClient()
			if tt.setupData != nil {
				tt.setupData(repo, storage)
			}
			service := NewService(repo, storage, 10*1024*1024)

			err := service.DeleteFile(context.Background(), tt.fileID, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				// Verify file was soft-deleted
				file, _ := repo.GetByFileID(context.Background(), tt.fileID)
				assert.Equal(t, "deleted", file.Status)
			}
		})
	}
}

func TestService_ListUserFiles(t *testing.T) {
	repo := newMockFileRepository()
	storage := newMockStorageClient()
	service := NewService(repo, storage, 10*1024*1024)

	// Setup: Create multiple files
	repo.files["file-1"] = &File{FileID: "file-1", UploaderID: 100, Status: "active"}
	repo.files["file-2"] = &File{FileID: "file-2", UploaderID: 100, Status: "active"}
	repo.files["file-3"] = &File{FileID: "file-3", UploaderID: 200, Status: "active"}
	repo.files["file-4"] = &File{FileID: "file-4", UploaderID: 100, Status: "deleted"}

	tests := []struct {
		name        string
		userID      int64
		limit       int32
		offset      int32
		expectCount int
	}{
		{
			name:        "list user 100 files",
			userID:      100,
			limit:       10,
			offset:      0,
			expectCount: 2,
		},
		{
			name:        "list user 200 files",
			userID:      200,
			limit:       10,
			offset:      0,
			expectCount: 1,
		},
		{
			name:        "list user with no files",
			userID:      999,
			limit:       10,
			offset:      0,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := service.ListUserFiles(context.Background(), tt.userID, tt.limit, tt.offset)

			require.NoError(t, err)
			assert.Len(t, files, tt.expectCount)
		})
	}
}
