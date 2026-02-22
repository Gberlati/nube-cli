package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// PageInfo contains parsed Link header URLs.
type PageInfo struct {
	Next  string
	Prev  string
	First string
	Last  string
}

// HasNext returns true if there is a next page.
func (p PageInfo) HasNext() bool { return p.Next != "" }

// ParseLinkHeader parses an RFC 5988 Link header into a PageInfo.
// Example: `<https://api.tiendanube.com/v1/123/products?page=2>; rel="next"`
func ParseLinkHeader(header string) PageInfo {
	var info PageInfo

	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		segments := strings.SplitN(part, ";", 2)
		if len(segments) != 2 {
			continue
		}

		rawURL := strings.TrimSpace(segments[0])
		rawURL = strings.TrimPrefix(rawURL, "<")
		rawURL = strings.TrimSuffix(rawURL, ">")

		rel := strings.TrimSpace(segments[1])
		rel = strings.TrimPrefix(rel, "rel=")
		rel = strings.Trim(rel, `"`)

		switch rel {
		case "next":
			info.Next = rawURL
		case "prev":
			info.Prev = rawURL
		case "first":
			info.First = rawURL
		case "last":
			info.Last = rawURL
		}
	}

	return info
}

// CollectAllPages follows pagination links to collect all items.
// The decode function is called for each page response to extract items.
func CollectAllPages[T any](
	ctx context.Context,
	client *Client,
	path string,
	query url.Values,
	decode func(*http.Response) ([]T, error),
) ([]T, error) {
	var all []T

	currentPath := path
	currentQuery := query

	for {
		resp, err := client.Get(ctx, currentPath, currentQuery) //nolint:bodyclose // decode callback closes body
		if err != nil {
			return nil, fmt.Errorf("fetch page: %w", err)
		}

		// Read Link header before decode closes the body.
		linkHeader := resp.Header.Get("Link")

		items, decodeErr := decode(resp)
		if decodeErr != nil {
			return nil, fmt.Errorf("decode page: %w", decodeErr)
		}

		all = append(all, items...)

		if linkHeader == "" {
			break
		}

		pageInfo := ParseLinkHeader(linkHeader)
		if !pageInfo.HasNext() {
			break
		}

		// Parse the next URL to extract path and query.
		nextURL, parseErr := url.Parse(pageInfo.Next)
		if parseErr != nil {
			return nil, fmt.Errorf("parse next page URL: %w", parseErr)
		}

		currentPath = nextURL.Path
		currentQuery = nextURL.Query()
	}

	return all, nil
}
