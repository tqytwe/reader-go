package booksource

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Service 书源服务
type Service struct {
	db *sql.DB
}

// NewService 创建书源服务
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// DB 返回底层数据库连接（统计/迁移用）
func (s *Service) DB() *sql.DB {
	return s.db
}

// Init 初始化数据库表
func (s *Service) Init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS book_sources (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		name          TEXT NOT NULL,
		base_url      TEXT,
		search_url    TEXT,
		book_info_url TEXT,
		toc_url       TEXT,
		content_url   TEXT,
		search_rule   TEXT,
		book_info_rule TEXT,
		toc_rule      TEXT,
		content_rule  TEXT,
		search_mode   TEXT DEFAULT 'default',
		book_info_mode TEXT DEFAULT 'default',
		toc_mode      TEXT DEFAULT 'default',
		content_mode  TEXT DEFAULT 'default',
		user_agent    TEXT,
		headers       TEXT,
		cookie        TEXT,
		timeout       INTEGER DEFAULT 15,
		enabled       INTEGER DEFAULT 1,
		"group"       TEXT DEFAULT '',
		"order"       INTEGER DEFAULT 0,
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_book_sources_enabled ON book_sources(enabled);
	CREATE INDEX IF NOT EXISTS idx_book_sources_group ON book_sources("group");
	CREATE INDEX IF NOT EXISTS idx_book_sources_name ON book_sources(name);
	`
	_, err := s.db.Exec(schema)
	return err
}

// List 列出所有书源
func (s *Service) List() ([]*BookSource, error) {
	rows, err := s.db.Query(`
		SELECT ` + bookSourceSelectCols + `
		FROM book_sources ORDER BY "order" DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list book sources: %w", err)
	}
	defer rows.Close()

	var sources []*BookSource
	for rows.Next() {
		bs, err := scanBookSourceErr(rows, "scan book source")
		if err != nil {
			return nil, err
		}
		sources = append(sources, bs)
	}
	return sources, rows.Err()
}

// GetByID 根据ID获取书源
func (s *Service) GetByID(id int64) (*BookSource, error) {
	row := s.db.QueryRow(`
		SELECT `+bookSourceSelectCols+`
		FROM book_sources WHERE id = ?
	`, id)
	bs, err := scanBookSource(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get book source by id: %w", err)
	}
	return bs, nil
}

