package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOIDCVerifier_ReturnsNilWhenNotInitialized(t *testing.T) {
	original := oidcVerifier
	oidcVerifier = nil
	defer func() { oidcVerifier = original }()

	assert.Nil(t, OIDCVerifier())
}
