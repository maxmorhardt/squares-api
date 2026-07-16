package config

import (
	"testing"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitNATS_InvalidURL(t *testing.T) {
	cfg := &model.AppConfig{}
	cfg.NATS.URL = "nats://127.0.0.1:1"

	conn, err := InitNATS(cfg)

	require.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "failed to connect to NATS")
}

func TestInitNATS_MalformedURL(t *testing.T) {
	cfg := &model.AppConfig{}
	cfg.NATS.URL = "not-a-valid-url"

	conn, err := InitNATS(cfg)

	require.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "failed to connect to NATS")
}
