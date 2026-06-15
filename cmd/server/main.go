package main

import (
	"log"
	"os"

	"reader-go/internal/web"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "6464"
	}

	server, err := web.NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Printf("Starting server on port %s", port)
	if err := server.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
