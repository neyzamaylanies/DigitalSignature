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
	maxRequestUploadSize = 20 << 20 // 20 MB
	requestDocumentDir   = "uploads/documents"
)

type signerInput struct {
	UserID     int      `json:"user_id"`
	Urutan     int      `json:"urutan"`
	PageNumber int      `json:"page_number"`
	KoordinatX *float64 `json:"koordinat_x,omitempty"`
	KoordinatY *float64 `json:"koordinat_y,omitempty"`
	Width      *float64 `json:"width,omitempty"`
	Height     *float64 `json:"height,omitempty"`
}

// AjukanTandaTangan uploads a document and creates signing requests for other users.
func AjukanTandaTangan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pengajuID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	defer drainBody(r)

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestUploadSize)
	if err := r.ParseMultipartForm(maxRequestUploadSize); err != nil {
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
	tipe := strings.TrimSpace(r.FormValue("tipe"))
	penandaTanganJSON := r.FormValue("penanda_tangan")

	if judul == "" || tipe == "" || penandaTanganJSON == "" {
		http.Error(w, "Judul, tipe, and penanda_tangan are required", http.StatusBadRequest)
		return
	}
	if !allowedTipeDokumen[tipe] {
		http.Error(w, "Tipe dokumen is not valid", http.StatusBadRequest)
		return
	}

	var penandaTangan []signerInput
	if err := json.Unmarshal([]byte(penandaTanganJSON), &penandaTangan); err != nil || len(penandaTangan) == 0 {
		http.Error(w, "penanda_tangan must be a non-empty JSON array", http.StatusBadRequest)
		return
	}

	seenUser := make(map[int]bool)
	seenUrutan := make(map[int]bool)
	for _, signer := range penandaTangan {
		if signer.UserID <= 0 || signer.PageNumber <= 0 || signer.Urutan <= 0 {
			http.Error(w, "Each signer must have user_id, urutan, and page_number", http.StatusBadRequest)
			return
		}
		if signer.KoordinatX != nil && (*signer.KoordinatX < 0 || *signer.KoordinatX > 100) {
			http.Error(w, "koordinat_x must be between 0 and 100", http.StatusBadRequest)
			return
		}
		if signer.KoordinatY != nil && (*signer.KoordinatY < 0 || *signer.KoordinatY > 100) {
			http.Error(w, "koordinat_y must be between 0 and 100", http.StatusBadRequest)
			return
		}
		if signer.Width != nil && *signer.Width <= 0 {
			http.Error(w, "width must be greater than 0", http.StatusBadRequest)
			return
		}
		if signer.Height != nil && *signer.Height <= 0 {
			http.Error(w, "height must be greater than 0", http.StatusBadRequest)
			return
		}
		if signer.UserID == pengajuID {
			http.Error(w, "Pengaju cannot be a signer for request document", http.StatusBadRequest)
			return
		}
		if seenUser[signer.UserID] {
			http.Error(w, "A signer may only be added once", http.StatusBadRequest)
			return
		}
		if seenUrutan[signer.Urutan] {
			http.Error(w, "Each signer must have a unique urutan", http.StatusBadRequest)
			return
		}
		seenUser[signer.UserID] = true
		seenUrutan[signer.Urutan] = true
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File tidak ditemukan di request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if header.Size > maxRequestUploadSize {
		http.Error(w, "Ukuran file maksimal 20 MB", http.StatusRequestEntityTooLarge)
		return
	}
	if strings.ToLower(filepath.Ext(header.Filename)) != ".pdf" {
		http.Error(w, "Hanya file PDF yang diperbolehkan", http.StatusBadRequest)
		return
	}

	pdfHeader := make([]byte, 5)
	if _, err := io.ReadFull(file, pdfHeader); err != nil || string(pdfHeader) != "%PDF-" {
		http.Error(w, "File bukan PDF yang valid", http.StatusBadRequest)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "Failed to read uploaded file", http.StatusInternalServerError)
		return
	}

	if err := os.MkdirAll(requestDocumentDir, 0755); err != nil {
		http.Error(w, "Failed to prepare upload directory", http.StatusInternalServerError)
		return
	}

	fileName, err := randomRequestPDFName()
	if err != nil {
		http.Error(w, "Failed to generate file name", http.StatusInternalServerError)
		return
	}
	storedPath := filepath.Join(requestDocumentDir, fileName)
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

	dokumenID, requests, err := createDocumentRequest(
		pengajuID, judul, deskripsi, pesan, tipe, filepath.ToSlash(storedPath), penandaTangan,
	)
	if err != nil {
		_ = os.Remove(storedPath)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":        "Document request created successfully",
		"dokumen_id":     dokumenID,
		"jenis":          "request",
		"status":         "menunggu_ttd",
		"permintaan_ttd": requests,
	})
}

func createDocumentRequest(pengajuID int, judul, deskripsi, pesan, tipe, filePath string, signers []signerInput) (int, []models.PermintaanTTD, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to start transaction")
	}
	defer tx.Rollback()

	for _, signer := range signers {
		var userExists bool
		if err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", signer.UserID).Scan(&userExists); err != nil {
			return 0, nil, fmt.Errorf("failed to validate signer")
		}
		if !userExists {
			return 0, nil, fmt.Errorf("signer with user_id %d was not found", signer.UserID)
		}
	}

	var dokumenID int
	err = tx.QueryRow(`
		INSERT INTO dokumen (user_id, judul, deskripsi, pesan, tipe, jenis, file_path, status)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, 'request', $6, 'menunggu_ttd')
		RETURNING id`,
		pengajuID, judul, deskripsi, pesan, tipe, filePath,
	).Scan(&dokumenID)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to save document data")
	}

	requests := make([]models.PermintaanTTD, 0, len(signers))
	for _, signer := range signers {
		var request models.PermintaanTTD
		var alasanTolak sql.NullString
		err := tx.QueryRow(`
			INSERT INTO permintaan_ttd (dokumen_id, user_id, urutan, page_number, koordinat_x, koordinat_y, width, height, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'menunggu')
			RETURNING id, dokumen_id, user_id, urutan, page_number, koordinat_x, koordinat_y, width, height, status, alasan_tolak, created_at`,
			dokumenID, signer.UserID, signer.Urutan, signer.PageNumber, signer.KoordinatX, signer.KoordinatY, signer.Width, signer.Height,
		).Scan(
			&request.ID,
			&request.DokumenID,
			&request.UserID,
			&request.Urutan,
			&request.PageNumber,
			&request.KoordinatX,
			&request.KoordinatY,
			&request.Width,
			&request.Height,
			&request.Status,
			&alasanTolak,
			&request.CreatedAt,
		)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to save signing request")
		}
		if alasanTolak.Valid {
			request.AlasanTolak = &alasanTolak.String
		}
		requests = append(requests, request)
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, fmt.Errorf("failed to save signing requests")
	}

	return dokumenID, requests, nil
}

func randomRequestPDFName() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.pdf", hex.EncodeToString(bytes)), nil
}
