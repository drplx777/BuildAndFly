package models

import "github.com/google/uuid"

type Role struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Role string    `json:"role" db:"role"`
}
