package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPresenceStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status PresenceStatus
		want   bool
	}{
		{"online is valid", StatusOnline, true},
		{"away is valid", StatusAway, true},
		{"busy is valid", StatusBusy, true},
		{"offline is valid", StatusOffline, true},
		{"empty string is invalid", "", false},
		{"random string is invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("PresenceStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPresence_Validate(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		presence Presence
		wantErr  bool
	}{
		{
			name: "valid presence",
			presence: Presence{
				UserID:    "user123",
				Status:    StatusOnline,
				Message:   "Working",
				LastSeen:  now,
				UpdatedAt: now,
				NodeID:    "node1",
			},
			wantErr: false,
		},
		{
			name: "empty user ID",
			presence: Presence{
				Status:    StatusOnline,
				LastSeen:  now,
				UpdatedAt: now,
				NodeID:    "node1",
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			presence: Presence{
				UserID:    "user123",
				Status:    "invalid",
				LastSeen:  now,
				UpdatedAt: now,
				NodeID:    "node1",
			},
			wantErr: true,
		},
		{
			name: "empty node ID",
			presence: Presence{
				UserID:    "user123",
				Status:    StatusOnline,
				LastSeen:  now,
				UpdatedAt: now,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.presence.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Presence.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPresence_IsExpired(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		presence Presence
		want     bool
	}{
		{
			name: "not expired with future TTL",
			presence: Presence{
				UpdatedAt: now.Add(-30 * time.Second),
				TTL:       time.Minute,
			},
			want: false,
		},
		{
			name: "expired with past TTL",
			presence: Presence{
				UpdatedAt: now.Add(-2 * time.Minute),
				TTL:       time.Minute,
			},
			want: true,
		},
		{
			name: "no TTL means never expired",
			presence: Presence{
				UpdatedAt: now.Add(-24 * time.Hour),
				TTL:       0,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.presence.IsExpired(); got != tt.want {
				t.Errorf("Presence.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPresence_JSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	
	original := Presence{
		UserID:    "user123",
		Status:    StatusBusy,
		Message:   "In a meeting",
		LastSeen:  now,
		UpdatedAt: now,
		NodeID:    "center-node-1",
		TTL:       time.Hour,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Presence: %v", err)
	}

	// Unmarshal back
	var unmarshaled Presence
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Presence: %v", err)
	}

	// Compare
	if unmarshaled.UserID != original.UserID {
		t.Errorf("UserID mismatch: got %v, want %v", unmarshaled.UserID, original.UserID)
	}
	if unmarshaled.Status != original.Status {
		t.Errorf("Status mismatch: got %v, want %v", unmarshaled.Status, original.Status)
	}
	if unmarshaled.Message != original.Message {
		t.Errorf("Message mismatch: got %v, want %v", unmarshaled.Message, original.Message)
	}
	if !unmarshaled.LastSeen.Equal(original.LastSeen) {
		t.Errorf("LastSeen mismatch: got %v, want %v", unmarshaled.LastSeen, original.LastSeen)
	}
	if !unmarshaled.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("UpdatedAt mismatch: got %v, want %v", unmarshaled.UpdatedAt, original.UpdatedAt)
	}
	if unmarshaled.NodeID != original.NodeID {
		t.Errorf("NodeID mismatch: got %v, want %v", unmarshaled.NodeID, original.NodeID)
	}
}

func TestPresenceResponse_JSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	
	response := PresenceResponse{
		Success: true,
		Data: map[string]Presence{
			"user1": {
				UserID:    "user1",
				Status:    StatusOnline,
				LastSeen:  now,
				UpdatedAt: now,
				NodeID:    "node1",
			},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal PresenceResponse: %v", err)
	}

	var unmarshaled PresenceResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal PresenceResponse: %v", err)
	}

	if unmarshaled.Success != response.Success {
		t.Errorf("Success mismatch: got %v, want %v", unmarshaled.Success, response.Success)
	}
	if len(unmarshaled.Data) != len(response.Data) {
		t.Errorf("Data length mismatch: got %v, want %v", len(unmarshaled.Data), len(response.Data))
	}
}