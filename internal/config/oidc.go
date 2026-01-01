package config

import (
	"context"
	"log/slog"

	"github.com/coreos/go-oidc/v3/oidc"
)

var OIDCVerifier *oidc.IDTokenVerifier

const authProvider = "https://login.maxstash.io/application/o/squares/"

func init() {
	provider, err := oidc.NewProvider(context.Background(), authProvider)
	if err != nil {
		slog.Error("failed to create oidc provider", "error", err)
		panic(err)
	}

	OIDCVerifier = provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
	slog.Info("oidc configuration initialized")
}