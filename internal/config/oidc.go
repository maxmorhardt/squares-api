package config

import (
	"context"
	"log/slog"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
)

var OIDCVerifier *oidc.IDTokenVerifier

const authProvider = "https://login.maxstash.io"

func InitOIDC(validateClientId bool) {
	clientID := os.Getenv("OIDC_CLIENT_ID")
	if clientID == "" {
		slog.Error("OIDC_CLIENT_ID environment variable is not set")
		panic("OIDC_CLIENT_ID environment variable is required")
	}

	provider, err := oidc.NewProvider(context.Background(), authProvider)
	if err != nil {
		slog.Error("failed to create oidc provider", "error", err)
		panic(err)
	}
	
	oidcConfig := &oidc.Config{
		SkipClientIDCheck: true,
	}

	if validateClientId {
		oidcConfig = &oidc.Config{
			ClientID: clientID,
		}
	}

	OIDCVerifier = provider.Verifier(oidcConfig)
	slog.Info("oidc configuration initialized")
}
