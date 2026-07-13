package main

import (
	"fmt"
	"log"
	"net/http"

	"digital-signature-api/config"
	"digital-signature-api/db"
	"digital-signature-api/handlers"
	"digital-signature-api/middleware"
	"digital-signature-api/utils"
)

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
	http.HandleFunc("/api/dokumen/ajukan", middleware.AuthMiddleware(cfg.JWTSecret, handlers.AjukanTandaTangan),
)
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("Digital Signature API is running"))
	})

	fmt.Println("Server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
