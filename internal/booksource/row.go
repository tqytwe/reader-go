package booksource

import (
	"database/sql"
	"fmt"
)

const bookSourceSelectCols = `id, name, base_url, search_url, book_info_url, toc_url, content_url,
		       search_rule, book_info_rule, toc_rule, content_rule,
		       search_mode, book_info_mode, toc_mode, content_mode,
		       user_agent, headers, cookie, timeout, enabled, "group", "order",
		       explore_url, explore_rule, explore_mode, login_url,
		       created_at, updated_at`

func scanBookSource(scanner interface {
	Scan(dest ...interface{}) error
}) (*BookSource, error) {
	var bs BookSource
	var headers, cookie sql.NullString
	err := scanner.Scan(
		&bs.ID, &bs.Name, &bs.BaseURL, &bs.SearchURL, &bs.BookInfoURL, &bs.TocURL, &bs.ContentURL,
		&bs.SearchRule, &bs.BookInfoRule, &bs.TocRule, &bs.ContentRule,
		&bs.SearchMode, &bs.BookInfoMode, &bs.TocMode, &bs.ContentMode,
		&bs.UserAgent, &headers, &cookie, &bs.Timeout, &bs.Enabled, &bs.Group, &bs.Order,
		&bs.ExploreURL, &bs.ExploreRule, &bs.ExploreMode, &bs.LoginURL,
		&bs.CreatedAt, &bs.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if headers.Valid {
		bs.Headers = headers.String
	}
	if cookie.Valid {
		bs.Cookie = cookie.String
	}
	return &bs, nil
}

func scanBookSourceErr(scanner interface {
	Scan(dest ...interface{}) error
}, wrap string) (*BookSource, error) {
	bs, err := scanBookSource(scanner)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", wrap, err)
	}
	return bs, nil
}
