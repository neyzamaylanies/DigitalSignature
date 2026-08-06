package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"digital-signature-api/db"
	"digital-signature-api/middleware"
	"digital-signature-api/utils"
)

// execer is satisfied by both *sql.DB and *sql.Tx, so insertLogAktivitas can
// be used both inside a transaction (approve/reject/self-sign) and outside
// one (upload, download, certificate management).
type execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// insertLogAktivitas records an entry into log_aktivitas. dokumenID is a
// pointer because the column has a foreign key to `dokumen` and some
// actions (certificate create/revoke) aren't tied to any document.
func insertLogAktivitas(exec execer, userID int, dokumenID *int, aksi, keterangan string) error {
	_, err := exec.Exec(`
		INSERT INTO log_aktivitas (user_id, dokumen_id, aksi, keterangan)
		VALUES ($1, $2, $3, NULLIF($4, ''))
	`, userID, dokumenID, aksi, keterangan)
	return err
}

// randomFinalPDFName generates the output filename for a stamped PDF.
func randomFinalPDFName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("final_%s.pdf", hex.EncodeToString(b)), nil
}

// getActiveSertifikatID returns the user's active certificate id, or
// sql.ErrNoRows if they don't have one.
func getActiveSertifikatID(q interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}, userID int) (int, error) {
	var id int
	err := q.QueryRow(
		`SELECT id FROM sertifikat WHERE user_id = $1 AND status = 'active' ORDER BY created_at DESC LIMIT 1`,
		userID,
	).Scan(&id)
	return id, err
}

// getOwnedTandaTanganPath returns the file_path of a tanda_tangan image,
// verifying it belongs to userID. Returns sql.ErrNoRows if not found or not owned.
func getOwnedTandaTanganPath(q interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}, tandaTanganID, userID int) (string, error) {
	var ownerID int
	var filePath string
	err := q.QueryRow(
		`SELECT user_id, file_path FROM tanda_tangan WHERE id = $1`,
		tandaTanganID,
	).Scan(&ownerID, &filePath)
	if err != nil {
		return "", err
	}
	if ownerID != userID {
		return "", sql.ErrNoRows
	}
	return filePath, nil
}

func insertTransaksiSertifikat(tx *sql.Tx, permintaanTTDID *int, dokumenID, userID int, sertifikatID *int, aksi, alasanTolak, fileResultPath string) error {
	_, err := tx.Exec(`
		INSERT INTO transaksi_sertifikat (permintaan_ttd_id, dokumen_id, user_id, sertifikat_id, aksi, alasan_tolak, file_result_path)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''))`,
		permintaanTTDID, dokumenID, userID, sertifikatID, aksi, alasanTolak, fileResultPath,
	)
	return err
}

// Approve (Cross Signing)

