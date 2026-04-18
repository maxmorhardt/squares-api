package bootstrap

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-playground/validator/v10"
)

func TestSetupValidators(t *testing.T) {
	setupValidators()

	v, ok := binding.Validator.Engine().(*validator.Validate)
	require.True(t, ok)

	// verify contestname validator is registered and works (gin uses "binding" tag)
	type cnField struct {
		Name string `binding:"contestname"`
	}
	assert.NoError(t, v.Struct(cnField{Name: "Valid-Name_1"}))
	assert.Error(t, v.Struct(cnField{Name: "<script>"}))

	// verify safestring validator is registered and works
	type ssField struct {
		Value string `binding:"safestring"`
	}
	assert.NoError(t, v.Struct(ssField{Value: "hello world"}))
	assert.Error(t, v.Struct(ssField{Value: "<bad>"}))
}

func newValidatorWithCustomRules(t *testing.T) *validator.Validate {
	t.Helper()
	v := validator.New()
	require.NoError(t, v.RegisterValidation("contestname", validateContestName))
	require.NoError(t, v.RegisterValidation("safestring", validateSafeString))
	return v
}

type contestNameField struct {
	Name string `validate:"contestname"`
}

type safeStringField struct {
	Value string `validate:"safestring"`
}

func TestValidateContestName(t *testing.T) {
	v := newValidatorWithCustomRules(t)

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"empty", "", true},
		{"alphanumeric", "SuperBowl2025", true},
		{"with spaces", "Super Bowl", true},
		{"with hyphens", "super-bowl", true},
		{"with underscores", "super_bowl", true},
		{"mixed", "My Contest-1_2", true},
		{"special chars", "bowl<script>", false},
		{"curly braces", "bowl{}", false},
		{"pipe", "a|b", false},
		{"backtick", "a`b", false},
		{"brackets", "a[b]", false},
		{"emoji", "bowl🏈", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(contestNameField{Name: tt.input})
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateSafeString(t *testing.T) {
	v := newValidatorWithCustomRules(t)

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"empty", "", true},
		{"normal text", "hello world", true},
		{"with numbers", "abc123", true},
		{"angle brackets", "<script>", false},
		{"curly braces", "{bad}", false},
		{"square brackets", "[bad]", false},
		{"backslash", `path\to`, false},
		{"pipe", "a|b", false},
		{"backtick", "a`b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(safeStringField{Value: tt.input})
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
