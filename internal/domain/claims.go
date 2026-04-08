package domain

import "github.com/google/uuid"

type Claims struct {
	UserID uuid.UUID
	Role   string
}
