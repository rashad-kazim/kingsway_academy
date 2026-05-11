package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"kingsway/backend/internal/domain"
)

func (p *Postgres) RegisterFile(ctx context.Context, file domain.FileObject) (domain.FileObject, error) {
	if file.Policy == "" && file.Category.IsValid() {
		file.Policy = domain.InferFilePolicy(file.Category, file.Purpose)
	}
	if !file.Category.IsValid() || !file.Policy.IsValid() || file.Policy.Category() != file.Category || file.OriginalFilename == "" || file.StorageBucket == "" || file.StorageKey == "" {
		return domain.FileObject{}, domain.ErrInvalidInput
	}
	if file.Category.MustPreserveOriginalBytes() && file.StoredSizeBytes != file.OriginalSizeBytes {
		return domain.FileObject{}, domain.ErrInvalidInput
	}
	createdAt := time.Now().UTC()
	if file.RetentionUntil == nil {
		file.RetentionUntil = domain.RetentionUntil(file.Purpose, createdAt)
	}

	var created domain.FileObject
	err := p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO files (
				branch_id, uploader_user_id, owner_type, owner_id, category, policy, purpose,
				original_filename, mime_type, original_size_bytes, stored_size_bytes,
				original_sha256, storage_bucket, storage_key, retention_until
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			RETURNING id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
				category, policy, purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
				original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
		`, file.BranchID, file.UploaderUserID, file.OwnerType, file.OwnerID, file.Category, file.Policy, file.Purpose, file.OriginalFilename, file.MimeType, file.OriginalSizeBytes, file.StoredSizeBytes, file.OriginalSHA256, file.StorageBucket, file.StorageKey, file.RetentionUntil)

		next, err := scanFileObject(row)
		if err != nil {
			return mapPostgresError(err)
		}
		created = next
		if err := insertOutboxTx(ctx, tx, "files.file.registered", created); err != nil {
			return mapPostgresError(err)
		}
		if created.RetentionUntil != nil {
			if err := insertOutboxTx(ctx, tx, "files.retention.scheduled", map[string]any{
				"file_id":         created.ID,
				"branch_id":       created.BranchID,
				"retention_until": created.RetentionUntil,
			}); err != nil {
				return mapPostgresError(err)
			}
		}

		return nil
	})
	if err != nil {
		return domain.FileObject{}, err
	}

	return created, nil
}

func (p *Postgres) GetFile(ctx context.Context, id string) (domain.FileObject, error) {
	row := p.queryRow(ctx, `
		SELECT id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
			category, coalesce(policy, ''), purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
			original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
		FROM files
		WHERE id = $1
	`, id)

	file, err := scanFileObject(row)
	if err != nil {
		return domain.FileObject{}, mapPostgresError(err)
	}

	return file, nil
}

func (p *Postgres) ListFiles(ctx context.Context, branchID string) ([]domain.FileObject, error) {
	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
			category, coalesce(policy, ''), purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
			original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
		FROM files
		WHERE deleted_at IS NULL AND ($1 = '' OR branch_id = $1::uuid)
		ORDER BY created_at DESC
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	files := make([]domain.FileObject, 0)
	for rows.Next() {
		file, err := scanFileObject(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return files, nil
}

func (p *Postgres) ListFilesPage(ctx context.Context, branchID string, ownerType string, ownerID string, purpose domain.FilePurpose, page domain.PageRequest) ([]domain.FileObject, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM files
		WHERE deleted_at IS NULL
			AND ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR owner_type = $2)
			AND ($3 = '' OR owner_id = $3::uuid)
			AND ($4 = '' OR purpose = $4)
	`, branchID, ownerType, ownerID, string(purpose)).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
			category, coalesce(policy, ''), purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
			original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
		FROM files
		WHERE deleted_at IS NULL
			AND ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR owner_type = $2)
			AND ($3 = '' OR owner_id = $3::uuid)
			AND ($4 = '' OR purpose = $4)
		ORDER BY created_at DESC
		LIMIT $5 OFFSET $6
	`, branchID, ownerType, ownerID, string(purpose), page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	files := make([]domain.FileObject, 0, page.Limit)
	for rows.Next() {
		file, err := scanFileObject(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return files, total, nil
}

func (p *Postgres) ListExpiredFiles(ctx context.Context, now time.Time, limit int) ([]domain.FileObject, error) {
	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
			category, coalesce(policy, ''), purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
			original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
		FROM files
		WHERE deleted_at IS NULL
			AND retention_until IS NOT NULL
			AND retention_until <= $1
		ORDER BY retention_until
		LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	files := make([]domain.FileObject, 0)
	for rows.Next() {
		file, err := scanFileObject(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return files, nil
}

func (p *Postgres) MarkFileDeleted(ctx context.Context, id string, deletedAt time.Time) (domain.FileObject, error) {
	var file domain.FileObject
	err := p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			UPDATE files
			SET deleted_at = $2
			WHERE id = $1
			RETURNING id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
				category, coalesce(policy, ''), purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
				original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
		`, id, deletedAt)

		next, err := scanFileObject(row)
		if err != nil {
			return mapPostgresError(err)
		}
		file = next
		if err := insertOutboxTx(ctx, tx, "files.retention.deleted", file); err != nil {
			return mapPostgresError(err)
		}

		return nil
	})
	if err != nil {
		return domain.FileObject{}, err
	}

	return file, nil
}
