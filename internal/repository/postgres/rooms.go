package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"room-booking/internal/domain"
)

type RoomRepo struct {
	pool *Pool
}

func NewRoomRepo(pool *Pool) *RoomRepo {
	return &RoomRepo{pool: pool}
}

func (r *RoomRepo) Create(ctx context.Context, room *domain.Room) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO rooms (id, name, description, capacity, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		room.ID, room.Name, room.Description, room.Capacity, room.CreatedAt,
	)
	return err
}

func (r *RoomRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, description, capacity, created_at FROM rooms WHERE id = $1`, id)

	room := &domain.Room{}
	err := row.Scan(&room.ID, &room.Name, &room.Description, &room.Capacity, &room.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRoomNotFound
		}
		return nil, err
	}
	return room, nil
}

func (r *RoomRepo) List(ctx context.Context) ([]*domain.Room, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, capacity, created_at FROM rooms ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*domain.Room
	for rows.Next() {
		room := &domain.Room{}
		if err := rows.Scan(&room.ID, &room.Name, &room.Description, &room.Capacity, &room.CreatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}
