package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	Pool *pgxpool.Pool
}

func NewPostgresRepo(url string) (*PostgresRepo, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %v", err)
	}

	log.Println("Connected to Postgres successfully.")
	return &PostgresRepo{Pool: pool}, nil
}
