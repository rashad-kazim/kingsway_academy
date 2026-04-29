package domain

import (
	"errors"
	"regexp"
	"strings"
)

const FINLength = 7

var finPattern = regexp.MustCompile(`^[A-Z0-9]{7}$`)

var ErrInvalidFIN = errors.New("fin must be 7 uppercase alphanumeric characters")

type FIN string

func ParseFIN(value string) (FIN, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if !finPattern.MatchString(normalized) {
		return "", ErrInvalidFIN
	}

	return FIN(normalized), nil
}

func (f FIN) String() string {
	return string(f)
}
