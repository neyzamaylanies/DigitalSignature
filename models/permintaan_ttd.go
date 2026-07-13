package models

import "time"

type PermintaanTTD struct {
	ID          int       `json:"id"`
	DokumenID   int       `json:"dokumen_id"`
	UserID      int       `json:"user_id"`
	Urutan      int       `json:"urutan"`
	PageNumber  int       `json:"page_number"`
	Width       float64   `json:"width"`
	Height      float64   `json:"height"`
	Status      string    `json:"status"`
	AlasanTolak string    `json:"alasan_tolak,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
