package models

import "time"

type PermintaanTTD struct {
	ID          int       `json:"id"`
	DokumenID   int       `json:"dokumen_id"`
	UserID      int       `json:"user_id"`
	Urutan      int       `json:"urutan"`
	PageNumber  int       `json:"page_number"`
	KoordinatX  *float64  `json:"koordinat_x,omitempty"`
	KoordinatY  *float64  `json:"koordinat_y,omitempty"`
	Width       *float64  `json:"width,omitempty"`
	Height      *float64  `json:"height,omitempty"`
	Status      string    `json:"status"`
	AlasanTolak *string   `json:"alasan_tolak,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
