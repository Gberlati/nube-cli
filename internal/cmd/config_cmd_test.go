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
