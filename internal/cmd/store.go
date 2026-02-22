package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gberlati/nube-cli/internal/api"
	"github.com/gberlati/nube-cli/internal/outfmt"
	"github.com/gberlati/nube-cli/internal/ui"
)

// StoreCmd groups store-related commands.
type StoreCmd struct {
	Get StoreGetCmd `cmd:"" default:"withargs" help:"Show store information"`
}

// StoreGetCmd fetches store info from the API.
type StoreGetCmd struct{}

func (c *StoreGetCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	client, err := newAPIClient(flags)
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, "store", nil) //nolint:bodyclose // DecodeResponse closes body
	if err != nil {
		return err
	}

	data, err := api.DecodeResponse[map[string]any](resp)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, data)
	}

	return writeResult(ctx, u,
		kv("id", jsonStr(data, "id")),
		kv("name", extractI18n(data, "name")),
		kv("email", jsonStr(data, "email")),
		kv("domain", jsonStr(data, "original_domain")),
		kv("plan", jsonStr(data, "plan_name")),
	)
}

// decodeList is a generic response decoder for paginated list endpoints.
func decodeList(resp *http.Response) ([]map[string]any, error) {
	return api.DecodeResponse[[]map[string]any](resp)
}

// jsonStr extracts a string-like value from a map, handling numbers too.
func jsonStr(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}

		return fmt.Sprintf("%g", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
