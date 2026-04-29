package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

type Store interface {
	RegisterFile(ctx context.Context, file domain.FileObject) (domain.FileObject, error)
	GetFile(ctx context.Context, id string) (domain.FileObject, error)
	ListFiles(ctx context.Context, branchID string) ([]domain.FileObject, error)
	ListExpiredFiles(ctx context.Context, now time.Time, limit int) ([]domain.FileObject, error)
	MarkFileDeleted(ctx context.Context, id string, deletedAt time.Time) (domain.FileObject, error)
}

type ObjectStorage interface {
	Put(ctx context.Context, bucket string, key string, content io.Reader, size int64, contentType string) error
	PresignedGet(ctx context.Context, bucket string, key string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, bucket string, key string) error
}

type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

type Service struct {
	store          Store
	storage        ObjectStorage
	events         EventPublisher
	standardBucket string
	specialBucket  string
}

type Option func(*Service)

func WithObjectStorage(storage ObjectStorage, standardBucket string, specialBucket string) Option {
	return func(s *Service) {
		s.storage = storage
		s.standardBucket = standardBucket
		s.specialBucket = specialBucket
	}
}

func WithEventPublisher(events EventPublisher) Option {
	return func(s *Service) {
		s.events = events
	}
}

type RegisterFileInput struct {
	BranchID          string              `json:"branch_id"`
	OwnerType         string              `json:"owner_type"`
	OwnerID           string              `json:"owner_id"`
	Category          domain.FileCategory `json:"category"`
	Purpose           domain.FilePurpose  `json:"purpose"`
	OriginalFilename  string              `json:"original_filename"`
	MimeType          string              `json:"mime_type"`
	OriginalSizeBytes int64               `json:"original_size_bytes"`
	StoredSizeBytes   int64               `json:"stored_size_bytes"`
	OriginalSHA256    string              `json:"original_sha256"`
	StorageBucket     string              `json:"storage_bucket"`
	StorageKey        string              `json:"storage_key"`
}

type UploadFileInput struct {
	BranchID         string
	OwnerType        string
	OwnerID          string
	Category         domain.FileCategory
	Purpose          domain.FilePurpose
	OriginalFilename string
	MimeType         string
	Size             int64
	Content          io.Reader
}

type DownloadURLResult struct {
	URL       string            `json:"url"`
	ExpiresAt time.Time         `json:"expires_at"`
	File      domain.FileObject `json:"file"`
}

type RetentionCleanupResult struct {
	DeletedCount int                 `json:"deleted_count"`
	Files        []domain.FileObject `json:"files"`
}

type RetentionWorkerOptions struct {
	Interval time.Duration
	Limit    int
	Logger   *zap.Logger
}

func NewService(store Store, options ...Option) *Service {
	service := &Service{store: store}
	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) RegisterFile(ctx context.Context, actor domain.Principal, input RegisterFileInput) (domain.FileObject, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist, domain.RoleTeacher); err != nil {
		return domain.FileObject{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.FileObject{}, err
	}
	if input.Purpose == domain.FilePurposeExamWriting && actor.Role != domain.RoleTeacher && actor.Role != domain.RoleOwner {
		return domain.FileObject{}, domain.ErrForbidden
	}
	if input.Category == domain.FileCategorySpecial && actor.Role == domain.RoleTeacher {
		return domain.FileObject{}, domain.ErrForbidden
	}

	file, err := s.store.RegisterFile(ctx, domain.FileObject{
		BranchID:          input.BranchID,
		UploaderUserID:    actor.UserID,
		OwnerType:         strings.TrimSpace(input.OwnerType),
		OwnerID:           strings.TrimSpace(input.OwnerID),
		Category:          input.Category,
		Purpose:           input.Purpose,
		OriginalFilename:  strings.TrimSpace(input.OriginalFilename),
		MimeType:          strings.TrimSpace(input.MimeType),
		OriginalSizeBytes: input.OriginalSizeBytes,
		StoredSizeBytes:   input.StoredSizeBytes,
		OriginalSHA256:    strings.TrimSpace(input.OriginalSHA256),
		StorageBucket:     strings.TrimSpace(input.StorageBucket),
		StorageKey:        strings.TrimSpace(input.StorageKey),
	})
	if err != nil {
		return domain.FileObject{}, err
	}
	s.publishFileEvents(ctx, file)

	return file, nil
}

