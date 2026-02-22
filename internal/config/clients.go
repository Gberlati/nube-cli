package config

import (
	"errors"
	"fmt"
	"strings"
)

const DefaultClientName = "default"

var errInvalidClientName = errors.New("invalid client name")

func NormalizeClientName(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	if name == "" {
		return "", fmt.Errorf("%w: empty", errInvalidClientName)
	}

	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}

		return "", fmt.Errorf("%w: %q", errInvalidClientName, raw)
	}

	return name, nil
}

func NormalizeClientNameOrDefault(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return DefaultClientName, nil
	}

	return NormalizeClientName(raw)
}
