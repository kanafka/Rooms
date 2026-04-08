package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"room-booking/internal/domain"
)

type BookingUsecase struct {
	bookings   domain.BookingRepository
	slots      domain.SlotRepository
	conference ConferenceService
}

func NewBookingUsecase(bookings domain.BookingRepository, slots domain.SlotRepository, conference ConferenceService) *BookingUsecase {
	return &BookingUsecase{
		bookings:   bookings,
		slots:      slots,
		conference: conference,
	}
}

func (u *BookingUsecase) CreateBooking(ctx context.Context, userID, slotID uuid.UUID, createConferenceLink bool) (*domain.Booking, error) {
	slot, err := u.slots.GetByID(ctx, slotID)
	if err != nil {
		if errors.Is(err, domain.ErrSlotNotFound) {
			return nil, domain.ErrSlotNotFound
		}
		return nil, err
	}

	if slot.Start.Before(time.Now()) {
		return nil, fmt.Errorf("%w: slot is in the past", domain.ErrInvalidRequest)
	}

	_, err = u.bookings.GetActiveBySlotID(ctx, slotID)
	if err == nil {
		return nil, domain.ErrSlotBooked
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	booking := &domain.Booking{
		ID:        uuid.New(),
		SlotID:    slotID,
		UserID:    userID,
		Status:    domain.BookingStatusActive,
		CreatedAt: time.Now().UTC(),
	}

	if err := u.bookings.Create(ctx, booking); err != nil {
		return nil, err
	}

	if createConferenceLink && u.conference != nil {
		link, err := u.conference.CreateLink(ctx, booking.ID)
		if err != nil {
			log.Printf("failed to create conference link for booking %s: %v", booking.ID, err)
		} else {
			if updateErr := u.bookings.UpdateConferenceLink(ctx, booking.ID, link); updateErr != nil {
				log.Printf("failed to save conference link for booking %s: %v", booking.ID, updateErr)
			} else {
				booking.ConferenceLink = &link
			}
		}
	}

	return booking, nil
}

func (u *BookingUsecase) ListAll(ctx context.Context, page, pageSize int) ([]*domain.Booking, *domain.Pagination, error) {
	bookings, total, err := u.bookings.ListAll(ctx, page, pageSize)
	if err != nil {
		return nil, nil, err
	}

	pagination := &domain.Pagination{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}

	if bookings == nil {
		bookings = []*domain.Booking{}
	}

	return bookings, pagination, nil
}

func (u *BookingUsecase) ListMy(ctx context.Context, userID uuid.UUID) ([]*domain.Booking, error) {
	bookings, err := u.bookings.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if bookings == nil {
		bookings = []*domain.Booking{}
	}
	return bookings, nil
}

func (u *BookingUsecase) CancelBooking(ctx context.Context, userID, bookingID uuid.UUID) (*domain.Booking, error) {
	booking, err := u.bookings.GetByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, domain.ErrBookingNotFound) {
			return nil, domain.ErrBookingNotFound
		}
		return nil, err
	}

	if booking.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if booking.Status == domain.BookingStatusCancelled {
		return booking, nil
	}

	if err := u.bookings.UpdateStatus(ctx, bookingID, domain.BookingStatusCancelled); err != nil {
		return nil, err
	}

	booking.Status = domain.BookingStatusCancelled
	return booking, nil
}
