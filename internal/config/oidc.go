package config

import (
	"context"
	"log/slog"

	"github.com/coreos/go-oidc/v3/oidc"
)

var oidcVerifier *oidc.IDTokenVerifier

func InitOIDC() {
	provider, err := oidc.NewProvider(context.Background(), Env().OIDC.Issuer)
	if err != nil {
		slog.Error("failed to create oidc provider", "error", err)
		panic(err)
	}
	
	oidcConfig := &oidc.Config{
		ClientID: Env().OIDC.ClientID,
	}
	oidcVerifier = provider.Verifier(oidcConfig)
	
	slog.Info("oidc configuration initialized")
}

func OIDCVerifier() *oidc.IDTokenVerifier {
	return oidcVerifier
}