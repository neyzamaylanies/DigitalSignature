package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	SelesaiCount       int `json:"selesai_count"`
	DitolakCount       int `json:"ditolak_count"`
	MenungguCount      int `json:"menunggu_count"`
}

// ListDokumenDiunggah returns all documents uploaded by the authenticated user
// (both jenis=self and jenis=request), with signer progress counts.
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
			COUNT(*) FILTER (WHERE pt.status = 'selesai') AS selesai_count,
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
			&item.TotalPenandaTangan, &item.SelesaiCount, &item.DitolakCount, &item.MenungguCount,
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
// including full signer progress (only populated when jenis = request).
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
	UserID      int      `json:"user_id"`
	Nama        string   `json:"nama"`
	Urutan      int      `json:"urutan"`
	PageNumber  int      `json:"page_number"`
	KoordinatX  *float64 `json:"koordinat_x,omitempty"`
	KoordinatY  *float64 `json:"koordinat_y,omitempty"`
	Width       *float64 `json:"width,omitempty"`
	Height      *float64 `json:"height,omitempty"`
	Status      string   `json:"status"`
	AlasanTolak *string  `json:"alasan_tolak,omitempty"`
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
		SELECT pt.user_id, u.name, pt.urutan, pt.page_number, pt.koordinat_x, pt.koordinat_y, pt.width, pt.height, pt.status, pt.alasan_tolak
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
		if err := rows.Scan(&signer.UserID, &signer.Nama, &signer.Urutan, &signer.PageNumber, &signer.KoordinatX, &signer.KoordinatY, &signer.Width, &signer.Height, &signer.Status, &alasanTolak); err != nil {
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

	// NOTE: sertifikat metadata belum ditambahkan di sini karena modul
	// sertifikat digital (Minggu 10) belum dibangun. Tinggal nambahin
	// JOIN ke tabel sertifikat/transaksi_sertifikat begitu endpoint-nya jadi.
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

// Download PDF (bagian dari Minggu 9: "link unduh PDF")
// Versi disk lokal Railway. Kalau nanti pindah ke R2, tinggal ganti
// bagian "serve dari disk" di bawah jadi generate presigned URL.

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9\-_]+`)

func sanitizeFilename(name string) string {
	cleaned := unsafeFilenameChars.ReplaceAllString(strings.TrimSpace(name), "_")
	cleaned = strings.Trim(cleaned, "_")
	if cleaned == "" {
		return "dokumen"
	}
	if len(cleaned) > 100 {
		cleaned = cleaned[:100]
	}
	return cleaned
}

// GetDokumenDownload streams the PDF of a document owned by the authenticated
// user directly from local disk storage.
func GetDokumenDownload(w http.ResponseWriter, r *http.Request) {
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

	var ownerUserID int
	var judul string
	var filePath string
	var finalFilePath sql.NullString

	err = db.DB.QueryRow(`
		SELECT user_id, judul, file_path, final_file_path
		FROM dokumen
		WHERE id = $1`,
		dokumenID,
	).Scan(&ownerUserID, &judul, &filePath, &finalFilePath)
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

	// Prefer the finalized file (set once fully signed); fall back to the
	// originally uploaded file otherwise.
	storedPath := filePath
	if finalFilePath.Valid && finalFilePath.String != "" {
		storedPath = finalFilePath.String
	}

	// Defense in depth: only allow serving files inside our own uploads dir,
	// even though storedPath always comes from our own DB, not user input.
	cleanPath := filepath.Clean(storedPath)
	if !strings.HasPrefix(cleanPath, "uploads"+string(filepath.Separator)) && cleanPath != "uploads" {
		http.Error(w, "Invalid file location", http.StatusInternalServerError)
		return
	}

	if _, err := os.Stat(cleanPath); err != nil {
		http.Error(w, "File is not available on the server (may have been lost after a redeploy)", http.StatusNotFound)
		return
	}

	downloadName := fmt.Sprintf("%s.pdf", sanitizeFilename(judul))

	_ = insertLogAktivitas(db.DB, userID, &dokumenID, "download",
		fmt.Sprintf("User mengunduh dokumen '%s'", judul))

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	http.ServeFile(w, r, cleanPath)
}
