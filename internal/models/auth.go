package models

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken stores hashed refresh tokens for rotation and revocation.
type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID  `gorm:"type:uuid;index;not null"`
	TokenID   string     `gorm:"type:varchar(64);index;not null"`
	TokenHash string     `gorm:"type:varchar(64);uniqueIndex;not null"`
	ExpiresAt time.Time  `gorm:"index;not null"`
	RevokedAt *time.Time `gorm:"index"`
	CreatedAt time.Time
}
