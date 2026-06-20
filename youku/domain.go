package youku

import (
	"context"
	"regexp"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the Youku kit driver.
type Domain struct{}

// Info describes the scheme, hostnames, and binary identity.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme:  "youku",
		Aliases: []string{"yk"},
		Hosts:   []string{Host, "www.youku.com", "v.youku.com"},
		Identity: kit.Identity{
			Binary: "youku",
			Short:  "Read public Youku (优酷) video data",
			Long: `youku reads public Youku (优酷, youku.com) video data over plain HTTPS,
shapes it into clean records, and prints output that pipes into the rest of
your tools. No API key or login required.

Quick start:
  youku search "旅行" -n 10       search for videos
  youku hot                       hot shows (20 results)
  youku hot --type movie -n 10    hot movies
  youku video <id>                full show metadata
  youku search "python" -o json   JSON output`,
			Site: Host,
			Repo: "https://github.com/tamnd/youku-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "shows",
		Summary: "Search shows by keyword",
		Args:    []kit.Arg{{Name: "query", Help: "search keyword"}},
	}, searchOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "hot",
		Group:   "shows",
		Summary: "List hot/trending shows",
		Args:    nil,
	}, hotOp)

	kit.Handle(app, kit.OpMeta{
		Name:     "video",
		Group:    "shows",
		Single:   true,
		Resolver: true,
		URIType:  "show",
		Summary:  "Fetch a show by ID",
		Args:     []kit.Arg{{Name: "id", Help: "show ID"}},
	}, videoOp)
}

// newClient builds a Client from the kit Config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- input structs ---

type searchInput struct {
	Query  string  `kit:"arg" help:"search keyword"`
	Limit  int     `kit:"flag,inherit" help:"max results" default:"20"`
	Client *Client `kit:"inject"`
}

type hotInput struct {
	Type   string  `kit:"flag" help:"content type: movie, tv, variety, documentary"`
	Limit  int     `kit:"flag,inherit" help:"max results" default:"20"`
	Client *Client `kit:"inject"`
}

type videoInput struct {
	ID     string  `kit:"arg" help:"show ID"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func searchOp(ctx context.Context, in searchInput, emit func(Show) error) error {
	if in.Query == "" {
		return errs.Usage("youku search: query argument is required")
	}
	shows, err := in.Client.Search(ctx, in.Query, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for _, s := range shows {
		if err := emit(s); err != nil {
			return err
		}
	}
	return nil
}

func hotOp(ctx context.Context, in hotInput, emit func(Show) error) error {
	shows, err := in.Client.Hot(ctx, in.Type, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for _, s := range shows {
		if err := emit(s); err != nil {
			return err
		}
	}
	return nil
}

func videoOp(ctx context.Context, in videoInput, emit func(*Show) error) error {
	id := cleanShowID(in.ID)
	if id == "" {
		return errs.Usage("youku video: show ID is required")
	}
	show, err := in.Client.VideoDetail(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	return emit(show)
}

// Classify turns any accepted input into the canonical (uriType, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", errs.Usage("youku: empty input")
	}
	// Full URL like https://v.youku.com/v_show/id_<id>.html
	if strings.Contains(input, "v_show/id_") {
		if id = extractShowID(input); id != "" {
			return "show", id, nil
		}
	}
	// Bare show ID
	if looksLikeShowID(input) {
		return "show", input, nil
	}
	return "", "", errs.Usage("youku: unrecognized reference: %q", input)
}

// Locate returns the canonical URL for a (uriType, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "show":
		return showPageURL(id), nil
	default:
		return "", errs.Usage("youku has no resource type %q", uriType)
	}
}

// mapErr converts library errors into kit error kinds.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if err == ErrNotFound {
		return errs.NotFound("%s", err.Error())
	}
	if err == ErrRateLimited {
		return errs.RateLimited("%s", err.Error())
	}
	return err
}

var showIDFromURLRE = regexp.MustCompile(`v_show/id_([^.?#/]+)`)
var showIDRE = regexp.MustCompile(`^[A-Za-z0-9_=+/-]{8,40}$`)

// extractShowID extracts the show ID from a Youku show URL.
func extractShowID(s string) string {
	m := showIDFromURLRE.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// looksLikeShowID returns true when s could be a Youku show ID.
// Youku IDs are either hex strings or base64-encoded IDs starting with X.
func looksLikeShowID(s string) bool {
	return showIDRE.MatchString(s)
}

// cleanShowID strips URL context from a show reference.
func cleanShowID(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "v_show/id_") {
		return extractShowID(s)
	}
	return s
}