// Create 创建书源
func (s *Service) Create(bs *BookSource) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	if bs.Timeout == 0 {
		bs.Timeout = 15
	}
	if bs.Order == 0 {
		var maxOrder int
		s.db.QueryRow("SELECT COALESCE(MAX(\"order\"), 0) FROM book_sources").Scan(&maxOrder)
		bs.Order = maxOrder + 1
	}

	result, err := s.db.Exec(`
		INSERT INTO book_sources (
			name, base_url, search_url, book_info_url, toc_url, content_url,
			search_rule, book_info_rule, toc_rule, content_rule,
			search_mode, book_info_mode, toc_mode, content_mode,
			user_agent, headers, cookie, timeout, enabled, "group", "order",
			explore_url, explore_rule, explore_mode, login_url,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, bs.Name, bs.BaseURL, bs.SearchURL, bs.BookInfoURL, bs.TocURL, bs.ContentURL,
		bs.SearchRule, bs.BookInfoRule, bs.TocRule, bs.ContentRule,
		bs.SearchMode, bs.BookInfoMode, bs.TocMode, bs.ContentMode,
		bs.UserAgent, bs.Headers, bs.Cookie, bs.Timeout, boolToInt(bs.Enabled), bs.Group, bs.Order,
		bs.ExploreURL, bs.ExploreRule, bs.ExploreMode, bs.LoginURL,
		now, now)
	if err != nil {
		return fmt.Errorf("create book source: %w", err)
	}
	id, _ := result.LastInsertId()
	bs.ID = id
	bs.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", now)
	bs.UpdatedAt = bs.CreatedAt
	return nil
}

// Update 更新书源
func (s *Service) Update(bs *BookSource) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(`
		UPDATE book_sources SET
			name = ?, base_url = ?, search_url = ?, book_info_url = ?, toc_url = ?, content_url = ?,
			search_rule = ?, book_info_rule = ?, toc_rule = ?, content_rule = ?,
			search_mode = ?, book_info_mode = ?, toc_mode = ?, content_mode = ?,
			user_agent = ?, headers = ?, cookie = ?, timeout = ?, enabled = ?, "group" = ?, "order" = ?,
			explore_url = ?, explore_rule = ?, explore_mode = ?, login_url = ?,
			updated_at = ?
		WHERE id = ?
	`, bs.Name, bs.BaseURL, bs.SearchURL, bs.BookInfoURL, bs.TocURL, bs.ContentURL,
		bs.SearchRule, bs.BookInfoRule, bs.TocRule, bs.ContentRule,
		bs.SearchMode, bs.BookInfoMode, bs.TocMode, bs.ContentMode,
		bs.UserAgent, bs.Headers, bs.Cookie, bs.Timeout, boolToInt(bs.Enabled), bs.Group, bs.Order,
		bs.ExploreURL, bs.ExploreRule, bs.ExploreMode, bs.LoginURL,
		now, bs.ID)
	if err != nil {
		return fmt.Errorf("update book source: %w", err)
	}
	bs.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", now)
	return nil
}

// Delete 删除书源
func (s *Service) Delete(id int64) error {
	result, err := s.db.Exec("DELETE FROM book_sources WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete book source: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("book source %d not found", id)
	}
	return nil
}

// DeleteBatch 批量删除书源
func (s *Service) DeleteBatch(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	// 构建 IN 子句占位符
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf("DELETE FROM book_sources WHERE id IN (%s)", strings.Join(placeholders, ","))
	result, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("batch delete book sources: %w", err)
	}
	return result.RowsAffected()
}

// Import 批量导入书源（兼容旧签名）
func (s *Service) Import(sources []*BookSource) (int, error) {
	r := s.ImportWithResult(sources)
	return r.Success, nil
}

// ImportWithResult 批量导入并返回错误明细
func (s *Service) ImportWithResult(sources []*BookSource) ImportResult {
	result := ImportResult{Total: len(sources), Errors: []string{}}
	if len(sources) == 0 {
		return result
	}

	tx, err := s.db.Begin()
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Failed = result.Total
		return result
	}
	defer tx.Rollback()

	now := time.Now().Format("2006-01-02 15:04:05")
	stmt, err := tx.Prepare(`
		INSERT INTO book_sources (
			name, base_url, search_url, book_info_url, toc_url, content_url,
			search_rule, book_info_rule, toc_rule, content_rule,
			search_mode, book_info_mode, toc_mode, content_mode,
			user_agent, headers, cookie, timeout, enabled, "group", "order",
			explore_url, explore_rule, explore_mode, login_url,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Failed = result.Total
		return result
	}
	defer stmt.Close()

	var maxOrder int
	tx.QueryRow("SELECT COALESCE(MAX(\"order\"), 0) FROM book_sources").Scan(&maxOrder)

	for _, bs := range sources {
		if bs.Name == "" {
			result.Failed++
			result.Errors = append(result.Errors, "empty book source name")
			continue
		}
		if bs.Timeout == 0 {
			bs.Timeout = 15
		}
		maxOrder++
		bs.Order = maxOrder

		_, err := stmt.Exec(
			bs.Name, bs.BaseURL, bs.SearchURL, bs.BookInfoURL, bs.TocURL, bs.ContentURL,
			bs.SearchRule, bs.BookInfoRule, bs.TocRule, bs.ContentRule,
			bs.SearchMode, bs.BookInfoMode, bs.TocMode, bs.ContentMode,
			bs.UserAgent, bs.Headers, bs.Cookie, bs.Timeout, boolToInt(bs.Enabled), bs.Group, bs.Order,
			bs.ExploreURL, bs.ExploreRule, bs.ExploreMode, bs.LoginURL,
			now, now,
		)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", bs.Name, err))
			continue
		}
		result.Success++
	}

	if err := tx.Commit(); err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	return result
}

