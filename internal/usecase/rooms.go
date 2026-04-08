package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"room-booking/internal/domain"
)

type RoomUsecase struct {
	rooms domain.RoomRepository
}

func NewRoomUsecase(rooms domain.RoomRepository) *RoomUsecase {
	return &RoomUsecase{rooms: rooms}
}

func (u *RoomUsecase) CreateRoom(ctx context.Context, name string, description *string, capacity *int) (*domain.Room, error) {
	room := &domain.Room{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Capacity:    capacity,
		CreatedAt:   time.Now().UTC(),
	}

	if err := u.rooms.Create(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

func (u *RoomUsecase) ListRooms(ctx context.Context) ([]*domain.Room, error) {
	return u.rooms.List(ctx)
}
