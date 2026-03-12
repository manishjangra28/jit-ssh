package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Team struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// AgentToken is a pre-shared secret that authenticates an agent against the control plane.
// Admin generates it, copies it into the agent config file once.
type AgentToken struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Token       string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"` // never in JSON
	Label       string     `gorm:"type:varchar(255)" json:"label"` // human-readable name
	ServerID    *uuid.UUID `gorm:"type:uuid" json:"server_id,omitempty"` // set after agent registers
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name         string     `gorm:"type:varchar(255);not null;default:''" json:"name"`
	Email        string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Role         string     `gorm:"type:varchar(50);not null;default:'developer'" json:"role"`
	Status       string     `gorm:"type:varchar(50);not null;default:'active'" json:"status"` // active, inactive
	PasswordHash string     `gorm:"type:text;default:''" json:"-"` // Never expose in JSON
	TeamID       *uuid.UUID `gorm:"type:uuid" json:"team_id,omitempty"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`

	Team *Team `gorm:"foreignKey:TeamID;constraint:OnDelete:SET NULL;" json:"team,omitempty"`
}

type Server struct {
	ID         uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Hostname   string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"hostname"`
	IP         string     `gorm:"type:varchar(45);not null" json:"ip"`
	InstanceID string     `gorm:"type:varchar(255)" json:"instance_id"`
	AgentID    string     `gorm:"type:varchar(255)" json:"agent_id"`
	Status     string     `gorm:"type:varchar(50);not null;default:'offline'" json:"status"`
	TeamID     *uuid.UUID `gorm:"type:uuid" json:"team_id,omitempty"`
	LastSeen   time.Time  `json:"last_seen"`

	Team *Team       `gorm:"foreignKey:TeamID;constraint:OnDelete:SET NULL;" json:"team,omitempty"`
	Tags []ServerTag `gorm:"foreignKey:ServerID" json:"tags"`
}

type ServerTag struct {
	ServerID uuid.UUID `gorm:"type:uuid;primaryKey" json:"server_id"`
	TagKey   string    `gorm:"type:varchar(100);primaryKey" json:"tag_key"`
	TagValue string    `gorm:"type:varchar(255);not null" json:"tag_value"`
}

type Cluster struct {
	ID   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	Type string    `gorm:"type:varchar(100)" json:"type"`
}

type AccessRequest struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	ServerID   uuid.UUID `gorm:"type:uuid;not null" json:"server_id"`
	PubKey     string    `gorm:"type:text;not null" json:"pub_key"`
	Sudo       bool      `gorm:"not null;default:false" json:"sudo"`
	Duration   string    `gorm:"type:interval;not null" json:"duration"`
	Status            string    `gorm:"type:varchar(50);not null;default:'pending'" json:"status"`
	RequestedPath     string    `gorm:"type:varchar(255)" json:"requested_path"`      // e.g. /home/ec2-user
	RequestedServices string    `gorm:"type:varchar(255)" json:"requested_services"` // e.g. docker,mysql
	ApprovedBy        uuid.UUID `gorm:"type:uuid" json:"approved_by,omitempty"`
	ExpiresAt         time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`

	User   User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"user"`
	Server Server `gorm:"foreignKey:ServerID;constraint:OnDelete:CASCADE;" json:"server"`
}

type AuditLog struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid" json:"user_id"`
	ServerID  uuid.UUID `gorm:"type:uuid" json:"server_id"`
	Action    string    `gorm:"type:varchar(255);not null" json:"action"`
	Timestamp time.Time `gorm:"autoCreateTime" json:"timestamp"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

func (t *Team) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return
}

func (s *Server) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}

func (c *Cluster) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return
}

func (a *AccessRequest) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}

func (al *AuditLog) BeforeCreate(tx *gorm.DB) (err error) {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}
	return
}
