package util

import (
	"regexp"
	"unicode"
)

var dangerousChars = regexp.MustCompile(`[<>{}[\]\\|` + "`" + `]`)

func IsSafeString(s string) bool {
	return !dangerousChars.MatchString(s)
}

func CapitalizeFirstLetter(err error) string {
	if err == nil {
		return ""
	}

	message := err.Error()
	if message == "" {
		return ""
	}

	runes := []rune(message)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}

	return string(runes)
}
