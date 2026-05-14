package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"testing"
	"time"

	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/store"
)

type fakeFileStore struct {
	expired []domain.FileObject
	file    domain.FileObject
	marked  []string
}

func (s *fakeFileStore) RegisterFile(context.Context, domain.FileObject) (domain.FileObject, error) {
	return domain.FileObject{}, nil
}

func (s *fakeFileStore) GetFile(_ context.Context, id string) (domain.FileObject, error) {
	if s.file.ID == id {
		return s.file, nil
	}

	return domain.FileObject{}, domain.ErrNotFound
}

func (s *fakeFileStore) ListFiles(context.Context, string) ([]domain.FileObject, error) {
	return nil, nil
}

func (s *fakeFileStore) ListFilesPage(context.Context, string, string, string, domain.FilePurpose, domain.PageRequest) ([]domain.FileObject, int, error) {
	return nil, 0, nil
}

func (s *fakeFileStore) ListExpiredFiles(_ context.Context, _ time.Time, limit int) ([]domain.FileObject, error) {
	if limit > 0 && len(s.expired) > limit {
		return s.expired[:limit], nil
	}

	return s.expired, nil
}

func (s *fakeFileStore) MarkFileDeleted(_ context.Context, id string, deletedAt time.Time) (domain.FileObject, error) {
	s.marked = append(s.marked, id)
	if s.file.ID == id {
		s.file.DeletedAt = &deletedAt
		return s.file, nil
	}
	for _, file := range s.expired {
		if file.ID == id {
			file.DeletedAt = &deletedAt
			return file, nil
		}
	}

	return domain.FileObject{}, domain.ErrNotFound
}

func TestDeleteFileRemovesObjectAndMarksDeleted(t *testing.T) {
	t.Parallel()

	repo := &fakeFileStore{
		file: domain.FileObject{
			ID:            "file-1",
			BranchID:      "branch-1",
			StorageBucket: "standard",
			StorageKey:    "branch-1/profile.jpg",
		},
	}
	storage := &fakeObjectStorage{}
	service := NewService(repo, WithObjectStorage(storage, "standard", "special"))

	deleted, err := service.DeleteFile(context.Background(), domain.Principal{UserID: "owner", Role: domain.RoleOwner}, "file-1")
	if err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}

	if deleted.DeletedAt == nil {
		t.Fatalf("expected deleted_at to be set: %#v", deleted)
	}
	if len(storage.deleted) != 1 || storage.deleted[0] != "standard/branch-1/profile.jpg" {
		t.Fatalf("unexpected deleted object list: %#v", storage.deleted)
	}
	if len(repo.marked) != 1 || repo.marked[0] != "file-1" {
		t.Fatalf("unexpected marked files: %#v", repo.marked)
	}
}

type fakeObjectStorage struct {
	deleted     []string
	putBucket   string
	putKey      string
	putContent  []byte
	contentType string
}

func (s *fakeObjectStorage) Put(_ context.Context, bucket string, key string, content io.Reader, _ int64, contentType string) error {
	body, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	s.putBucket = bucket
	s.putKey = key
	s.putContent = body
	s.contentType = contentType
	return nil
}

func (s *fakeObjectStorage) PresignedGet(context.Context, string, string, time.Duration) (string, error) {
	return "", nil
}

func (s *fakeObjectStorage) Delete(_ context.Context, bucket string, key string) error {
	s.deleted = append(s.deleted, bucket+"/"+key)
	return nil
}

func TestCleanupExpiredFilesSystemDeletesExpiredObjects(t *testing.T) {
	t.Parallel()

	store := &fakeFileStore{expired: []domain.FileObject{
		{ID: "file-1", StorageBucket: "standard", StorageKey: "branch/file-1.txt"},
		{ID: "file-2", StorageBucket: "standard", StorageKey: "branch/file-2.txt"},
	}}
	storage := &fakeObjectStorage{}
	service := NewService(store, WithObjectStorage(storage, "standard", "special"))

	result, err := service.CleanupExpiredFilesSystem(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected cleanup error: %v", err)
	}

	if result.DeletedCount != 1 {
		t.Fatalf("expected one deleted file, got %d", result.DeletedCount)
	}
	if len(storage.deleted) != 1 || storage.deleted[0] != "standard/branch/file-1.txt" {
		t.Fatalf("unexpected deleted objects: %#v", storage.deleted)
	}
	if len(store.marked) != 1 || store.marked[0] != "file-1" {
		t.Fatalf("unexpected marked files: %#v", store.marked)
	}
}

