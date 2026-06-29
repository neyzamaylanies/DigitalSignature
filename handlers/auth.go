package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"digital-signature-api/config"
	"digital-signature-api/db"
	"digital-signature-api/middleware"
	"digital-signature-api/models"
	"digital-signature-api/utils"

	"github.com/golang-jwt/jwt/v5"
)

// func Register(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodPost {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	var input struct {
// 		Name     string `json:"name"`
// 		Email    string `json:"email"`
// 		Password string `json:"password"`
// 	}

// 	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	if input.Name == "" || input.Email == "" || input.Password == "" {
// 		http.Error(w, "Name, email, and password are required", http.StatusBadRequest)
// 		return
// 	}

// 	hashedPassword, err := utils.HashPassword(input.Password)
// 	if err != nil {
// 		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
// 		return
// 	}

// 	var user models.User
// 	err = db.DB.QueryRow(
// 		"INSERT INTO users (name, email, password, role) VALUES ($1, $2, $3, $4) RETURNING id, name, email, role, created_at",
// 		input.Name, input.Email, hashedPassword, "user",
// 	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.CreatedAt)

// 	if err != nil {
// 		http.Error(w, "Email already exists or failed to create user", http.StatusBadRequest)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusCreated)
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"message": "User registered successfully",
// 		"user":    user,
// 	})
// }

func Login(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var input struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var user models.User
		var hashedPassword string
		err := db.DB.QueryRow(
			"SELECT id, name, email, password, role, created_at FROM users WHERE email = $1",
			input.Email,
		).Scan(&user.ID, &user.Name, &user.Email, &hashedPassword, &user.Role, &user.CreatedAt)

		if err != nil {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		if !utils.CheckPassword(input.Password, hashedPassword) {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":   user.ID,
			"user_role": user.Role,
			"exp":       time.Now().Add(24 * time.Hour).Unix(),
		})

		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Login successful",
			"token":   tokenString,
			"user":    user,
		})
	}
}

func Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logout successful",
	})
}

func Profile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(middleware.UserIDKey).(int)

	var user models.User
	err := db.DB.QueryRow(
		"SELECT id, name, email, role, created_at FROM users WHERE id = $1",
		userID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.CreatedAt)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
