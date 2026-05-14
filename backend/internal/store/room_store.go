package store

import (
	"context"
	"strings"

	"kingsway/backend/internal/domain"
)

func (p *Postgres) CreateRoom(ctx context.Context, room domain.Room) (domain.Room, error) {
	if strings.TrimSpace(room.Name) == "" {
		return domain.Room{}, domain.ErrInvalidInput
	}
	if room.Capacity <= 0 {
		room.Capacity = 1
	}

	row := p.queryRow(ctx, `
		INSERT INTO rooms (branch_id, name, capacity, is_active)
		VALUES ($1, $2, $3, true)
		RETURNING id::text, branch_id::text, name, capacity, is_active, created_at, updated_at
	`, room.BranchID, strings.TrimSpace(room.Name), room.Capacity)

	created, err := scanRoom(row)
	if err != nil {
		return domain.Room{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) UpdateRoom(ctx context.Context, room domain.Room) (domain.Room, error) {
	if strings.TrimSpace(room.Name) == "" {
		return domain.Room{}, domain.ErrInvalidInput
	}
	if room.Capacity <= 0 {
		room.Capacity = 1
	}

	row := p.queryRow(ctx, `
		UPDATE rooms
		SET name = $2, capacity = $3, updated_at = now()
		WHERE id = $1 AND is_active = true
		RETURNING id::text, branch_id::text, name, capacity, is_active, created_at, updated_at
	`, room.ID, strings.TrimSpace(room.Name), room.Capacity)

	updated, err := scanRoom(row)
	if err != nil {
		return domain.Room{}, mapPostgresError(err)
	}

	return updated, nil
}

func (p *Postgres) ListRooms(ctx context.Context, branchID string) ([]domain.Room, error) {
	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, name, capacity, is_active, created_at, updated_at
		FROM rooms
		WHERE is_active = true AND ($1 = '' OR branch_id = $1::uuid)
		ORDER BY name
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	rooms := make([]domain.Room, 0)
	for rows.Next() {
		room, err := scanRoom(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return rooms, nil
}

func (p *Postgres) ListRoomsPage(ctx context.Context, branchID string, page domain.PageRequest) ([]domain.Room, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM rooms
		WHERE is_active = true AND ($1 = '' OR branch_id = $1::uuid)
	`, branchID).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, name, capacity, is_active, created_at, updated_at
		FROM rooms
		WHERE is_active = true AND ($1 = '' OR branch_id = $1::uuid)
		ORDER BY name
		LIMIT $2 OFFSET $3
	`, branchID, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	rooms := make([]domain.Room, 0, page.Limit)
	for rows.Next() {
		room, err := scanRoom(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return rooms, total, nil
}

func (p *Postgres) GetRoom(ctx context.Context, id string) (domain.Room, error) {
	row := p.queryRow(ctx, `
		SELECT id::text, branch_id::text, name, capacity, is_active, created_at, updated_at
		FROM rooms
		WHERE id = $1
	`, id)

	room, err := scanRoom(row)
	if err != nil {
		return domain.Room{}, mapPostgresError(err)
	}

	return room, nil
}

func (p *Postgres) DeactivateRoom(ctx context.Context, id string) (domain.Room, error) {
	row := p.queryRow(ctx, `
		UPDATE rooms
		SET is_active = false, updated_at = now()
		WHERE id = $1
		RETURNING id::text, branch_id::text, name, capacity, is_active, created_at, updated_at
	`, id)

	room, err := scanRoom(row)
	if err != nil {
		return domain.Room{}, mapPostgresError(err)
	}

	return room, nil
}

func (p *Postgres) roomBranch(ctx context.Context, roomID string) (string, error) {
	var branchID string
	err := p.queryRow(ctx, `SELECT branch_id::text FROM rooms WHERE id = $1`, roomID).Scan(&branchID)
	if err != nil {
		return "", mapPostgresError(err)
	}

	return branchID, nil
}
