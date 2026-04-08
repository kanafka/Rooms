package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"room-booking/internal/domain"
)

type ScheduleRepo struct {
	pool *Pool
}

func NewScheduleRepo(pool *Pool) *ScheduleRepo {
	return &ScheduleRepo{pool: pool}
}

func (r *ScheduleRepo) Create(ctx context.Context, schedule *domain.Schedule) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time)
		 VALUES ($1, $2, $3, $4, $5)`,
		schedule.ID, schedule.RoomID, schedule.DaysOfWeek, schedule.StartTime, schedule.EndTime,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return domain.ErrScheduleExists
		}
		return err
	}
	return nil
}

func (r *ScheduleRepo) GetByRoomID(ctx context.Context, roomID uuid.UUID) (*domain.Schedule, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, room_id, days_of_week, start_time, end_time FROM schedules WHERE room_id = $1`, roomID)

	s := &domain.Schedule{}
	err := row.Scan(&s.ID, &s.RoomID, &s.DaysOfWeek, &s.StartTime, &s.EndTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return s, nil
}
