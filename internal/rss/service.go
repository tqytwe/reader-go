package rss

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Service provides RSS feed management
type Service struct {
	db     *sql.DB
	parser *Parser
}

// NewService creates a new RSS service
func NewService(db *sql.DB) *Service {
	return &Service{
		db:     db,
		parser: NewParser(),
	}
}

// Init initializes the RSS database tables
func (s *Service) Init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS rss_feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		link TEXT,
		description TEXT,
		feed_url TEXT UNIQUE NOT NULL,
		site_url TEXT,
		icon_url TEXT,
		feed_type INTEGER DEFAULT 0,
		rss_group TEXT DEFAULT '',
		enabled INTEGER DEFAULT 1,
		last_fetch INTEGER DEFAULT 0,
		parse_rules TEXT DEFAULT '',
		created_at INTEGER DEFAULT (strftime('%s','now')),
		updated_at INTEGER DEFAULT (strftime('%s','now'))
	);

	CREATE TABLE IF NOT EXISTS rss_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		feed_id INTEGER NOT NULL,
		guid TEXT NOT NULL,
		title TEXT NOT NULL,
		link TEXT,
		description TEXT,
		content TEXT,
		author TEXT,
		published_at INTEGER DEFAULT 0,
		is_read INTEGER DEFAULT 0,
		is_starred INTEGER DEFAULT 0,
		created_at INTEGER DEFAULT (strftime('%s','now')),
		FOREIGN KEY (feed_id) REFERENCES rss_feeds(id) ON DELETE CASCADE,
		UNIQUE(feed_id, guid)
	);

	CREATE INDEX IF NOT EXISTS idx_rss_items_feed_id ON rss_items(feed_id);
	CREATE INDEX IF NOT EXISTS idx_rss_items_published ON rss_items(published_at);
	CREATE INDEX IF NOT EXISTS idx_rss_feeds_group ON rss_feeds(rss_group);
	`
	_, err := s.db.Exec(schema)
	return err
}

// AddFeed adds a new RSS feed subscription
func (s *Service) AddFeed(feed *Feed) error {
	now := time.Now()
	nowUnix := now.Unix()

	result, err := s.db.Exec(`
		INSERT INTO rss_feeds (title, link, description, feed_url, site_url, icon_url, feed_type, rss_group, enabled, last_fetch, parse_rules, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, feed.Title, feed.Link, feed.Description, feed.FeedURL, feed.SiteURL, feed.IconURL, feed.FeedType, feed.Group, boolToInt(feed.Enabled), int64(0), feed.ParseRules, nowUnix, nowUnix)
	if err != nil {
		return fmt.Errorf("add feed: %w", err)
	}

	id, _ := result.LastInsertId()
	feed.ID = id
	feed.CreatedAt = now
	feed.UpdatedAt = now
	feed.LastFetch = time.Time{}

	return nil
}

// ListFeeds returns all RSS feeds
func (s *Service) ListFeeds() ([]*Feed, error) {
	rows, err := s.db.Query(`
		SELECT id, title, link, description, feed_url, site_url, icon_url, feed_type, rss_group, enabled, last_fetch, parse_rules, created_at, updated_at
		FROM rss_feeds ORDER BY rss_group, title
	`)
	if err != nil {
		return nil, fmt.Errorf("list feeds: %w", err)
	}
	defer rows.Close()

	var feeds []*Feed
	for rows.Next() {
		var f Feed
		var lastFetch, createdAt, updatedAt int64
		var parseRules sql.NullString
		if err := rows.Scan(&f.ID, &f.Title, &f.Link, &f.Description, &f.FeedURL, &f.SiteURL, &f.IconURL, &f.FeedType, &f.Group, &f.Enabled, &lastFetch, &parseRules, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan feed: %w", err)
		}
		f.LastFetch = time.Unix(lastFetch, 0)
		if parseRules.Valid {
			f.ParseRules = parseRules.String
		}
		f.CreatedAt = time.Unix(createdAt, 0)
		f.UpdatedAt = time.Unix(updatedAt, 0)
		feeds = append(feeds, &f)
	}
	return feeds, rows.Err()
}

