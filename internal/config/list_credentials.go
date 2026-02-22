package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CredentialInfo describes a stored credentials file.
type CredentialInfo struct {
	Client  string
	Path    string
	Default bool
}

// ListClientCredentials returns all stored credential files.
func ListClientCredentials() ([]CredentialInfo, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("read config dir: %w", err)
	}

	var out []CredentialInfo

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()

		if name == "credentials.json" {
			out = append(out, CredentialInfo{
				Client:  DefaultClientName,
				Path:    filepath.Join(dir, name),
				Default: true,
			})

			continue
		}

		if strings.HasPrefix(name, "credentials-") && strings.HasSuffix(name, ".json") {
			client := strings.TrimSuffix(strings.TrimPrefix(name, "credentials-"), ".json")
			out = append(out, CredentialInfo{
				Client:  client,
				Path:    filepath.Join(dir, name),
				Default: false,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Client < out[j].Client })

	return out, nil
}
