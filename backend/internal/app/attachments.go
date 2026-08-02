package app

import (
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/TelecomDep/ejournal_backend/internal/db"
)

type AttachmentUploadRequest struct {
	Filename string `json:"filename"`
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type"`
}

// UploadAttachment accepts teacher file uploads (PDF, images, ZIP, RAR, DOCX, etc.)
// and saves them to the database (or S3/disk for larger files).
func (s *Service) UploadAttachment(sessionToken string, filename string, r io.Reader, size int64, mimeType string) Response {
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	filename = strings.TrimSpace(filename)
	if filename == "" {
		return Response{OK: false, Error: "filename is required"}
	}

	if size <= 0 || size > 50*1024*1024 { // 50MB max file size limit for attachments
		return Response{OK: false, Error: "file size must be between 1 byte and 50 MiB"}
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return Response{OK: false, Error: "failed to read uploaded attachment"}
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if mimeType == "" {
		switch ext {
		case ".pdf":
			mimeType = "application/pdf"
		case ".zip":
			mimeType = "application/zip"
		case ".rar":
			mimeType = "application/x-rar-compressed"
		case ".png":
			mimeType = "image/png"
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		default:
			mimeType = "application/octet-stream"
		}
	}

	att := db.Attachment{
		OwnerID:     user.ID,
		Filename:    filename,
		FileSize:    size,
		MimeType:    mimeType,
		StorageType: "db",
		Data:        data,
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	id, err := s.store.Users.SaveAttachment(ctx, att)
	if err != nil {
		return Response{OK: false, Error: "failed to save attachment"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"id":        id,
			"filename":  filename,
			"file_size": size,
			"mime_type": mimeType,
		},
	}
}

// GetAttachmentByID retrieves an attachment by ID.
func (s *Service) GetAttachmentByID(id int64) (db.Attachment, bool, error) {
	if id <= 0 {
		return db.Attachment{}, false, errors.New("invalid attachment id")
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	return s.store.Users.GetAttachmentByID(ctx, id)
}
