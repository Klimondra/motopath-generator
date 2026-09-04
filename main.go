package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"motopath-generator/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("/api/route", api.Handler)
	mux.HandleFunc("/api/health", api.Handler)

	// Fallback to static files in public/ (for local dev when not served by Vercel CDN)
	fs := http.FileServer(http.Dir("public"))
	mux.Handle("/", fs)

	fmt.Printf("🚀 MotoPath running on port %s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
