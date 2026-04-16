package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	PostgresURL  string
	ClickhouseURL string
	RedisHost     string
	RedisPort     string
	DBGeoipPath   string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	port := os.Getenv("API_SERVER_PORT")
	if port == "" {
		port = "3334"
	}

	pgHost := os.Getenv("POSTGRES_HOST")
	if pgHost == "" {
		pgHost = "localhost"
	}
	pgPort := os.Getenv("POSTGRES_PORT")
	if pgPort == "" {
		pgPort = "5432"
	}
	pgUser := os.Getenv("POSTGRES_USER")
	if pgUser == "" {
		pgUser = "postgres"
	}
	pgPass := os.Getenv("POSTGRES_PASSWORD")
	if pgPass == "" {
		pgPass = "postgres"
	}
	pgDB := os.Getenv("POSTGRES_DB")
	if pgDB == "" {
		pgDB = "openpanel"
	}

	pgURL := "postgres://" + pgUser + ":" + pgPass + "@" + pgHost + ":" + pgPort + "/" + pgDB

	chHost := os.Getenv("CLICKHOUSE_HOST")
	if chHost == "" {
		chHost = "localhost"
	}
	chPort := os.Getenv("CLICKHOUSE_PORT")
	if chPort == "" {
		chPort = "9000"
	}
	chURL := chHost + ":" + chPort

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	dbGeoipPath := os.Getenv("DB_GEOIP_PATH")
	if dbGeoipPath == "" {
		dbGeoipPath = "/usr/share/GeoIP"
	}

	return &Config{
		Port:          port,
		PostgresURL:   pgURL,
		ClickhouseURL: chURL,
		RedisHost:     redisHost,
		RedisPort:     redisPort,
		DBGeoipPath:   dbGeoipPath,
	}
}
