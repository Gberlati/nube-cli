package config

import "testing"

func setupConfigDir(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
}