type setujuiInput struct {
	TandaTanganID int `json:"tanda_tangan_id"`
}

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

	var input setujuiInput
	_ = json.NewDecoder(r.Body).Decode(&input)
	if input.TandaTanganID <= 0 {
		http.Error(w, "tanda_tangan_id is required: pick which signature image to stamp with", http.StatusBadRequest)
		return
	}

	if err := expireDueCertificates(db.DB); err != nil {
		http.Error(w, "Failed to check certificate", http.StatusInternalServerError)
		return
	}

	sertifikatID, err := getActiveSertifikatID(db.DB, userID)
	if err == sql.ErrNoRows {
		http.Error(w, "Anda harus memiliki sertifikat digital aktif sebelum bisa menandatangani dokumen", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Failed to check certificate", http.StatusInternalServerError)
		return
	}

	tandaTanganPath, err := getOwnedTandaTanganPath(db.DB, input.TandaTanganID, userID)
	if err == sql.ErrNoRows {
		http.Error(w, "Signature image not found or does not belong to you", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, "Failed to check signature image", http.StatusInternalServerError)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var dokumenID, ownerUserID, urutan, pageNumber int
	var status string
	var koordinatX, koordinatY, width, height *float64
	err = tx.QueryRow(`
		SELECT dokumen_id, user_id, urutan, page_number, koordinat_x, koordinat_y, width, height, status
		FROM permintaan_ttd
		WHERE id = $1
		FOR UPDATE`,
		permintaanID,
	).Scan(&dokumenID, &ownerUserID, &urutan, &pageNumber, &koordinatX, &koordinatY, &width, &height, &status)
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
	if koordinatX == nil || koordinatY == nil || width == nil || height == nil {
		http.Error(w, "Signing request is missing position data (koordinat_x/koordinat_y/width/height)", http.StatusBadRequest)
		return
	}

	var belumGiliran bool
	if err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM permintaan_ttd
			WHERE dokumen_id = $1 AND urutan < $2 AND status <> 'selesai'
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

	var filePath string
	var finalFilePath sql.NullString
	if err := tx.QueryRow(`SELECT file_path, final_file_path FROM dokumen WHERE id = $1`, dokumenID).
		Scan(&filePath, &finalFilePath); err != nil {
		http.Error(w, "Failed to fetch document file", http.StatusInternalServerError)
		return
	}
	sourcePath := filePath
	if finalFilePath.Valid && finalFilePath.String != "" {
		sourcePath = finalFilePath.String
	}

	outputName, err := randomFinalPDFName()
	if err != nil {
		http.Error(w, "Failed to generate output file name", http.StatusInternalServerError)
		return
	}
	outputPath := filepath.Join(documentDir, outputName)

	if err := utils.StampSignatureOnPDF(sourcePath, outputPath, tandaTanganPath, pageNumber, *koordinatX, *koordinatY, *width, *height); err != nil {
		http.Error(w, fmt.Sprintf("Failed to stamp signature on PDF: %v", err), http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(
		`UPDATE permintaan_ttd SET status = 'selesai' WHERE id = $1`,
		permintaanID,
	); err != nil {
		http.Error(w, "Failed to update signing request", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(
		`UPDATE dokumen SET final_file_path = $1 WHERE id = $2`,
		filepath.ToSlash(outputPath), dokumenID,
	); err != nil {
		http.Error(w, "Failed to update document file", http.StatusInternalServerError)
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
			`UPDATE dokumen SET status = 'selesai' WHERE id = $1`,
			dokumenID,
		); err != nil {
			http.Error(w, "Failed to finalize document", http.StatusInternalServerError)
			return
		}
	}

	permintaanIDCopy := permintaanID
	sertifikatIDCopy := sertifikatID
	if err := insertTransaksiSertifikat(tx, &permintaanIDCopy, dokumenID, userID, &sertifikatIDCopy, "approve", "", filepath.ToSlash(outputPath)); err != nil {
		http.Error(w, "Failed to record certificate transaction", http.StatusInternalServerError)
		return
	}

	if err := insertLogAktivitas(
		tx, userID, &dokumenID, "sign",
		"User menyetujui dan menandatangani permintaan dokumen",
	); err != nil {
		http.Error(w, "Failed to save activity log", http.StatusInternalServerError)
		return
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

// Reject (Cross Signing)

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
			WHERE dokumen_id = $1 AND urutan < $2 AND status <> 'selesai'
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

	permintaanIDCopy := permintaanID
	if err := insertTransaksiSertifikat(tx, &permintaanIDCopy, dokumenID, userID, nil, "reject", input.AlasanTolak, ""); err != nil {
		http.Error(w, "Failed to record certificate transaction", http.StatusInternalServerError)
		return
	}

	if err := insertLogAktivitas(
		tx, userID, &dokumenID, "reject",
		"Permintaan tanda tangan ditolak: "+input.AlasanTolak,
	); err != nil {
		http.Error(w, "Failed to save activity log", http.StatusInternalServerError)
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

type tandaTanganiSendiriInput struct {
	PageNumber int     `json:"page_number"`
	KoordinatX float64 `json:"koordinat_x"`
	KoordinatY float64 `json:"koordinat_y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
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

	var input tandaTanganiSendiriInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if input.PageNumber <= 0 {
		http.Error(w, "page_number must be a positive number", http.StatusBadRequest)
		return
	}
	if input.KoordinatX < 0 || input.KoordinatX > 100 || input.KoordinatY < 0 || input.KoordinatY > 100 {
		http.Error(w, "koordinat_x and koordinat_y must be between 0 and 100", http.StatusBadRequest)
		return
	}
	if input.Width <= 0 || input.Width > 100 || input.Height <= 0 || input.Height > 100 {
		http.Error(w, "width and height must be greater than 0 and not exceed 100", http.StatusBadRequest)
		return
	}
	if input.KoordinatX+input.Width > 100 || input.KoordinatY+input.Height > 100 {
		http.Error(w, "Signature area exceeds page boundaries", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var ownerUserID int
	var jenis, status string
	err = tx.QueryRow(`
		SELECT user_id, jenis, status
		FROM dokumen
		WHERE id = $1
		FOR UPDATE`,
		dokumenID,
	).Scan(&ownerUserID, &jenis, &status)
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
	if jenis != "self" {
		http.Error(w, "This document is not a self-signing document", http.StatusBadRequest)
		return
	}
	if status != "draft" {
		http.Error(w, "Document has already started or finished the signing process", http.StatusBadRequest)
		return
	}

	var permintaanID int
	err = tx.QueryRow(`
		INSERT INTO permintaan_ttd (dokumen_id, user_id, urutan, page_number, koordinat_x, koordinat_y, width, height, status)
		VALUES ($1, $2, 1, $3, $4, $5, $6, $7, 'menunggu')
		RETURNING id`,
		dokumenID, userID, input.PageNumber, input.KoordinatX, input.KoordinatY, input.Width, input.Height,
	).Scan(&permintaanID)
	if err != nil {
		http.Error(w, "Failed to create signing request", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(
		`UPDATE dokumen SET status = 'proses_ttd' WHERE id = $1`,
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
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Signing request initiated. Call POST /api/permintaan-ttd/{id}/setujui to finish signing.",
		"dokumen_id":    dokumenID,
		"permintaan_id": permintaanID,
		"status":        "proses_ttd",
	})
}
