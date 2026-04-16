package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/openpanel-dev/openpanel-api/internal/buffers"
	"github.com/openpanel-dev/openpanel-api/internal/config"
	"github.com/openpanel-dev/openpanel-api/internal/cron"
	"github.com/openpanel-dev/openpanel-api/internal/handlers"
	"github.com/openpanel-dev/openpanel-api/internal/repository"
	"github.com/openpanel-dev/openpanel-api/internal/services"
)

func main() {
	log.Println("Initializing specific Configuration...")
	cfg := config.LoadConfig()

	log.Println("Initializing UA Parser...")
	if err := services.InitUAParser(); err != nil {
		log.Printf("Warning: Failed to initialize UA parser: %v", err)
	}

	log.Println("Connecting to Postgres...")
	pgRepo, err := repository.NewPostgresRepo(cfg.PostgresURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to pg: %v", err)
	}

	log.Println("Connecting to Clickhouse...")
	chRepo, err := repository.NewClickhouseRepo(cfg.ClickhouseURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to clickhouse: %v", err)
	}

	log.Println("Initializing Buffers...")
	b := buffers.InitBuffers(chRepo)

	log.Println("Initializing Asynq Cron & Workers...")
	cronManager := cron.NewManager(cfg, b, pgRepo)
	cronManager.Start()

	api := handlers.NewAPI(b)
	
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
	}))
	r.Use(middleware.Timeout(60 * time.Second))

	r.Mount("/", api.Router())

	log.Printf("Starting OpenPanel Go API explicitly on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
