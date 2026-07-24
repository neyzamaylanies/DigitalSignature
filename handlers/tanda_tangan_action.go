package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"digital-signature-api/db"
	"digital-signature-api/middleware"
)

func SetujuiPermintaanTTD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	permintaanID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || permintaanID <= 0 {
		http.Error(w, "Invalid permintaan_ttd id", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var dokumenID, ownerUserID, urutan int
	var status string
	err = tx.QueryRow(`
		SELECT dokumen_id, user_id, urutan, status
		FROM permintaan_ttd
		WHERE id = $1
		FOR UPDATE`,
		permintaanID,
	).Scan(&dokumenID, &ownerUserID, &urutan, &status)
	if err == sql.ErrNoRows {
		http.Error(w, "Signing request not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to fetch signing request", http.StatusInternalServerError)
		return
	}

	if ownerUserID != userID {
		http.Error(w, "Forbidden: this signing request does not belong to you", http.StatusForbidden)
		return
	}
	if status != "menunggu" {
		http.Error(w, "Signing request is no longer pending", http.StatusBadRequest)
		return
	}

	var belumGiliran bool
	if err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM permintaan_ttd
			WHERE dokumen_id = $1 AND urutan < $2 AND status <> 'disetujui'
		)`,
		dokumenID, urutan,
	).Scan(&belumGiliran); err != nil {
		http.Error(w, "Failed to validate signing order", http.StatusInternalServerError)
		return
	}
	if belumGiliran {
		http.Error(w, "Belum giliran Anda untuk menandatangani dokumen ini", http.StatusForbidden)
		return
	}

	if _, err := tx.Exec(
		`UPDATE permintaan_ttd SET status = 'disetujui' WHERE id = $1`,
		permintaanID,
	); err != nil {
		http.Error(w, "Failed to update signing request", http.StatusInternalServerError)
		return
	}

	var masihAdaMenunggu bool
	if err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM permintaan_ttd
			WHERE dokumen_id = $1 AND status = 'menunggu'
		)`,
		dokumenID,
	).Scan(&masihAdaMenunggu); err != nil {
		http.Error(w, "Failed to check remaining signers", http.StatusInternalServerError)
		return
	}

	dokumenSelesai := !masihAdaMenunggu
	if dokumenSelesai {
		if _, err := tx.Exec(
			`UPDATE dokumen SET status = 'selesai', final_file_path = file_path WHERE id = $1`,
			dokumenID,
		); err != nil {
			http.Error(w, "Failed to finalize document", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to save changes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Signing request approved successfully",
		"permintaan_id":   permintaanID,
		"dokumen_id":      dokumenID,
		"dokumen_selesai": dokumenSelesai,
	})
}

type tolakInput struct {
	AlasanTolak string `json:"alasan_tolak"`
}

func TolakPermintaanTTD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	permintaanID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || permintaanID <= 0 {
		http.Error(w, "Invalid permintaan_ttd id", http.StatusBadRequest)
		return
	}

	var input tolakInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	input.AlasanTolak = strings.TrimSpace(input.AlasanTolak)
	if input.AlasanTolak == "" {
		http.Error(w, "alasan_tolak is required", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var dokumenID, ownerUserID, urutan int
	var status string
	err = tx.QueryRow(`
		SELECT dokumen_id, user_id, urutan, status
		FROM permintaan_ttd
		WHERE id = $1
		FOR UPDATE`,
		permintaanID,
	).Scan(&dokumenID, &ownerUserID, &urutan, &status)
	if err == sql.ErrNoRows {
		http.Error(w, "Signing request not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to fetch signing request", http.StatusInternalServerError)
		return
	}

	if ownerUserID != userID {
		http.Error(w, "Forbidden: this signing request does not belong to you", http.StatusForbidden)
		return
	}
	if status != "menunggu" {
		http.Error(w, "Signing request is no longer pending", http.StatusBadRequest)
		return
	}

	var belumGiliran bool
	if err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM permintaan_ttd
			WHERE dokumen_id = $1 AND urutan < $2 AND status <> 'disetujui'
		)`,
		dokumenID, urutan,
	).Scan(&belumGiliran); err != nil {
		http.Error(w, "Failed to validate signing order", http.StatusInternalServerError)
		return
	}
	if belumGiliran {
		http.Error(w, "Belum giliran Anda untuk memproses dokumen ini", http.StatusForbidden)
		return
	}

	if _, err := tx.Exec(
		`UPDATE permintaan_ttd SET status = 'ditolak', alasan_tolak = $1 WHERE id = $2`,
		input.AlasanTolak, permintaanID,
	); err != nil {
		http.Error(w, "Failed to update signing request", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(
		`UPDATE dokumen SET status = 'ditolak' WHERE id = $1`,
		dokumenID,
	); err != nil {
		http.Error(w, "Failed to update document status", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to save changes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Signing request rejected successfully",
		"permintaan_id": permintaanID,
		"dokumen_id":    dokumenID,
	})
}

func TandaTanganiSendiri(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	dokumenID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || dokumenID <= 0 {
		http.Error(w, "Invalid dokumen id", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var ownerUserID int
	var tipe, status string
	err = tx.QueryRow(`
		SELECT user_id, tipe, status
		FROM dokumen
		WHERE id = $1
		FOR UPDATE`,
		dokumenID,
	).Scan(&ownerUserID, &tipe, &status)
	if err == sql.ErrNoRows {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to fetch document", http.StatusInternalServerError)
		return
	}

	if ownerUserID != userID {
		http.Error(w, "Forbidden: this document does not belong to you", http.StatusForbidden)
		return
	}
	if tipe != "sendiri" {
		http.Error(w, "This document is not a self-signing document", http.StatusBadRequest)
		return
	}
	if status != "draft" {
		http.Error(w, "Document is not in draft status", http.StatusBadRequest)
		return
	}

	if _, err := tx.Exec(
		`UPDATE dokumen SET status = 'selesai', final_file_path = file_path WHERE id = $1`,
		dokumenID,
	); err != nil {
		http.Error(w, "Failed to finalize document", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to save changes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Document signed successfully",
		"dokumen_id": dokumenID,
	})
}