// GetFeed returns a feed by ID
func (s *Service) GetFeed(id int64) (*Feed, error) {
	var f Feed
	var lastFetch, createdAt, updatedAt int64
	var parseRules sql.NullString
	err := s.db.QueryRow(`
		SELECT id, title, link, description, feed_url, site_url, icon_url, feed_type, rss_group, enabled, last_fetch, parse_rules, created_at, updated_at
		FROM rss_feeds WHERE id = ?
	`, id).Scan(&f.ID, &f.Title, &f.Link, &f.Description, &f.FeedURL, &f.SiteURL, &f.IconURL, &f.FeedType, &f.Group, &f.Enabled, &lastFetch, &parseRules, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get feed: %w", err)
	}
	f.LastFetch = time.Unix(lastFetch, 0)
	if parseRules.Valid {
		f.ParseRules = parseRules.String
	}
	f.CreatedAt = time.Unix(createdAt, 0)
	f.UpdatedAt = time.Unix(updatedAt, 0)
	return &f, nil
}

// DeleteFeed deletes a feed and its items
func (s *Service) DeleteFeed(id int64) error {
	result, err := s.db.Exec("DELETE FROM rss_feeds WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete feed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("feed %d not found", id)
	}
	return nil
}

// FetchFeed fetches and parses a feed, returning new items
func (s *Service) FetchFeed(feed *Feed) ([]*FeedItem, error) {
	return s.FetchFeedCtx(context.Background(), feed)
}

// PreviewFeed 仅解析订阅源，不写入 feed 或 item 数据。
func (s *Service) PreviewFeed(ctx context.Context, feed *Feed) (*ParseResult, string, error) {
	useLegado := strings.TrimSpace(feed.GetRuleArticles()) != ""
	if useLegado {
		result, err := s.parser.ParseLegadoCtx(ctx, feed)
		return result, feed.FeedURL, err
	}
	return s.parser.ParseWithDiscoveryCtx(ctx, feed.FeedURL)
}

