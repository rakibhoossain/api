package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/services"
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

	geo := services.GetGeoLocation(ip)
	saltCurrent := "saltX" // TODO: inject postgres salts

	devIdOverride := ""
	if devId, ok := body.Properties["__deviceId"].(string); ok {
		devIdOverride = devId
	}

	deviceId := devIdOverride
	if deviceId == "" {
		deviceId = services.GenerateDeviceID(saltCurrent, projectId, ip, ua)
	}
	sessionId := services.GetSessionID(projectId, deviceId, time.Now().UnixMilli(), 1000*60*30, 1000*60)

	a.buffers.EventBuffer.Add(map[string]interface{}{
		"projectId": projectId,
		"deviceId":  deviceId,
		"sessionId": sessionId,
		"geo":       geo,
		"event":     body,
	})

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("ok"))
}
