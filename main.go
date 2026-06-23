package main

import (
	"fmt"
	"log"
	"net/http"

	"digital-signature-api/config"
	"digital-signature-api/db"
)

func main() {
	cfg := config.Load()
	db.Init(cfg)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Digital Signature API is running!")
	})

	log.Println("Server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
