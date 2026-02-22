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

// CustomerCmd groups customer-related commands.
type CustomerCmd struct {
	List CustomerListCmd `cmd:"" help:"List customers"`
	Get  CustomerGetCmd  `cmd:"" help:"Get a customer by ID"`
}

// CustomerListCmd lists customers with pagination and filters.
type CustomerListCmd struct {
	PaginationFlags `embed:""`

	SinceID    string `help:"Return customers after this ID" name:"since-id"`
	Query      string `help:"Search query" short:"q" name:"q"`
	Email      string `help:"Filter by email" name:"email"`
	CreatedMin string `help:"Created after (ISO 8601)" name:"created-at-min"`
	CreatedMax string `help:"Created before (ISO 8601)" name:"created-at-max"`
	UpdatedMin string `help:"Updated after (ISO 8601)" name:"updated-at-min"`
	UpdatedMax string `help:"Updated before (ISO 8601)" name:"updated-at-max"`
	Fields     string `help:"Comma-separated fields to return from API" name:"fields"`
}

func (c *CustomerListCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	client, err := newAPIClient(flags)
	if err != nil {
		return err
	}

	q := url.Values{}
	c.Apply(q)
	addQueryParam(q, "since_id", c.SinceID)
	addQueryParam(q, "q", c.Query)
	addQueryParam(q, "email", c.Email)
	addQueryParam(q, "created_at_min", c.CreatedMin)
	addQueryParam(q, "created_at_max", c.CreatedMax)
	addQueryParam(q, "updated_at_min", c.UpdatedMin)
	addQueryParam(q, "updated_at_max", c.UpdatedMax)
	addQueryParam(q, "fields", c.Fields)

	var items []map[string]any

	if c.WantsAllPages() {
		items, err = api.CollectAllPages(ctx, client, "customers", q, decodeList)
	} else {
		var resp *http.Response
		resp, err = client.Get(ctx, "customers", q) //nolint:bodyclose // decodeList closes body
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

	_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE\tCREATED")

	for _, cust := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", //nolint:gosec // tabwriter, not HTML
			jsonStr(cust, "id"),
			jsonStr(cust, "name"),
			jsonStr(cust, "email"),
			jsonStr(cust, "phone"),
			jsonStr(cust, "created_at"),
		)
	}

	_ = u

	return nil
}

// CustomerGetCmd fetches a single customer by ID.
type CustomerGetCmd struct {
	CustomerID string `arg:"" name:"customer-id" help:"Customer ID"`
	Fields     string `help:"Comma-separated fields to return from API" name:"fields"`
}

func (c *CustomerGetCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	client, err := newAPIClient(flags)
	if err != nil {
		return err
	}

	q := url.Values{}
	addQueryParam(q, "fields", c.Fields)

	resp, err := client.Get(ctx, "customers/"+c.CustomerID, q) //nolint:bodyclose // DecodeResponse closes body
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
		kv("name", jsonStr(data, "name")),
		kv("email", jsonStr(data, "email")),
		kv("phone", jsonStr(data, "phone")),
		kv("created_at", jsonStr(data, "created_at")),
		kv("updated_at", jsonStr(data, "updated_at")),
	)
}
