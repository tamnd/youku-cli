package youku

import (
	"context"
	"fmt"
)

// VideoDetail fetches full metadata for a show by its show ID using OGP scraping.
func (c *Client) VideoDetail(ctx context.Context, showID string) (*Show, error) {
	if showID == "" {
		return nil, fmt.Errorf("youku: empty show ID")
	}
	url := c.cfg.ShowBase + showID + ".html"
	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	show := parseShowFromHTML(showID, string(body))
	if show.Title == "" {
		return nil, ErrNotFound
	}
	return show, nil
}

// parseShowFromHTML builds a Show from OGP tags in show page HTML.
func parseShowFromHTML(id, html string) *Show {
	ogp := parseOGP(html)
	title := cleanTitle(ogp["og:title"])
	if title == "" {
		title = cleanTitle(ogp["twitter:title"])
	}
	description := ogp["og:description"]
	if description == "" {
		description = ogp["twitter:description"]
	}
	coverURL := ogp["og:image"]
	if coverURL == "" {
		coverURL = ogp["twitter:image"]
	}
	ogType := ogp["og:type"]
	canonicalURL := ogp["og:url"]
	if canonicalURL == "" {
		canonicalURL = showPageURL(id)
	}

	return &Show{
		ID:          id,
		Title:       title,
		Type:        ogType,
		Description: description,
		CoverURL:    coverURL,
		URL:         canonicalURL,
	}
}
