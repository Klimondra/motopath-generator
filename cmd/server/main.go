package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"motopath-generator/internal/api"
	"motopath-generator/internal/overpass"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Locate static files directory
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		// Try relative paths
		candidates := []string{"./web/static", "./static", "../web/static"}
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				staticDir = c
				break
			}
		}
		if staticDir == "" {
			staticDir = "./web/static"
		}
	}
	absStaticDir, _ := filepath.Abs(staticDir)
	log.Printf("[Server] Serving static files from: %s", absStaticDir)

	// Initialize Overpass client and API handler
	opClient := overpass.NewClient(35 * time.Second)
	handler := api.NewHandler(opClient)

	// Register routes on Go 1.22+ ServeMux
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, absStaticDir)

	// Middleware pipeline
	rootHandler := api.RecoveryMiddleware(
		api.CorsMiddleware(
			api.LoggerMiddleware(mux),
		),
	)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      rootHandler,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown channel
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("=====================================================")
		log.Printf(" 🏎️  MotoPath Generator (Off-Road GoKart Routing) ")
		log.Printf(" 🌐 Server running at http://localhost:%s", port)
		log.Printf("=====================================================")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[Server] Fatal error: %v", err)
		}
	}()

	<-stopChan
	log.Println("[Server] Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Server] Shutdown error: %v", err)
	}

	fmt.Println("[Server] Goodbye!")
}
