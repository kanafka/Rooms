package usecase

import (
	"context"

	"github.com/google/uuid"
)

type ConferenceService interface {
	CreateLink(ctx context.Context, bookingID uuid.UUID) (string, error)
}

type MockConferenceService struct{}

func (m *MockConferenceService) CreateLink(_ context.Context, bookingID uuid.UUID) (string, error) {
	return "https://conference.example.com/meeting/" + bookingID.String(), nil
}
