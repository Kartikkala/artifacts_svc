package config

import (
	"os"
)

type DatabaseConfig struct {
	User     string
	Password string
	Hostname string
	DBName   string
	Port     uint16
	Timezone string
}
type AppConfig struct {
	WorkerInactivityKillDurationSecs uint16
}

type Config struct {
	App      *AppConfig
	Database *DatabaseConfig
}

func NewConfig() *Config {
	return &Config{
		App: &AppConfig{
			WorkerInactivityKillDurationSecs: 30,
		},
		Database: &DatabaseConfig{
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASS"),
			DBName:   os.Getenv("POSTGRES_DBNAME"),
			Hostname: "127.0.0.1",
			Port:     uint16(5432),
			Timezone: "Asia/Kolkata",
		},
	}
}
