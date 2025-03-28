package helpers

import (
	"errors"
	"strings"
)

func GetSplitPart(target string, separate string, index int) (string, error) {
	parts := strings.Split(target, separate)
	if index >= len(parts) {
		return "", errors.New("index out of range")
	}
	return parts[index], nil
}
