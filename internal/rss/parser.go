package rss

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"time"
)

// Parser parses RSS 2.0 and Atom 1.0 feeds
type Parser struct {
	client *http.Client
}

// NewParser creates a new RSS parser
func NewParser() *Parser {
	return &Parser{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewParserWithClient creates a parser with a custom HTTP client
func NewParserWithClient(client *http.Client) *Parser {
	return &Parser{client: client}
}

// ParseResult holds the parsed feed data
type ParseResult struct {
	Title       string
	Link        string
	Description string
	FeedType    FeedType
	Items       []*FeedItem
}

// Parse fetches and parses an RSS feed from the given URL
func (p *Parser) Parse(feedURL string) (*ParseResult, error) {
	return p.ParseWithContext(context.Background(), feedURL)
}

// ParseWithContext fetches and parses an RSS feed with context
func (p *Parser) ParseWithContext(ctx context.Context, feedURL string) (*ParseResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return p.ParseBytes(data)
}

// ParseBytes parses RSS/Atom data from bytes
func (p *Parser) ParseBytes(data []byte) (*ParseResult, error) {
	// Detect feed type
	feedType := p.detectFeedType(data)

	switch feedType {
	case FeedTypeRSS2:
		return p.parseRSS2(data)
	case FeedTypeAtom:
		return p.parseAtom(data)
	default:
		return nil, ErrUnknownFeedType
	}
}

// detectFeedType detects the feed type from XML content
func (p *Parser) detectFeedType(data []byte) FeedType {
	// Check for RSS 2.0
	if bytes.Contains(data, []byte("<rss")) {
		return FeedTypeRSS2
	}
	// Check for Atom
	if bytes.Contains(data, []byte("<feed")) {
		return FeedTypeAtom
	}
	return FeedTypeUnknown
}

// RSS2 feed structures
type rss2Channel struct {
	XMLName     xml.Name `xml:"rss"`
	Version     string   `xml:"version,attr"`
	Title       string   `xml:"channel>title"`
	Link        string   `xml:"channel>link"`
	Description string   `xml:"channel>description"`
	Items       []rss2Item `xml:"channel>item"`
}

type rss2Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Content     string `xml:"content:encoded"`
	Author      string `xml:"author"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func (p *Parser) parseRSS2(data []byte) (*ParseResult, error) {
	var channel rss2Channel
	if err := xml.Unmarshal(data, &channel); err != nil {
		return nil, err
	}

	items := make([]*FeedItem, 0, len(channel.Items))
	for _, item := range channel.Items {
		pubDate := p.parseTime(item.PubDate)

		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}

		content := item.Content
		if content == "" {
			content = item.Description
		}

		items = append(items, &FeedItem{
			GUID:        guid,
			Title:       cleanText(item.Title),
			Link:        item.Link,
			Description: cleanText(item.Description),
			Content:     content,
			Author:      cleanText(item.Author),
			PublishedAt: pubDate,
		})
	}

	return &ParseResult{
		Title:       cleanText(channel.Title),
		Link:        channel.Link,
		Description: cleanText(channel.Description),
		FeedType:    FeedTypeRSS2,
		Items:       items,
	}, nil
}

// Atom feed structures
type atomFeed struct {
	XMLName  xml.Name    `xml:"feed"`
	Title    string      `xml:"title"`
	Link     []atomLink  `xml:"link"`
	Subtitle string      `xml:"subtitle"`
	Entries  []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href  string `xml:"href,attr"`
	Rel   string `xml:"rel,attr"`
	Type  string `xml:"type,attr"`
}

type atomEntry struct {
	Title     string      `xml:"title"`
	Link      []atomLink  `xml:"link"`
	Summary   string      `xml:"summary"`
	Content   string      `xml:"content"`
	Author    atomAuthor  `xml:"author"`
	Published string      `xml:"published"`
	Updated   string      `xml:"updated"`
	ID        string      `xml:"id"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

func (p *Parser) parseAtom(data []byte) (*ParseResult, error) {
	var feed atomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	// Find the alternate link (typically the website home)
	var siteURL string
	for _, link := range feed.Link {
		if link.Rel == "alternate" || link.Rel == "" {
			siteURL = link.Href
			break
		}
	}

	items := make([]*FeedItem, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		pubDate := p.parseTime(entry.Published)
		if pubDate.IsZero() {
			pubDate = p.parseTime(entry.Updated)
		}

		// Find the alternate link for the entry
		var itemLink string
		for _, link := range entry.Link {
			if link.Rel == "alternate" || link.Rel == "" {
				itemLink = link.Href
				break
			}
		}

		content := entry.Content
		if content == "" {
			content = entry.Summary
		}

		items = append(items, &FeedItem{
			GUID:        entry.ID,
			Title:       cleanText(entry.Title),
			Link:        itemLink,
			Description: cleanText(entry.Summary),
			Content:     content,
			Author:      cleanText(entry.Author.Name),
			PublishedAt: pubDate,
		})
	}

	return &ParseResult{
		Title:       cleanText(feed.Title),
		Link:        siteURL,
		Description: cleanText(feed.Subtitle),
		FeedType:    FeedTypeAtom,
		Items:       items,
	}, nil
}

// parseTime parses various time formats used in RSS/Atom
func (p *Parser) parseTime(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

// cleanText removes leading/trailing whitespace and normalizes spaces
func cleanText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

// ErrUnknownFeedType is returned when the feed type cannot be determined
var ErrUnknownFeedType = &FeedTypeError{Message: "unknown feed type"}

type FeedTypeError struct {
	Message string
}

func (e *FeedTypeError) Error() string {
	return e.Message
}