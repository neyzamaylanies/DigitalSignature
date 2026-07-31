package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"digital-signature-api/db"
	"digital-signature-api/middleware"
	"digital-signature-api/models"
)

const (
	maxSignatureImageSize = 2 << 20 // 2 MB
	signatureImageDir     = "uploads/signatures"
)

var allowedTipeTandaTangan = map[string]bool{
	"signature": true,
	"paraf":     true,
}

func randomImageName(originalName string) (string, error) {
	ext := strings.ToLower(filepath.Ext(originalName))

	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes) + ext, nil
}

func validateSignatureImage(file multipart.File, header *multipart.FileHeader) (int, error) {
	if header.Size > maxSignatureImageSize {
		return http.StatusRequestEntityTooLarge, errors.New("ukuran gambar maksimal 2 MB")
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return http.StatusBadRequest, errors.New("hanya file PNG, JPG, atau JPEG yang diperbolehkan")
	}

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return http.StatusInternalServerError, errors.New("failed to read uploaded file")
	}

	contentType := http.DetectContentType(buffer[:n])
	if contentType != "image/png" && contentType != "image/jpeg" {
		return http.StatusBadRequest, errors.New("file bukan gambar PNG/JPEG yang valid")
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return http.StatusInternalServerError, errors.New("failed to read uploaded file")
	}

	return http.StatusOK, nil
}

func saveSignatureImageFile(file multipart.File, originalName string) (string, error) {
	if err := os.MkdirAll(signatureImageDir, 0755); err != nil {
		return "", err
	}

	fileName, err := randomImageName(originalName)
	if err != nil {
		return "", err
	}

	storedPath := filepath.Join(signatureImageDir, fileName)

	destination, err := os.Create(storedPath)
	if err != nil {
		return "", err
	}

	_, copyErr := io.Copy(destination, file)
	closeErr := destination.Close()
	if copyErr != nil {
		_ = os.Remove(storedPath)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(storedPath)
		return "", closeErr
	}

	return filepath.ToSlash(storedPath), nil
}

func UploadTandaTangan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	defer drainBody(r)

	r.Body = http.MaxBytesReader(w, r.Body, maxSignatureImageSize)
	if err := r.ParseMultipartForm(maxSignatureImageSize); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "Ukuran gambar maksimal 2 MB", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}

	tipe := strings.TrimSpace(r.FormValue("tipe"))
	if tipe == "" {
		http.Error(w, "Tipe is required", http.StatusBadRequest)
		return
	}
	if !allowedTipeTandaTangan[tipe] {
		http.Error(w, "Tipe must be signature or paraf", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File gambar tidak ditemukan di request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if statusCode, err := validateSignatureImage(file, header); err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	storedPath, err := saveSignatureImageFile(file, header.Filename)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	var tandaTangan models.TandaTangan
	err = db.DB.QueryRow(`
		INSERT INTO tanda_tangan (user_id, tipe, file_path)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, tipe, file_path, created_at`,
		userID, tipe, storedPath,
	).Scan(
		&tandaTangan.ID,
		&tandaTangan.UserID,
		&tandaTangan.Tipe,
		&tandaTangan.FilePath,
		&tandaTangan.CreatedAt,
	)
	if err != nil {
		_ = os.Remove(storedPath)
		http.Error(w, "Failed to save signature image data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Signature image uploaded successfully",
		"tanda_tangan": tandaTangan,
	})
}

func ListTandaTangan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, user_id, tipe, file_path, created_at
		FROM tanda_tangan
		WHERE user_id = $1
		ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		http.Error(w, "Failed to retrieve signature images", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tandaTanganList := make([]models.TandaTangan, 0)
	for rows.Next() {
		var t models.TandaTangan
		if err := rows.Scan(&t.ID, &t.UserID, &t.Tipe, &t.FilePath, &t.CreatedAt); err != nil {
			http.Error(w, "Failed to read signature image data", http.StatusInternalServerError)
			return
		}
		tandaTanganList = append(tandaTanganList, t)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read signature image data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Signature images retrieved successfully",
		"tanda_tangan": tandaTanganList,
	})
}

