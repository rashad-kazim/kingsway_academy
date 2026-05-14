package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
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
	ListFilesPage(ctx context.Context, branchID string, ownerType string, ownerID string, purpose domain.FilePurpose, page domain.PageRequest) ([]domain.FileObject, int, error)
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

const maxUploadContentBytes int64 = 15 << 20

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
	Policy            domain.FilePolicy   `json:"policy"`
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
	Policy           domain.FilePolicy
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
	policy, category, err := normalizeUploadPolicy(input.Policy, input.Category, input.Purpose)
	if err != nil {
		return domain.FileObject{}, err
	}
	input.Policy = policy
	input.Category = category
	if input.Category == domain.FileCategorySpecial && actor.Role == domain.RoleTeacher {
		return domain.FileObject{}, domain.ErrForbidden
	}

	file, err := s.store.RegisterFile(ctx, domain.FileObject{
		BranchID:          input.BranchID,
		UploaderUserID:    actor.UserID,
		OwnerType:         strings.TrimSpace(input.OwnerType),
		OwnerID:           strings.TrimSpace(input.OwnerID),
		Category:          input.Category,
		Policy:            input.Policy,
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
	if input.Content == nil || input.Size < 0 || input.Size > maxUploadContentBytes {
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
	policy, category, err := normalizeUploadPolicy(input.Policy, input.Category, input.Purpose)
	if err != nil {
		return domain.FileObject{}, err
	}
	input.Policy = policy
	input.Category = category
	if input.Category == domain.FileCategorySpecial && actor.Role == domain.RoleTeacher {
		return domain.FileObject{}, domain.ErrForbidden
	}

	bucket := s.bucketFor(input.Category)
	if bucket == "" {
		return domain.FileObject{}, domain.ErrInvalidInput
	}

	upload, err := prepareUploadContent(input)
	if err != nil {
		return domain.FileObject{}, err
	}
	key := storageKey(input.BranchID, input.Category, upload.filename)
	if err := s.storage.Put(ctx, bucket, key, bytes.NewReader(upload.content), upload.storedSize, upload.mimeType); err != nil {
		return domain.FileObject{}, err
	}

	file, err := s.store.RegisterFile(ctx, domain.FileObject{
		BranchID:          input.BranchID,
		UploaderUserID:    actor.UserID,
		OwnerType:         strings.TrimSpace(input.OwnerType),
		OwnerID:           strings.TrimSpace(input.OwnerID),
		Category:          input.Category,
		Policy:            input.Policy,
		Purpose:           input.Purpose,
		OriginalFilename:  upload.filename,
		MimeType:          upload.mimeType,
		OriginalSizeBytes: input.Size,
		StoredSizeBytes:   upload.storedSize,
		OriginalSHA256:    upload.originalSHA256,
		StorageBucket:     bucket,
		StorageKey:        key,
	})
	if err != nil {
		return domain.FileObject{}, err
	}
	s.publishFileEvents(ctx, file)

	return file, nil
}

type preparedUpload struct {
	content        []byte
	filename       string
	mimeType       string
	storedSize     int64
	originalSHA256 string
}

func prepareUploadContent(input UploadFileInput) (preparedUpload, error) {
	original, err := io.ReadAll(io.LimitReader(input.Content, maxUploadContentBytes+1))
	if err != nil {
		return preparedUpload{}, err
	}
	if int64(len(original)) > maxUploadContentBytes {
		return preparedUpload{}, domain.ErrInvalidInput
	}

	hash := sha256.Sum256(original)
	upload := preparedUpload{
		content:        original,
		filename:       strings.TrimSpace(input.OriginalFilename),
		mimeType:       normalizedUploadMIME(input.MimeType, original, input.OriginalFilename),
		storedSize:     int64(len(original)),
		originalSHA256: hex.EncodeToString(hash[:]),
	}

	if !isProfileImageUpload(input) {
		return upload, nil
	}
	if !isAllowedProfileImageType(upload.mimeType) {
		return preparedUpload{}, domain.ErrInvalidInput
	}

	optimized, err := optimizeProfileImage(original, upload.filename)
	if err != nil {
		return preparedUpload{}, domain.ErrInvalidInput
	}
	upload.content = optimized.content
	upload.filename = optimized.filename
	upload.mimeType = optimized.mimeType
	upload.storedSize = int64(len(optimized.content))

	return upload, nil
}

func isProfileImageUpload(input UploadFileInput) bool {
	return input.Policy == domain.FilePolicyStandardUI ||
		input.Category == domain.FileCategoryStandard &&
			input.Purpose == domain.FilePurposeProfile
}

func normalizeUploadPolicy(policy domain.FilePolicy, category domain.FileCategory, purpose domain.FilePurpose) (domain.FilePolicy, domain.FileCategory, error) {
	if policy == "" {
		if !category.IsValid() {
			return "", "", domain.ErrInvalidInput
		}
		return domain.InferFilePolicy(category, purpose), category, nil
	}
	if !policy.IsValid() {
		return "", "", domain.ErrInvalidInput
	}

	expectedCategory := policy.Category()
	if category != "" && category != expectedCategory {
		return "", "", domain.ErrInvalidInput
	}

	return policy, expectedCategory, nil
}

func isAllowedProfileImageType(mimeType string) bool {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func normalizedUploadMIME(mimeType string, content []byte, filename string) string {
	raw := strings.TrimSpace(strings.Split(mimeType, ";")[0])
	switch strings.ToLower(raw) {
	case "image/jpeg", "image/jpg", "image/pjpeg":
		return "image/jpeg"
	case "image/png", "image/x-png":
		return "image/png"
	case "image/webp":
		return "image/webp"
	case "", "application/octet-stream":
		// Some browsers/OS integrations omit image MIME. Fall through to sniffing.
	default:
		return raw
	}

	detected := strings.ToLower(strings.TrimSpace(http.DetectContentType(content)))
	switch detected {
	case "image/jpeg", "image/png", "image/webp":
		return detected
	}
	if isWebP(content) {
		return "image/webp"
	}

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return detected
	}
}

func isWebP(content []byte) bool {
	return len(content) >= 12 &&
		content[0] == 'R' &&
		content[1] == 'I' &&
		content[2] == 'F' &&
		content[3] == 'F' &&
		content[8] == 'W' &&
		content[9] == 'E' &&
		content[10] == 'B' &&
		content[11] == 'P'
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

func (s *Service) ListFilesPage(ctx context.Context, actor domain.Principal, branchID string, ownerType string, ownerID string, purpose domain.FilePurpose, page domain.PageRequest) ([]domain.FileObject, int, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if actor.IsOwner() && branchID == "" {
		return s.store.ListFilesPage(ctx, "", strings.TrimSpace(ownerType), strings.TrimSpace(ownerID), purpose, page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}

	return s.store.ListFilesPage(ctx, branchID, strings.TrimSpace(ownerType), strings.TrimSpace(ownerID), purpose, page)
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

func (s *Service) DeleteFile(ctx context.Context, actor domain.Principal, fileID string) (domain.FileObject, error) {
	if s.storage == nil {
		return domain.FileObject{}, domain.ErrInvalidInput
	}

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
	if err := s.storage.Delete(ctx, file.StorageBucket, file.StorageKey); err != nil {
		return domain.FileObject{}, err
	}

	deleted, err := s.store.MarkFileDeleted(ctx, file.ID, time.Now().UTC())
	if err != nil {
		return domain.FileObject{}, err
	}
	if s.events != nil {
		_ = s.events.Publish(ctx, "files.file.deleted", deleted)
	}

	return deleted, nil
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

func (s *Service) StartRetentionWorker(ctx context.Context, options RetentionWorkerOptions) <-chan struct{} {
	done := make(chan struct{})
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
		defer close(done)
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

	return done
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
	b.Grow(len(name))
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
