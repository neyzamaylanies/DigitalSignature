package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"digital-signature-api/db"
	"digital-signature-api/middleware"
	"digital-signature-api/models"
)

const (
	defaultLogAktivitasLimit = 50
	maxLogAktivitasLimit     = 200
)

// parseLogAktivitasLimit reads an optional ?limit= query param, falling back
// to a sane default and capping it so nobody can request the entire table.
func parseLogAktivitasLimit(r *http.Request) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return defaultLogAktivitasLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLogAktivitasLimit
	}
	if n > maxLogAktivitasLimit {
		return maxLogAktivitasLimit
	}
	return n
}

// ListLogAktivitas returns the authenticated user's own activity log only.
// Sesuai flow: "User -> hanya lihat log milik sendiri."
func ListLogAktivitas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit := parseLogAktivitasLimit(r)

	aksiFilter := r.URL.Query().Get("aksi")

	query := `
		SELECT id, user_id, dokumen_id, aksi, keterangan, created_at
		FROM log_aktivitas
		WHERE user_id = $1`
	args := []interface{}{userID}

	if aksiFilter != "" {
		query += " AND aksi = $2"
		args = append(args, aksiFilter)
	}
	query += " ORDER BY created_at DESC LIMIT " + strconv.Itoa(limit)

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to fetch activity log", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	logs := make([]models.LogAktivitas, 0)
	for rows.Next() {
		var l models.LogAktivitas
		if err := rows.Scan(&l.ID, &l.UserID, &l.DokumenID, &l.Aksi, &l.Keterangan, &l.CreatedAt); err != nil {
			http.Error(w, "Failed to read activity log", http.StatusInternalServerError)
			return
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read activity log", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"log_aktivitas": logs,
	})
}

// ListSemuaLogAktivitas returns activity logs across ALL users. Admin only.
// Sesuai flow: "Admin -> bisa lihat log semua user."
// Optional ?user_id= to filter down to one user's log from the admin view.
func ListSemuaLogAktivitas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := parseLogAktivitasLimit(r)
	aksiFilter := r.URL.Query().Get("aksi")
	userIDFilter := r.URL.Query().Get("user_id")

	query := `
		SELECT l.id, l.user_id, u.name, l.dokumen_id, l.aksi, l.keterangan, l.created_at
		FROM log_aktivitas l
		JOIN users u ON u.id = l.user_id
		WHERE 1=1`
	args := []interface{}{}

	if userIDFilter != "" {
		uid, err := strconv.Atoi(userIDFilter)
		if err != nil || uid <= 0 {
			http.Error(w, "Invalid user_id filter", http.StatusBadRequest)
			return
		}
		args = append(args, uid)
		query += " AND l.user_id = $" + strconv.Itoa(len(args))
	}
	if aksiFilter != "" {
		args = append(args, aksiFilter)
		query += " AND l.aksi = $" + strconv.Itoa(len(args))
	}
	query += " ORDER BY l.created_at DESC LIMIT " + strconv.Itoa(limit)

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to fetch activity log", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	logs := make([]models.LogAktivitas, 0)
	for rows.Next() {
		var l models.LogAktivitas
		if err := rows.Scan(&l.ID, &l.UserID, &l.UserName, &l.DokumenID, &l.Aksi, &l.Keterangan, &l.CreatedAt); err != nil {
			http.Error(w, "Failed to read activity log", http.StatusInternalServerError)
			return
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read activity log", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"log_aktivitas": logs,
	})
}
