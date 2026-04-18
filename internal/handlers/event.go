package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/openpanel-dev/openpanel-api/internal/models"
)

func (a *API) handleEvent(w http.ResponseWriter, r *http.Request) {
	var body models.DeprecatedPostEventPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := models.Validate.Struct(&body); err != nil {
		http.Error(w, "validation error", http.StatusBadRequest)
		return
	}

	projectId := r.Header.Get("x-project-id")
	if projectId == "" {
		http.Error(w, "missing projectId", http.StatusBadRequest)
		return
	}

	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = strings.Split(r.RemoteAddr, ":")[0]
	}
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		ua = "unknown/1.0"
	}

	// Convert deprecated payload to standard track payload
	trackPayload := models.TrackPayload{
		Name:       body.Name,
		Properties: body.Properties,
		ProfileID:  body.ProfileID,
		Timestamp:  body.Timestamp,
	}

	_, err := a.ingestion.ProcessEvent(r.Context(), projectId, ip, ua, trackPayload, nil)
	if err != nil {
		http.Error(w, "ingestion error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("ok"))
}
