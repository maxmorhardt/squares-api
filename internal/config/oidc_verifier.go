package config

import (
	"context"
	"log/slog"

	"github.com/coreos/go-oidc/v3/oidc"
)

var (
	OIDCVerifier *oidc.IDTokenVerifier
)

func init() {
	provider, err := oidc.NewProvider(context.Background(), "https://auth.maxstash.io/realms/maxstash")
	if err != nil {
		slog.Error("unable to create oidc provider", "err", err)
		panic(err)
	}

	OIDCVerifier = provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
}