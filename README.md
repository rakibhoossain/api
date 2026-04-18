# OpenPanel Analytics API (Go)

High-performance analytics backend built with Go, Redis, and PostgreSQL.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                  External Clients                                │
│         (SDKs, Dashboard, E-commerce)                 │
└───────────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                  API Server (Gin)                          │
│                  Port: 3334                               │
│                                                           │
│  /track      - Event tracking                            │
│  /event     - Event ingestion (deprecated)             │
│  /profile   - Profile management                    │
│  /live      - Real-time events (WebSocket)           │
│  /ai        - AI queries                       │
│  /health*   - Health checks                     │
└───────────────────────────────┬─────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            ▼               ▼               ▼
       PostgreSQL       Redis          WebSocket
       (Salts)      (Buffers)      (Live Events)
       (Projects)   (Sessions)
```

## Project Structure

```
api/
├── README.md                    # This file
├── schema.sql                 # PostgreSQL schema + ClickHouse
├── docker-compose.yml         # Production ready compose
├── Dockerfile
├── go.mod / go.sum
├── cmd/server/
│   └── main.go              # Entry point
└── internal/
    ├── config/
    │   └── config.go       # Configuration
    ├── repository/
    │   └── repository.go # Data access layer
    ├── services/
    │   └── services.go  # Business logic
    ├── buffers/
    │   └── redis.go    # Redis buffer management
    ├── cron/
    │   └── cron.go   # Background jobs (6 tasks)
    ├── websocket/
    │   └── websocket.go # WebSocket for live events
    └── handlers/
        └── handlers.go  # HTTP handlers
```

## API Routes

| Route | Method | Handler |
|-------|--------|---------|
| `/track` | POST | TrackHandler.Track |
| `/track/device-id` | GET | TrackHandler.FetchDeviceID |
| `/event` | POST | EventHandler.PostEvent |
| `/profile/:id` | GET | ProfileHandler.GetProfile |
| `/live` | GET | WebSocket Handler |
| `/ai/query` | POST | AIHandler.Query |
| `/healthcheck` | GET | HealthHandler.HealthCheck |
| `/healthz/live` | GET | HealthHandler.Liveness |
| `/healthz/ready` | GET | HealthHandler.Readiness |

## Cron Jobs (6 tasks)

| Task | Interval | Description |
|------|---------|-------------|
| `flush_events` | 10s | Flush event buffer to storage |
| `flush_profiles` | 10s | Flush profile buffer to storage |
| `flush_sessions` | 10s | Flush session buffer to storage |
| `flush_profile_backfill` | 30s | Profile backfill updates |
| `flush_replay` | 10s | Session replay buffer |
| `salt_rotation` | Daily | Rotate salts |

## Database Schema

PostgreSQL tables:
- `projects` - Analytics projects
- `users` - User accounts
- `salts` - Device ID salts

ClickHouse tables (in schema.sql):
- `events` - Track events
- `sessions` - User sessions (VersionedCollapsingMergeTree)
- `profiles` - User profiles (ReplacingMergeTree)

## Redis Keyspace Notifications

```go
redis.Config("SET", "notify-keyspace-events", "Ex")
```

Used to detect expired session cache keys.

## Auth

- **SDK tracking**: `OpenPanel-Project-Id` header
- **Dashboard**: JWT token (for future)

## Running

```bash
cd api
go mod tidy
go run cmd/server/main.go
```

## Testing with cURL

Once the server is running, you can manually trigger events using `cURL`. 

**1. Track an Event:**
```sh
curl -X POST http://localhost:3334/track \
  -H "Content-Type: application/json" \
  -H "OpenPanel-Project-Id: test-project-id" \
  -d '{
    "type": "track",
    "payload": {
      "name": "button_click",
      "properties": {
        "button_name": "checkout",
        "__path": "/cart"
      }
    }
  }'
```

**2. Track a Revenue Event:**
```sh
curl -X POST http://localhost:3334/track \
  -H "Content-Type: application/json" \
  -H "OpenPanel-Project-Id: test-project-id" \
  -d '{
    "type": "track",
    "payload": {
      "name": "revenue",
      "properties": {
        "__revenue": 99.99,
        "currency": "USD"
      }
    }
  }'
```

**3. Identify a User Profile:**
```sh
curl -X POST http://localhost:3334/track \
  -H "Content-Type: application/json" \
  -H "OpenPanel-Project-Id: test-project-id" \
  -d '{
    "type": "identify",
    "payload": {
      "profileId": "user_123",
      "firstName": "John",
      "lastName": "Doe",
      "email": "john@example.com"
    }
  }'
```


Or with Docker:
```bash
docker-compose up -d
```

- API: http://localhost:3334
- Legacy: http://localhost:3333

## Comparing with Legacy

| Aspect | Legacy (Node.js) | New (Go) |
|--------|-----------------|----------|
| Port | 3333 | 3334 |
| Router | Fastify | Gin |
| Queue | BullMQ | Custom cron |
| WebSocket | - | gorilla/websocket |
| Schema | Prisma | SQL scripts |