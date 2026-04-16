package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickhouseRepo struct {
	Conn driver.Conn
}

func NewClickhouseRepo(addr string) (*ClickhouseRepo, error) {
	ctx := context.Background()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to connect to clickhouse: %v", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping clickhouse: %v", err)
	}

	log.Println("Connected to Clickhouse successfully.")
	return &ClickhouseRepo{Conn: conn}, nil
}
