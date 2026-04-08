package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"room-booking/internal/domain"
)

type jwtClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type AuthUsecase struct {
	users     domain.UserRepository
	jwtSecret []byte
	adminUUID uuid.UUID
	userUUID  uuid.UUID
}

func NewAuthUsecase(users domain.UserRepository, jwtSecret string, adminUUID, userUUID uuid.UUID) *AuthUsecase {
	return &AuthUsecase{
		users:     users,
		jwtSecret: []byte(jwtSecret),
		adminUUID: adminUUID,
		userUUID:  userUUID,
	}
}

func (u *AuthUsecase) DummyLogin(role string) (string, error) {
	var userID uuid.UUID
	switch domain.Role(role) {
	case domain.RoleAdmin:
		userID = u.adminUUID
	case domain.RoleUser:
		userID = u.userUUID
	default:
		return "", fmt.Errorf("%w: role must be admin or user", domain.ErrInvalidRequest)
	}
	return u.generateToken(userID, role)
}

func (u *AuthUsecase) Register(ctx context.Context, email, password, role string) (*domain.User, error) {
	if domain.Role(role) != domain.RoleAdmin && domain.Role(role) != domain.RoleUser {
		return nil, fmt.Errorf("%w: invalid role", domain.ErrInvalidRequest)
	}

	_, err := u.users.GetByEmail(ctx, email)
	if err == nil {
		return nil, domain.ErrEmailTaken
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.Role(role),
		CreatedAt:    time.Now().UTC(),
	}

	if err := u.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *AuthUsecase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := u.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", domain.ErrInvalidCredentials
	}

	return u.generateToken(user.ID, string(user.Role))
}

func (u *AuthUsecase) ValidateToken(tokenStr string) (*domain.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return u.jwtSecret, nil
	})
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	c, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrUnauthorized
	}

	return &domain.Claims{UserID: c.UserID, Role: c.Role}, nil
}

func (u *AuthUsecase) generateToken(userID uuid.UUID, role string) (string, error) {
	c := &jwtClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
