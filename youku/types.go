package youku

// Show holds all the public metadata for a single Youku video show.
type Show struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Year        string `json:"year,omitempty"`
	Score       string `json:"score,omitempty"`
	Duration    string `json:"duration,omitempty"`
	CoverURL    string `json:"cover_url"`
	URL         string `json:"url"`
}

// --- Search API response types ---

// SearchResponse is the top-level JSON from the Youku search API.
type SearchResponse struct {
	Message  string   `json:"message"`
	More     bool     `json:"more"`
	PageData PageData `json:"pageData"`
}

// PageData holds pagination info and the search result blocks.
type PageData struct {
	Total  int           `json:"total"`
	Pg     int           `json:"pg"`
	Pz     int           `json:"pz"`
	IsEnd  int           `json:"isEnd"`
	Status []StatusBlock `json:"status"`
}

// StatusBlock is one result block in the search response.
// Type 1027 is the primary video result block.
type StatusBlock struct {
	Type   int          `json:"type"`
	ShowID *string      `json:"showId"` // pointer to handle null
	Action *ActionData  `json:"action,omitempty"`
	Data   []ShowCard   `json:"data,omitempty"`
}

// ActionData contains title and showId in the action block.
type ActionData struct {
	ShowID *string  `json:"showId"`
	Title  string   `json:"title"`
}

// ShowCard is a show entry within a status block's data list.
type ShowCard struct {
	ShowID    *string `json:"showId"`
	ShowTitle string  `json:"showTitle"`
	ImgURL    string  `json:"imgUrl"`
	Score     string  `json:"score"`
	Year      string  `json:"year"`
	Type      string  `json:"type"`
}
