package util

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func InitialsFromName(name string) string {
	var initials strings.Builder
	count := 0
	for part := range strings.FieldsSeq(name) {
		r, _ := utf8.DecodeRuneInString(part)
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			continue
		}

		initials.WriteRune(unicode.ToUpper(r))
		count++
		if count >= 3 {
			break
		}
	}

	return initials.String()
}
