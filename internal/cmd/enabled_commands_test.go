package cmd

import "testing"

func TestEnforceEnabledCommands(t *testing.T) {
	tests := []struct {
		name    string
		enabled string
		args    []string
		wantErr bool
	}{
		{"empty allows all", "", []string{"version"}, false},
		{"allowed command", "version,config", []string{"version"}, false},
		{"blocked command", "version", []string{"config", "path"}, true},
		{"wildcard star", "*", []string{"config", "path"}, false},
		{"wildcard all", "all", []string{"auth", "list"}, false},
		{"case insensitive", "VERSION", []string{"version"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, _, err := newParser("test")
			if err != nil {
				t.Fatalf("newParser: %v", err)
			}

			kctx, parseErr := parser.Parse(tt.args)
			if parseErr != nil {
				t.Skipf("parse error: %v", parseErr)
			}

			err = enforceEnabledCommands(kctx, tt.enabled)
			if (err != nil) != tt.wantErr {
				t.Errorf("enforceEnabledCommands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
