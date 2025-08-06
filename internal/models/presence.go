package models

import (
	"errors"
	"time"
)

// PresenceStatus represents the presence status of a user
type PresenceStatus string

const (
	StatusOnline  PresenceStatus = "online"
	StatusAway    PresenceStatus = "away"
	StatusBusy    PresenceStatus = "busy"
	StatusOffline PresenceStatus = "offline"
)

// IsValid checks if the presence status is valid
func (ps PresenceStatus) IsValid() bool {
	switch ps {
	case StatusOnline, StatusAway, StatusBusy, StatusOffline:
		return true
	default:
		return false
	}
}

// Presence represents a user's presence information
type Presence struct {
	UserID    string         `json:"user_id"`
	Status    PresenceStatus `json:"status"`
	Message   string         `json:"message,omitempty"`
	LastSeen  time.Time      `json:"last_seen"`
	UpdatedAt time.Time      `json:"updated_at"`
	NodeID    string         `json:"node_id"`
	TTL       time.Duration  `json:"ttl,omitempty"`
}

// Validate validates the presence data
func (p *Presence) Validate() error {
	if p.UserID == "" {
		return errors.New("user_id is required")
	}
	if !p.Status.IsValid() {
		return errors.New("invalid status")
	}
	if p.NodeID == "" {
		return errors.New("node_id is required")
	}
	return nil
}

// IsExpired checks if the presence has expired based on TTL
func (p *Presence) IsExpired() bool {
	if p.TTL == 0 {
		return false // No TTL means never expires
	}
	return time.Since(p.UpdatedAt) > p.TTL
}

// PresenceResponse represents the API response format
type PresenceResponse struct {
	Success bool                `json:"success"`
	Data    map[string]Presence `json:"data,omitempty"`
	Error   string              `json:"error,omitempty"`
}
