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

// CategoryCmd groups category-related commands.
type CategoryCmd struct {
	List CategoryListCmd `cmd:"" help:"List categories"`
	Get  CategoryGetCmd  `cmd:"" help:"Get a category by ID"`
}

// CategoryListCmd lists categories with pagination and filters.
type CategoryListCmd struct {
	PaginationFlags `embed:""`

	SinceID    string `help:"Return categories after this ID" name:"since-id"`
	Language   string `help:"Filter by language code" name:"language"`
	Handle     string `help:"Filter by URL handle" name:"handle"`
	ParentID   string `help:"Filter by parent category ID" name:"parent-id"`
	CreatedMin string `help:"Created after (ISO 8601)" name:"created-at-min"`
	CreatedMax string `help:"Created before (ISO 8601)" name:"created-at-max"`
	UpdatedMin string `help:"Updated after (ISO 8601)" name:"updated-at-min"`
	UpdatedMax string `help:"Updated before (ISO 8601)" name:"updated-at-max"`
	Fields     string `help:"Comma-separated fields to return from API" name:"fields"`
}

func (c *CategoryListCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	client, err := newAPIClient(flags)
	if err != nil {
		return err
	}

	q := url.Values{}
	c.Apply(q)
	addQueryParam(q, "since_id", c.SinceID)
	addQueryParam(q, "language", c.Language)
	addQueryParam(q, "handle", c.Handle)
	addQueryParam(q, "parent_id", c.ParentID)
	addQueryParam(q, "created_at_min", c.CreatedMin)
	addQueryParam(q, "created_at_max", c.CreatedMax)
	addQueryParam(q, "updated_at_min", c.UpdatedMin)
	addQueryParam(q, "updated_at_max", c.UpdatedMax)
	addQueryParam(q, "fields", c.Fields)

	var items []map[string]any

	if c.WantsAllPages() {
		items, err = api.CollectAllPages(ctx, client, "categories", q, decodeList)
	} else {
		var resp *http.Response
		resp, err = client.Get(ctx, "categories", q) //nolint:bodyclose // decodeList closes body
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

	_, _ = fmt.Fprintln(w, "ID\tNAME\tHANDLE\tPARENT\tSUBCATEGORIES")

	for _, cat := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n", //nolint:gosec // tabwriter, not HTML
			jsonStr(cat, "id"),
			extractI18n(cat, "name"),
			extractI18n(cat, "handle"),
			jsonStr(cat, "parent"),
			countSubcategories(cat),
		)
	}

	_ = u

	return nil
}

// CategoryGetCmd fetches a single category by ID.
type CategoryGetCmd struct {
	CategoryID string `arg:"" name:"category-id" help:"Category ID"`
	Fields     string `help:"Comma-separated fields to return from API" name:"fields"`
}

func (c *CategoryGetCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	client, err := newAPIClient(flags)
	if err != nil {
		return err
	}

	q := url.Values{}
	addQueryParam(q, "fields", c.Fields)

	resp, err := client.Get(ctx, "categories/"+c.CategoryID, q) //nolint:bodyclose // DecodeResponse closes body
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
		kv("parent", jsonStr(data, "parent")),
		kv("subcategories", countSubcategories(data)),
		kv("created_at", jsonStr(data, "created_at")),
		kv("updated_at", jsonStr(data, "updated_at")),
	)
}

func countSubcategories(cat map[string]any) int {
	subs, ok := cat["subcategories"]
	if !ok {
		return 0
	}

	arr, ok := subs.([]any)
	if !ok {
		return 0
	}

	return len(arr)
}
