package migrate

import (
	"database/sql"
	"fmt"
)

// Migrate applies schema migrations idempotently.
func Migrate(db *sql.DB) error {
	if err := ensureRSSSchema(db); err != nil {
		return err
	}
	if err := addColumn(db, "book_sources", "explore_url", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumn(db, "book_sources", "explore_rule", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumn(db, "book_sources", "explore_mode", "TEXT DEFAULT 'default'"); err != nil {
		return err
	}
	if err := addColumn(db, "book_sources", "login_url", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumn(db, "rss_feeds", "parse_rules", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS source_stats (
			source_id INTEGER PRIMARY KEY,
			last_success INTEGER DEFAULT 0,
			last_error INTEGER DEFAULT 0,
			error_count INTEGER DEFAULT 0,
			success_count INTEGER DEFAULT 0,
			updated_at INTEGER DEFAULT (strftime('%s','now'))
		);
	`)
	return err
}

func ensureRSSSchema(db *sql.DB) error {
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
	_, err := db.Exec(schema)
	return err
}

func addColumn(db *sql.DB, table, column, typedef string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, typedef))
	return err
}
