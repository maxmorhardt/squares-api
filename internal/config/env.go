package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/maxmorhardt/squares-api/internal/model"
)

func LoadEnv() (*model.AppConfig, error) {
	_ = godotenv.Load()

	cfg := &model.AppConfig{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return cfg, nil
}
