package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"digital-signature-api/db"
	"digital-signature-api/middleware"
)

var validDokumenStatus = map[string]bool{
	"draft":        true,
	"menunggu_ttd": true,
	"selesai":      true,
	"ditolak":      true,
}

type DokumenDiunggahItem struct {
	ID            int       `json:"id"`
	Judul         string    `json:"judul"`
	Jenis         string    `json:"jenis"`
	Deskripsi     *string   `json:"deskripsi,omitempty"`
	Tipe          string    `json:"tipe"`
	Status        string    `json:"status"`
	FilePath      string    `json:"file_path"`
	FinalFilePath *string   `json:"final_file_path,omitempty"`
	CreatedAt     time.Time `json:"created_at"`

	TotalPenandaTangan int `json:"total_penanda_tangan"`
	DisetujuiCount     int `json:"disetujui_count"`
	DitolakCount       int `json:"ditolak_count"`
	MenungguCount      int `json:"menunggu_count"`
}

// ListDokumenDiunggah returns all documents uploaded by the authenticated user
// (both tipe=sendiri and tipe=pihak_lain), with signer progress counts.
func ListDokumenDiunggah(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	if statusFilter != "" && !validDokumenStatus[statusFilter] {
		http.Error(w, "Status filter is not valid", http.StatusBadRequest)
		return
	}

	query := `
		SELECT
			d.id, d.judul, d.jenis, d.deskripsi, d.tipe, d.status,
			d.file_path, d.final_file_path, d.created_at,
			COUNT(pt.id) AS total_penanda_tangan,
			COUNT(*) FILTER (WHERE pt.status = 'disetujui') AS disetujui_count,
			COUNT(*) FILTER (WHERE pt.status = 'ditolak') AS ditolak_count,
			COUNT(*) FILTER (WHERE pt.status = 'menunggu') AS menunggu_count
		FROM dokumen d
		LEFT JOIN permintaan_ttd pt ON pt.dokumen_id = d.id
		WHERE d.user_id = $1`

	args := []interface{}{userID}
	if statusFilter != "" {
		query += " AND d.status = $2"
		args = append(args, statusFilter)
	}
	query += " GROUP BY d.id ORDER BY d.created_at DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to fetch documents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := make([]DokumenDiunggahItem, 0)
	for rows.Next() {
		var item DokumenDiunggahItem
		var deskripsi, finalFilePath sql.NullString

		if err := rows.Scan(
			&item.ID, &item.Judul, &item.Jenis, &deskripsi, &item.Tipe, &item.Status,
			&item.FilePath, &finalFilePath, &item.CreatedAt,
			&item.TotalPenandaTangan, &item.DisetujuiCount, &item.DitolakCount, &item.MenungguCount,
		); err != nil {
			http.Error(w, "Failed to read documents", http.StatusInternalServerError)
			return
		}

		if deskripsi.Valid {
			item.Deskripsi = &deskripsi.String
		}
		if finalFilePath.Valid {
			item.FinalFilePath = &finalFilePath.String
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read documents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"dokumen": items,
	})
}

// DokumenDiunggahDetail is the detail response for a single uploaded document,
// including full signer progress (only populated when tipe = pihak_lain).
type DokumenDiunggahDetail struct {
	ID            int       `json:"id"`
	Judul         string    `json:"judul"`
	Jenis         string    `json:"jenis"`
	Deskripsi     *string   `json:"deskripsi,omitempty"`
	Pesan         *string   `json:"pesan,omitempty"`
	Tipe          string    `json:"tipe"`
	Status        string    `json:"status"`
	FilePath      string    `json:"file_path"`
	FinalFilePath *string   `json:"final_file_path,omitempty"`
	CreatedAt     time.Time `json:"created_at"`

	PenandaTangan []SignerDetail `json:"penanda_tangan"`
}

type SignerDetail struct {
	UserID      int     `json:"user_id"`
	Nama        string  `json:"nama"`
	Urutan      int     `json:"urutan"`
	PageNumber  int     `json:"page_number"`
	Status      string  `json:"status"`
	AlasanTolak *string `json:"alasan_tolak,omitempty"`
}

// GetDokumenDiunggahDetail returns detail of one uploaded document owned by
// the authenticated user, along with the progress of every signer assigned to it.
func GetDokumenDiunggahDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	var detail DokumenDiunggahDetail
	var deskripsi, pesan, finalFilePath sql.NullString
	var ownerUserID int

	err = db.DB.QueryRow(`
		SELECT id, user_id, judul, jenis, deskripsi, pesan, tipe, status, file_path, final_file_path, created_at
		FROM dokumen
		WHERE id = $1`,
		dokumenID,
	).Scan(
		&detail.ID, &ownerUserID, &detail.Judul, &detail.Jenis, &deskripsi, &pesan,
		&detail.Tipe, &detail.Status, &detail.FilePath, &finalFilePath, &detail.CreatedAt,
	)
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

	if deskripsi.Valid {
		detail.Deskripsi = &deskripsi.String
	}
	if pesan.Valid {
		detail.Pesan = &pesan.String
	}
	if finalFilePath.Valid {
		detail.FinalFilePath = &finalFilePath.String
	}

	rows, err := db.DB.Query(`
		SELECT pt.user_id, u.name, pt.urutan, pt.page_number, pt.status, pt.alasan_tolak
		FROM permintaan_ttd pt
		JOIN users u ON u.id = pt.user_id
		WHERE pt.dokumen_id = $1
		ORDER BY pt.urutan ASC`,
		dokumenID,
	)
	if err != nil {
		http.Error(w, "Failed to fetch signer progress", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	signers := make([]SignerDetail, 0)
	for rows.Next() {
		var signer SignerDetail
		var alasanTolak sql.NullString
		if err := rows.Scan(&signer.UserID, &signer.Nama, &signer.Urutan, &signer.PageNumber, &signer.Status, &alasanTolak); err != nil {
			http.Error(w, "Failed to read signer progress", http.StatusInternalServerError)
			return
		}
		if alasanTolak.Valid {
			signer.AlasanTolak = &alasanTolak.String
		}
		signers = append(signers, signer)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read signer progress", http.StatusInternalServerError)
		return
	}
	detail.PenandaTangan = signers

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

type DokumenDitandatanganiItem struct {
	ID            int       `json:"id"`
	Judul         string    `json:"judul"`
	Jenis         string    `json:"jenis"`
	Tipe          string    `json:"tipe"`
	FilePath      string    `json:"file_path"`
	FinalFilePath *string   `json:"final_file_path,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// ListDokumenDitandatangani returns the history of documents owned by the
// authenticated user that have reached status = 'selesai', ready for download.
func ListDokumenDitandatangani(w http.ResponseWriter, r *http.Request) {
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
		SELECT id, judul, jenis, tipe, file_path, final_file_path, created_at
		FROM dokumen
		WHERE user_id = $1 AND status = 'selesai'
		ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		http.Error(w, "Failed to fetch documents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := make([]DokumenDitandatanganiItem, 0)
	for rows.Next() {
		var item DokumenDitandatanganiItem
		var finalFilePath sql.NullString

		if err := rows.Scan(
			&item.ID, &item.Judul, &item.Jenis, &item.Tipe, &item.FilePath, &finalFilePath, &item.CreatedAt,
		); err != nil {
			http.Error(w, "Failed to read documents", http.StatusInternalServerError)
			return
		}
		if finalFilePath.Valid {
			item.FinalFilePath = &finalFilePath.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read documents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"dokumen": items,
	})
}
