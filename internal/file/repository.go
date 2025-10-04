package file

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// File 文件元数据
type File struct {
	ID          int64     `json:"id"`
	FileID      string    `json:"file_id"`
	UploaderID  int64     `json:"uploader_id"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	StorageKey  string    `json:"storage_key"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// Repository 文件仓储
type Repository struct {
	db *sql.DB
}

// NewRepository 创建文件仓储
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建文件记录
func (r *Repository) Create(ctx context.Context, file *File) error {
	query := `
		INSERT INTO files (file_id, uploader_id, file_name, file_size, content_type, storage_key, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		file.FileID,
		file.UploaderID,
		file.FileName,
		file.FileSize,
		file.ContentType,
		file.StorageKey,
		file.Status,
		time.Now(),
	).Scan(&file.ID, &file.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取文件
func (r *Repository) GetByID(ctx context.Context, id int64) (*File, error) {
	query := `
		SELECT id, file_id, uploader_id, file_name, file_size, content_type, storage_key, status, created_at, deleted_at
		FROM files
		WHERE id = $1 AND deleted_at IS NULL
	`

	file := &File{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&file.ID,
		&file.FileID,
		&file.UploaderID,
		&file.FileName,
		&file.FileSize,
		&file.ContentType,
		&file.StorageKey,
		&file.Status,
		&file.CreatedAt,
		&file.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return file, nil
}

// GetByFileID 根据 file_id 获取文件
func (r *Repository) GetByFileID(ctx context.Context, fileID string) (*File, error) {
	query := `
		SELECT id, file_id, uploader_id, file_name, file_size, content_type, storage_key, status, created_at, deleted_at
		FROM files
		WHERE file_id = $1 AND deleted_at IS NULL
	`

	file := &File{}
	err := r.db.QueryRowContext(ctx, query, fileID).Scan(
		&file.ID,
		&file.FileID,
		&file.UploaderID,
		&file.FileName,
		&file.FileSize,
		&file.ContentType,
		&file.StorageKey,
		&file.Status,
		&file.CreatedAt,
		&file.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return file, nil
}

// UpdateStatus 更新文件状态
func (r *Repository) UpdateStatus(ctx context.Context, fileID string, status string) error {
	query := `
		UPDATE files
		SET status = $1
		WHERE file_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, status, fileID)
	if err != nil {
		return fmt.Errorf("failed to update file status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("file not found")
	}

	return nil
}

// Delete 软删除文件
func (r *Repository) Delete(ctx context.Context, fileID string) error {
	query := `
		UPDATE files
		SET deleted_at = $1
		WHERE file_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), fileID)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("file not found")
	}

	return nil
}

// ListByUploader 获取用户上传的文件列表
func (r *Repository) ListByUploader(ctx context.Context, uploaderID int64, limit, offset int32) ([]*File, error) {
	query := `
		SELECT id, file_id, uploader_id, file_name, file_size, content_type, storage_key, status, created_at, deleted_at
		FROM files
		WHERE uploader_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, uploaderID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		file := &File{}
		err := rows.Scan(
			&file.ID,
			&file.FileID,
			&file.UploaderID,
			&file.FileName,
			&file.FileSize,
			&file.ContentType,
			&file.StorageKey,
			&file.Status,
			&file.CreatedAt,
			&file.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, file)
	}

	return files, nil
}