// RecordSourceStat 记录书源调用结果（T-101）
func (s *Service) RecordSourceStat(sourceID int64, success bool) {
	now := time.Now().Unix()
	if success {
		_, _ = s.db.Exec(`
			INSERT INTO source_stats (source_id, last_success, success_count, updated_at)
			VALUES (?, ?, 1, ?)
			ON CONFLICT(source_id) DO UPDATE SET
				last_success=excluded.last_success,
				success_count=success_count+1,
				updated_at=excluded.updated_at
		`, sourceID, now, now)
	} else {
		_, _ = s.db.Exec(`
			INSERT INTO source_stats (source_id, last_error, error_count, updated_at)
			VALUES (?, ?, 1, ?)
			ON CONFLICT(source_id) DO UPDATE SET
				last_error=excluded.last_error,
				error_count=error_count+1,
				updated_at=excluded.updated_at
		`, sourceID, now, now)
	}
}

// DeleteAll 删除所有书源（合集导入全量替换时使用）
func (s *Service) DeleteAll() error {
	_, err := s.db.Exec("DELETE FROM book_sources")
	return err
}

// BatchSetEnabledByTarget 按 JS 依赖批量设置启用状态
// target: "js" | "nonJs" | "all"
func (s *Service) BatchSetEnabledByTarget(target string, enabled bool) (updated int, stats EnablePolicyResult, err error) {
	sources, err := s.List()
	if err != nil {
		return 0, stats, err
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, bs := range sources {
		js := RequiresJSSearch(bs)
		match := target == "all" ||
			(target == "js" && js) ||
			(target == "nonJs" && !js)
		if !match || bs.Enabled == enabled {
			continue
		}
		if _, err := s.db.Exec(
			`UPDATE book_sources SET enabled = ?, updated_at = ? WHERE id = ?`,
			boolToInt(enabled), now, bs.ID,
		); err != nil {
			return updated, stats, fmt.Errorf("batch update %s: %w", bs.Name, err)
		}
		bs.Enabled = enabled
		updated++
	}
	for _, bs := range sources {
		if RequiresJSSearch(bs) {
			stats.JSRequired++
		} else {
			stats.NonJS++
		}
		if bs.Enabled {
			stats.Enabled++
		} else {
			stats.Disabled++
		}
	}
	return updated, stats, nil
}

// ApplyEnablePolicyExisting 对已有书源重新应用「仅启用无 JS」策略
func (s *Service) ApplyEnablePolicyExisting(enableOnlyNonJS bool) (updated int, stats EnablePolicyResult, err error) {
	sources, err := s.List()
	if err != nil {
		return 0, stats, err
	}
	stats = ApplyEnablePolicy(sources, enableOnlyNonJS)
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, bs := range sources {
		if _, err := s.db.Exec(
			`UPDATE book_sources SET enabled = ?, updated_at = ? WHERE id = ?`,
			boolToInt(bs.Enabled), now, bs.ID,
		); err != nil {
			return updated, stats, fmt.Errorf("apply policy %s: %w", bs.Name, err)
		}
		updated++
	}
	return updated, stats, nil
}

// ListEnabled 列出所有启用的书源
func (s *Service) ListEnabled() ([]*BookSource, error) {
	rows, err := s.db.Query(`
		SELECT ` + bookSourceSelectCols + `
		FROM book_sources WHERE enabled = 1 ORDER BY "order" DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list enabled book sources: %w", err)
	}
	defer rows.Close()

	var sources []*BookSource
	for rows.Next() {
		bs, err := scanBookSourceErr(rows, "scan book source")
		if err != nil {
			return nil, err
		}
		sources = append(sources, bs)
	}
	return sources, rows.Err()
}

// ParseHeaders 解析Headers JSON字符串
func ParseHeaders(s string) (map[string]string, error) {
	if s == "" {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// boolToInt converts bool to int (1/0) for SQLite
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
