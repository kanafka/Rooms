package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"room-booking/internal/domain"
)

type ScheduleUsecase struct {
	schedules domain.ScheduleRepository
	rooms     domain.RoomRepository
}

func NewScheduleUsecase(schedules domain.ScheduleRepository, rooms domain.RoomRepository) *ScheduleUsecase {
	return &ScheduleUsecase{
		schedules: schedules,
		rooms:     rooms,
	}
}

func (u *ScheduleUsecase) CreateSchedule(ctx context.Context, roomID uuid.UUID, daysOfWeek []int, startTime, endTime string) (*domain.Schedule, error) {
	if _, err := u.rooms.GetByID(ctx, roomID); err != nil {
		if errors.Is(err, domain.ErrRoomNotFound) {
			return nil, domain.ErrRoomNotFound
		}
		return nil, err
	}

	for _, d := range daysOfWeek {
		if d < 1 || d > 7 {
			return nil, fmt.Errorf("%w: days_of_week must be between 1 and 7", domain.ErrInvalidRequest)
		}
	}

	if !isValidTime(startTime) || !isValidTime(endTime) {
		return nil, fmt.Errorf("%w: times must be in HH:MM format", domain.ErrInvalidRequest)
	}

	if startTime >= endTime {
		return nil, fmt.Errorf("%w: start_time must be before end_time", domain.ErrInvalidRequest)
	}

	schedule := &domain.Schedule{
		ID:         uuid.New(),
		RoomID:     roomID,
		DaysOfWeek: daysOfWeek,
		StartTime:  startTime,
		EndTime:    endTime,
	}

	if err := u.schedules.Create(ctx, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

func isValidTime(t string) bool {
	if len(t) != 5 || t[2] != ':' {
		return false
	}
	hh := t[0:2]
	mm := t[3:5]
	for _, c := range hh + mm {
		if c < '0' || c > '9' {
			return false
		}
	}
	h := int(hh[0]-'0')*10 + int(hh[1]-'0')
	m := int(mm[0]-'0')*10 + int(mm[1]-'0')
	return h >= 0 && h <= 23 && m >= 0 && m <= 59
}
