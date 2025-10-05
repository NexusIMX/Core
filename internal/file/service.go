package file

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// Service 文件服务
type Service struct {
	repo    FileRepository
	storage StorageClient
	maxSize int64
}

// NewService 创建文件服务
func NewService(repo FileRepository, storage StorageClient, maxSize int64) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		maxSize: maxSize,
	}
}

// UploadFile 上传文件
func (s *Service) UploadFile(ctx context.Context, uploaderID int64, fileName string, fileSize int64, contentType string, fileData io.Reader) (*File, error) {
	// 检查文件大小
	if fileSize > s.maxSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", s.maxSize)
	}

	// 生成文件 ID
	fileID := uuid.New().String()

	// 生成 S3 存储 key: uploads/{year}/{month}/{day}/{uuid}{ext}
	now := time.Now()
	ext := filepath.Ext(fileName)
	storageKey := fmt.Sprintf("uploads/%d/%02d/%02d/%s%s",
		now.Year(), now.Month(), now.Day(), fileID, ext)

	// 上传到对象存储
	if err := s.storage.Upload(ctx, storageKey, fileData, contentType); err != nil {
		return nil, fmt.Errorf("failed to upload to storage: %w", err)
	}

	// 创建文件记录
	file := &File{
		FileID:      fileID,
		UploaderID:  uploaderID,
		FileName:    fileName,
		FileSize:    fileSize,
		ContentType: contentType,
		StorageKey:  storageKey,
		Status:      "active",
	}

	if err := s.repo.Create(ctx, file); err != nil {
		// 如果数据库插入失败，尝试删除存储文件
		_ = s.storage.Delete(ctx, storageKey)
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	return file, nil
}

// GetFile 获取文件信息
func (s *Service) GetFile(ctx context.Context, fileID string) (*File, error) {
	return s.repo.GetByFileID(ctx, fileID)
}

// DownloadFile 下载文件
func (s *Service) DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, *File, error) {
	// 获取文件信息
	file, err := s.repo.GetByFileID(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}

	// 从对象存储下载
	body, err := s.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download from storage: %w", err)
	}

	return body, file, nil
}

// GetDownloadURL 获取下载链接（预签名 URL）
func (s *Service) GetDownloadURL(ctx context.Context, fileID string) (string, error) {
	// 获取文件信息
	file, err := s.repo.GetByFileID(ctx, fileID)
	if err != nil {
		return "", err
	}

	// 生成预签名 URL
	url, err := s.storage.GetPresignedURL(ctx, file.StorageKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return url, nil
}

// DeleteFile 删除文件
func (s *Service) DeleteFile(ctx context.Context, fileID string, userID int64) error {
	// 获取文件信息
	file, err := s.repo.GetByFileID(ctx, fileID)
	if err != nil {
		return err
	}

	// 检查权限（只有上传者可以删除）
	if file.UploaderID != userID {
		return fmt.Errorf("permission denied: only uploader can delete this file")
	}

	// 软删除数据库记录
	if err := s.repo.Delete(ctx, fileID); err != nil {
		return err
	}

	// 从对象存储删除（异步，失败也不影响）
	go func() {
		_ = s.storage.Delete(context.Background(), file.StorageKey)
	}()

	return nil
}

// ListUserFiles 获取用户上传的文件列表
func (s *Service) ListUserFiles(ctx context.Context, userID int64, limit, offset int32) ([]*File, error) {
	return s.repo.ListByUploader(ctx, userID, limit, offset)
}
