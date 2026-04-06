package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNATS_ReturnsNilWhenNotInitialized(t *testing.T) {
	original := natsConn
	natsConn = nil
	defer func() { natsConn = original }()

	assert.Nil(t, NATS())
}

func TestCloseNATS_NilConnection(t *testing.T) {
	original := natsConn
	natsConn = nil
	defer func() { natsConn = original }()

	// Should not panic
	assert.NotPanics(t, func() {
		CloseNATS()
	})
}
