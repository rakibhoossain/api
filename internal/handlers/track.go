package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/openpanel-dev/openpanel-api/internal/models"
)

// Extract headers similar to TS `getStringHeaders`
func getStringHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	keys := []string{"User-Agent", "Openpanel-Sdk-Name", "Openpanel-Sdk-Version", "Openpanel-Client-Id", "Request-Id"}
	for _, k := range keys {
		val := r.Header.Get(k)
		if val != "" {
			headers[strings.ToLower(k)] = val
		}
	}
	return headers
}

func (a *API) handleTrack(w http.ResponseWriter, r *http.Request) {
	// Parse root payload
	var body models.TrackHandlerPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := models.Validate.Struct(&body); err != nil {
		http.Error(w, "validation error", http.StatusBadRequest)
		return
	}

	if body.Type == "alias" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 400, "error": "Bad Request", "message": "Alias is not supported"})
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

	var deviceId, sessionId string

	// Handle different types using the new IngestionService where applicable
	switch body.Type {
	case "track":
		var p models.TrackPayload
		json.Unmarshal(body.Payload, &p)
		
		res, err := a.ingestion.ProcessEvent(r.Context(), projectId, ip, ua, p, getStringHeaders(r))
		if err != nil {
			http.Error(w, "ingestion error", http.StatusInternalServerError)
			return
		}
		deviceId = res.DeviceID
		sessionId = res.SessionID

	case "identify":
		var p models.IdentifyPayload
		json.Unmarshal(body.Payload, &p)
		a.buffers.ProfileBuffer.Add(p)

	case "increment":
		var p models.IncrementPayload
		json.Unmarshal(body.Payload, &p)
		a.buffers.ProfileBuffer.Add(p)

	case "decrement":
		var p models.DecrementPayload
		json.Unmarshal(body.Payload, &p)
		a.buffers.ProfileBuffer.Add(p)

	case "group":
		var p models.GroupPayload
		json.Unmarshal(body.Payload, &p)
		// Mocked for now, same as original
		
	case "assign_group":
		var p models.AssignGroupPayload
		json.Unmarshal(body.Payload, &p)
	
	case "replay":
		var p models.ReplayPayload
		json.Unmarshal(body.Payload, &p)
		// SessionId might be missing if replay is called separately, but usually it follows track
		a.buffers.ReplayBuffer.Add(map[string]interface{}{
			"project_id": projectId,
			"payload":    p.Payload,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"deviceId":  deviceId,
		"sessionId": sessionId,
	})
}
