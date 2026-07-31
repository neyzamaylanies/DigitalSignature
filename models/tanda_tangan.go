package models

import "time"

type TandaTangan struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Tipe      string    `json:"tipe"` // signature | paraf
	FilePath  string    `json:"file_path"`
	CreatedAt time.Time `json:"created_at"`
}
