package youku

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(searchBase, showBase string) *Client {
	cfg := DefaultConfig()
	cfg.SearchBase = searchBase
	cfg.ShowBase = showBase
	cfg.Rate = 0
	cfg.Retries = 0
	return NewClient(cfg)
}

func strPtr(s string) *string { return &s }

func makeSearchResponse(showID, title string, isEnd int) SearchResponse {
	return SearchResponse{
		Message: "success",
		More:    isEnd != 1,
		PageData: PageData{
			Total: 1,
			Pg:    1,
			Pz:    1,
			IsEnd: isEnd,
			Status: []StatusBlock{
				{
					Type:   1027,
					ShowID: strPtr(showID),
					Action: &ActionData{
						ShowID: strPtr(showID),
						Title:  title,
					},
				},
			},
		},
	}
}

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := makeSearchResponse("show001", "旅行视频", 1)
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, srv.URL+"/v_show/id_")
	shows, err := c.Search(context.Background(), "旅行", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(shows) == 0 {
		t.Fatal("Search: got 0 shows")
	}
	if shows[0].ID != "show001" {
		t.Errorf("ID = %q", shows[0].ID)
	}
	if shows[0].Title != "旅行视频" {
		t.Errorf("Title = %q", shows[0].Title)
	}
}

func TestSearchShowCards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := SearchResponse{
			Message: "success",
			PageData: PageData{
				IsEnd: 1,
				Status: []StatusBlock{
					{
						Type: 1052,
						Data: []ShowCard{
							{ShowID: strPtr("card001"), ShowTitle: "Python教程", Year: "2022", Score: "9.0"},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, srv.URL+"/v_show/id_")
	shows, err := c.Search(context.Background(), "python", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(shows) == 0 {
		t.Fatal("Search from ShowCards: got 0 shows")
	}
	if shows[0].ID != "card001" {
		t.Errorf("ID = %q", shows[0].ID)
	}
	if shows[0].Year != "2022" {
		t.Errorf("Year = %q", shows[0].Year)
	}
}

func TestSearchRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "error"})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, srv.URL+"/v_show/id_")
	_, err := c.Search(context.Background(), "test", 5)
	if err != ErrRateLimited {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestVideoDetail(t *testing.T) {
	const sampleHTML = `<!DOCTYPE html>
<html>
<head>
<meta property="og:title" content="Python编程教程—优酷">
<meta property="og:description" content="全面的Python教程">
<meta property="og:image" content="https://example.com/cover.jpg">
<meta property="og:type" content="video.other">
<meta property="og:url" content="https://v.youku.com/v_show/id_testid.html">
</head>
<body>Test</body>
</html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(sampleHTML))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, srv.URL+"/v_show/id_")
	show, err := c.VideoDetail(context.Background(), "testid")
	if err != nil {
		t.Fatalf("VideoDetail: %v", err)
	}
	if show.Title != "Python编程教程" {
		t.Errorf("Title = %q", show.Title)
	}
	if show.Description != "全面的Python教程" {
		t.Errorf("Description = %q", show.Description)
	}
}

func TestVideoDetailNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Return page with no OGP title
		w.Write([]byte("<html><head><title>404</title></head><body>Not found</body></html>"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, srv.URL+"/v_show/id_")
	_, err := c.VideoDetail(context.Background(), "nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestParseOGP(t *testing.T) {
	html := `<meta property="og:title" content="Test Title">
<meta property="og:description" content="Test desc">`
	ogp := parseOGP(html)
	if ogp["og:title"] != "Test Title" {
		t.Errorf("og:title = %q", ogp["og:title"])
	}
	if ogp["og:description"] != "Test desc" {
		t.Errorf("og:description = %q", ogp["og:description"])
	}
}

func TestCleanTitle(t *testing.T) {
	cases := []struct{ in, want string }{
		{"测试剧集—优酷", "测试剧集"},
		{"测试剧集_优酷", "测试剧集"},
		{"测试剧集 - 优酷", "测试剧集"},
		{"测试剧集", "测试剧集"},
	}
	for _, tc := range cases {
		got := cleanTitle(tc.in)
		if got != tc.want {
			t.Errorf("cleanTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPercentEncode(t *testing.T) {
	got := percentEncode("hello world")
	if !strings.Contains(got, "%") {
		t.Errorf("percentEncode(%q) = %q, expected percent-encoded", "hello world", got)
	}
	got2 := percentEncode("python")
	if got2 != "python" {
		t.Errorf("percentEncode(%q) = %q, want python", "python", got2)
	}
}
