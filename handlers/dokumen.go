package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"digital-signature-api/db"
	"digital-signature-api/middleware"
	"digital-signature-api/models"
)

const (
	maxUploadSize = 20 << 20 // 20 MiB
	documentDir   = "uploads/documents"
)

var allowedJenis = map[string]bool{
	"Akademik":      true,
	"Keuangan":      true,
	"SDM":           true,
	"Hukum & Legal": true,
	"Operasional":   true,
	"Kerjasama":     true,
	"Kemahasiswaan": true,
}

// UploadDokumen uploads a PDF owned by the authenticated user for self signing.
func UploadDokumen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "Ukuran file maksimal 20 MB", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}

	judul := strings.TrimSpace(r.FormValue("judul"))
	deskripsi := strings.TrimSpace(r.FormValue("deskripsi"))
	pesan := strings.TrimSpace(r.FormValue("pesan"))
	jenis := strings.TrimSpace(r.FormValue("jenis"))

	if judul == "" || jenis == "" {
		http.Error(w, "Judul and jenis are required", http.StatusBadRequest)
		return
	}
	if !allowedJenis[jenis] {
		http.Error(w, "Jenis dokumen is not valid", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File tidak ditemukan di request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if header.Size > maxUploadSize {
		http.Error(w, "Ukuran file maksimal 20 MB", http.StatusRequestEntityTooLarge)
		return
	}
	if strings.ToLower(filepath.Ext(header.Filename)) != ".pdf" {
		http.Error(w, "Hanya file PDF yang diperbolehkan", http.StatusBadRequest)
		return
	}

	fileHeader := make([]byte, 5)
	if _, err := io.ReadFull(file, fileHeader); err != nil || string(fileHeader) != "%PDF-" {
		http.Error(w, "File bukan PDF yang valid", http.StatusBadRequest)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "Failed to read uploaded file", http.StatusInternalServerError)
		return
	}

	if err := os.MkdirAll(documentDir, 0755); err != nil {
		http.Error(w, "Failed to prepare upload directory", http.StatusInternalServerError)
		return
	}

	fileName, err := randomPDFName()
	if err != nil {
		http.Error(w, "Failed to generate file name", http.StatusInternalServerError)
		return
	}
	storedPath := filepath.Join(documentDir, fileName)
	destination, err := os.Create(storedPath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	_, copyErr := io.Copy(destination, file)
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(storedPath)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	dokumen, err := insertDokumen(userID, judul, deskripsi, pesan, jenis, filepath.ToSlash(storedPath))
	if err != nil {
		_ = os.Remove(storedPath)
		http.Error(w, "Failed to save document data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Document uploaded successfully",
		"dokumen": dokumen,
	})
}

func insertDokumen(userID int, judul, deskripsi, pesan, jenis, filePath string) (models.Dokumen, error) {
	var dokumen models.Dokumen
	var deskripsiDB, pesanDB, finalFilePathDB sql.NullString

	err := db.DB.QueryRow(`
		INSERT INTO dokumen (user_id, judul, deskripsi, pesan, jenis, tipe, file_path, status)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, 'sendiri', $6, 'draft')
		RETURNING id, user_id, judul, deskripsi, pesan, jenis, tipe, file_path, final_file_path, status, created_at`,
		userID, judul, deskripsi, pesan, jenis, filePath,
	).Scan(
		&dokumen.ID,
		&dokumen.UserID,
		&dokumen.Judul,
		&deskripsiDB,
		&pesanDB,
		&dokumen.Jenis,
		&dokumen.Tipe,
		&dokumen.FilePath,
		&finalFilePathDB,
		&dokumen.Status,
		&dokumen.CreatedAt,
	)
	if err != nil {
		return models.Dokumen{}, err
	}

	if deskripsiDB.Valid {
		dokumen.Deskripsi = &deskripsiDB.String
	}
	if pesanDB.Valid {
		dokumen.Pesan = &pesanDB.String
	}
	if finalFilePathDB.Valid {
		dokumen.FinalFilePath = &finalFilePathDB.String
	}

	return dokumen, nil
}

func randomPDFName() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.pdf", hex.EncodeToString(bytes)), nil
}
