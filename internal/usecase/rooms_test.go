package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"room-booking/internal/domain"
	"room-booking/internal/usecase"
)

type MockRoomRepo struct {
	mock.Mock
}

func (m *MockRoomRepo) Create(ctx context.Context, room *domain.Room) error {
	args := m.Called(ctx, room)
	return args.Error(0)
}

func (m *MockRoomRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Room), args.Error(1)
}

func (m *MockRoomRepo) List(ctx context.Context) ([]*domain.Room, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Room), args.Error(1)
}

func TestCreateRoom_Success(t *testing.T) {
	roomRepo := new(MockRoomRepo)
	uc := usecase.NewRoomUsecase(roomRepo)

	desc := "A nice room"
	cap := 10
	roomRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Room")).Return(nil)

	room, err := uc.CreateRoom(context.Background(), "Conference Room A", &desc, &cap)
	require.NoError(t, err)
	require.NotNil(t, room)
	assert.Equal(t, "Conference Room A", room.Name)
	assert.Equal(t, &desc, room.Description)
	assert.Equal(t, &cap, room.Capacity)
	assert.NotEqual(t, uuid.Nil, room.ID)

	roomRepo.AssertExpectations(t)
}

func TestListRooms_Empty(t *testing.T) {
	roomRepo := new(MockRoomRepo)
	uc := usecase.NewRoomUsecase(roomRepo)

	roomRepo.On("List", mock.Anything).Return([]*domain.Room{}, nil)

	rooms, err := uc.ListRooms(context.Background())
	require.NoError(t, err)
	assert.Empty(t, rooms)

	roomRepo.AssertExpectations(t)
}

func TestListRooms_Multiple(t *testing.T) {
	roomRepo := new(MockRoomRepo)
	uc := usecase.NewRoomUsecase(roomRepo)

	expected := []*domain.Room{
		{ID: uuid.New(), Name: "Room A"},
		{ID: uuid.New(), Name: "Room B"},
		{ID: uuid.New(), Name: "Room C"},
	}
	roomRepo.On("List", mock.Anything).Return(expected, nil)

	rooms, err := uc.ListRooms(context.Background())
	require.NoError(t, err)
	assert.Len(t, rooms, 3)
	assert.Equal(t, "Room A", rooms[0].Name)
	assert.Equal(t, "Room B", rooms[1].Name)
	assert.Equal(t, "Room C", rooms[2].Name)

	roomRepo.AssertExpectations(t)
}
