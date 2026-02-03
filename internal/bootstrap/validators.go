package bootstrap

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func setupValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("contestname", validateContestName)
		_ = v.RegisterValidation("safestring", validateSafeString)
	}
}

func validateContestName(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	if len(name) == 0 {
		return true
	}

	matches, _ := regexp.MatchString(`^[A-Za-z0-9\s\-_]+$`, name)
	return matches
}

// Blocks: < > { } [ ] \ | ` and other control characters
func validateSafeString(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	if len(str) == 0 {
		return true
	}

	dangerousChars := regexp.MustCompile(`[<>{}[\]\\|` + "`" + `]`)
	return !dangerousChars.MatchString(str)
}
