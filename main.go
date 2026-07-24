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

	// permintaan ttd reject, approve, sign
	http.HandleFunc("POST /api/permintaan-ttd/{id}/setujui", middleware.AuthMiddleware(cfg.JWTSecret, handlers.SetujuiPermintaanTTD))
	http.HandleFunc("POST /api/permintaan-ttd/{id}/tolak", middleware.AuthMiddleware(cfg.JWTSecret, handlers.TolakPermintaanTTD))
	http.HandleFunc("POST /api/dokumen/{id}/tanda-tangani", middleware.AuthMiddleware(cfg.JWTSecret, handlers.TandaTanganiSendiri))

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
