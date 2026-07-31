package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"digital-signature-api/db"
	"digital-signature-api/middleware"
	"digital-signature-api/models"
)

func generateSerialNumber(userID int) string {
	return fmt.Sprintf("CERT-%d-%d", userID, time.Now().UnixNano())
}

func generatePublicKey(userID int) (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("SIMULATED-PUBLIC-KEY-USER-%d-%s", userID, hex.EncodeToString(randomBytes)), nil
}

func CreateSertifikat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var activeCount int
	if err := db.DB.QueryRow(
		`SELECT COUNT(*) FROM sertifikat WHERE user_id = $1 AND status = 'active'`,
		userID,
	).Scan(&activeCount); err != nil {
		http.Error(w, "Failed to check existing certificate", http.StatusInternalServerError)
		return
	}

	if activeCount > 0 {
		http.Error(w, "User already has an active certificate", http.StatusBadRequest)
		return
	}

	publicKey, err := generatePublicKey(userID)
	if err != nil {
		http.Error(w, "Failed to generate certificate key", http.StatusInternalServerError)
		return
	}

	serialNumber := generateSerialNumber(userID)
	validFrom := time.Now()
	validUntil := validFrom.AddDate(1, 0, 0)

	var sertifikat models.Sertifikat
	err = db.DB.QueryRow(`
		INSERT INTO sertifikat (user_id, serial_number, public_key, status, valid_from, valid_until)
		VALUES ($1, $2, $3, 'active', $4, $5)
		RETURNING id, user_id, serial_number, public_key, status, valid_from, valid_until, created_at`,
		userID, serialNumber, publicKey, validFrom, validUntil,
	).Scan(
		&sertifikat.ID,
		&sertifikat.UserID,
		&sertifikat.SerialNumber,
		&sertifikat.PublicKey,
		&sertifikat.Status,
		&sertifikat.ValidFrom,
		&sertifikat.ValidUntil,
		&sertifikat.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Failed to create certificate", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Certificate created successfully",
		"sertifikat": sertifikat,
	})
}

func ListSertifikat(w http.ResponseWriter, r *http.Request) {
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
		SELECT id, user_id, serial_number, public_key, status, valid_from, valid_until, created_at
		FROM sertifikat
		WHERE user_id = $1
		ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		http.Error(w, "Failed to retrieve certificates", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	sertifikatList := make([]models.Sertifikat, 0)
	for rows.Next() {
		var s models.Sertifikat
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.SerialNumber, &s.PublicKey,
			&s.Status, &s.ValidFrom, &s.ValidUntil, &s.CreatedAt,
		); err != nil {
			http.Error(w, "Failed to read certificate data", http.StatusInternalServerError)
			return
		}
		sertifikatList = append(sertifikatList, s)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read certificate data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Certificates retrieved successfully",
		"sertifikat": sertifikatList,
	})
}

func RevokeSertifikat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sertifikatID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || sertifikatID <= 0 {
		http.Error(w, "Invalid sertifikat id", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var ownerUserID int
	var status string
	err = tx.QueryRow(
		`SELECT user_id, status FROM sertifikat WHERE id = $1 FOR UPDATE`,
		sertifikatID,
	).Scan(&ownerUserID, &status)
	if err == sql.ErrNoRows {
		http.Error(w, "Certificate not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to fetch certificate", http.StatusInternalServerError)
		return
	}

	if ownerUserID != userID {
		http.Error(w, "Forbidden: this certificate does not belong to you", http.StatusForbidden)
		return
	}
	if status == "revoked" {
		http.Error(w, "Certificate is already revoked", http.StatusBadRequest)
		return
	}

	if _, err := tx.Exec(
		`UPDATE sertifikat SET status = 'revoked' WHERE id = $1`,
		sertifikatID,
	); err != nil {
		http.Error(w, "Failed to revoke certificate", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to save changes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Certificate revoked successfully",
		"sertifikat_id": sertifikatID,
	})
}
