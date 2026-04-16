package cron

import (
	"log"

	"github.com/hibiken/asynq"
	"github.com/openpanel-dev/openpanel-api/internal/buffers"
	"github.com/openpanel-dev/openpanel-api/internal/config"
	"github.com/openpanel-dev/openpanel-api/internal/tasks"
)

type Manager struct {
	server    *asynq.Server
	scheduler *asynq.Scheduler
	mux       *asynq.ServeMux
	b         *buffers.Buffers
}

func NewManager(cfg *config.Config, b *buffers.Buffers) *Manager {
	redisConnOpt := asynq.RedisClientOpt{
		Addr: cfg.RedisHost + ":" + cfg.RedisPort,
	}

	server := asynq.NewServer(redisConnOpt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
	})

	scheduler := asynq.NewScheduler(redisConnOpt, nil)
	mux := asynq.NewServeMux()

	return &Manager{
		server:    server,
		scheduler: scheduler,
		mux:       mux,
		b:         b,
	}
}

func (m *Manager) Start() {
	// Register Handlers
	m.mux.HandleFunc(tasks.TypeFlushEvents, tasks.HandleFlushEventsTask(m.b))
	m.mux.HandleFunc(tasks.TypeFlushProfiles, tasks.HandleFlushProfilesTask(m.b))
	m.mux.HandleFunc(tasks.TypeFlushSessions, tasks.HandleFlushSessionsTask(m.b))
	m.mux.HandleFunc(tasks.TypeFlushProfileBackfill, tasks.HandleFlushProfileBackfillTask(m.b))
	m.mux.HandleFunc(tasks.TypeFlushReplay, tasks.HandleFlushReplayTask(m.b))
	m.mux.HandleFunc(tasks.TypeSalt, tasks.HandleSaltTask())

	// Register cron jobs to run e.g. every minute
	if _, err := m.scheduler.Register("* * * * *", asynq.NewTask(tasks.TypeFlushEvents, nil)); err != nil {
		log.Printf("Scheduler error events: %v", err)
	}
	if _, err := m.scheduler.Register("* * * * *", asynq.NewTask(tasks.TypeFlushProfiles, nil)); err != nil {
		log.Printf("Scheduler error profiles: %v", err)
	}
	if _, err := m.scheduler.Register("* * * * *", asynq.NewTask(tasks.TypeFlushSessions, nil)); err != nil {
		log.Printf("Scheduler error sessions: %v", err)
	}
	// Add other cron registrations here similarly ...

	go func() {
		log.Println("Starting Asynq worker server")
		if err := m.server.Run(m.mux); err != nil {
			log.Fatalf("Could not run Asynq server: %v", err)
		}
	}()

	go func() {
		log.Println("Starting Asynq scheduler")
		if err := m.scheduler.Run(); err != nil {
			log.Fatalf("Could not run Asynq scheduler: %v", err)
		}
	}()
}
