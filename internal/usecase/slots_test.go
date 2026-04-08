package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"room-booking/internal/domain"
	"room-booking/internal/usecase"
)

type MockScheduleRepo struct {
	mock.Mock
}

func (m *MockScheduleRepo) Create(ctx context.Context, schedule *domain.Schedule) error {
	args := m.Called(ctx, schedule)
	return args.Error(0)
}

func (m *MockScheduleRepo) GetByRoomID(ctx context.Context, roomID uuid.UUID) (*domain.Schedule, error) {
	args := m.Called(ctx, roomID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Schedule), args.Error(1)
}

func newSlotUsecase(slotRepo *MockSlotRepo, roomRepo *MockRoomRepo, scheduleRepo *MockScheduleRepo) *usecase.SlotUsecase {
	return usecase.NewSlotUsecase(slotRepo, roomRepo, scheduleRepo)
}

func TestGenerateSlots_NoSchedule(t *testing.T) {
	slotRepo := new(MockSlotRepo)
	roomRepo := new(MockRoomRepo)
	scheduleRepo := new(MockScheduleRepo)
	uc := newSlotUsecase(slotRepo, roomRepo, scheduleRepo)

	roomID := uuid.New()
	room := &domain.Room{ID: roomID, Name: "Test Room"}
	date := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)

	roomRepo.On("GetByID", mock.Anything, roomID).Return(room, nil)
	scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(nil, domain.ErrNotFound)

	slots, err := uc.GenerateAndGetAvailable(context.Background(), roomID, date)
	require.NoError(t, err)
	assert.Empty(t, slots)

	roomRepo.AssertExpectations(t)
	scheduleRepo.AssertExpectations(t)
}

func TestGenerateSlots_WrongDayOfWeek(t *testing.T) {
	slotRepo := new(MockSlotRepo)
	roomRepo := new(MockRoomRepo)
	scheduleRepo := new(MockScheduleRepo)
	uc := newSlotUsecase(slotRepo, roomRepo, scheduleRepo)

	roomID := uuid.New()
	room := &domain.Room{ID: roomID, Name: "Test Room"}
	date := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	schedule := &domain.Schedule{
		ID:         uuid.New(),
		RoomID:     roomID,
		DaysOfWeek: []int{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "18:00",
	}

	roomRepo.On("GetByID", mock.Anything, roomID).Return(room, nil)
	scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)

	slots, err := uc.GenerateAndGetAvailable(context.Background(), roomID, date)
	require.NoError(t, err)
	assert.Empty(t, slots)

	roomRepo.AssertExpectations(t)
	scheduleRepo.AssertExpectations(t)
}

func TestGenerateSlots_CorrectDay(t *testing.T) {
	slotRepo := new(MockSlotRepo)
	roomRepo := new(MockRoomRepo)
	scheduleRepo := new(MockScheduleRepo)
	uc := newSlotUsecase(slotRepo, roomRepo, scheduleRepo)

	roomID := uuid.New()
	room := &domain.Room{ID: roomID, Name: "Test Room"}
	date := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)

	schedule := &domain.Schedule{
		ID:         uuid.New(),
		RoomID:     roomID,
		DaysOfWeek: []int{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "11:00",
	}

	expectedSlots := []*domain.Slot{
		{ID: uuid.New(), RoomID: roomID, Start: time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)},
		{ID: uuid.New(), RoomID: roomID, Start: time.Date(2026, 4, 6, 9, 30, 0, 0, time.UTC)},
		{ID: uuid.New(), RoomID: roomID, Start: time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)},
		{ID: uuid.New(), RoomID: roomID, Start: time.Date(2026, 4, 6, 10, 30, 0, 0, time.UTC)},
	}

	roomRepo.On("GetByID", mock.Anything, roomID).Return(room, nil)
	scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)
	slotRepo.On("UpsertSlots", mock.Anything, mock.AnythingOfType("[]*domain.Slot")).Return(nil)
	slotRepo.On("GetAvailableByRoomAndDate", mock.Anything, roomID, mock.AnythingOfType("time.Time")).Return(expectedSlots, nil)

	slots, err := uc.GenerateAndGetAvailable(context.Background(), roomID, date)
	require.NoError(t, err)
	assert.Len(t, slots, 4)

	roomRepo.AssertExpectations(t)
	scheduleRepo.AssertExpectations(t)
	slotRepo.AssertExpectations(t)
}
