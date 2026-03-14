package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProtectedUser represents a system account that the JIT agent is forbidden from locking or modifying.
// This prevents accidental lockout of critical accounts like 'root', 'ubuntu', or custom service accounts.
type ProtectedUser struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Username  string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"username"`
	Reason    string    `gorm:"type:text" json:"reason"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// BeforeCreate hook to ensure a UUID is generated if not provided
func (p *ProtectedUser) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}
