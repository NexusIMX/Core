package file

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	files           map[string]*File
	createFunc      func(ctx context.Context, file *File) error
	getByFileIDFunc func(ctx context.Context, fileID string) (*File, error)
	deleteFunc      func(ctx context.Context, fileID string) error
	listFunc        func(ctx context.Context, userID int64, limit, offset int32) ([]*File, error)
}

func newMockRepository() *MockRepository {
	return &MockRepository{
		files: make(map[string]*File),
	}
}

func (m *MockRepository) Create(ctx context.Context, file *File) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, file)
	}
	m.files[file.FileID] = file
	return nil
}

func (m *MockRepository) GetByFileID(ctx context.Context, fileID string) (*File, error) {
	if m.getByFileIDFunc != nil {
		return m.getByFileIDFunc(ctx, fileID)
	}
	file, ok := m.files[fileID]
	if !ok {
		return nil, errors.New("file not found")
	}
	return file, nil
}

func (m *MockRepository) Delete(ctx context.Context, fileID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, fileID)
	}
	delete(m.files, fileID)
	return nil
}

func (m *MockRepository) ListByUploader(ctx context.Context, userID int64, limit, offset int32) ([]*File, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, userID, limit, offset)
	}
	var files []*File
	for _, file := range m.files {
		if file.UploaderID == userID {
			files = append(files, file)
		}
	}
	return files, nil
}

// MockS3Client is a mock implementation of S3 client
type MockS3Client struct {
	uploadFunc         func(ctx context.Context, key string, body io.Reader, contentType string) error
	downloadFunc       func(ctx context.Context, key string) (io.ReadCloser, error)
	deleteFunc         func(ctx context.Context, key string) error
	getPresignedURLFunc func(ctx context.Context, key string) (string, error)
	storage            map[string][]byte
}

func newMockS3Client() *MockS3Client {
	return &MockS3Client{
		storage: make(map[string][]byte),
	}
}

func (m *MockS3Client) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
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

func (m *MockS3Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, key)
	}
	data, ok := m.storage[key]
	if !ok {
		return nil, errors.New("file not found in S3")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *MockS3Client) Delete(ctx context.Context, key string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, key)
	}
	delete(m.storage, key)
	return nil
}

func (m *MockS3Client) GetPresignedURL(ctx context.Context, key string) (string, error) {
	if m.getPresignedURLFunc != nil {
		return m.getPresignedURLFunc(ctx, key)
	}
	return "https://s3.example.com/" + key, nil
}

func TestNewService(t *testing.T) {
	repo := newMockRepository()
	s3Client := newMockS3Client()
	maxSize := int64(524288000) // 500MB

	service := NewService(repo, s3Client, maxSize)

	assert.NotNil(t, service)
	assert.NotNil(t, service.repo)
	assert.NotNil(t, service.s3Client)
	assert.Equal(t, maxSize, service.maxSize)
}

