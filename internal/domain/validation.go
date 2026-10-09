package domain

import "unicode/utf8"

func validateText(field, value string, maxLength int) []Violation {
	length := utf8.RuneCountInString(value)
	if length == 0 {
		return []Violation{{Field: field, Description: "must not be empty"}}
	}
	if length > maxLength {
		return []Violation{{Field: field, Description: "is too long"}}
	}
	return nil
}
