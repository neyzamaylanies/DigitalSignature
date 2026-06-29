package utils

import (
	"database/sql"
	"log"
)

func SeedUsers(db *sql.DB) {
	users := []struct {
		Name     string
		Email    string
		Password string
		Role     string
	}{
		{"Administrator Testing", "admin@uai.ac.id", "admin123!", "admin"},
		{"User Testing 1", "user1@uai.ac.id", "user123!", "user"},
		{"User Testing 2", "user2@uai.ac.id", "user123!", "user"},
	}

	for _, u := range users {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", u.Email).Scan(&exists)
		if err != nil {
			log.Printf("Error checking user %s: %v", u.Email, err)
			continue
		}

		if exists {
			log.Printf("User %s already exists, skipping...", u.Email)
			continue
		}

		hashedPassword, err := HashPassword(u.Password)
		if err != nil {
			log.Printf("Error hashing password for %s: %v", u.Email, err)
			continue
		}

		_, err = db.Exec(
			"INSERT INTO users (name, email, password, role) VALUES ($1, $2, $3, $4)",
			u.Name, u.Email, hashedPassword, u.Role,
		)
		if err != nil {
			log.Printf("Error seeding user %s: %v", u.Email, err)
			continue
		}

		log.Printf("Seeded user: %s (%s)", u.Name, u.Role)
	}
}
