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

// OrderCmd groups order-related commands.
type OrderCmd struct {
	List OrderListCmd `cmd:"" help:"List orders"`
	Get  OrderGetCmd  `cmd:"" help:"Get an order by ID"`
}

// OrderListCmd lists orders with pagination and filters.
type OrderListCmd struct {
	PaginationFlags `embed:""`

	SinceID        string `help:"Return orders after this ID" name:"since-id"`
	Status         string `help:"Filter by status (open/closed/cancelled)" name:"status"`
	PaymentStatus  string `help:"Filter by payment status (pending/authorized/paid/voided/refunded)" name:"payment-status"`
	ShippingStatus string `help:"Filter by shipping status (unpacked/shipped/unshipped/delivered)" name:"shipping-status"`
	Channels       string `help:"Filter by sales channel" name:"channels"`
	CreatedMin     string `help:"Created after (ISO 8601)" name:"created-at-min"`
	CreatedMax     string `help:"Created before (ISO 8601)" name:"created-at-max"`
	UpdatedMin     string `help:"Updated after (ISO 8601)" name:"updated-at-min"`
	UpdatedMax     string `help:"Updated before (ISO 8601)" name:"updated-at-max"`
	CustomerIDs    string `help:"Comma-separated customer IDs" name:"customer-ids"`
	Query          string `help:"Search query" short:"q" name:"q"`
	Fields         string `help:"Comma-separated fields to return from API" name:"fields"`
	Aggregates     string `help:"Comma-separated aggregates to include" name:"aggregates"`
}

func (c *OrderListCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	client, err := newAPIClient(flags)
	if err != nil {
		return err
	}

	q := url.Values{}
	c.Apply(q)
	addQueryParam(q, "since_id", c.SinceID)
	addQueryParam(q, "status", c.Status)
	addQueryParam(q, "payment_status", c.PaymentStatus)
	addQueryParam(q, "shipping_status", c.ShippingStatus)
	addQueryParam(q, "channels", c.Channels)
	addQueryParam(q, "created_at_min", c.CreatedMin)
	addQueryParam(q, "created_at_max", c.CreatedMax)
	addQueryParam(q, "updated_at_min", c.UpdatedMin)
	addQueryParam(q, "updated_at_max", c.UpdatedMax)
	addQueryParam(q, "customer_ids", c.CustomerIDs)
	addQueryParam(q, "q", c.Query)
	addQueryParam(q, "fields", c.Fields)
	addQueryParam(q, "aggregates", c.Aggregates)

	var items []map[string]any

	if c.WantsAllPages() {
		items, err = api.CollectAllPages(ctx, client, "orders", q, decodeList)
	} else {
		var resp *http.Response
		resp, err = client.Get(ctx, "orders", q) //nolint:bodyclose // decodeList closes body
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

	_, _ = fmt.Fprintln(w, "ID\tNUMBER\tSTATUS\tPAYMENT\tSHIPPING\tTOTAL\tCREATED")

	for _, o := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", //nolint:gosec // tabwriter, not HTML
			jsonStr(o, "id"),
			jsonStr(o, "number"),
			jsonStr(o, "status"),
			jsonStr(o, "payment_status"),
			jsonStr(o, "shipping_status"),
			jsonStr(o, "total"),
			jsonStr(o, "created_at"),
		)
	}

	_ = u

	return nil
}

// OrderGetCmd fetches a single order by ID.
type OrderGetCmd struct {
	OrderID    string `arg:"" name:"order-id" help:"Order ID"`
	Fields     string `help:"Comma-separated fields to return from API" name:"fields"`
	Aggregates string `help:"Comma-separated aggregates to include" name:"aggregates"`
}

func (c *OrderGetCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	client, err := newAPIClient(flags)
	if err != nil {
		return err
	}

	q := url.Values{}
	addQueryParam(q, "fields", c.Fields)
	addQueryParam(q, "aggregates", c.Aggregates)

	resp, err := client.Get(ctx, "orders/"+c.OrderID, q) //nolint:bodyclose // DecodeResponse closes body
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
		kv("number", jsonStr(data, "number")),
		kv("status", jsonStr(data, "status")),
		kv("payment_status", jsonStr(data, "payment_status")),
		kv("shipping_status", jsonStr(data, "shipping_status")),
		kv("total", jsonStr(data, "total")),
		kv("currency", jsonStr(data, "currency")),
		kv("created_at", jsonStr(data, "created_at")),
		kv("updated_at", jsonStr(data, "updated_at")),
	)
}
