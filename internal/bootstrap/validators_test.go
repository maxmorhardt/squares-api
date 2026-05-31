package bootstrap

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupValidators(t *testing.T) {
	setupValidators()

	v, ok := binding.Validator.Engine().(*validator.Validate)
	require.True(t, ok)

	// safestring
	assert.NoError(t, v.Var("hello world", "safestring"))
	assert.NoError(t, v.Var("", "safestring"), "empty is allowed")
	assert.Error(t, v.Var("bad<script>", "safestring"))

	// contestname
	assert.NoError(t, v.Var("My-Contest_1", "contestname"))
	assert.NoError(t, v.Var("", "contestname"), "empty is allowed")
	assert.Error(t, v.Var("bad@name!", "contestname"))
}
