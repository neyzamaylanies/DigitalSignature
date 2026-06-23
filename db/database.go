package db

import (
	"database/sql"
	"fmt"
	"log"

	"digital-signature-api/config"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init(cfg config.Config) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal("Database not reachable:", err)
	}

	log.Println("Database connected!")
}
