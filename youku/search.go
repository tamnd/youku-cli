package youku

import (
	"context"
	"fmt"
)

const searchPageSize = 15

// Search returns up to limit shows matching keyword.
// It pages through the Youku JSON search API and extracts Show records from
// the result blocks.
func (c *Client) Search(ctx context.Context, keyword string, limit int) ([]Show, error) {
	if limit <= 0 {
		limit = 20
	}
	if keyword == "" {
		return nil, fmt.Errorf("youku search: keyword is required")
	}

	seen := map[string]bool{}
	var out []Show
	page := 1

	for len(out) < limit {
		url := fmt.Sprintf("%s?keyword=%s&site=1&source=&atm=&appid=340&pageNo=%d&pageSize=%d",
			c.cfg.SearchBase, percentEncode(keyword), page, searchPageSize)

		var resp SearchResponse
		if err := c.getJSON(ctx, url, &resp); err != nil {
			if len(out) > 0 {
				break
			}
			return nil, err
		}

		if resp.Message != "success" {
			if len(out) > 0 {
				break
			}
			return nil, ErrRateLimited
		}

		shows := extractShowsFromResponse(resp)
		for _, s := range shows {
			if len(out) >= limit {
				break
			}
			if s.ID == "" || seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			out = append(out, s)
		}

		if resp.PageData.IsEnd == 1 || len(shows) == 0 {
			break
		}
		page++
	}

	return out, nil
}

// extractShowsFromResponse pulls Show records from the complex search response.
func extractShowsFromResponse(resp SearchResponse) []Show {
	var out []Show
	for _, block := range resp.PageData.Status {
		// Try ActionData first (type 1027 blocks)
		if block.Action != nil && block.Action.ShowID != nil && *block.Action.ShowID != "" {
			show := Show{
				ID:    *block.Action.ShowID,
				Title: block.Action.Title,
				URL:   showPageURL(*block.Action.ShowID),
			}
			out = append(out, show)
			continue
		}
		// Try block-level ShowID with action title
		if block.ShowID != nil && *block.ShowID != "" {
			title := ""
			if block.Action != nil {
				title = block.Action.Title
			}
			show := Show{
				ID:    *block.ShowID,
				Title: title,
				URL:   showPageURL(*block.ShowID),
			}
			out = append(out, show)
			continue
		}
		// Try Data list (ShowCard entries)
		for _, card := range block.Data {
			if card.ShowID == nil || *card.ShowID == "" {
				continue
			}
			show := Show{
				ID:       *card.ShowID,
				Title:    card.ShowTitle,
				Type:     card.Type,
				Year:     card.Year,
				Score:    card.Score,
				CoverURL: card.ImgURL,
				URL:      showPageURL(*card.ShowID),
			}
			out = append(out, show)
		}
	}
	return out
}

// percentEncode encodes a string for use as a URL query value.
func percentEncode(s string) string {
	encoded := make([]byte, 0, len(s)*3)
	for i := 0; i < len(s); i++ {
		b := s[i]
		if isUnreservedByte(b) {
			encoded = append(encoded, b)
		} else {
			encoded = append(encoded, '%', nibbleToHex(b>>4), nibbleToHex(b&0xf))
		}
	}
	return string(encoded)
}

func isUnreservedByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') ||
		b == '-' || b == '_' || b == '.' || b == '~'
}

func nibbleToHex(v byte) byte {
	if v < 10 {
		return '0' + v
	}
	return 'A' + v - 10
}
