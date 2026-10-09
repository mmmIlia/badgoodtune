package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	UsernameMinLength = 3
	UsernameMaxLength = 32
	PasswordMinLength = 12
	PasswordMaxLength = 128
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

type User struct {
	ID        ID
	Username  string
	CreatedAt time.Time
}

func NormalizeUsername(username string) string {
	return strings.TrimSpace(username)
}

func ValidateRegistration(username, password string) (string, error) {
	normalized := NormalizeUsername(username)
	violations := validateUsername(normalized)

	passwordLength := utf8.RuneCountInString(password)
	if passwordLength < PasswordMinLength || passwordLength > PasswordMaxLength {
		violations = append(violations, Violation{
			Field:       "password",
			Description: fmt.Sprintf("must contain between %d and %d characters", PasswordMinLength, PasswordMaxLength),
		})
	}

	return normalized, NewValidationError(violations...)
}

func ValidateLogin(username, password string) (string, error) {
	normalized := NormalizeUsername(username)
	violations := make([]Violation, 0, 2)
	if normalized == "" || utf8.RuneCountInString(normalized) > UsernameMaxLength {
		violations = append(violations, Violation{Field: "username", Description: "is invalid"})
	}
	if password == "" || utf8.RuneCountInString(password) > PasswordMaxLength {
		violations = append(violations, Violation{Field: "password", Description: "is invalid"})
	}
	return normalized, NewValidationError(violations...)
}

func validateUsername(username string) []Violation {
	length := utf8.RuneCountInString(username)
	if length < UsernameMinLength || length > UsernameMaxLength {
		return []Violation{{
			Field:       "username",
			Description: fmt.Sprintf("must contain between %d and %d characters", UsernameMinLength, UsernameMaxLength),
		}}
	}
	if !usernamePattern.MatchString(username) {
		return []Violation{{
			Field:       "username",
			Description: "may contain only ASCII letters, digits, and underscore",
		}}
	}
	return nil
}
