package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/services"
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

func getIdentity(payload models.TrackHandlerPayload) (*models.IdentifyPayload, string) {
	if payload.Type == "track" {
		var track models.TrackPayload
		if err := json.Unmarshal(payload.Payload, &track); err == nil {
			if identifyVal, ok := track.Properties["__identify"]; ok {
				identBytes, err := json.Marshal(identifyVal)
				if err == nil {
					var identity models.IdentifyPayload
					if err := json.Unmarshal(identBytes, &identity); err == nil {
						return &identity, models.ConvertProfileID(identity.ProfileID)
					}
				}
			}
			profId := models.ConvertProfileID(track.ProfileID)
			if profId != "" {
				return &models.IdentifyPayload{ProfileID: profId}, profId
			}
		}
	}
	return nil, ""
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

	// projectId usually attached via middleware client identification. 
	// Assuming it's in header or context for now.
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

	// Salts hardcoded fallback initially, replaced by Postgres query once repo injected
	saltCurrent := "saltX"

	overrideDeviceId := ""
	if body.Type == "track" {
		var track models.TrackPayload
		if json.Unmarshal(body.Payload, &track) == nil {
			if ipOverride, ok := track.Properties["__ip"].(string); ok && ipOverride != "" {
				ip = ipOverride
			}
			if devIdOverride, ok := track.Properties["__deviceId"].(string); ok && devIdOverride != "" {
				overrideDeviceId = devIdOverride
			}
		}
	}

	geo := services.GetGeoLocation(ip)

	deviceId := overrideDeviceId
	sessionId := ""
	if deviceId == "" {
		deviceId = services.GenerateDeviceID(saltCurrent, projectId, ip, ua)
		sessionId = services.GetSessionID(projectId, deviceId, time.Now().UnixMilli(), 1000*60*30, 1000*60)
	}

	// Dispatch logic based on Type
	switch body.Type {
	case "track":
		var p models.TrackPayload
		json.Unmarshal(body.Payload, &p)
		
		// Event Buffer
		a.buffers.EventBuffer.Add(map[string]interface{}{
			"projectId": projectId,
			"deviceId":  deviceId,
			"sessionId": sessionId,
			"geo":       geo,
			"event":     p,
		})

	case "identify":
		var p models.IdentifyPayload
		json.Unmarshal(body.Payload, &p)
		a.buffers.ProfileBuffer.Add(p)

	case "increment":
		var p models.IncrementPayload
		json.Unmarshal(body.Payload, &p)
		a.buffers.ProfileBuffer.Add(p) // Should dispatch to increment handler or specify intent

	case "decrement":
		var p models.DecrementPayload
		json.Unmarshal(body.Payload, &p)
		a.buffers.ProfileBuffer.Add(p)

	case "group":
		var p models.GroupPayload
		json.Unmarshal(body.Payload, &p)
		// No group buffer yet, can drop or mock

	case "assign_group":
		var p models.AssignGroupPayload
		json.Unmarshal(body.Payload, &p)
	
	case "replay":
		var p models.ReplayPayload
		json.Unmarshal(body.Payload, &p)
		a.buffers.ReplayBuffer.Add(map[string]interface{}{
			"project_id": projectId,
			"session_id": sessionId,
			"payload":    p.Payload,
		})
	}

	// Return standard 200 { deviceId, sessionId }
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"deviceId":  deviceId,
		"sessionId": sessionId,
	})
}
