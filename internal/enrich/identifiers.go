package enrich

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"

	"golang.org/x/crypto/sha3"
)

// GenerateDeviceID mirrors the openpanel legacy shake256 algorithm.
// data string format: `${ua}:${ip}:${origin}:${salt}`
func GenerateDeviceID(salt, origin, ip, ua string) string {
	if ua == "" {
		return ""
	}
	input := fmt.Sprintf("%s:%s:%s:%s", ua, ip, origin, salt)
	
	// Create shake256 hasher
	hasher := sha3.NewShake256()
	hasher.Write([]byte(input))
	
	// Node.js crypto.createHash('shake256', { outputLength: 16 }).digest('hex')
	// Output length 16 bytes = 32 hex chars
	buf := make([]byte, 16)
	hasher.Read(buf)
	
	return hex.EncodeToString(buf)
}

// GetSessionID generates a deterministic session id for (projectId, deviceId) within a time window.
// - windowMs: 30 minutes by default  (1000 * 60 * 30)
// - graceMs: 1 minute by default     (1000 * 60)
// - Output: base64url, 128-bit (16 bytes) truncated from SHA-256
func GetSessionID(projectId, deviceId string, eventMs int64, windowMs, graceMs int64) string {
	if eventMs == 0 {
		return "" // or default time
	}
	if windowMs <= 0 {
		windowMs = 5 * 60 * 1000 // default 5 mins
	}
	if graceMs < 0 || graceMs >= windowMs {
		graceMs = 60 * 1000 // default 1 min
	}

	bucket := int64(math.Floor(float64(eventMs) / float64(windowMs)))
	offset := eventMs - (bucket * windowMs)

	chosenBucket := bucket
	if offset < graceMs {
		chosenBucket = bucket - 1
	}

	input := fmt.Sprintf("sess:v1:%s:%s:%d", projectId, deviceId, chosenBucket)

	hasher := sha256.New()
	hasher.Write([]byte(input))
	digest := hasher.Sum(nil)

	// Truncate to 16 bytes
	truncated := digest[:16]

	// Use base64url encoding perfectly matching NodeJS base64url replacements.
	return base64.RawURLEncoding.EncodeToString(truncated)
}
