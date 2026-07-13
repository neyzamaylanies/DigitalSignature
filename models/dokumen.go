package models

import "time"

type Dokumen struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	Judul         string    `json:"judul"`
	Deskripsi     *string   `json:"deskripsi,omitempty"`
	Pesan         *string   `json:"pesan,omitempty"`
	Jenis         string    `json:"jenis"`
	Tipe          string    `json:"tipe"`
	FilePath      string    `json:"file_path"`
	FinalFilePath *string   `json:"final_file_path,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
