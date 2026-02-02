package validators

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

func ValidateContestName(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	if len(name) == 0 {
		return true
	}

	matches, _ := regexp.MatchString(`^[A-Za-z0-9\s\-_]+$`, name)
	return matches
}

// Blocks: < > { } [ ] \ | ` and other control characters
func ValidateSafeString(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	if len(str) == 0 {
		return true
	}

	// reject dangerous characters
	dangerousChars := regexp.MustCompile(`[<>{}[\]\\|` + "`" + `]`)
	return !dangerousChars.MatchString(str)
}
