package outfmt_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/gberlati/nube-cli/internal/outfmt"
)

func TestFromFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		json      bool
		plain     bool
		wantJSON  bool
		wantPlain bool
		wantErr   bool
	}{
		{"default", false, false, false, false, false},
		{"json", true, false, true, false, false},
		{"plain", false, true, false, true, false},
		{"both errors", true, true, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mode, err := outfmt.FromFlags(tt.json, tt.plain)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FromFlags() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				return
			}

			if mode.JSON != tt.wantJSON {
				t.Errorf("JSON = %v, want %v", mode.JSON, tt.wantJSON)
			}

			if mode.Plain != tt.wantPlain {
				t.Errorf("Plain = %v, want %v", mode.Plain, tt.wantPlain)
			}
		})
	}
}

func TestFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		jsonEnv   string
		plainEnv  string
		wantJSON  bool
		wantPlain bool
	}{
		{"empty", "", "", false, false},
		{"json true", "1", "", true, false},
		{"plain true", "", "true", false, true},
		{"json yes", "yes", "", true, false},
		{"json on", "on", "", true, false},
		{"json y", "y", "", true, false},
		{"invalid value", "no", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NUBE_JSON", tt.jsonEnv)
			t.Setenv("NUBE_PLAIN", tt.plainEnv)

			mode := outfmt.FromEnv()

			if mode.JSON != tt.wantJSON {
				t.Errorf("JSON = %v, want %v", mode.JSON, tt.wantJSON)
			}

			if mode.Plain != tt.wantPlain {
				t.Errorf("Plain = %v, want %v", mode.Plain, tt.wantPlain)
			}
		})
	}
}

func TestContextRoundtrip(t *testing.T) {
	t.Parallel()

	mode := outfmt.Mode{JSON: true}
	ctx := outfmt.WithMode(context.Background(), mode)

	got := outfmt.FromContext(ctx)
	if got.JSON != true {
		t.Error("expected JSON=true after roundtrip")
	}

	if outfmt.IsJSON(ctx) != true {
		t.Error("IsJSON should return true")
	}

	if outfmt.IsPlain(ctx) != false {
		t.Error("IsPlain should return false")
	}
}

func TestFromContext_Empty(t *testing.T) {
	t.Parallel()

	mode := outfmt.FromContext(context.Background())
	if mode.JSON || mode.Plain {
		t.Error("empty context should return zero Mode")
	}
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	data := map[string]string{"key": "value"}

	if err := outfmt.WriteJSON(context.Background(), &buf, data); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got["key"] != "value" {
		t.Errorf("got %q, want %q", got["key"], "value")
	}

	// Verify indented output
	if !bytes.Contains(buf.Bytes(), []byte("  ")) {
		t.Error("expected indented JSON output")
	}
}

func TestKeyValuePayload(t *testing.T) {
	t.Parallel()

	p := outfmt.KeyValuePayload("name", "foo")
	if p["key"] != "name" {
		t.Errorf("key = %v, want %q", p["key"], "name")
	}

	if p["value"] != "foo" {
		t.Errorf("value = %v, want %q", p["value"], "foo")
	}
}

func TestKeysPayload(t *testing.T) {
	t.Parallel()

	keys := []string{"a", "b"}
	p := outfmt.KeysPayload(keys)

	got, ok := p["keys"].([]string)
	if !ok {
		t.Fatal("keys not []string")
	}

	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("keys = %v, want [a b]", got)
	}
}

func TestPathPayload(t *testing.T) {
	t.Parallel()

	p := outfmt.PathPayload("/foo/bar")
	if p["path"] != "/foo/bar" {
		t.Errorf("path = %v, want %q", p["path"], "/foo/bar")
	}
}

func TestJSONTransformContext(t *testing.T) {
	t.Parallel()

	transform := outfmt.JSONTransform{Select: []string{"id", "name"}}
	ctx := outfmt.WithJSONTransform(context.Background(), transform)

	got := outfmt.JSONTransformFromContext(ctx)
	if len(got.Select) != 2 || got.Select[0] != "id" || got.Select[1] != "name" {
		t.Errorf("Select = %v, want [id name]", got.Select)
	}
}

func TestJSONTransformFromContext_Empty(t *testing.T) {
	t.Parallel()

	got := outfmt.JSONTransformFromContext(context.Background())
	if len(got.Select) != 0 {
		t.Errorf("empty context should return empty transform, got %v", got)
	}
}

func TestApplyJSONTransform_Object(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"id":   float64(1),
		"name": "Widget",
		"desc": "A fine widget",
	}

	result := outfmt.ApplyJSONTransform(data, outfmt.JSONTransform{Select: []string{"id", "name"}})

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["id"] != float64(1) {
		t.Errorf("id = %v, want 1", m["id"])
	}

	if m["name"] != "Widget" {
		t.Errorf("name = %v, want Widget", m["name"])
	}

	if _, has := m["desc"]; has {
		t.Error("desc should be filtered out")
	}
}

func TestApplyJSONTransform_Array(t *testing.T) {
	t.Parallel()

	data := []any{
		map[string]any{"id": float64(1), "name": "A", "extra": "x"},
		map[string]any{"id": float64(2), "name": "B", "extra": "y"},
	}

	result := outfmt.ApplyJSONTransform(data, outfmt.JSONTransform{Select: []string{"id", "name"}})

	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}

	if len(arr) != 2 {
		t.Fatalf("expected 2 items, got %d", len(arr))
	}

	first := arr[0].(map[string]any)
	if first["id"] != float64(1) || first["name"] != "A" {
		t.Errorf("first = %v", first)
	}

	if _, has := first["extra"]; has {
		t.Error("extra should be filtered out")
	}
}

func TestApplyJSONTransform_NestedPath(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"id": float64(1),
		"name": map[string]any{
			"en": "English",
			"es": "Spanish",
		},
		"price": float64(99),
	}

	result := outfmt.ApplyJSONTransform(data, outfmt.JSONTransform{Select: []string{"id", "name.en"}})

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["id"] != float64(1) {
		t.Errorf("id = %v, want 1", m["id"])
	}

	if m["name.en"] != "English" {
		t.Errorf("name.en = %v, want English", m["name.en"])
	}

	if _, has := m["price"]; has {
		t.Error("price should be filtered out")
	}
}

func TestApplyJSONTransform_MissingFields(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"id": float64(1),
	}

	result := outfmt.ApplyJSONTransform(data, outfmt.JSONTransform{Select: []string{"id", "nonexistent"}})

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["id"] != float64(1) {
		t.Errorf("id = %v, want 1", m["id"])
	}

	if _, has := m["nonexistent"]; has {
		t.Error("nonexistent should not be in result")
	}
}

func TestApplyJSONTransform_NoSelect(t *testing.T) {
	t.Parallel()

	data := map[string]any{"id": float64(1)}
	result := outfmt.ApplyJSONTransform(data, outfmt.JSONTransform{})

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if m["id"] != float64(1) {
		t.Errorf("id = %v, want 1", m["id"])
	}
}

func TestWriteJSON_WithSelect(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"id":   float64(1),
		"name": "Widget",
		"desc": "A fine widget",
	}

	ctx := outfmt.WithJSONTransform(context.Background(), outfmt.JSONTransform{Select: []string{"id"}})

	var buf bytes.Buffer
	if err := outfmt.WriteJSON(ctx, &buf, data); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got["id"] != float64(1) {
		t.Errorf("id = %v, want 1", got["id"])
	}

	if _, has := got["name"]; has {
		t.Error("name should be filtered out by --select")
	}
}
