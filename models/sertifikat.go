package models

import "time"

type Sertifikat struct {
	ID           int        `json:"id"`
	UserID       int        `json:"user_id"`
	SerialNumber string     `json:"serial_number"`
	PublicKey    string     `json:"public_key"`
	Status       string     `json:"status"` // active | expired | revoked
	ValidFrom    *time.Time `json:"valid_from,omitempty"`
	ValidUntil   *time.Time `json:"valid_until,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
