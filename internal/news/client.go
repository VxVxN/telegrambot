package news

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FeedURL is the Go blog Atom feed.
const FeedURL = "https://go.dev/blog/feed.atom"

// Item is a single news entry from the feed.
type Item struct {
	Title     string
	URL       string
	Author    string
	Summary   string
	Published time.Time
}

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	Author    string     `xml:"author>name"`
	Summary   string     `xml:"summary"`
	Published string     `xml:"published"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type Client struct {
	http *http.Client
	url  string
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 10 * time.Second},
		url:  FeedURL,
	}
}

// FetchLatest returns feed items, newest first. If limit > 0, at most limit items
// are returned.
func (c *Client) FetchLatest(limit int) ([]Item, error) {
	body, err := c.get(c.url)
	if err != nil {
		return nil, err
	}

	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}

	items := make([]Item, 0, len(feed.Entries))
	for _, e := range feed.Entries {
		items = append(items, Item{
			Title:     e.Title,
			URL:       e.link(),
			Author:    e.Author,
			Summary:   e.Summary,
			Published: parseTime(e.Published),
		})
	}

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

// link picks the alternate (human-readable) link, falling back to the first one.
func (e atomEntry) link() string {
	for _, l := range e.Links {
		if l.Rel == "alternate" || l.Rel == "" {
			return l.Href
		}
	}
	if len(e.Links) > 0 {
		return e.Links[0].Href
	}
	return ""
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (c *Client) get(url string) ([]byte, error) {
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
