package config

import (
	"testing"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitOIDC_InvalidIssuer(t *testing.T) {
	cfg := &model.AppConfig{}
	cfg.OIDC.Issuer = "http://127.0.0.1:1/nonexistent/"
	cfg.OIDC.ClientID = "test-client"

	verifier, err := InitOIDC(cfg)

	require.Error(t, err)
	assert.Nil(t, verifier)
	assert.Contains(t, err.Error(), "failed to create oidc provider")
}

func TestInitOIDC_MalformedIssuerURL(t *testing.T) {
	cfg := &model.AppConfig{}
	cfg.OIDC.Issuer = "not-a-valid-url"
	cfg.OIDC.ClientID = "test-client"

	verifier, err := InitOIDC(cfg)

	require.Error(t, err)
	assert.Nil(t, verifier)
	assert.Contains(t, err.Error(), "failed to create oidc provider")
}
