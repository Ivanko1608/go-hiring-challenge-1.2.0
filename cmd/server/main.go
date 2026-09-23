package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/mytheresa/go-hiring-challenge/app/catalog"
	"github.com/mytheresa/go-hiring-challenge/app/categories"
	"github.com/mytheresa/go-hiring-challenge/app/database"
	"github.com/mytheresa/go-hiring-challenge/models"
)

// How long in-flight requests get to finish after SIGINT/SIGTERM before the
// server closes their connections.
const shutdownTimeout = 10 * time.Second

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}

	// signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize database connection
	db, close := database.New(
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_PORT"),
	)
	defer close()

	// Initialize handlers
	productsRepo := models.NewProductsRepository(db)
	catalogHandler := catalog.NewHandler(productsRepo)
	categoriesRepo := models.NewCategoriesRepository(db)
	categoriesHandler := categories.NewHandler(categoriesRepo)

	// Set up routing
	mux := http.NewServeMux()
	mux.HandleFunc("GET /catalog", catalogHandler.List)
	mux.HandleFunc("GET /catalog/{code}", catalogHandler.Get)
	mux.HandleFunc("GET /categories", categoriesHandler.List)
	mux.HandleFunc("POST /categories", categoriesHandler.Create)

	// Set up the HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf("localhost:%s", os.Getenv("HTTP_PORT")),
		Handler: mux,
	}

	// Start the server
	go func() {
		log.Printf("Starting server on http://%s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %s", err)
		}
	}()

	<-ctx.Done()
	stop()
	log.Println("Shutting down server...")

	// ctx is already cancelled here, so draining in-flight requests needs a fresh deadline.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown failed: %s", err)
		return
	}

	log.Println("Server stopped gracefully")
}
