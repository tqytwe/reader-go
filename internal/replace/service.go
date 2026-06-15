package replace

import (
	"database/sql"
	"fmt"
	"regexp"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Service 替换规则服务
type Service struct {
	db *sql.DB
}

// NewService 创建替换规则服务
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Init 初始化数据库表
func (s *Service) Init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS replace_rules (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		name             TEXT NOT NULL,
		pattern          TEXT NOT NULL,
		replacement      TEXT DEFAULT '',
		scope            TEXT DEFAULT 'all',
		case_insensitive INTEGER DEFAULT 0,
		enabled          INTEGER DEFAULT 1,
		"order"          INTEGER DEFAULT 0,
		created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_replace_rules_enabled ON replace_rules(enabled);
	CREATE INDEX IF NOT EXISTS idx_replace_rules_scope ON replace_rules(scope);
	`
	_, err := s.db.Exec(schema)
	return err
}

// List 列出所有替换规则
func (s *Service) List() ([]*ReplaceRule, error) {
	rows, err := s.db.Query(`
		SELECT id, name, pattern, replacement, scope, case_insensitive, enabled, "order", created_at, updated_at
		FROM replace_rules ORDER BY "order" DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list replace rules: %w", err)
	}
	defer rows.Close()

	var rules []*ReplaceRule
	for rows.Next() {
		var r ReplaceRule
		if err := rows.Scan(
			&r.ID, &r.Name, &r.Pattern, &r.Replacement, &r.Scope,
			&r.CaseInsensitive, &r.Enabled, &r.Order, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan replace rule: %w", err)
		}
		rules = append(rules, &r)
	}
	return rules, rows.Err()
}

// ListEnabled 列出所有启用的替换规则
func (s *Service) ListEnabled() ([]*ReplaceRule, error) {
	rules, err := s.List()
	if err != nil {
		return nil, err
	}
	var enabled []*ReplaceRule
	for _, r := range rules {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	return enabled, nil
}

// GetByID 根据ID获取替换规则
func (s *Service) GetByID(id int64) (*ReplaceRule, error) {
	var r ReplaceRule
	err := s.db.QueryRow(`
		SELECT id, name, pattern, replacement, scope, case_insensitive, enabled, "order", created_at, updated_at
		FROM replace_rules WHERE id = ?
	`, id).Scan(
		&r.ID, &r.Name, &r.Pattern, &r.Replacement, &r.Scope,
		&r.CaseInsensitive, &r.Enabled, &r.Order, &r.CreatedAt, &r.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get replace rule by id: %w", err)
	}
	return &r, nil
}

// Create 创建替换规则
func (s *Service) Create(r *ReplaceRule) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	if r.Order == 0 {
		var maxOrder int
		s.db.QueryRow("SELECT COALESCE(MAX(\"order\"), 0) FROM replace_rules").Scan(&maxOrder)
		r.Order = maxOrder + 1
	}
	if r.Scope == "" {
		r.Scope = ScopeAll
	}

	result, err := s.db.Exec(`
		INSERT INTO replace_rules (
			name, pattern, replacement, scope, case_insensitive, enabled, "order",
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.Name, r.Pattern, r.Replacement, r.Scope, boolToInt(r.CaseInsensitive),
		boolToInt(r.Enabled), r.Order, now, now)
	if err != nil {
		return fmt.Errorf("create replace rule: %w", err)
	}
	id, _ := result.LastInsertId()
	r.ID = id
	r.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", now)
	r.UpdatedAt = r.CreatedAt
	return nil
}

// Update 更新替换规则
func (s *Service) Update(r *ReplaceRule) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(`
		UPDATE replace_rules SET
			name = ?, pattern = ?, replacement = ?, scope = ?, case_insensitive = ?,
			enabled = ?, "order" = ?, updated_at = ?
		WHERE id = ?
	`, r.Name, r.Pattern, r.Replacement, r.Scope, boolToInt(r.CaseInsensitive),
		boolToInt(r.Enabled), r.Order, now, r.ID)
	if err != nil {
		return fmt.Errorf("update replace rule: %w", err)
	}
	r.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", now)
	return nil
}

// Delete 删除替换规则
func (s *Service) Delete(id int64) error {
	result, err := s.db.Exec("DELETE FROM replace_rules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete replace rule: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("replace rule %d not found", id)
	}
	return nil
}

// Apply 应用替换规则到文本
func ApplyRule(r *ReplaceRule, text string) string {
	if !r.Enabled || r.Pattern == "" {
		return text
	}

	opts := ""
	if r.CaseInsensitive {
		opts = "(?i)"
	}

	regex, err := regexp.Compile(opts + r.Pattern)
	if err != nil {
		// 编译失败则退化为普通字符串替换
		return regexp.QuoteMeta(r.Pattern)
	}

	return regex.ReplaceAllString(text, r.Replacement)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
