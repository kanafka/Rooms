package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"room-booking/internal/domain"
	"room-booking/internal/usecase"
)

var (
	testAdminUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testUserUUID  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func newAuthUsecase(userRepo *MockUserRepo) *usecase.AuthUsecase {
	return usecase.NewAuthUsecase(userRepo, "testsecret", testAdminUUID, testUserUUID)
}

func TestDummyLogin_Admin(t *testing.T) {
	userRepo := new(MockUserRepo)
	uc := newAuthUsecase(userRepo)

	token, err := uc.DummyLogin("admin")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := uc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, testAdminUUID, claims.UserID)
	assert.Equal(t, "admin", claims.Role)
}

func TestDummyLogin_User(t *testing.T) {
	userRepo := new(MockUserRepo)
	uc := newAuthUsecase(userRepo)

	token, err := uc.DummyLogin("user")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := uc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, testUserUUID, claims.UserID)
	assert.Equal(t, "user", claims.Role)
}

func TestDummyLogin_InvalidRole(t *testing.T) {
	userRepo := new(MockUserRepo)
	uc := newAuthUsecase(userRepo)

	_, err := uc.DummyLogin("superuser")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidRequest))
}

func TestRegister_Success(t *testing.T) {
	userRepo := new(MockUserRepo)
	uc := newAuthUsecase(userRepo)

	userRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, domain.ErrNotFound)
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)

	user, err := uc.Register(context.Background(), "test@example.com", "password123", "user")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, domain.RoleUser, user.Role)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, "password123", user.PasswordHash)

	userRepo.AssertExpectations(t)
}

func TestRegister_EmailTaken(t *testing.T) {
	userRepo := new(MockUserRepo)
	uc := newAuthUsecase(userRepo)

	existingUser := &domain.User{
		ID:    uuid.New(),
		Email: "taken@example.com",
		Role:  domain.RoleUser,
	}
	userRepo.On("GetByEmail", mock.Anything, "taken@example.com").Return(existingUser, nil)

	_, err := uc.Register(context.Background(), "taken@example.com", "password123", "user")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrEmailTaken))

	userRepo.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	userRepo := new(MockUserRepo)
	uc := newAuthUsecase(userRepo)

	userRepo.On("GetByEmail", mock.Anything, "login@example.com").Return(nil, domain.ErrNotFound).Once()
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()

	registeredUser, err := uc.Register(context.Background(), "login@example.com", "mypassword", "user")
	require.NoError(t, err)

	userRepo.On("GetByEmail", mock.Anything, "login@example.com").Return(registeredUser, nil).Once()

	token, err := uc.Login(context.Background(), "login@example.com", "mypassword")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := uc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, registeredUser.ID, claims.UserID)

	userRepo.AssertExpectations(t)
}

func TestLogin_WrongPassword(t *testing.T) {
	userRepo := new(MockUserRepo)
	uc := newAuthUsecase(userRepo)

	userRepo.On("GetByEmail", mock.Anything, "user@example.com").Return(nil, domain.ErrNotFound).Once()
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()

	registeredUser, err := uc.Register(context.Background(), "user@example.com", "correctpassword", "user")
	require.NoError(t, err)

	userRepo.On("GetByEmail", mock.Anything, "user@example.com").Return(registeredUser, nil).Once()

	_, err = uc.Login(context.Background(), "user@example.com", "wrongpassword")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidCredentials))

	userRepo.AssertExpectations(t)
}
