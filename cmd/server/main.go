package main

import (
	"log"
	"folio/internal/config"
	"folio/internal/server"
)

func main() {
	cfg := config.Load()

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
