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