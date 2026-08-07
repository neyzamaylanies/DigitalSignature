package models

import "time"

type LogAktivitas struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	UserName   *string   `json:"user_name,omitempty"`
	DokumenID  *int      `json:"dokumen_id,omitempty"`
	Aksi       string    `json:"aksi"`
	Keterangan *string   `json:"keterangan,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