func getOwnedTandaTangan(id int, userID int) (models.TandaTangan, bool, error) {
	var t models.TandaTangan
	err := db.DB.QueryRow(`
		SELECT id, user_id, tipe, file_path, created_at
		FROM tanda_tangan
		WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.UserID, &t.Tipe, &t.FilePath, &t.CreatedAt)

	if err == sql.ErrNoRows {
		return t, false, nil
	}
	if err != nil {
		return t, false, err
	}

	if t.UserID != userID {
		return t, false, nil
	}

	return t, true, nil
}

func GetTandaTanganDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid tanda tangan id", http.StatusBadRequest)
		return
	}

	tandaTangan, found, err := getOwnedTandaTangan(id, userID)
	if err != nil {
		http.Error(w, "Failed to retrieve signature image", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Signature image not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Signature image detail retrieved successfully",
		"tanda_tangan": tandaTangan,
	})
}

func PreviewTandaTangan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid tanda tangan id", http.StatusBadRequest)
		return
	}

	tandaTangan, found, err := getOwnedTandaTangan(id, userID)
	if err != nil {
		http.Error(w, "Failed to retrieve signature image", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Signature image not found", http.StatusNotFound)
		return
	}

	if _, err := os.Stat(tandaTangan.FilePath); err != nil {
		http.Error(w, "Signature image file not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, tandaTangan.FilePath)
}

func UpdateTandaTangan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid tanda tangan id", http.StatusBadRequest)
		return
	}

	existing, found, err := getOwnedTandaTangan(id, userID)
	if err != nil {
		http.Error(w, "Failed to retrieve signature image", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Signature image not found", http.StatusNotFound)
		return
	}

	defer drainBody(r)

	r.Body = http.MaxBytesReader(w, r.Body, maxSignatureImageSize)
	if err := r.ParseMultipartForm(maxSignatureImageSize); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "Ukuran gambar maksimal 2 MB", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}

	tipe := strings.TrimSpace(r.FormValue("tipe"))
	if tipe == "" {
		tipe = existing.Tipe
	} else if !allowedTipeTandaTangan[tipe] {
		http.Error(w, "Tipe must be signature or paraf", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File gambar tidak ditemukan di request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if statusCode, err := validateSignatureImage(file, header); err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	storedPath, err := saveSignatureImageFile(file, header.Filename)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	var updated models.TandaTangan
	err = db.DB.QueryRow(`
		UPDATE tanda_tangan
		SET tipe = $1, file_path = $2
		WHERE id = $3
		RETURNING id, user_id, tipe, file_path, created_at`,
		tipe, storedPath, existing.ID,
	).Scan(
		&updated.ID,
		&updated.UserID,
		&updated.Tipe,
		&updated.FilePath,
		&updated.CreatedAt,
	)
	if err != nil {
		_ = os.Remove(storedPath)
		http.Error(w, "Failed to update signature image data", http.StatusInternalServerError)
		return
	}

	if existing.FilePath != "" && existing.FilePath != updated.FilePath {
		_ = os.Remove(existing.FilePath)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Signature image updated successfully",
		"tanda_tangan": updated,
	})
}

func DeleteTandaTangan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid tanda tangan id", http.StatusBadRequest)
		return
	}

	tandaTangan, found, err := getOwnedTandaTangan(id, userID)
	if err != nil {
		http.Error(w, "Failed to retrieve signature image", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Signature image not found", http.StatusNotFound)
		return
	}

	if _, err := db.DB.Exec(`DELETE FROM tanda_tangan WHERE id = $1`, tandaTangan.ID); err != nil {
		http.Error(w, "Failed to delete signature image", http.StatusInternalServerError)
		return
	}

	if tandaTangan.FilePath != "" {
		_ = os.Remove(tandaTangan.FilePath)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Signature image deleted successfully",
	})
}