func TestService_UploadFile(t *testing.T) {
	tests := []struct {
		name        string
		uploaderID  int64
		fileName    string
		fileSize    int64
		contentType string
		fileData    []byte
		maxSize     int64
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "successful upload",
			uploaderID:  100,
			fileName:    "test.jpg",
			fileSize:    1024,
			contentType: "image/jpeg",
			fileData:    []byte("fake image data"),
			maxSize:     524288000,
			wantErr:     false,
		},
		{
			name:        "file too large",
			uploaderID:  100,
			fileName:    "large.mp4",
			fileSize:    524288001,
			contentType: "video/mp4",
			fileData:    []byte("large file"),
			maxSize:     524288000,
			wantErr:     true,
			errMsg:      "file size exceeds maximum",
		},
		{
			name:        "small text file",
			uploaderID:  200,
			fileName:    "document.txt",
			fileSize:    100,
			contentType: "text/plain",
			fileData:    []byte("Hello, World!"),
			maxSize:     524288000,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			s3Client := newMockS3Client()
			service := NewService(repo, s3Client, tt.maxSize)

			fileReader := bytes.NewReader(tt.fileData)
			file, err := service.UploadFile(
				context.Background(),
				tt.uploaderID,
				tt.fileName,
				tt.fileSize,
				tt.contentType,
				fileReader,
			)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
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
	repo := newMockRepository()
	s3Client := newMockS3Client()
	service := NewService(repo, s3Client, 524288000)

	// Setup: Create a file
	testFile := &File{
		FileID:      "test-file-id",
		UploaderID:  100,
		FileName:    "test.jpg",
		FileSize:    1024,
		ContentType: "image/jpeg",
		StorageKey:  "uploads/2024/01/01/test.jpg",
		Status:      "active",
		CreatedAt:   time.Now(),
	}
	repo.files[testFile.FileID] = testFile

	tests := []struct {
		name    string
		fileID  string
		wantErr bool
	}{
		{
			name:    "file found",
			fileID:  "test-file-id",
			wantErr: false,
		},
		{
			name:    "file not found",
			fileID:  "non-existent",
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
	repo := newMockRepository()
	s3Client := newMockS3Client()
	service := NewService(repo, s3Client, 524288000)

	// Setup: Create file and upload to S3
	storageKey := "uploads/2024/01/01/test.jpg"
	testData := []byte("test file content")
	s3Client.storage[storageKey] = testData

	testFile := &File{
		FileID:      "test-file-id",
		UploaderID:  100,
		FileName:    "test.jpg",
		FileSize:    int64(len(testData)),
		ContentType: "image/jpeg",
		StorageKey:  storageKey,
		Status:      "active",
	}
	repo.files[testFile.FileID] = testFile

	body, file, err := service.DownloadFile(context.Background(), "test-file-id")

	require.NoError(t, err)
	assert.NotNil(t, body)
	assert.NotNil(t, file)
	assert.Equal(t, "test-file-id", file.FileID)

	// Read and verify content
	content, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, testData, content)

	body.Close()
}

func TestService_GetDownloadURL(t *testing.T) {
	repo := newMockRepository()
	s3Client := newMockS3Client()
	service := NewService(repo, s3Client, 524288000)

	// Setup: Create file
	testFile := &File{
		FileID:      "test-file-id",
		UploaderID:  100,
		FileName:    "test.jpg",
		StorageKey:  "uploads/2024/01/01/test.jpg",
		Status:      "active",
	}
	repo.files[testFile.FileID] = testFile

	url, err := service.GetDownloadURL(context.Background(), "test-file-id")

	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, "s3.example.com")
}

func TestService_DeleteFile(t *testing.T) {
	tests := []struct {
		name       string
		fileID     string
		userID     int64
		setupFile  *File
		wantErr    bool
		errMsg     string
	}{
		{
			name:   "successful delete by uploader",
			fileID: "file-1",
			userID: 100,
			setupFile: &File{
				FileID:      "file-1",
				UploaderID:  100,
				FileName:    "test.jpg",
				StorageKey:  "uploads/test.jpg",
				Status:      "active",
			},
			wantErr: false,
		},
		{
			name:   "delete permission denied",
			fileID: "file-2",
			userID: 200,
			setupFile: &File{
				FileID:      "file-2",
				UploaderID:  100,
				FileName:    "test.jpg",
				StorageKey:  "uploads/test.jpg",
				Status:      "active",
			},
			wantErr: true,
			errMsg:  "permission denied",
		},
		{
			name:    "file not found",
			fileID:  "non-existent",
			userID:  100,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			s3Client := newMockS3Client()
			service := NewService(repo, s3Client, 524288000)

			if tt.setupFile != nil {
				repo.files[tt.setupFile.FileID] = tt.setupFile
				s3Client.storage[tt.setupFile.StorageKey] = []byte("test data")
			}

			err := service.DeleteFile(context.Background(), tt.fileID, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				// Verify file was deleted from repository
				_, exists := repo.files[tt.fileID]
				assert.False(t, exists)
			}
		})
	}
}

func TestService_ListUserFiles(t *testing.T) {
	repo := newMockRepository()
	s3Client := newMockS3Client()
	service := NewService(repo, s3Client, 524288000)

	// Setup: Create multiple files for different users
	files := []*File{
		{FileID: "file-1", UploaderID: 100, FileName: "file1.jpg"},
		{FileID: "file-2", UploaderID: 100, FileName: "file2.jpg"},
		{FileID: "file-3", UploaderID: 200, FileName: "file3.jpg"},
		{FileID: "file-4", UploaderID: 100, FileName: "file4.jpg"},
	}

	for _, file := range files {
		repo.files[file.FileID] = file
	}

	tests := []struct {
		name          string
		userID        int64
		limit         int32
		offset        int32
		expectedCount int
	}{
		{
			name:          "get user 100 files",
			userID:        100,
			limit:         10,
			offset:        0,
			expectedCount: 3,
		},
		{
			name:          "get user 200 files",
			userID:        200,
			limit:         10,
			offset:        0,
			expectedCount: 1,
		},
		{
			name:          "user with no files",
			userID:        300,
			limit:         10,
			offset:        0,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := service.ListUserFiles(context.Background(), tt.userID, tt.limit, tt.offset)

			require.NoError(t, err)
			assert.Len(t, files, tt.expectedCount)

			// Verify all files belong to the user
			for _, file := range files {
				assert.Equal(t, tt.userID, file.UploaderID)
			}
		})
	}
}

func TestService_UploadFile_S3Failure(t *testing.T) {
	repo := newMockRepository()
	s3Client := newMockS3Client()
	service := NewService(repo, s3Client, 524288000)

	// Mock S3 upload failure
	s3Client.uploadFunc = func(ctx context.Context, key string, body io.Reader, contentType string) error {
		return errors.New("S3 upload failed")
	}

	fileReader := bytes.NewReader([]byte("test data"))
	file, err := service.UploadFile(
		context.Background(),
		100,
		"test.jpg",
		100,
		"image/jpeg",
		fileReader,
	)

	assert.Error(t, err)
	assert.Nil(t, file)
	assert.Contains(t, err.Error(), "failed to upload to S3")
}

func TestService_UploadFile_DatabaseFailure(t *testing.T) {
	repo := newMockRepository()
	s3Client := newMockS3Client()
	service := NewService(repo, s3Client, 524288000)

	// Mock database create failure
	repo.createFunc = func(ctx context.Context, file *File) error {
		return errors.New("database error")
	}

	fileReader := bytes.NewReader([]byte("test data"))
	file, err := service.UploadFile(
		context.Background(),
		100,
		"test.jpg",
		100,
		"image/jpeg",
		fileReader,
	)

	assert.Error(t, err)
	assert.Nil(t, file)
	assert.Contains(t, err.Error(), "failed to create file record")
}
