package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/openpanel-dev/openpanel-api/internal/buffers"
	"github.com/openpanel-dev/openpanel-api/internal/services"
	"github.com/openpanel-dev/openpanel-api/internal/websocket"
)

type API struct {
	router    *chi.Mux
	buffers   *buffers.Buffers
	ingestion *services.IngestionService
}

func NewAPI(b *buffers.Buffers, ingestion *services.IngestionService) *API {
	api := &API{
		router:    chi.NewRouter(),
		buffers:   b,
		ingestion: ingestion,
	}
	api.setupRoutes()
	return api
}

func (a *API) Router() *chi.Mux {
	return a.router
}

func (a *API) setupRoutes() {
	a.router.Post("/event", a.handleEvent)
	a.router.Post("/profile", a.handleProfile)
	a.router.Post("/track", a.handleTrack)
	a.router.Post("/ai", a.handleAI)

	// Health and liveness
	a.router.Get("/healthcheck", a.handleHealthCheck)
	a.router.Get("/healthz/live", a.handleHealthCheck)
	a.router.Get("/healthz/ready", a.handleHealthCheck)

	// Websocket for live view
	a.router.HandleFunc("/live", websocket.HandleLiveConnect)
}




func (a *API) handleAI(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ai response mocked"})
}

func (a *API) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
