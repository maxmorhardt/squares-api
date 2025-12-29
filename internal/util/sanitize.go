package util

import (
	"regexp"
	"strings"
	"unicode"
)

func SanitizeInput(input string) string {
	// trim leading and trailing whitespace
	input = strings.TrimSpace(input)

	// remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// remove/replace email header injection characters (CR, LF)
	input = strings.ReplaceAll(input, "\r", "")
	input = strings.ReplaceAll(input, "\n", " ")

	// remove other control characters except tab
	input = removeControlCharacters(input)

	return input
}

func removeControlCharacters(input string) string {
	var result strings.Builder
	result.Grow(len(input))

	for _, r := range input {
		// allow printable characters, spaces, and tabs
		if unicode.IsPrint(r) || r == '\t' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

func SanitizeForHTML(input string) string {
	replacements := map[string]string{
		"&":  "&amp;",
		"<":  "&lt;",
		">":  "&gt;",
		"\"": "&#34;",
		"'":  "&#39;",
	}

	for old, new := range replacements {
		input = strings.ReplaceAll(input, old, new)
	}

	return input
}

func NormalizeWhitespace(input string) string {
	// replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	return spaceRegex.ReplaceAllString(strings.TrimSpace(input), " ")
}
