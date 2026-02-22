package outfmt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Mode struct {
	JSON  bool
	Plain bool
}

type ParseError struct{ msg string }

func (e *ParseError) Error() string { return e.msg }

func FromFlags(jsonOut bool, plainOut bool) (Mode, error) {
	if jsonOut && plainOut {
		return Mode{}, &ParseError{msg: "invalid output mode (cannot combine --json and --plain)"}
	}

	return Mode{JSON: jsonOut, Plain: plainOut}, nil
}

func FromEnv() Mode {
	return Mode{
		JSON:  envBool("NUBE_JSON"),
		Plain: envBool("NUBE_PLAIN"),
	}
}

type ctxKey struct{}

func WithMode(ctx context.Context, mode Mode) context.Context {
	return context.WithValue(ctx, ctxKey{}, mode)
}

func FromContext(ctx context.Context) Mode {
	if v := ctx.Value(ctxKey{}); v != nil {
		if m, ok := v.(Mode); ok {
			return m
		}
	}

	return Mode{}
}

func IsJSON(ctx context.Context) bool  { return FromContext(ctx).JSON }
func IsPlain(ctx context.Context) bool { return FromContext(ctx).Plain }

// JSONTransform configures JSON output transformations.
type JSONTransform struct {
	// Select projects objects to only the requested fields (comma-separated; supports dot paths).
	// When applied to a list, it projects each element.
	Select []string
}

type transformCtxKey struct{}

// WithJSONTransform stores a JSONTransform in the context.
func WithJSONTransform(ctx context.Context, t JSONTransform) context.Context {
	return context.WithValue(ctx, transformCtxKey{}, t)
}

// JSONTransformFromContext retrieves the JSONTransform from the context.
func JSONTransformFromContext(ctx context.Context) JSONTransform {
	if v := ctx.Value(transformCtxKey{}); v != nil {
		if t, ok := v.(JSONTransform); ok {
			return t
		}
	}

	return JSONTransform{}
}

// WriteJSON encodes v as indented JSON. If a JSONTransform is in the context,
// it applies field selection before encoding.
func WriteJSON(ctx context.Context, w io.Writer, v any) error {
	transform := JSONTransformFromContext(ctx)
	if len(transform.Select) > 0 {
		v = ApplyJSONTransform(v, transform)
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	return nil
}

// ApplyJSONTransform applies field selection to the given data.
// Data must be JSON-compatible (maps, slices, primitives).
func ApplyJSONTransform(data any, transform JSONTransform) any {
	if len(transform.Select) == 0 {
		return data
	}

	// Convert structured types to map[string]any for field selection.
	normalized := normalizeForSelect(data)

	return selectFields(normalized, transform.Select)
}

func normalizeForSelect(v any) any {
	// If already a map or slice, return as-is.
	switch v.(type) {
	case map[string]any, []any:
		return v
	}

	// Marshal/unmarshal to convert struct types to maps.
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}

	var result any
	if json.Unmarshal(b, &result) != nil {
		return v
	}

	return result
}

func selectFields(v any, fields []string) any {
	switch vv := v.(type) {
	case []any:
		out := make([]any, 0, len(vv))
		for _, it := range vv {
			out = append(out, selectFieldsFromItem(it, fields))
		}

		return out
	default:
		return selectFieldsFromItem(v, fields)
	}
}

func selectFieldsFromItem(v any, fields []string) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}

	out := make(map[string]any, len(fields))

	for _, f := range fields {
		if val, ok := getAtPath(m, f); ok {
			out[f] = val
		}
	}

	return out
}

func getAtPath(v any, path string) (any, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}

	segs := strings.Split(path, ".")
	cur := v

	for _, seg := range segs {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return nil, false
		}

		switch c := cur.(type) {
		case map[string]any:
			next, ok := c[seg]
			if !ok {
				return nil, false
			}

			cur = next
		case []any:
			i, err := strconv.Atoi(seg)
			if err != nil || i < 0 || i >= len(c) {
				return nil, false
			}

			cur = c[i]
		default:
			return nil, false
		}
	}

	return cur, true
}

func KeyValuePayload(key string, value any) map[string]any {
	return map[string]any{
		"key":   key,
		"value": value,
	}
}

func KeysPayload(keys []string) map[string]any {
	return map[string]any{
		"keys": keys,
	}
}

func PathPayload(path string) map[string]any {
	return map[string]any{
		"path": path,
	}
}

func envBool(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
