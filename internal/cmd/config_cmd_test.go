package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConfigPath(t *testing.T) {
	setupConfigDir(t)
	buf := captureStdout(t)

	err := Execute([]string{"config", "path"})
	if err != nil {
		t.Fatalf("Execute error = %v", err)
	}

	if !strings.Contains(buf.String(), "config.json") {
		t.Errorf("output = %q, want containing config.json", buf.String())
	}
}

func TestConfigPath_JSON(t *testing.T) {
	setupConfigDir(t)
	buf := captureStdout(t)

	err := Execute([]string{"config", "path", "--json"})
	if err != nil {
		t.Fatalf("Execute error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (output: %q)", err, buf.String())
	}

	if got["path"] == nil {
		t.Error("expected path in JSON output")
	}
}

func TestConfigKeys(t *testing.T) {
	setupConfigDir(t)
	buf := captureStdout(t)

	err := Execute([]string{"config", "keys"})
	if err != nil {
		t.Fatalf("Execute error = %v", err)
	}

	if !strings.Contains(buf.String(), "keyring_backend") {
		t.Errorf("output = %q, want containing keyring_backend", buf.String())
	}
}

func TestConfigGetSetRoundtrip(t *testing.T) {
	setupConfigDir(t)

	// Set
	buf := captureStdout(t)
	err := Execute([]string{"config", "set", "keyring_backend", "file"})
	if err != nil {
		t.Fatalf("set error = %v", err)
	}
	_ = buf.String()
}

func TestConfigGetSetRoundtrip_Get(t *testing.T) {
	setupConfigDir(t)

	// Set first
	_ = captureStdout(t)
	_ = Execute([]string{"config", "set", "keyring_backend", "file"})

	// Now get
	buf2 := captureStdout(t)
	err := Execute([]string{"config", "get", "keyring_backend"})
	if err != nil {
		t.Fatalf("get error = %v", err)
	}

	if !strings.Contains(buf2.String(), "file") {
		t.Errorf("get output = %q, want containing 'file'", buf2.String())
	}
}

func TestConfigUnset(t *testing.T) {
	setupConfigDir(t)

	// Set then unset
	_ = captureStdout(t)
	_ = Execute([]string{"config", "set", "keyring_backend", "file"})

	buf := captureStdout(t)
	err := Execute([]string{"config", "unset", "keyring_backend"})
	if err != nil {
		t.Fatalf("unset error = %v", err)
	}

	if !strings.Contains(buf.String(), "Unset") {
		t.Errorf("output = %q, want containing 'Unset'", buf.String())
	}
}

func TestConfigList(t *testing.T) {
	setupConfigDir(t)
	buf := captureStdout(t)

	err := Execute([]string{"config", "list"})
	if err != nil {
		t.Fatalf("Execute error = %v", err)
	}

	if !strings.Contains(buf.String(), "Config file") {
		t.Errorf("output = %q, want containing 'Config file'", buf.String())
	}
}

func TestConfigGet_InvalidKey(t *testing.T) {
	setupConfigDir(t)
	_ = captureStdout(t)

	err := Execute([]string{"config", "get", "nonexistent_key"})
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}
