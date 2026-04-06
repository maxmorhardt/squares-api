package util

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSafeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"safe alphanumeric", "hello123", true},
		{"safe with spaces", "hello world", true},
		{"safe with punctuation", "hello, world!", true},
		{"safe with hyphens", "my-name", true},
		{"empty string", "", true},
		{"contains <", "hello<world", false},
		{"contains >", "hello>world", false},
		{"contains {", "hello{world", false},
		{"contains }", "hello}world", false},
		{"contains [", "hello[world", false},
		{"contains ]", "hello]world", false},
		{"contains backslash", `hello\world`, false},
		{"contains pipe", "hello|world", false},
		{"contains backtick", "hello`world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSafeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCapitalizeFirstLetter(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, ""},
		{"empty message", errors.New(""), ""},
		{"lowercase start", errors.New("something went wrong"), "Something went wrong"},
		{"uppercase start", errors.New("Already uppercase"), "Already uppercase"},
		{"single char", errors.New("a"), "A"},
		{"number start", errors.New("404 not found"), "404 not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapitalizeFirstLetter(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
