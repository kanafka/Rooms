package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"room-booking/internal/domain"
)

type BookingRepo struct {
	pool *Pool
}

func NewBookingRepo(pool *Pool) *BookingRepo {
	return &BookingRepo{pool: pool}
}

func (r *BookingRepo) Create(ctx context.Context, booking *domain.Booking) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO bookings (id, slot_id, user_id, status, conference_link, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		booking.ID, booking.SlotID, booking.UserID, string(booking.Status), booking.ConferenceLink, booking.CreatedAt,
	)
	return err
}

func (r *BookingRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, slot_id, user_id, status, conference_link, created_at FROM bookings WHERE id = $1`, id)

	b := &domain.Booking{}
	err := row.Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrBookingNotFound
		}
		return nil, err
	}
	return b, nil
}

func (r *BookingRepo) GetActiveBySlotID(ctx context.Context, slotID uuid.UUID) (*domain.Booking, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, slot_id, user_id, status, conference_link, created_at
		 FROM bookings WHERE slot_id = $1 AND status = 'active'`, slotID)

	b := &domain.Booking{}
	err := row.Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (r *BookingRepo) ListAll(ctx context.Context, page, pageSize int) ([]*domain.Booking, int, error) {
	offset := (page - 1) * pageSize

	rows, err := r.pool.Query(ctx,
		`SELECT id, slot_id, user_id, status, conference_link, created_at, COUNT(*) OVER() as total
		 FROM bookings
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var bookings []*domain.Booking
	var total int
	for rows.Next() {
		b := &domain.Booking{}
		if err := rows.Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt, &total); err != nil {
			return nil, 0, err
		}
		bookings = append(bookings, b)
	}
	return bookings, total, rows.Err()
}

func (r *BookingRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Booking, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT b.id, b.slot_id, b.user_id, b.status, b.conference_link, b.created_at
		 FROM bookings b
		 JOIN slots s ON s.id = b.slot_id
		 WHERE b.user_id = $1 AND s.start_time > NOW()
		 ORDER BY s.start_time ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*domain.Booking
	for rows.Next() {
		b := &domain.Booking{}
		if err := rows.Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *BookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.BookingStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE bookings SET status = $1 WHERE id = $2`,
		string(status), id,
	)
	return err
}

func (r *BookingRepo) UpdateConferenceLink(ctx context.Context, id uuid.UUID, link string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE bookings SET conference_link = $1 WHERE id = $2`,
		link, id,
	)
	return err
}
