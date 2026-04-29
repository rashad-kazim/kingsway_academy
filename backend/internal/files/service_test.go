package files

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/store"
)

type fakeFileStore struct {
	expired []domain.FileObject
	marked  []string
}

func (s *fakeFileStore) RegisterFile(context.Context, domain.FileObject) (domain.FileObject, error) {
	return domain.FileObject{}, nil
}

func (s *fakeFileStore) GetFile(context.Context, string) (domain.FileObject, error) {
	return domain.FileObject{}, nil
}

func (s *fakeFileStore) ListFiles(context.Context, string) ([]domain.FileObject, error) {
	return nil, nil
}

func (s *fakeFileStore) ListExpiredFiles(_ context.Context, _ time.Time, limit int) ([]domain.FileObject, error) {
	if limit > 0 && len(s.expired) > limit {
		return s.expired[:limit], nil
	}

	return s.expired, nil
}

func (s *fakeFileStore) MarkFileDeleted(_ context.Context, id string, deletedAt time.Time) (domain.FileObject, error) {
	s.marked = append(s.marked, id)
	for _, file := range s.expired {
		if file.ID == id {
			file.DeletedAt = &deletedAt
			return file, nil
		}
	}

	return domain.FileObject{}, domain.ErrNotFound
}

type fakeObjectStorage struct {
	deleted []string
}

func (s *fakeObjectStorage) Put(context.Context, string, string, io.Reader, int64, string) error {
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
