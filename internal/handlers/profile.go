package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/services"
)

func (a *API) handleProfile(w http.ResponseWriter, r *http.Request) {
	var body models.DeprecatedUpdateProfilePayload
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

	geo := services.GetGeoLocation(ip)

	// Enrich payload properly
	if body.Properties == nil {
		body.Properties = make(map[string]interface{})
	}
	body.Properties["country"] = geo.Country
	body.Properties["city"] = geo.City
	body.Properties["region"] = geo.Region

	a.buffers.ProfileBuffer.Add(body)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(body.ProfileID))
}
