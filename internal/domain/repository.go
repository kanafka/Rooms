package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type RoomRepository interface {
	Create(ctx context.Context, room *Room) error
	GetByID(ctx context.Context, id uuid.UUID) (*Room, error)
	List(ctx context.Context) ([]*Room, error)
}

type ScheduleRepository interface {
	Create(ctx context.Context, schedule *Schedule) error
	GetByRoomID(ctx context.Context, roomID uuid.UUID) (*Schedule, error)
}

type SlotRepository interface {
	UpsertSlots(ctx context.Context, slots []*Slot) error
	GetAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]*Slot, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Slot, error)
}

type BookingRepository interface {
	Create(ctx context.Context, booking *Booking) error
	GetByID(ctx context.Context, id uuid.UUID) (*Booking, error)
	GetActiveBySlotID(ctx context.Context, slotID uuid.UUID) (*Booking, error)
	ListAll(ctx context.Context, page, pageSize int) ([]*Booking, int, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*Booking, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status BookingStatus) error
	UpdateConferenceLink(ctx context.Context, id uuid.UUID, link string) error
}
