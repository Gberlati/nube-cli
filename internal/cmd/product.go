package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/gberlati/nube-cli/internal/api"
	"github.com/gberlati/nube-cli/internal/outfmt"
	"github.com/gberlati/nube-cli/internal/ui"
)

// ProductCmd groups product-related commands.
type ProductCmd struct {
	List     ProductListCmd     `cmd:"" help:"List products"`
	Get      ProductGetCmd      `cmd:"" help:"Get a product by ID"`
	GetBySku ProductGetBySkuCmd `cmd:"" name:"get-by-sku" help:"Get a product by SKU"`
}

// ProductListCmd lists products with pagination and filters.
type ProductListCmd struct {
	PaginationFlags `embed:""`

	IDs          string `help:"Comma-separated product IDs" name:"ids"`
	SinceID      string `help:"Return products after this ID" name:"since-id"`
	Query        string `help:"Search query" short:"q" name:"q"`
	Handle       string `help:"Filter by URL handle" name:"handle"`
	CategoryID   string `help:"Filter by category ID" name:"category-id"`
	Published    string `help:"Filter by published status (true/false)" name:"published"`
	FreeShipping string `help:"Filter by free shipping (true/false)" name:"free-shipping"`
	CreatedMin   string `help:"Created after (ISO 8601)" name:"created-at-min"`
	CreatedMax   string `help:"Created before (ISO 8601)" name:"created-at-max"`
	UpdatedMin   string `help:"Updated after (ISO 8601)" name:"updated-at-min"`
	UpdatedMax   string `help:"Updated before (ISO 8601)" name:"updated-at-max"`
	SortBy       string `help:"Sort field (e.g. created-at-ascending)" name:"sort-by"`
	Fields       string `help:"Comma-separated fields to return from API" name:"fields"`
}

func (c *ProductListCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	client, err := newAPIClient(flags)
	if err != nil {
		return err
	}

	q := url.Values{}
	c.Apply(q)
	addQueryParam(q, "ids", c.IDs)
	addQueryParam(q, "since_id", c.SinceID)
	addQueryParam(q, "q", c.Query)
	addQueryParam(q, "handle", c.Handle)
	addQueryParam(q, "category_id", c.CategoryID)
	addQueryParam(q, "published", c.Published)
	addQueryParam(q, "free_shipping", c.FreeShipping)
	addQueryParam(q, "created_at_min", c.CreatedMin)
	addQueryParam(q, "created_at_max", c.CreatedMax)
	addQueryParam(q, "updated_at_min", c.UpdatedMin)
	addQueryParam(q, "updated_at_max", c.UpdatedMax)
	addQueryParam(q, "sort_by", c.SortBy)
	addQueryParam(q, "fields", c.Fields)

	var items []map[string]any

	if c.WantsAllPages() {
		items, err = api.CollectAllPages(ctx, client, "products", q, decodeList)
	} else {
		var resp *http.Response
		resp, err = client.Get(ctx, "products", q) //nolint:bodyclose // decodeList closes body
		if err == nil {
			items, err = decodeList(resp)
		}
	}

	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, items)
	}

	w, done := tableWriter(ctx)
	defer done()

	_, _ = fmt.Fprintln(w, "ID\tNAME\tHANDLE\tPUBLISHED\tVARIANTS\tPRICE")

	for _, p := range items {
		name := extractI18n(p, "name")
		variants := countVariants(p)
		price := firstVariantPrice(p)

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n", //nolint:gosec // tabwriter, not HTML
			jsonStr(p, "id"),
			name,
			extractI18n(p, "handle"),
			jsonStr(p, "published"),
			variants,
			price,
		)
	}

	_ = u

	return nil
}

// ProductGetCmd fetches a single product by ID.
type ProductGetCmd struct {
	ProductID string `arg:"" name:"product-id" help:"Product ID"`
	Fields    string `help:"Comma-separated fields to return from API" name:"fields"`
}

func (c *ProductGetCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	client, err := newAPIClient(flags)
	if err != nil {
		return err
	}

	q := url.Values{}
	addQueryParam(q, "fields", c.Fields)

	resp, err := client.Get(ctx, "products/"+c.ProductID, q) //nolint:bodyclose // DecodeResponse closes body
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
		kv("handle", extractI18n(data, "handle")),
		kv("published", jsonStr(data, "published")),
		kv("variants", countVariants(data)),
		kv("created_at", jsonStr(data, "created_at")),
		kv("updated_at", jsonStr(data, "updated_at")),
	)
}

// ProductGetBySkuCmd fetches a product by SKU.
type ProductGetBySkuCmd struct {
	SKU string `arg:"" name:"sku" help:"Product variant SKU"`
}

func (c *ProductGetBySkuCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	client, err := newAPIClient(flags)
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, "products/sku/"+c.SKU, nil) //nolint:bodyclose // DecodeResponse closes body
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
		kv("handle", extractI18n(data, "handle")),
		kv("sku", c.SKU),
	)
}

func countVariants(p map[string]any) int {
	variants, ok := p["variants"]
	if !ok {
		return 0
	}

	arr, ok := variants.([]any)
	if !ok {
		return 0
	}

	return len(arr)
}

func firstVariantPrice(p map[string]any) string {
	variants, ok := p["variants"]
	if !ok {
		return ""
	}

	arr, ok := variants.([]any)
	if !ok || len(arr) == 0 {
		return ""
	}

	first, ok := arr[0].(map[string]any)
	if !ok {
		return ""
	}

	return jsonStr(first, "price")
}
