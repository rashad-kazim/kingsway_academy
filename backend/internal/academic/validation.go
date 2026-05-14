package academic

import (
	"strings"
	"time"

	"kingsway/backend/internal/domain"
)

func normalizeClockTime(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if _, err := time.Parse("15:04", value); err != nil {
		return "", domain.ErrInvalidInput
	}

	return value, nil
}

func normalizeBirthDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	parsed, err := time.Parse("02/01/2006", value)
	if err != nil {
		return "", domain.ErrInvalidInput
	}

	return parsed.Format("02/01/2006"), nil
}

func isHireDateOnOrAfterBirthDate(birthDate string, hiredAt string) bool {
	if birthDate == "" || hiredAt == "" {
		return true
	}
	birth, err := time.Parse("02/01/2006", birthDate)
	if err != nil {
		return false
	}
	hire, err := time.Parse("02/01/2006", hiredAt)
	if err != nil {
		return false
	}

	return !hire.Before(birth)
}
