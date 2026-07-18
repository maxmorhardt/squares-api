package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialsFromName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single name", "Max", "M"},
		{"first and last", "Max Morhardt", "MM"},
		{"three parts", "Max John Morhardt", "MJM"},
		{"caps at three", "Max John Robert Morhardt", "MJR"},
		{"extra whitespace", "  Max   Morhardt  ", "MM"},
		{"lowercase input", "max morhardt", "MM"},
		{"leading number", "3 Doors Down", "3DD"},
		{"skips punctuation-led part", "Max -Morhardt", "M"},
		{"empty string", "", ""},
		{"whitespace only", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, InitialsFromName(tt.input))
		})
	}
}
