package httpapi

import "github.com/mmmIlia/badgoodtune/internal/domain"

func missingBodyField(field string) error {
	return domain.NewValidationError(domain.Violation{Field: field, Description: "is required"})
}

func stringPointer(value string) *string { return &value }

func valueOrZero[T ~int](value *T) int {
	if value == nil {
		return 0
	}
	return int(*value)
}

func valueOrEmpty[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}
