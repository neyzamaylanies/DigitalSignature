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

var validPermintaanStatus = map[string]bool{
	"menunggu":  true,
	"disetujui": true,
	"ditolak":   true,
}

type PermintaanTTDListItem struct {
	ID          int       `json:"id"`
	DokumenID   int       `json:"dokumen_id"`
	Urutan      int       `json:"urutan"`
	PageNumber  int       `json:"page_number"`
	Status      string    `json:"status"`
	AlasanTolak *string   `json:"alasan_tolak,omitempty"`
	CreatedAt   time.Time `json:"created_at"`

	DokumenJudul     string  `json:"dokumen_judul"`
	DokumenJenis     string  `json:"dokumen_jenis"`
	DokumenDeskripsi *string `json:"dokumen_deskripsi,omitempty"`
	PengajuID        int     `json:"pengaju_id"`
	PengajuNama      string  `json:"pengaju_nama"`
}

func ListPermintaanTTD(w http.ResponseWriter, r *http.Request) {
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
	if statusFilter != "" && !validPermintaanStatus[statusFilter] {
		http.Error(w, "Status filter is not valid", http.StatusBadRequest)
		return
	}

	query := `
		SELECT
			pt.id, pt.dokumen_id, pt.urutan, pt.page_number, pt.status,
			pt.alasan_tolak, pt.created_at,
			d.judul, d.jenis, d.deskripsi,
			d.user_id, u.name
		FROM permintaan_ttd pt
		JOIN dokumen d ON d.id = pt.dokumen_id
		JOIN users u ON u.id = d.user_id
		WHERE pt.user_id = $1`

	args := []interface{}{userID}
	if statusFilter != "" {
		query += " AND pt.status = $2"
		args = append(args, statusFilter)
	}
	query += " ORDER BY pt.created_at DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to fetch signing requests", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := make([]PermintaanTTDListItem, 0)
	for rows.Next() {
		var item PermintaanTTDListItem
		var alasanTolak, dokumenDeskripsi sql.NullString

		if err := rows.Scan(
			&item.ID, &item.DokumenID, &item.Urutan, &item.PageNumber, &item.Status,
			&alasanTolak, &item.CreatedAt,
			&item.DokumenJudul, &item.DokumenJenis, &dokumenDeskripsi,
			&item.PengajuID, &item.PengajuNama,
		); err != nil {
			http.Error(w, "Failed to read signing requests", http.StatusInternalServerError)
			return
		}

		if alasanTolak.Valid {
			item.AlasanTolak = &alasanTolak.String
		}
		if dokumenDeskripsi.Valid {
			item.DokumenDeskripsi = &dokumenDeskripsi.String
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read signing requests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"permintaan_ttd": items,
	})
}

type SignerProgress struct {
	UserID int    `json:"user_id"`
	Nama   string `json:"nama"`
	Urutan int    `json:"urutan"`
	Status string `json:"status"`
	Milik  bool   `json:"milik_saya"`
}

type PermintaanTTDDetail struct {
	ID          int       `json:"id"`
	DokumenID   int       `json:"dokumen_id"`
	Urutan      int       `json:"urutan"`
	PageNumber  int       `json:"page_number"`
	Width       *float64  `json:"width,omitempty"`
	Height      *float64  `json:"height,omitempty"`
	Status      string    `json:"status"`
	AlasanTolak *string   `json:"alasan_tolak,omitempty"`
	CreatedAt   time.Time `json:"created_at"`

	DokumenJudul     string  `json:"dokumen_judul"`
	DokumenJenis     string  `json:"dokumen_jenis"`
	DokumenDeskripsi *string `json:"dokumen_deskripsi,omitempty"`
	DokumenPesan     *string `json:"dokumen_pesan,omitempty"`
	DokumenFilePath  string  `json:"dokumen_file_path"`
	PengajuID        int     `json:"pengaju_id"`
	PengajuNama      string  `json:"pengaju_nama"`

	PenandaTangan []SignerProgress `json:"penanda_tangan"`
	GiliranSaya   bool             `json:"giliran_saya"`
}

func GetPermintaanTTDDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idParam := r.PathValue("id")
	permintaanID, err := strconv.Atoi(idParam)
	if err != nil || permintaanID <= 0 {
		http.Error(w, "Invalid permintaan_ttd id", http.StatusBadRequest)
		return
	}

	var detail PermintaanTTDDetail
	var alasanTolak, dokumenDeskripsi, dokumenPesan sql.NullString
	var ownerUserID int

	err = db.DB.QueryRow(`
		SELECT
			pt.id, pt.dokumen_id, pt.user_id, pt.urutan, pt.page_number, pt.width, pt.height,
			pt.status, pt.alasan_tolak, pt.created_at,
			d.judul, d.jenis, d.deskripsi, d.pesan, d.file_path,
			d.user_id, u.name
		FROM permintaan_ttd pt
		JOIN dokumen d ON d.id = pt.dokumen_id
		JOIN users u ON u.id = d.user_id
		WHERE pt.id = $1`,
		permintaanID,
	).Scan(
		&detail.ID, &detail.DokumenID, &ownerUserID, &detail.Urutan, &detail.PageNumber,
		&detail.Width, &detail.Height,
		&detail.Status, &alasanTolak, &detail.CreatedAt,
		&detail.DokumenJudul, &detail.DokumenJenis, &dokumenDeskripsi, &dokumenPesan, &detail.DokumenFilePath,
		&detail.PengajuID, &detail.PengajuNama,
	)
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

	if alasanTolak.Valid {
		detail.AlasanTolak = &alasanTolak.String
	}
	if dokumenDeskripsi.Valid {
		detail.DokumenDeskripsi = &dokumenDeskripsi.String
	}
	if dokumenPesan.Valid {
		detail.DokumenPesan = &dokumenPesan.String
	}

	rows, err := db.DB.Query(`
		SELECT pt.user_id, u.name, pt.urutan, pt.status
		FROM permintaan_ttd pt
		JOIN users u ON u.id = pt.user_id
		WHERE pt.dokumen_id = $1
		ORDER BY pt.urutan ASC`,
		detail.DokumenID,
	)
	if err != nil {
		http.Error(w, "Failed to fetch signer progress", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	signers := make([]SignerProgress, 0)
	for rows.Next() {
		var signer SignerProgress
		if err := rows.Scan(&signer.UserID, &signer.Nama, &signer.Urutan, &signer.Status); err != nil {
			http.Error(w, "Failed to read signer progress", http.StatusInternalServerError)
			return
		}
		signer.Milik = signer.UserID == userID
		signers = append(signers, signer)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read signer progress", http.StatusInternalServerError)
		return
	}
	detail.PenandaTangan = signers

	giliran := detail.Status == "menunggu"
	if giliran {
		for _, signer := range signers {
			if signer.Urutan < detail.Urutan && signer.Status != "disetujui" {
				giliran = false
				break
			}
		}
	}
	detail.GiliranSaya = giliran

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}