func (s *Service) UploadFile(ctx context.Context, actor domain.Principal, input UploadFileInput) (domain.FileObject, error) {
	if s.storage == nil {
		return domain.FileObject{}, domain.ErrInvalidInput
	}
	if input.Content == nil || input.Size < 0 {
		return domain.FileObject{}, domain.ErrInvalidInput
	}
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist, domain.RoleTeacher); err != nil {
		return domain.FileObject{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.FileObject{}, err
	}
	if input.Purpose == domain.FilePurposeExamWriting && actor.Role != domain.RoleTeacher && actor.Role != domain.RoleOwner {
		return domain.FileObject{}, domain.ErrForbidden
	}
	if input.Category == domain.FileCategorySpecial && actor.Role == domain.RoleTeacher {
		return domain.FileObject{}, domain.ErrForbidden
	}

	bucket := s.bucketFor(input.Category)
	if bucket == "" {
		return domain.FileObject{}, domain.ErrInvalidInput
	}

	hash := sha256.New()
	key := storageKey(input.BranchID, input.Category, input.OriginalFilename)
	content := io.TeeReader(input.Content, hash)
	if err := s.storage.Put(ctx, bucket, key, content, input.Size, input.MimeType); err != nil {
		return domain.FileObject{}, err
	}

	file, err := s.store.RegisterFile(ctx, domain.FileObject{
		BranchID:          input.BranchID,
		UploaderUserID:    actor.UserID,
		OwnerType:         strings.TrimSpace(input.OwnerType),
		OwnerID:           strings.TrimSpace(input.OwnerID),
		Category:          input.Category,
		Purpose:           input.Purpose,
		OriginalFilename:  strings.TrimSpace(input.OriginalFilename),
		MimeType:          strings.TrimSpace(input.MimeType),
		OriginalSizeBytes: input.Size,
		StoredSizeBytes:   input.Size,
		OriginalSHA256:    hex.EncodeToString(hash.Sum(nil)),
		StorageBucket:     bucket,
		StorageKey:        key,
	})
	if err != nil {
		return domain.FileObject{}, err
	}
	s.publishFileEvents(ctx, file)

	return file, nil
}

func (s *Service) ListFiles(ctx context.Context, actor domain.Principal, branchID string) ([]domain.FileObject, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.store.ListFiles(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	return s.store.ListFiles(ctx, branchID)
}

func (s *Service) GetFile(ctx context.Context, actor domain.Principal, fileID string) (domain.FileObject, error) {
	file, err := s.store.GetFile(ctx, strings.TrimSpace(fileID))
	if err != nil {
		return domain.FileObject{}, err
	}
	if file.DeletedAt != nil {
		return domain.FileObject{}, domain.ErrNotFound
	}
	if err := auth.RequireBranch(actor, file.BranchID); err != nil {
		return domain.FileObject{}, err
	}

	return file, nil
}

func (s *Service) CreateDownloadURL(ctx context.Context, actor domain.Principal, fileID string) (DownloadURLResult, error) {
	if s.storage == nil {
		return DownloadURLResult{}, domain.ErrInvalidInput
	}

	file, err := s.store.GetFile(ctx, strings.TrimSpace(fileID))
	if err != nil {
		return DownloadURLResult{}, err
	}
	if file.DeletedAt != nil {
		return DownloadURLResult{}, domain.ErrNotFound
	}
	if err := auth.RequireBranch(actor, file.BranchID); err != nil {
		return DownloadURLResult{}, err
	}

	expiry := 15 * time.Minute
	url, err := s.storage.PresignedGet(ctx, file.StorageBucket, file.StorageKey, expiry)
	if err != nil {
		return DownloadURLResult{}, err
	}

	return DownloadURLResult{
		URL:       url,
		ExpiresAt: time.Now().UTC().Add(expiry),
		File:      file,
	}, nil
}

func (s *Service) CleanupExpiredFiles(ctx context.Context, actor domain.Principal, limit int) (RetentionCleanupResult, error) {
	if s.storage == nil {
		return RetentionCleanupResult{}, domain.ErrInvalidInput
	}
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return RetentionCleanupResult{}, err
	}

	return s.CleanupExpiredFilesSystem(ctx, limit)
}

