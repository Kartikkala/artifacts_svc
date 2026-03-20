package config

import (
	"os"

	"github.com/sirkartik/artifacts_svc/internal/utils"
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
	ArtifactUpstreamAddress          string
	ArtifactUpstreamPort             uint16
	ArtifactUpstreamProtocol         utils.Protocol
	ArtifactUpstreamEndpoint         string
}

type Config struct {
	App      *AppConfig
	Database *DatabaseConfig
}

func NewConfig() *Config {
	return &Config{
		App: &AppConfig{
			WorkerInactivityKillDurationSecs: 30,
			ArtifactUpstreamAddress:          "127.0.0.1",
			ArtifactUpstreamEndpoint:         "/hls",
			ArtifactUpstreamPort:             9009,
			ArtifactUpstreamProtocol:         utils.HTTP,
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
