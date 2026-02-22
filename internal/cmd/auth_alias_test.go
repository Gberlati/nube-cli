package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAuthAlias_SetList(t *testing.T) {
	setupConfigDir(t)
	buf := captureStdout(t)

	err := Execute([]string{"auth", "alias", "set", "prod", "store@example.com"})
	if err != nil {
		t.Fatalf("set error = %v", err)
	}

	if !strings.Contains(buf.String(), "prod") {
		t.Errorf("set output = %q, want containing 'prod'", buf.String())
	}
}

func TestAuthAlias_List(t *testing.T) {
	setupConfigDir(t)
	_ = captureStdout(t)
	_ = Execute([]string{"auth", "alias", "set", "prod", "store@example.com"})

	buf := captureStdout(t)
	err := Execute([]string{"auth", "alias", "list"})
	if err != nil {
		t.Fatalf("list error = %v", err)
	}

	if !strings.Contains(buf.String(), "prod") {
		t.Errorf("list output = %q, want containing 'prod'", buf.String())
	}
}

func TestAuthAlias_Unset(t *testing.T) {
	setupConfigDir(t)
	_ = captureStdout(t)
	_ = Execute([]string{"auth", "alias", "set", "prod", "store@example.com"})

	buf := captureStdout(t)
	err := Execute([]string{"auth", "alias", "unset", "prod"})
	if err != nil {
		t.Fatalf("unset error = %v", err)
	}
	_ = buf.String()
}

func TestAuthAlias_Unset_NotFound(t *testing.T) {
	setupConfigDir(t)
	_ = captureStdout(t)

	err := Execute([]string{"auth", "alias", "unset", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for not found alias")
	}
}

func TestAuthAlias_Set_AtValidation(t *testing.T) {
	setupConfigDir(t)
	_ = captureStdout(t)

	err := Execute([]string{"auth", "alias", "set", "user@bad", "store@example.com"})
	if err == nil {
		t.Fatal("expected error for @ in alias")
	}
}

func TestAuthAlias_Set_ReservedName(t *testing.T) {
	setupConfigDir(t)
	_ = captureStdout(t)

	err := Execute([]string{"auth", "alias", "set", "auto", "store@example.com"})
	if err == nil {
		t.Fatal("expected error for reserved alias name")
	}
}

func TestAuthAlias_JSON(t *testing.T) {
	setupConfigDir(t)

	_ = captureStdout(t)
	_ = Execute([]string{"auth", "alias", "set", "prod", "store@example.com"})

	buf := captureStdout(t)
	err := Execute([]string{"auth", "alias", "list", "--json"})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (output: %q)", err, buf.String())
	}

	if got["aliases"] == nil {
		t.Error("expected aliases in JSON output")
	}
}