func (s *Service) CleanupExpiredFilesSystem(ctx context.Context, limit int) (RetentionCleanupResult, error) {
	if s.storage == nil {
		return RetentionCleanupResult{}, domain.ErrInvalidInput
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	expired, err := s.store.ListExpiredFiles(ctx, time.Now().UTC(), limit)
	if err != nil {
		return RetentionCleanupResult{}, err
	}

	deleted := make([]domain.FileObject, 0, len(expired))
	for _, file := range expired {
		if err := s.storage.Delete(ctx, file.StorageBucket, file.StorageKey); err != nil {
			return RetentionCleanupResult{}, err
		}
		deletedFile, err := s.store.MarkFileDeleted(ctx, file.ID, time.Now().UTC())
		if err != nil {
			return RetentionCleanupResult{}, err
		}
		deleted = append(deleted, deletedFile)
		if s.events != nil {
			_ = s.events.Publish(ctx, "files.retention.deleted", deletedFile)
		}
	}

	return RetentionCleanupResult{DeletedCount: len(deleted), Files: deleted}, nil
}

func (s *Service) StartRetentionWorker(ctx context.Context, options RetentionWorkerOptions) {
	interval := options.Interval
	if interval <= 0 {
		interval = time.Hour
	}
	limit := options.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	log := options.Logger
	if log == nil {
		log = zap.NewNop()
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("retention worker stopped")
				return
			case <-ticker.C:
				result, err := s.CleanupExpiredFilesSystem(ctx, limit)
				if err != nil {
					log.Error("retention cleanup failed", zap.Error(err))
					continue
				}
				if result.DeletedCount > 0 {
					log.Info("retention cleanup deleted files", zap.Int("deleted_count", result.DeletedCount))
				}
			}
		}
	}()
}

func (s *Service) bucketFor(category domain.FileCategory) string {
	switch category {
	case domain.FileCategoryStandard:
		return s.standardBucket
	case domain.FileCategorySpecial:
		return s.specialBucket
	default:
		return ""
	}
}

func storageKey(branchID string, category domain.FileCategory, filename string) string {
	today := time.Now().UTC()
	return fmt.Sprintf(
		"%s/%s/%04d/%02d/%s-%s",
		branchID,
		category,
		today.Year(),
		int(today.Month()),
		uuid.NewString(),
		safeFilename(filename),
	)
}

func safeFilename(filename string) string {
	name := path.Base(strings.TrimSpace(filename))
	if name == "" || name == "." || name == "/" {
		return "upload.bin"
	}

	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}

	safe := b.String()
	if safe == "" {
		return "upload.bin"
	}

	return safe
}

func (s *Service) publishFileEvents(ctx context.Context, file domain.FileObject) {
	if s.events == nil {
		return
	}

	_ = s.events.Publish(ctx, "files.file.registered", file)
	if file.RetentionUntil != nil {
		_ = s.events.Publish(ctx, "files.retention.scheduled", map[string]any{
			"file_id":         file.ID,
			"branch_id":       file.BranchID,
			"retention_until": file.RetentionUntil,
		})
	}
}
