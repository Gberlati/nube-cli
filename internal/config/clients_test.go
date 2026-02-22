package config

import "testing"

func TestNormalizeClientName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"lowercase", "myapp", "myapp", false},
		{"uppercase becomes lowercase", "MyApp", "myapp", false},
		{"with dashes", "my-app", "my-app", false},
		{"with underscores", "my_app", "my_app", false},
		{"with dots", "my.app", "my.app", false},
		{"with numbers", "app123", "app123", false},
		{"empty", "", "", true},
		{"whitespace only", "   ", "", true},
		{"invalid chars spaces", "my app", "", true},
		{"invalid chars special", "my@app", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeClientName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeClientName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("NormalizeClientName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeClientNameOrDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty returns default", "", DefaultClientName, false},
		{"whitespace returns default", "   ", DefaultClientName, false},
		{"valid name", "myapp", "myapp", false},
		{"invalid chars", "my@app", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeClientNameOrDefault(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeClientNameOrDefault(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("NormalizeClientNameOrDefault(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
