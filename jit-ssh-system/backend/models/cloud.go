package models

import (
	"time"

	"gorm.io/gorm"
)

// CloudProviderType defines the supported cloud providers
type CloudProviderType string

const (
	ProviderAWS   CloudProviderType = "aws"
	ProviderGCP   CloudProviderType = "gcp"
	ProviderAzure CloudProviderType = "azure"
)

// CloudIntegration represents a configured connection to a cloud provider
type CloudIntegration struct {
	ID                   string            `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name                 string            `gorm:"type:varchar(255);not null;uniqueIndex:idx_cloud_int_name_deleted,where:deleted_at IS NULL" json:"name"`
	Provider             CloudProviderType `gorm:"type:varchar(50);not null" json:"provider"`       // aws, gcp, azure
	EncryptedCredentials []byte            `gorm:"type:bytea;not null" json:"-"`                    // Never expose via JSON API
	Status               string            `gorm:"type:varchar(50);default:'active'" json:"status"` // active, error, inactive
	Metadata             string            `gorm:"type:jsonb" json:"metadata"`                      // JSON string for non-secret configs (Region, TenantID, IdentityStoreID)
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
	DeletedAt            gorm.DeletedAt    `gorm:"index" json:"-"`
}

// CloudAccessRequest represents a user's request for temporary cloud access
type CloudAccessRequest struct {
	ID               string           `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID           string           `gorm:"type:uuid;not null" json:"user_id"`
	User             User             `gorm:"foreignKey:UserID" json:"user"`
	ApproverID       *string          `gorm:"type:uuid" json:"approver_id,omitempty"`
	IntegrationID    string           `gorm:"type:uuid;not null" json:"integration_id"`
	Integration      CloudIntegration `gorm:"foreignKey:IntegrationID" json:"integration"`
	TargetGroupID    string           `gorm:"type:varchar(255);not null" json:"target_group_id"`
	TargetGroupName  string           `gorm:"type:varchar(255);not null" json:"target_group_name"`
	Status           string           `gorm:"type:varchar(50);default:'pending'" json:"status"` // pending, approved, active, revoked, expired, failed
	Reason           string           `gorm:"type:text" json:"reason"`
	DurationHours    int              `gorm:"not null;default:1" json:"duration_hours"`
	RequiresPassword bool             `gorm:"default:false" json:"requires_password"`
	RequiresKeys     bool             `gorm:"default:false" json:"requires_keys"`
	ConsoleURL       string           `gorm:"type:varchar(255)" json:"console_url,omitempty"`
	TempPassword     string           `gorm:"type:varchar(255)" json:"temp_password,omitempty"`
	TempAccessKey    string           `gorm:"type:varchar(255)" json:"temp_access_key,omitempty"`
	TempSecretKey    string           `gorm:"type:varchar(255)" json:"temp_secret_key,omitempty"`
	ApprovedAt       *time.Time       `json:"approved_at,omitempty"`
	ExpiresAt        *time.Time       `json:"expires_at,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}
