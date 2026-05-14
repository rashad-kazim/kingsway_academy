package store

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"kingsway/backend/internal/domain"
)

func (p *Postgres) CreateStaffMember(ctx context.Context, user domain.User, staff domain.StaffMember) (domain.StaffMember, error) {
	user.Email = normalizeEmail(user.Email)
	if user.BranchID == "" || user.Email == "" || user.PasswordHash == "" || user.FirstName == "" || user.LastName == "" {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	if user.Role != domain.RoleReceptionist {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}

	var created domain.StaffMember
	err := p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		createdUser, err := scanUser(tx.QueryRow(ctx, `
			INSERT INTO users (branch_id, role, email, password_hash, first_name, last_name, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, true)
			RETURNING id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, token_version, coalesce(last_login_at::text, ''), created_at, updated_at
		`, user.BranchID, user.Role, user.Email, user.PasswordHash, strings.TrimSpace(user.FirstName), strings.TrimSpace(user.LastName)))
		if err != nil {
			return mapPostgresError(err)
		}

		var staffID string
		err = tx.QueryRow(ctx, `
			INSERT INTO staff_profiles (branch_id, user_id, role, birth_date, gender, phone, address, hired_at, salary_amount_azn, profile_photo_file_id)
			VALUES ($1, $2, $3, CASE WHEN $4 = '' THEN NULL ELSE to_date($4, 'DD/MM/YYYY') END, $5, $6, $7, CASE WHEN $8 = '' THEN NULL ELSE to_date($8, 'DD/MM/YYYY') END, $9, nullif($10, '')::uuid)
			RETURNING id::text
		`, createdUser.BranchID, createdUser.ID, domain.RoleReceptionist, strings.TrimSpace(staff.BirthDate), strings.TrimSpace(staff.Gender), strings.TrimSpace(staff.Phone), strings.TrimSpace(staff.Address), strings.TrimSpace(staff.HiredAt), staff.SalaryAmountAZN, strings.TrimSpace(staff.ProfilePhotoFileID)).Scan(&staffID)
		if err != nil {
			return mapPostgresError(err)
		}

		next, err := p.staffMemberByIDTx(ctx, tx, staffID)
		if err != nil {
			return err
		}
		created = next
		return nil
	})
	if err != nil {
		return domain.StaffMember{}, err
	}

	return created, nil
}

func (p *Postgres) UpdateStaffMember(ctx context.Context, staff domain.StaffMember, passwordHash string) (domain.StaffMember, error) {
	if staff.ID == "" || staff.FirstName == "" || staff.LastName == "" || staff.Email == "" || staff.SalaryAmountAZN < 0 {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}

	var updated domain.StaffMember
	err := p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		current, err := p.staffMemberByIDTx(ctx, tx, staff.ID)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			UPDATE users
			SET branch_id = $2,
				email = $3,
				first_name = $4,
				last_name = $5,
				password_hash = CASE WHEN $6 = '' THEN password_hash ELSE $6 END,
				is_active = $7,
				token_version = token_version + CASE WHEN $6 = '' AND is_active = $7 AND branch_id = $2 THEN 0 ELSE 1 END,
				updated_at = now()
			WHERE id = $1
		`, current.UserID, staff.BranchID, normalizeEmail(staff.Email), strings.TrimSpace(staff.FirstName), strings.TrimSpace(staff.LastName), strings.TrimSpace(passwordHash), staff.IsActive)
		if err != nil {
			return mapPostgresError(err)
		}

		_, err = tx.Exec(ctx, `
			UPDATE staff_profiles
			SET branch_id = $2,
				birth_date = CASE WHEN $3 = '' THEN NULL ELSE to_date($3, 'DD/MM/YYYY') END,
				gender = $4,
				phone = $5,
				address = $6,
				hired_at = CASE WHEN $7 = '' THEN NULL ELSE to_date($7, 'DD/MM/YYYY') END,
				salary_amount_azn = $8,
				profile_photo_file_id = nullif($9, '')::uuid,
				updated_at = now()
			WHERE id = $1
		`, staff.ID, staff.BranchID, strings.TrimSpace(staff.BirthDate), strings.TrimSpace(staff.Gender), strings.TrimSpace(staff.Phone), strings.TrimSpace(staff.Address), strings.TrimSpace(staff.HiredAt), staff.SalaryAmountAZN, strings.TrimSpace(staff.ProfilePhotoFileID))
		if err != nil {
			return mapPostgresError(err)
		}

		next, err := p.staffMemberByIDTx(ctx, tx, staff.ID)
		if err != nil {
			return err
		}
		updated = next
		return nil
	})
	if err != nil {
		return domain.StaffMember{}, err
	}

	return updated, nil
}

func (p *Postgres) DeleteStaffMember(ctx context.Context, id string) (domain.StaffMember, error) {
	var staff domain.StaffMember
	err := p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		current, err := p.staffMemberByIDTx(ctx, tx, id)
		if err != nil {
			return err
		}
		staff = current
		if _, err := tx.Exec(ctx, `UPDATE staff_profiles SET profile_photo_file_id = NULL WHERE id = $1`, id); err != nil {
			return mapPostgresError(err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM notifications WHERE recipient_user_id = $1`, staff.UserID); err != nil {
			return mapPostgresError(err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM staff_profiles WHERE id = $1`, id); err != nil {
			return mapPostgresError(err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, staff.UserID); err != nil {
			return mapPostgresError(err)
		}
		return nil
	})
	if err != nil {
		return domain.StaffMember{}, err
	}

	return staff, nil
}

func (p *Postgres) GetStaffMember(ctx context.Context, id string) (domain.StaffMember, error) {
	row := p.queryRow(ctx, staffMemberSelect()+` WHERE sp.id = $1`, id)
	staff, err := scanStaffMember(row)
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	return staff, nil
}

func (p *Postgres) ListStaffMembers(ctx context.Context, branchID string) ([]domain.StaffMember, error) {
	rows, err := p.query(ctx, staffMemberSelect()+`
		WHERE sp.branch_id = $1
		ORDER BY u.last_name, u.first_name
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	staff := make([]domain.StaffMember, 0)
	for rows.Next() {
		member, err := scanStaffMember(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		staff = append(staff, member)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return staff, nil
}

func (p *Postgres) ListStaffMembersPage(ctx context.Context, branchID string, page domain.PageRequest) ([]domain.StaffMember, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM staff_profiles sp
		JOIN users u ON u.id = sp.user_id
		WHERE sp.branch_id = $1
	`, branchID).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, staffMemberSelect()+`
		WHERE sp.branch_id = $1
		ORDER BY u.last_name, u.first_name
		LIMIT $2 OFFSET $3
	`, branchID, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	staff := make([]domain.StaffMember, 0, page.Limit)
	for rows.Next() {
		member, err := scanStaffMember(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		staff = append(staff, member)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return staff, total, nil
}

func (p *Postgres) staffMemberByIDTx(ctx context.Context, tx pgx.Tx, id string) (domain.StaffMember, error) {
	staff, err := scanStaffMember(tx.QueryRow(ctx, staffMemberSelect()+` WHERE sp.id = $1`, id))
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	return staff, nil
}
