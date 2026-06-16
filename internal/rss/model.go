package rss

import "time"

// FeedType represents the type of RSS feed
type FeedType int // 0=unknown, 1=RSS2.0, 2=Atom, 3=HTML

const (
	FeedTypeUnknown FeedType = 0
	FeedTypeRSS2    FeedType = 1
	FeedTypeAtom    FeedType = 2
	FeedTypeHTML    FeedType = 3 // HTML页面自动解析
)

// Feed represents an RSS subscription source
type Feed struct {
	ID          int64     `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Link        string    `json:"link" db:"link"`
	Description string    `json:"description" db:"description"`
	FeedURL     string    `json:"feedUrl" db:"feed_url"` // Subscription source URL
	SiteURL     string    `json:"siteUrl" db:"site_url"` // Website URL
	IconURL     string    `json:"iconUrl" db:"icon_url"`
	FeedType    FeedType  `json:"feedType" db:"feed_type"`
	Group       string    `json:"group" db:"rss_group"`
	Enabled     bool      `json:"enabled" db:"enabled"`
	LastFetch   time.Time `json:"lastFetch" db:"last_fetch"`
	ParseRules  string    `json:"parseRules" db:"parse_rules"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// FeedItem represents a single article/item from an RSS feed
type FeedItem struct {
	ID          int64     `json:"id" db:"id"`
	FeedID      int64     `json:"feedId" db:"feed_id"`
	GUID        string    `json:"guid" db:"guid"` // Unique identifier
	Title       string    `json:"title" db:"title"`
	Link        string    `json:"link" db:"link"`
	Description string    `json:"description" db:"description"`
	Content     string    `json:"content" db:"content"`
	Author      string    `json:"author" db:"author"`
	PublishedAt time.Time `json:"publishedAt" db:"published_at"`
	IsRead      bool      `json:"isRead" db:"is_read"`
	IsStarred   bool      `json:"isStarred" db:"is_starred"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

// Subscription represents a user's subscription to a feed
type Subscription struct {
	ID        int64     `json:"id" db:"id"`
	FeedID    int64     `json:"feedId" db:"feed_id"`
	UserID    int64     `json:"userId" db:"user_id"` // Personal use, fixed to 1
	Group     string    `json:"group" db:"rss_group"`
	Order     int       `json:"order" db:"rss_order"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}