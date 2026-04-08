package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"room-booking/internal/domain"
	"room-booking/internal/usecase"
)

type MockBookingRepo struct {
	mock.Mock
}

func (m *MockBookingRepo) Create(ctx context.Context, booking *domain.Booking) error {
	args := m.Called(ctx, booking)
	return args.Error(0)
}

func (m *MockBookingRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}

func (m *MockBookingRepo) GetActiveBySlotID(ctx context.Context, slotID uuid.UUID) (*domain.Booking, error) {
	args := m.Called(ctx, slotID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}

func (m *MockBookingRepo) ListAll(ctx context.Context, page, pageSize int) ([]*domain.Booking, int, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.Booking), args.Int(1), args.Error(2)
}

func (m *MockBookingRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Booking, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Booking), args.Error(1)
}

func (m *MockBookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.BookingStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockBookingRepo) UpdateConferenceLink(ctx context.Context, id uuid.UUID, link string) error {
	args := m.Called(ctx, id, link)
	return args.Error(0)
}

type MockSlotRepo struct {
	mock.Mock
}

func (m *MockSlotRepo) UpsertSlots(ctx context.Context, slots []*domain.Slot) error {
	args := m.Called(ctx, slots)
	return args.Error(0)
}

func (m *MockSlotRepo) GetAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]*domain.Slot, error) {
	args := m.Called(ctx, roomID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Slot), args.Error(1)
}

func (m *MockSlotRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Slot, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Slot), args.Error(1)
}

func newBookingUsecase(bookingRepo *MockBookingRepo, slotRepo *MockSlotRepo) *usecase.BookingUsecase {
	return usecase.NewBookingUsecase(bookingRepo, slotRepo, &usecase.MockConferenceService{})
}

func TestCreateBooking_Success(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	slotRepo := new(MockSlotRepo)
	uc := newBookingUsecase(bookingRepo, slotRepo)

	slotID := uuid.New()
	userID := uuid.New()
	futureTime := time.Now().Add(2 * time.Hour)

	slot := &domain.Slot{
		ID:     slotID,
		RoomID: uuid.New(),
		Start:  futureTime,
		End:    futureTime.Add(30 * time.Minute),
	}

	slotRepo.On("GetByID", mock.Anything, slotID).Return(slot, nil)
	bookingRepo.On("GetActiveBySlotID", mock.Anything, slotID).Return(nil, domain.ErrNotFound)
	bookingRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Booking")).Return(nil)

	booking, err := uc.CreateBooking(context.Background(), userID, slotID, false)
	require.NoError(t, err)
	require.NotNil(t, booking)
	assert.Equal(t, slotID, booking.SlotID)
	assert.Equal(t, userID, booking.UserID)
	assert.Equal(t, domain.BookingStatusActive, booking.Status)

	bookingRepo.AssertExpectations(t)
	slotRepo.AssertExpectations(t)
}

func TestCreateBooking_SlotNotFound(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	slotRepo := new(MockSlotRepo)
	uc := newBookingUsecase(bookingRepo, slotRepo)

	slotID := uuid.New()
	slotRepo.On("GetByID", mock.Anything, slotID).Return(nil, domain.ErrSlotNotFound)

	_, err := uc.CreateBooking(context.Background(), uuid.New(), slotID, false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrSlotNotFound))

	slotRepo.AssertExpectations(t)
}

func TestCreateBooking_SlotInPast(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	slotRepo := new(MockSlotRepo)
	uc := newBookingUsecase(bookingRepo, slotRepo)

	slotID := uuid.New()
	pastTime := time.Now().Add(-2 * time.Hour)

	slot := &domain.Slot{
		ID:     slotID,
		RoomID: uuid.New(),
		Start:  pastTime,
		End:    pastTime.Add(30 * time.Minute),
	}
	slotRepo.On("GetByID", mock.Anything, slotID).Return(slot, nil)

	_, err := uc.CreateBooking(context.Background(), uuid.New(), slotID, false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidRequest))

	slotRepo.AssertExpectations(t)
}

func TestCreateBooking_AlreadyBooked(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	slotRepo := new(MockSlotRepo)
	uc := newBookingUsecase(bookingRepo, slotRepo)

	slotID := uuid.New()
	futureTime := time.Now().Add(2 * time.Hour)

	slot := &domain.Slot{
		ID:     slotID,
		RoomID: uuid.New(),
		Start:  futureTime,
		End:    futureTime.Add(30 * time.Minute),
	}
	existingBooking := &domain.Booking{
		ID:     uuid.New(),
		SlotID: slotID,
		Status: domain.BookingStatusActive,
	}

	slotRepo.On("GetByID", mock.Anything, slotID).Return(slot, nil)
	bookingRepo.On("GetActiveBySlotID", mock.Anything, slotID).Return(existingBooking, nil)

	_, err := uc.CreateBooking(context.Background(), uuid.New(), slotID, false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrSlotBooked))

	bookingRepo.AssertExpectations(t)
	slotRepo.AssertExpectations(t)
}

func TestCancelBooking_Success(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	slotRepo := new(MockSlotRepo)
	uc := newBookingUsecase(bookingRepo, slotRepo)

	userID := uuid.New()
	bookingID := uuid.New()
	booking := &domain.Booking{
		ID:     bookingID,
		UserID: userID,
		Status: domain.BookingStatusActive,
	}

	bookingRepo.On("GetByID", mock.Anything, bookingID).Return(booking, nil)
	bookingRepo.On("UpdateStatus", mock.Anything, bookingID, domain.BookingStatusCancelled).Return(nil)

	result, err := uc.CancelBooking(context.Background(), userID, bookingID)
	require.NoError(t, err)
	assert.Equal(t, domain.BookingStatusCancelled, result.Status)

	bookingRepo.AssertExpectations(t)
}

func TestCancelBooking_Idempotent(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	slotRepo := new(MockSlotRepo)
	uc := newBookingUsecase(bookingRepo, slotRepo)

	userID := uuid.New()
	bookingID := uuid.New()
	booking := &domain.Booking{
		ID:     bookingID,
		UserID: userID,
		Status: domain.BookingStatusCancelled,
	}

	bookingRepo.On("GetByID", mock.Anything, bookingID).Return(booking, nil)

	result, err := uc.CancelBooking(context.Background(), userID, bookingID)
	require.NoError(t, err)
	assert.Equal(t, domain.BookingStatusCancelled, result.Status)

	bookingRepo.AssertNotCalled(t, "UpdateStatus")
	bookingRepo.AssertExpectations(t)
}

func TestCancelBooking_NotOwner(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	slotRepo := new(MockSlotRepo)
	uc := newBookingUsecase(bookingRepo, slotRepo)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	bookingID := uuid.New()
	booking := &domain.Booking{
		ID:     bookingID,
		UserID: ownerID,
		Status: domain.BookingStatusActive,
	}

	bookingRepo.On("GetByID", mock.Anything, bookingID).Return(booking, nil)

	_, err := uc.CancelBooking(context.Background(), otherUserID, bookingID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden))

	bookingRepo.AssertExpectations(t)
}

func TestCancelBooking_NotFound(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	slotRepo := new(MockSlotRepo)
	uc := newBookingUsecase(bookingRepo, slotRepo)

	bookingID := uuid.New()
	bookingRepo.On("GetByID", mock.Anything, bookingID).Return(nil, domain.ErrBookingNotFound)

	_, err := uc.CancelBooking(context.Background(), uuid.New(), bookingID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBookingNotFound))

	bookingRepo.AssertExpectations(t)
}
