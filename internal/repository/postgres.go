package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

func (r *PostgresRepo) RotateSalts(ctx context.Context) error {
	newSaltBytes := make([]byte, 16)
	if _, err := rand.Read(newSaltBytes); err != nil {
		return err
	}
	newSalt := hex.EncodeToString(newSaltBytes)

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get latest salt to keep as previous
	var previousSalt string
	err = tx.QueryRow(ctx, `SELECT salt FROM salts ORDER BY "createdAt" DESC LIMIT 1`).Scan(&previousSalt)
	
	// Ignore ErrNoRows as it just means this is the first salt
	
	_, err = tx.Exec(ctx, `INSERT INTO salts (salt, "createdAt", "updatedAt") VALUES ($1, NOW(), NOW())`, newSalt)
	if err != nil {
		return err
	}

	saltsToKeep := []string{newSalt}
	if previousSalt != "" {
		saltsToKeep = append(saltsToKeep, previousSalt)
	}

	_, err = tx.Exec(ctx, `DELETE FROM salts WHERE salt != ALL($1)`, saltsToKeep)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepo) GetSalts(ctx context.Context) (current, previous string, err error) {
	rows, err := r.Pool.Query(ctx, `SELECT salt FROM salts ORDER BY "createdAt" DESC LIMIT 2`)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()

	var salts []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil {
			salts = append(salts, s)
		}
	}

	if len(salts) == 0 {
		return "", "", fmt.Errorf("no salts found in database")
	}

	current = salts[0]
	if len(salts) > 1 {
		previous = salts[1]
	} else {
		previous = current
	}

	return current, previous, nil
}
