package bootstrap

import (
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type Dependencies struct {
	Config       *model.AppConfig
	DB           *gorm.DB
	NATS         *nats.Conn
	OIDCVerifier *oidc.IDTokenVerifier
}

func BuildDependencies() (*Dependencies, error) {
	cfg, err := config.LoadEnv()
	if err != nil {
		return nil, err
	}

	db, err := config.InitDB(cfg)
	if err != nil {
		return nil, err
	}

	nc, err := config.InitNATS(cfg)
	if err != nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}

	oidcVerifier, err := config.InitOIDC(cfg)
	if err != nil {
		nc.Close()
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}

	return &Dependencies{
		Config:       cfg,
		DB:           db,
		NATS:         nc,
		OIDCVerifier: oidcVerifier,
	}, nil
}
