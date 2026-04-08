package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrRoomNotFound       = errors.New("room not found")
	ErrSlotNotFound       = errors.New("slot not found")
	ErrBookingNotFound    = errors.New("booking not found")
	ErrScheduleExists     = errors.New("schedule already exists")
	ErrSlotBooked         = errors.New("slot already booked")
	ErrForbidden          = errors.New("forbidden")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
