package config

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Application struct {
	DB  *gorm.DB
	Cfg *Config
}

func NewApp() (*Application, error) {
	Cfg := NewConfig()
	DBCfg := Cfg.Database

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=%s",
		DBCfg.Hostname,
		DBCfg.User,
		DBCfg.Password,
		DBCfg.DBName,
		DBCfg.Port,
		DBCfg.Timezone,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to database: %w", err)
	}

	return &Application{
		DB:  db,
		Cfg: Cfg,
	}, nil
}