// FetchFeedCtx fetches and parses a feed with context for timeout control
func (s *Service) FetchFeedCtx(ctx context.Context, feed *Feed) ([]*FeedItem, error) {
	var result *ParseResult
	var err error
	var resolvedURL string
	useLegado := strings.TrimSpace(feed.GetRuleArticles()) != ""

	// #region agent log
	fetchDebugLog("service.go:FetchFeed", "fetch start", "H3", "pre-fix", map[string]interface{}{
		"feedId": feed.ID, "feedURL": feed.FeedURL, "useLegado": useLegado,
	})
	// #endregion

	if useLegado {
		result, err = s.parser.ParseLegadoCtx(ctx, feed)
	} else {
		result, resolvedURL, err = s.parser.ParseWithDiscoveryCtx(ctx, feed.FeedURL)
		if err == nil && resolvedURL != "" && resolvedURL != feed.FeedURL && isHTTPURL(resolvedURL) {
			feed.FeedURL = resolvedURL
		}
	}
	if err != nil {
		// #region agent log
		fetchDebugLog("service.go:FetchFeed", "fetch failed", "H3", "pre-fix", map[string]interface{}{
			"feedId": feed.ID, "error": err.Error(), "useLegado": useLegado, "resolvedURL": resolvedURL,
		})
		// #endregion
		return nil, fmt.Errorf("parse feed: %w", err)
	}

	// Update feed metadata if not set
	if feed.Title == "" && result.Title != "" {
		feed.Title = result.Title
	}
	if feed.Link == "" && result.Link != "" {
		feed.Link = result.Link
	}
	if feed.SiteURL == "" && result.Link != "" {
		feed.SiteURL = result.Link
	}
	if feed.Description == "" && result.Description != "" {
		feed.Description = result.Description
	}
	feed.FeedType = result.FeedType

	now := time.Now()
	feed.LastFetch = now
	nowUnix := now.Unix()

	// Update feed in database
	_, err = s.db.Exec(`
		UPDATE rss_feeds SET title = ?, link = ?, description = ?, feed_url = ?, site_url = ?, feed_type = ?, last_fetch = ?, updated_at = ?
		WHERE id = ?
	`, feed.Title, feed.Link, feed.Description, feed.FeedURL, feed.SiteURL, feed.FeedType, nowUnix, nowUnix, feed.ID)
	if err != nil {
		return nil, fmt.Errorf("update feed: %w", err)
	}

	// Insert new items (skip existing based on guid)
	var newItems []*FeedItem
	for _, item := range result.Items {
		item.FeedID = feed.ID

		res, err := s.db.Exec(`
			INSERT OR IGNORE INTO rss_items (feed_id, guid, title, link, description, content, author, published_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, item.FeedID, item.GUID, item.Title, item.Link, item.Description, item.Content, item.Author, item.PublishedAt.Unix(), now.Unix())
		if err != nil {
			continue
		}

		id, _ := res.LastInsertId()
		if id > 0 {
			item.CreatedAt = now
			item.ID = id
			newItems = append(newItems, item)
		}
	}

	return newItems, nil
}

// GetItems returns items for a feed with pagination metadata.
func (s *Service) GetItems(feedID int64, page, pageSize int) ([]*FeedItem, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM rss_items WHERE feed_id = ?`, feedID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count items: %w", err)
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	rows, err := s.db.Query(`
		SELECT id, feed_id, guid, title, link, description, content, author, published_at, is_read, is_starred, created_at
		FROM rss_items
		WHERE feed_id = ?
		ORDER BY
			CASE WHEN published_at <= 0 THEN 1 ELSE 0 END,
			published_at DESC,
			id DESC
		LIMIT ? OFFSET ?
	`, feedID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get items: %w", err)
	}
	defer rows.Close()

	var items []*FeedItem
	for rows.Next() {
		var item FeedItem
		var publishedAt, createdAt int64
		if err := rows.Scan(&item.ID, &item.FeedID, &item.GUID, &item.Title, &item.Link, &item.Description, &item.Content, &item.Author, &publishedAt, &item.IsRead, &item.IsStarred, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("scan item: %w", err)
		}
		item.PublishedAt = time.Unix(publishedAt, 0)
		item.CreatedAt = time.Unix(createdAt, 0)
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// MarkAsRead marks an item as read
func (s *Service) MarkAsRead(itemID int64) error {
	_, err := s.db.Exec("UPDATE rss_items SET is_read = 1 WHERE id = ?", itemID)
	if err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}
	return nil
}

// ToggleStar toggles the starred status of an item
func (s *Service) ToggleStar(itemID int64) error {
	_, err := s.db.Exec("UPDATE rss_items SET is_starred = NOT is_starred WHERE id = ?", itemID)
	if err != nil {
		return fmt.Errorf("toggle star: %w", err)
	}
	return nil
}

// UpsertFeed inserts or updates a feed by feed_url (Legado 同 URL 覆盖)
func (s *Service) UpsertFeed(feed *Feed) error {
	var existingID int64
	err := s.db.QueryRow("SELECT id FROM rss_feeds WHERE feed_url = ?", feed.FeedURL).Scan(&existingID)
	if err == sql.ErrNoRows {
		return s.AddFeed(feed)
	}
	if err != nil {
		return fmt.Errorf("lookup feed: %w", err)
	}

	now := time.Now()
	nowUnix := now.Unix()
	_, err = s.db.Exec(`
		UPDATE rss_feeds SET
			title = ?, link = ?, description = ?, site_url = ?, icon_url = ?,
			feed_type = ?, rss_group = ?, enabled = ?, parse_rules = ?, updated_at = ?
		WHERE id = ?
	`, feed.Title, feed.Link, feed.Description, feed.SiteURL, feed.IconURL,
		feed.FeedType, feed.Group, boolToInt(feed.Enabled), feed.ParseRules, nowUnix, existingID)
	if err != nil {
		return fmt.Errorf("update feed: %w", err)
	}
	feed.ID = existingID
	feed.UpdatedAt = now
	return nil
}

// boolToInt converts bool to int (1/0) for SQLite
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
