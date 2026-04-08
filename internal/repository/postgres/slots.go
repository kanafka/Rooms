package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"room-booking/internal/domain"
)

type SlotRepo struct {
	pool *Pool
}

func NewSlotRepo(pool *Pool) *SlotRepo {
	return &SlotRepo{pool: pool}
}

func (r *SlotRepo) UpsertSlots(ctx context.Context, slots []*domain.Slot) error {
	for _, slot := range slots {
		_, err := r.pool.Exec(ctx,
			`INSERT INTO slots (id, room_id, start_time, end_time)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (room_id, start_time) DO NOTHING`,
			slot.ID, slot.RoomID, slot.Start, slot.End,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *SlotRepo) GetAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]*domain.Slot, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.id, s.room_id, s.start_time, s.end_time
		 FROM slots s
		 LEFT JOIN bookings b ON b.slot_id = s.id AND b.status = 'active'
		 WHERE s.room_id = $1
		   AND DATE(s.start_time AT TIME ZONE 'UTC') = $2
		   AND b.id IS NULL
		 ORDER BY s.start_time ASC`,
		roomID, date.UTC().Format("2006-01-02"),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []*domain.Slot
	for rows.Next() {
		slot := &domain.Slot{}
		if err := rows.Scan(&slot.ID, &slot.RoomID, &slot.Start, &slot.End); err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	return slots, rows.Err()
}

func (r *SlotRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Slot, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, room_id, start_time, end_time FROM slots WHERE id = $1`, id)

	slot := &domain.Slot{}
	err := row.Scan(&slot.ID, &slot.RoomID, &slot.Start, &slot.End)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSlotNotFound
		}
		return nil, err
	}
	return slot, nil
}
