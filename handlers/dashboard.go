package handlers

import (
	"encoding/json"
	"net/http"

	"digital-signature-api/db"
	"digital-signature-api/middleware"
)

func Dashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var jumlahDokumen int
	db.DB.QueryRow(
		"SELECT COUNT(*) FROM dokumen WHERE user_id = $1",
		userID,
	).Scan(&jumlahDokumen)

	var jumlahPermintaan int
	db.DB.QueryRow(
		"SELECT COUNT(*) FROM permintaan_ttd WHERE user_id = $1 AND status = 'menunggu'",
		userID,
	).Scan(&jumlahPermintaan)

	var jumlahDitandatangani int
	db.DB.QueryRow(
		"SELECT COUNT(*) FROM dokumen WHERE user_id = $1 AND status = 'selesai'",
		userID,
	).Scan(&jumlahDitandatangani)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dokumen_saya":           jumlahDokumen,
		"permintaan_menunggu":    jumlahPermintaan,
		"dokumen_ditandatangani": jumlahDitandatangani,
	})
}
