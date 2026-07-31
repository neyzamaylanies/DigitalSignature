package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"digital-signature-api/config"
	"digital-signature-api/db"
	"digital-signature-api/handlers"
	"digital-signature-api/middleware"
	"digital-signature-api/utils"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// if r.Method == http.MethodOptions {
		// 	w.WriteHeader(http.StatusOK)
		// 	return
		// }

		next.ServeHTTP(w, r)
	})
}

func main() {
	cfg := config.Load()
	db.Init(cfg)
	utils.SeedUsers(db.DB)

	// Auth routes
	http.HandleFunc("/api/auth/login", handlers.Login(cfg))
	http.HandleFunc("/api/auth/logout", middleware.AuthMiddleware(cfg.JWTSecret, handlers.Logout))
	http.HandleFunc("/api/auth/profile", middleware.AuthMiddleware(cfg.JWTSecret, handlers.Profile))

	// User management (admin only)
	http.HandleFunc("/api/users", middleware.AdminMiddleware(cfg.JWTSecret, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetUsers(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// Dashboard
	http.HandleFunc("/api/dashboard", middleware.AuthMiddleware(cfg.JWTSecret, handlers.Dashboard))

	http.HandleFunc("/api/dokumen/upload", middleware.AuthMiddleware(cfg.JWTSecret, handlers.UploadDokumen))
	http.HandleFunc("/api/dokumen/ajukan", middleware.AuthMiddleware(cfg.JWTSecret, handlers.AjukanTandaTangan))
	// http.HandleFunc("/api/dokumen", middleware.AuthMiddleware(cfg.JWTSecret, handlers.GetDokumen))
	// http.HandleFunc("/api/dokumen/", middleware.AuthMiddleware(cfg.JWTSecret, func(w http.ResponseWriter, r *http.Request) {
	// 	switch r.Method {
	// 	case http.MethodGet:
	// 		handlers.GetDokumenByID(w, r)
	// 	case http.MethodPut:
	// 		handlers.UpdateDokumen(w, r)
	// 	default:
	// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	// 	}
	// }))

	// permintaan_ttd view
	http.HandleFunc("GET /api/permintaan-ttd", middleware.AuthMiddleware(cfg.JWTSecret, handlers.ListPermintaanTTD))
	http.HandleFunc("GET /api/permintaan-ttd/{id}", middleware.AuthMiddleware(cfg.JWTSecret, handlers.GetPermintaanTTDDetail))

	// dokumen diunggah & dokumen ditandatangani
	http.HandleFunc("GET /api/dokumen/diunggah", middleware.AuthMiddleware(cfg.JWTSecret, handlers.ListDokumenDiunggah))
	http.HandleFunc("GET /api/dokumen/diunggah/{id}", middleware.AuthMiddleware(cfg.JWTSecret, handlers.GetDokumenDiunggahDetail))
	http.HandleFunc("GET /api/dokumen/ditandatangani", middleware.AuthMiddleware(cfg.JWTSecret, handlers.ListDokumenDitandatangani))

	// unduh PDF
	http.HandleFunc("GET /api/dokumen/{id}/download", middleware.AuthMiddleware(cfg.JWTSecret, handlers.GetDokumenDownload))

	// permintaan ttd reject, approve, sign
	http.HandleFunc("POST /api/permintaan-ttd/{id}/setujui", middleware.AuthMiddleware(cfg.JWTSecret, handlers.SetujuiPermintaanTTD))
	http.HandleFunc("POST /api/permintaan-ttd/{id}/tolak", middleware.AuthMiddleware(cfg.JWTSecret, handlers.TolakPermintaanTTD))
	http.HandleFunc("POST /api/dokumen/{id}/tanda-tangani", middleware.AuthMiddleware(cfg.JWTSecret, handlers.TandaTanganiSendiri))

	// sertifikat digital
	http.HandleFunc("POST /api/sertifikat", middleware.AuthMiddleware(cfg.JWTSecret, handlers.CreateSertifikat))
	http.HandleFunc("GET /api/sertifikat", middleware.AuthMiddleware(cfg.JWTSecret, handlers.ListSertifikat))
	http.HandleFunc("POST /api/sertifikat/{id}/revoke", middleware.AuthMiddleware(cfg.JWTSecret, handlers.RevokeSertifikat))

	// gambar tanda tangan & paraf
	http.HandleFunc("POST /api/tanda-tangan", middleware.AuthMiddleware(cfg.JWTSecret, handlers.UploadTandaTangan))
	http.HandleFunc("GET /api/tanda-tangan", middleware.AuthMiddleware(cfg.JWTSecret, handlers.ListTandaTangan))
	http.HandleFunc("GET /api/tanda-tangan/{id}", middleware.AuthMiddleware(cfg.JWTSecret, handlers.GetTandaTanganDetail))
	http.HandleFunc("GET /api/tanda-tangan/{id}/preview", middleware.AuthMiddleware(cfg.JWTSecret, handlers.PreviewTandaTangan))
	http.HandleFunc("PUT /api/tanda-tangan/{id}", middleware.AuthMiddleware(cfg.JWTSecret, handlers.UpdateTandaTangan))
	http.HandleFunc("DELETE /api/tanda-tangan/{id}", middleware.AuthMiddleware(cfg.JWTSecret, handlers.DeleteTandaTangan))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("Digital Signature API is running"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server running on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, corsMiddleware(http.DefaultServeMux)))
}