func TestCleanupExpiredFilesRequiresOwnerForHTTPPath(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeFileStore{}, WithObjectStorage(&fakeObjectStorage{}, "standard", "special"))
	_, err := service.CleanupExpiredFiles(context.Background(), domain.Principal{Role: domain.RoleReceptionist}, 100)
	if err == nil {
		t.Fatal("expected forbidden error")
	}
}

func TestRegisterFileRoleRules(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	service := NewService(repo)

	branch, err := repo.CreateBranch(ctx, domain.Branch{Name: "Main", Slug: "file-rbac-main"})
	if err != nil {
		t.Fatal(err)
	}
	teacherUser, err := repo.CreateUser(ctx, domain.User{BranchID: branch.ID, Role: domain.RoleTeacher, Email: "file-teacher@test.local", PasswordHash: "hash", FirstName: "Teacher", LastName: "User"})
	if err != nil {
		t.Fatal(err)
	}
	receptionistUser, err := repo.CreateUser(ctx, domain.User{BranchID: branch.ID, Role: domain.RoleReceptionist, Email: "file-receptionist@test.local", PasswordHash: "hash", FirstName: "Receptionist", LastName: "User"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.RegisterFile(ctx, domain.Principal{UserID: teacherUser.ID, BranchID: branch.ID, Role: domain.RoleTeacher}, RegisterFileInput{
		BranchID:          branch.ID,
		OwnerType:         "student",
		OwnerID:           teacherUser.ID,
		Category:          domain.FileCategorySpecial,
		Purpose:           domain.FilePurposePassport,
		OriginalFilename:  "passport.pdf",
		MimeType:          "application/pdf",
		OriginalSizeBytes: 12,
		StoredSizeBytes:   12,
		OriginalSHA256:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		StorageBucket:     "special",
		StorageKey:        "special/passport.pdf",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected teacher special-file forbidden error, got %v", err)
	}

	_, err = service.RegisterFile(ctx, domain.Principal{UserID: receptionistUser.ID, BranchID: branch.ID, Role: domain.RoleReceptionist}, RegisterFileInput{
		BranchID:          branch.ID,
		OwnerType:         "exam",
		OwnerID:           receptionistUser.ID,
		Category:          domain.FileCategoryStandard,
		Purpose:           domain.FilePurposeExamWriting,
		OriginalFilename:  "writing.pdf",
		MimeType:          "application/pdf",
		OriginalSizeBytes: 12,
		StoredSizeBytes:   12,
		OriginalSHA256:    "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		StorageBucket:     "standard",
		StorageKey:        "standard/writing.pdf",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected receptionist writing-file forbidden error, got %v", err)
	}
}

func TestUploadStandardProfileImageOptimizesToScreenSizedJPEG(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	branch, err := repo.CreateBranch(ctx, domain.Branch{Name: "Main", Slug: "profile-main"})
	if err != nil {
		t.Fatal(err)
	}
	storage := &fakeObjectStorage{}
	service := NewService(repo, WithObjectStorage(storage, "standard", "special"))
	original := testJPEG(t, 1800, 1200)
	hash := sha256.Sum256(original)

	file, err := service.UploadFile(ctx, domain.Principal{UserID: "owner", Role: domain.RoleOwner}, UploadFileInput{
		BranchID:         branch.ID,
		OwnerType:        "branch",
		OwnerID:          branch.ID,
		Category:         domain.FileCategoryStandard,
		Purpose:          domain.FilePurposeProfile,
		OriginalFilename: "branch-profile.jpeg",
		MimeType:         "image/jpeg",
		Size:             int64(len(original)),
		Content:          bytes.NewReader(original),
	})
	if err != nil {
		t.Fatalf("unexpected upload error: %v", err)
	}

	if file.MimeType != "image/jpeg" || file.OriginalFilename != "branch-profile.jpg" {
		t.Fatalf("expected optimized jpeg metadata, got %q %q", file.MimeType, file.OriginalFilename)
	}
	if file.OriginalSizeBytes != int64(len(original)) || file.StoredSizeBytes != int64(len(storage.putContent)) {
		t.Fatalf("unexpected size metadata: original=%d stored=%d put=%d", file.OriginalSizeBytes, file.StoredSizeBytes, len(storage.putContent))
	}
	if file.StoredSizeBytes >= file.OriginalSizeBytes {
		t.Fatalf("expected optimized file to be smaller: original=%d stored=%d", file.OriginalSizeBytes, file.StoredSizeBytes)
	}
	if file.OriginalSHA256 != hex.EncodeToString(hash[:]) {
		t.Fatalf("expected original hash to be recorded, got %s", file.OriginalSHA256)
	}
	if storage.putBucket != "standard" || storage.contentType != "image/jpeg" {
		t.Fatalf("unexpected storage write: bucket=%q content_type=%q", storage.putBucket, storage.contentType)
	}

	optimized, err := jpeg.Decode(bytes.NewReader(storage.putContent))
	if err != nil {
		t.Fatalf("stored content is not jpeg: %v", err)
	}
	if optimized.Bounds().Dx() != profileImageSize || optimized.Bounds().Dy() != profileImageSize {
		t.Fatalf("expected %dx%d optimized image, got %dx%d", profileImageSize, profileImageSize, optimized.Bounds().Dx(), optimized.Bounds().Dy())
	}
}

func TestUploadSmallStandardProfileImageUpscalesToScreenSize(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	branch, err := repo.CreateBranch(ctx, domain.Branch{Name: "Small", Slug: "profile-small"})
	if err != nil {
		t.Fatal(err)
	}
	storage := &fakeObjectStorage{}
	service := NewService(repo, WithObjectStorage(storage, "standard", "special"))
	original := testJPEG(t, 320, 320)

	_, err = service.UploadFile(ctx, domain.Principal{UserID: "owner", Role: domain.RoleOwner}, UploadFileInput{
		BranchID:         branch.ID,
		OwnerType:        "branch",
		OwnerID:          branch.ID,
		Category:         domain.FileCategoryStandard,
		Purpose:          domain.FilePurposeProfile,
		OriginalFilename: "small.jpg",
		MimeType:         "image/jpeg",
		Size:             int64(len(original)),
		Content:          bytes.NewReader(original),
	})
	if err != nil {
		t.Fatalf("unexpected upload error: %v", err)
	}

	optimized, err := jpeg.Decode(bytes.NewReader(storage.putContent))
	if err != nil {
		t.Fatalf("stored content is not jpeg: %v", err)
	}
	if optimized.Bounds().Dx() != profileImageSize || optimized.Bounds().Dy() != profileImageSize {
		t.Fatalf("expected small image to upscale to %dx%d, got %dx%d", profileImageSize, profileImageSize, optimized.Bounds().Dx(), optimized.Bounds().Dy())
	}
}

func TestUploadStandardProfileImageSniffsOctetStreamJPEG(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	branch, err := repo.CreateBranch(ctx, domain.Branch{Name: "Sniffed", Slug: "profile-sniffed"})
	if err != nil {
		t.Fatal(err)
	}
	storage := &fakeObjectStorage{}
	service := NewService(repo, WithObjectStorage(storage, "standard", "special"))
	original := testJPEG(t, 500, 500)

	file, err := service.UploadFile(ctx, domain.Principal{UserID: "owner", Role: domain.RoleOwner}, UploadFileInput{
		BranchID:         branch.ID,
		OwnerType:        "branch",
		OwnerID:          branch.ID,
		Category:         domain.FileCategoryStandard,
		Purpose:          domain.FilePurposeProfile,
		OriginalFilename: "profile.jpg",
		MimeType:         "application/octet-stream",
		Size:             int64(len(original)),
		Content:          bytes.NewReader(original),
	})
	if err != nil {
		t.Fatalf("unexpected upload error: %v", err)
	}
	if file.MimeType != "image/jpeg" || storage.contentType != "image/jpeg" {
		t.Fatalf("expected sniffed jpeg content type, got file=%q storage=%q", file.MimeType, storage.contentType)
	}
}

func TestUploadStandardProfileImageAcceptsPNG(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	branch, err := repo.CreateBranch(ctx, domain.Branch{Name: "PNG", Slug: "profile-png"})
	if err != nil {
		t.Fatal(err)
	}
	storage := &fakeObjectStorage{}
	service := NewService(repo, WithObjectStorage(storage, "standard", "special"))
	original := testPNG(t, 500, 420)

	file, err := service.UploadFile(ctx, domain.Principal{UserID: "owner", Role: domain.RoleOwner}, UploadFileInput{
		BranchID:         branch.ID,
		OwnerType:        "branch",
		OwnerID:          branch.ID,
		Category:         domain.FileCategoryStandard,
		Purpose:          domain.FilePurposeProfile,
		OriginalFilename: "profile.png",
		MimeType:         "image/png",
		Size:             int64(len(original)),
		Content:          bytes.NewReader(original),
	})
	if err != nil {
		t.Fatalf("unexpected upload error: %v", err)
	}
	if file.MimeType != "image/jpeg" || storage.contentType != "image/jpeg" {
		t.Fatalf("expected optimized jpeg content type, got file=%q storage=%q", file.MimeType, storage.contentType)
	}
}

func TestUploadStandardProfileImageAcceptsJPEGAlias(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	branch, err := repo.CreateBranch(ctx, domain.Branch{Name: "Alias", Slug: "profile-alias"})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, WithObjectStorage(&fakeObjectStorage{}, "standard", "special"))
	original := testJPEG(t, 500, 500)

	file, err := service.UploadFile(ctx, domain.Principal{UserID: "owner", Role: domain.RoleOwner}, UploadFileInput{
		BranchID:         branch.ID,
		OwnerType:        "branch",
		OwnerID:          branch.ID,
		Category:         domain.FileCategoryStandard,
		Purpose:          domain.FilePurposeProfile,
		OriginalFilename: "profile.jpg",
		MimeType:         "image/jpg",
		Size:             int64(len(original)),
		Content:          bytes.NewReader(original),
	})
	if err != nil {
		t.Fatalf("unexpected upload error: %v", err)
	}
	if file.MimeType != "image/jpeg" {
		t.Fatalf("expected normalized jpeg content type, got %q", file.MimeType)
	}
}

func TestUploadStandardProfileImageRejectsUnsupportedImageType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	branch, err := repo.CreateBranch(ctx, domain.Branch{Name: "Unsupported", Slug: "profile-unsupported"})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, WithObjectStorage(&fakeObjectStorage{}, "standard", "special"))

	_, err = service.UploadFile(ctx, domain.Principal{UserID: "owner", Role: domain.RoleOwner}, UploadFileInput{
		BranchID:         branch.ID,
		OwnerType:        "branch",
		OwnerID:          branch.ID,
		Category:         domain.FileCategoryStandard,
		Purpose:          domain.FilePurposeProfile,
		OriginalFilename: "profile.gif",
		MimeType:         "image/gif",
		Size:             6,
		Content:          bytes.NewReader([]byte("GIF89a")),
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input for unsupported profile image type, got %v", err)
	}
}

func testJPEG(t *testing.T, width int, height int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: uint8((x + y) % 255),
				A: 255,
			})
		}
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}

	return out.Bytes()
}

func testPNG(t *testing.T, width int, height int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: uint8((x + y) % 255),
				A: 255,
			})
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}

	return out.Bytes()
}
