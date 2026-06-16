package youku

import "context"

// hotKeywords maps content type to search keywords that surface hot results.
var hotKeywords = map[string][]string{
	"movie":       {"热门电影", "口碑电影"},
	"tv":          {"热门电视剧", "热播剧"},
	"variety":     {"热门综艺", "热播综艺"},
	"documentary": {"热门纪录片", "纪录片"},
	"":            {"热门视频", "热播"},
}

// Hot returns up to limit hot/trending shows by content type.
// It uses the search API with predefined popular keywords per type,
// deduplicating by show ID.
func (c *Client) Hot(ctx context.Context, contentType string, limit int) ([]Show, error) {
	if limit <= 0 {
		limit = 20
	}

	keywords, ok := hotKeywords[contentType]
	if !ok {
		keywords = hotKeywords[""]
	}

	seen := map[string]bool{}
	var out []Show

	for _, kw := range keywords {
		if len(out) >= limit {
			break
		}
		shows, err := c.Search(ctx, kw, limit*2)
		if err != nil {
			continue
		}
		for _, s := range shows {
			if len(out) >= limit {
				break
			}
			if seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			out = append(out, s)
		}
	}

	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
