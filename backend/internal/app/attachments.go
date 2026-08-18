package app

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/TelecomDep/ejournal_backend/internal/db"
)

const (
	maxAttachmentSize  = int64(50 * 1024 * 1024)
	maxAttachmentQuota = int64(250 * 1024 * 1024)
)

var allowedAttachmentExtensions = map[string]bool{
	".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".webp": true,
	".docx": true, ".xlsx": true, ".pptx": true, ".zip": true, ".rar": true,
}

func attachmentMIMEAllowed(ext, detected string) bool {
	detected = strings.ToLower(strings.TrimSpace(strings.Split(detected, ";")[0]))
	switch ext {
	case ".pdf":
		return detected == "application/pdf"
	case ".png":
		return detected == "image/png"
	case ".jpg", ".jpeg":
		return detected == "image/jpeg"
	case ".webp":
		return detected == "image/webp"
	case ".docx", ".xlsx", ".pptx", ".zip":
		return detected == "application/zip" || detected == "application/octet-stream"
	case ".rar":
		return detected == "application/vnd.rar" || detected == "application/x-rar-compressed" || detected == "application/octet-stream"
	default:
		return false
	}
}

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
	if user.Role != RoleTeacher && user.Role != RoleAdmin {
		return Response{OK: false, Error: "forbidden: teacher or admin role required"}
	}

	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "" || filename == "." || strings.ContainsAny(filename, "\r\n") {
		return Response{OK: false, Error: "filename is required"}
	}
	if len(filename) > 255 {
		return Response{OK: false, Error: "filename is too long"}
	}

	if size <= 0 || size > maxAttachmentSize {
		return Response{OK: false, Error: "file size must be between 1 byte and 50 MiB"}
	}

	data, err := io.ReadAll(io.LimitReader(r, maxAttachmentSize+1))
	if err != nil {
		return Response{OK: false, Error: "failed to read uploaded attachment"}
	}
	if int64(len(data)) > maxAttachmentSize || int64(len(data)) != size {
		return Response{OK: false, Error: "uploaded file size does not match request"}
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedAttachmentExtensions[ext] {
		return Response{OK: false, Error: "attachment type is not allowed"}
	}
	detectedMIME := http.DetectContentType(data)
	if !attachmentMIMEAllowed(ext, detectedMIME) {
		return Response{OK: false, Error: "attachment content does not match its extension"}
	}
	mimeType = strings.Split(detectedMIME, ";")[0]

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
	usedBytes, err := s.store.Users.AttachmentBytesByOwner(ctx, user.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to check attachment quota"}
	}
	if usedBytes+int64(len(data)) > maxAttachmentQuota {
		return Response{OK: false, Error: "attachment quota exceeded"}
	}

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
func (s *Service) GetAttachmentByID(sessionToken string, id int64) (db.Attachment, bool, error) {
	if id <= 0 {
		return db.Attachment{}, false, errors.New("invalid attachment id")
	}
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return db.Attachment{}, false, err
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	return s.store.Users.GetAttachmentByIDForOwner(ctx, id, user.ID)
}
