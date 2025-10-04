package file

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/dollarkillerx/im-system/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler HTTP 处理器
type Handler struct {
	service *Service
}

// NewHandler 创建处理器
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// UploadFile 上传文件
// POST /v1/files
func (h *Handler) UploadFile(c *gin.Context) {
	// 从 context 获取用户 ID（由中间件注入）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// 解析 multipart form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		logger.Log.Error("Failed to open uploaded file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process file"})
		return
	}
	defer src.Close()

	// 上传文件
	fileRecord, err := h.service.UploadFile(
		c.Request.Context(),
		userID.(int64),
		file.Filename,
		file.Size,
		file.Header.Get("Content-Type"),
		src,
	)
	if err != nil {
		logger.Log.Error("Failed to upload file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_id":      fileRecord.FileID,
		"file_name":    fileRecord.FileName,
		"file_size":    fileRecord.FileSize,
		"content_type": fileRecord.ContentType,
		"created_at":   fileRecord.CreatedAt.Unix(),
	})
}

// GetFileInfo 获取文件信息
// GET /v1/files/:id
func (h *Handler) GetFileInfo(c *gin.Context) {
	fileID := c.Param("id")

	file, err := h.service.GetFile(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_id":      file.FileID,
		"file_name":    file.FileName,
		"file_size":    file.FileSize,
		"content_type": file.ContentType,
		"uploader_id":  file.UploaderID,
		"status":       file.Status,
		"created_at":   file.CreatedAt.Unix(),
	})
}

// DownloadFile 下载文件
// GET /v1/files/:id/download
func (h *Handler) DownloadFile(c *gin.Context) {
	fileID := c.Param("id")

	body, file, err := h.service.DownloadFile(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	defer body.Close()

	// 设置响应头
	c.Header("Content-Type", file.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.FileName))
	c.Header("Content-Length", strconv.FormatInt(file.FileSize, 10))

	// 流式传输文件
	_, err = io.Copy(c.Writer, body)
	if err != nil {
		logger.Log.Error("Failed to stream file", zap.Error(err))
	}
}

// GetDownloadURL 获取下载链接
// GET /v1/files/:id/url
func (h *Handler) GetDownloadURL(c *gin.Context) {
	fileID := c.Param("id")

	url, err := h.service.GetDownloadURL(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

// DeleteFile 删除文件
// DELETE /v1/files/:id
func (h *Handler) DeleteFile(c *gin.Context) {
	fileID := c.Param("id")

	// 从 context 获取用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	err := h.service.DeleteFile(c.Request.Context(), fileID, userID.(int64))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file deleted successfully"})
}

// ListUserFiles 获取用户文件列表
// GET /v1/files
func (h *Handler) ListUserFiles(c *gin.Context) {
	// 从 context 获取用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// 解析分页参数
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 32)
	offset, _ := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 32)

	files, err := h.service.ListUserFiles(c.Request.Context(), userID.(int64), int32(limit), int32(offset))
	if err != nil {
		logger.Log.Error("Failed to list files", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list files"})
		return
	}

	var fileList []gin.H
	for _, file := range files {
		fileList = append(fileList, gin.H{
			"file_id":      file.FileID,
			"file_name":    file.FileName,
			"file_size":    file.FileSize,
			"content_type": file.ContentType,
			"status":       file.Status,
			"created_at":   file.CreatedAt.Unix(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"files": fileList,
		"total": len(fileList),
	})
}
