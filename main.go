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

	// User management (admin only - RBAC)
	// http.HandleFunc("/api/users", middleware.AdminMiddleware(cfg.JWTSecret, func(w http.ResponseWriter, r *http.Request) {
	// 	switch r.Method {
	// 	case http.MethodPost:
	// 		handlers.CreateUser(w, r)
	// 	case http.MethodGet:
	// 		handlers.GetUsers(w, r)
	// 	default:
	// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	// 	}
	// }))

	fmt.Println("Server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
