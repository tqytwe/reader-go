package shelf

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Service 书架服务
type Service struct {
	db *sql.DB
}

// NewService 创建书架服务
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Init 初始化数据库表
func (s *Service) Init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS shelf_books (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		book_key         TEXT NOT NULL UNIQUE,
		name             TEXT NOT NULL,
		author           TEXT DEFAULT '',
		cover_url        TEXT DEFAULT '',
		summary          TEXT DEFAULT '',
		source_id        INTEGER DEFAULT 0,
		source_name      TEXT DEFAULT '',
		current_chapter  TEXT DEFAULT '',
		last_read_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
		read_count       INTEGER DEFAULT 0,
		note             TEXT DEFAULT '',
		"order"          INTEGER DEFAULT 0,
		created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_shelf_books_book_key ON shelf_books(book_key);
	CREATE INDEX IF NOT EXISTS idx_shelf_books_source_id ON shelf_books(source_id);
	CREATE INDEX IF NOT EXISTS idx_shelf_books_last_read ON shelf_books(last_read_at DESC);
	`
	_, err := s.db.Exec(schema)
	return err
}

// List 获取书架所有书籍
func (s *Service) List() ([]*ShelfBook, error) {
	rows, err := s.db.Query(`
		SELECT id, book_key, name, author, cover_url, summary, source_id, source_name,
		       current_chapter, last_read_at, read_count, note, "order", created_at, updated_at
		FROM shelf_books ORDER BY "order" DESC, last_read_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list shelf books: %w", err)
	}
	defer rows.Close()

	var books []*ShelfBook
	for rows.Next() {
		var sb ShelfBook
		if err := rows.Scan(
			&sb.ID, &sb.BookKey, &sb.Name, &sb.Author, &sb.CoverURL, &sb.Summary,
			&sb.SourceID, &sb.SourceName, &sb.CurrentChapter, &sb.LastReadAt,
			&sb.ReadCount, &sb.Note, &sb.Order, &sb.CreatedAt, &sb.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan shelf book: %w", err)
		}
		// 确保 LastReadAt 不为零
		if sb.LastReadAt.IsZero() {
			sb.LastReadAt = sb.CreatedAt
		}
		books = append(books, &sb)
	}
	return books, rows.Err()
}

// GetByID 根据主键获取书架书籍
func (s *Service) GetByID(id int64) (*ShelfBook, error) {
	var sb ShelfBook
	err := s.db.QueryRow(`
		SELECT id, book_key, name, author, cover_url, summary, source_id, source_name,
		       current_chapter, last_read_at, read_count, note, "order", created_at, updated_at
		FROM shelf_books WHERE id = ?
	`, id).Scan(
		&sb.ID, &sb.BookKey, &sb.Name, &sb.Author, &sb.CoverURL, &sb.Summary,
		&sb.SourceID, &sb.SourceName, &sb.CurrentChapter, &sb.LastReadAt,
		&sb.ReadCount, &sb.Note, &sb.Order, &sb.CreatedAt, &sb.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get shelf book by id: %w", err)
	}
	if sb.LastReadAt.IsZero() {
		sb.LastReadAt = sb.CreatedAt
	}
	return &sb, nil
}

// GetByBookKey 根据bookKey获取书架书籍
func (s *Service) GetByBookKey(bookKey string) (*ShelfBook, error) {
	var sb ShelfBook
	err := s.db.QueryRow(`
		SELECT id, book_key, name, author, cover_url, summary, source_id, source_name,
		       current_chapter, last_read_at, read_count, note, "order", created_at, updated_at
		FROM shelf_books WHERE book_key = ?
	`, bookKey).Scan(
		&sb.ID, &sb.BookKey, &sb.Name, &sb.Author, &sb.CoverURL, &sb.Summary,
		&sb.SourceID, &sb.SourceName, &sb.CurrentChapter, &sb.LastReadAt,
		&sb.ReadCount, &sb.Note, &sb.Order, &sb.CreatedAt, &sb.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get shelf book by key: %w", err)
	}
	if sb.LastReadAt.IsZero() {
		sb.LastReadAt = sb.CreatedAt
	}
	return &sb, nil
}

// Add 添加到书架（如果已存在则更新）
func (s *Service) Add(sb *ShelfBook) error {
	now := time.Now().Format("2006-01-02 15:04:05")

	// 检查是否已存在
	existing, _ := s.GetByBookKey(sb.BookKey)
	if existing != nil {
		// 更新现有记录
		return s.updateInternal(existing.ID, sb, now)
	}

	// 获取最大order
	var maxOrder int
	s.db.QueryRow("SELECT COALESCE(MAX(\"order\"), 0) FROM shelf_books").Scan(&maxOrder)
	sb.Order = maxOrder + 1

	result, err := s.db.Exec(`
		INSERT INTO shelf_books (
			book_key, name, author, cover_url, summary, source_id, source_name,
			current_chapter, last_read_at, read_count, note, "order", created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sb.BookKey, sb.Name, sb.Author, sb.CoverURL, sb.Summary,
		sb.SourceID, sb.SourceName, sb.CurrentChapter, now, sb.ReadCount, sb.Note, sb.Order, now, now)
	if err != nil {
		return fmt.Errorf("add to shelf: %w", err)
	}
	id, _ := result.LastInsertId()
	sb.ID = id
	sb.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", now)
	sb.UpdatedAt = sb.CreatedAt
	sb.LastReadAt = sb.CreatedAt
	return nil
}

// Update 更新书架书籍
func (s *Service) Update(sb *ShelfBook) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	return s.updateInternal(sb.ID, sb, now)
}

func (s *Service) updateInternal(id int64, sb *ShelfBook, now string) error {
	_, err := s.db.Exec(`
		UPDATE shelf_books SET
			name = ?, author = ?, cover_url = ?, summary = ?, source_id = ?, source_name = ?,
			current_chapter = ?, last_read_at = ?, read_count = ?, note = ?, "order" = ?,
			updated_at = ?
		WHERE id = ?
	`, sb.Name, sb.Author, sb.CoverURL, sb.Summary,
		sb.SourceID, sb.SourceName, sb.CurrentChapter,
		formatTime(sb.LastReadAt), sb.ReadCount, sb.Note, sb.Order,
		now, id)
	if err != nil {
		return fmt.Errorf("update shelf book: %w", err)
	}
	sb.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", now)
	return nil
}

// Remove 从书架移除
func (s *Service) Remove(id int64) error {
	result, err := s.db.Exec("DELETE FROM shelf_books WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("remove from shelf: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("shelf book %d not found", id)
	}
	return nil
}

// RemoveByBookKey 根据bookKey从书架移除
func (s *Service) RemoveByBookKey(bookKey string) error {
	result, err := s.db.Exec("DELETE FROM shelf_books WHERE book_key = ?", bookKey)
	if err != nil {
		return fmt.Errorf("remove from shelf by key: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("shelf book with key %q not found", bookKey)
	}
	return nil
}

// Stats 获取书架统计
func (s *Service) Stats() (*ShelfStats, error) {
	stats := &ShelfStats{}

	// Total
	s.db.QueryRow("SELECT COUNT(*) FROM shelf_books").Scan(&stats.Total)

	// Reading (有阅读进度的)
	s.db.QueryRow("SELECT COUNT(*) FROM shelf_books WHERE current_chapter != '' AND current_chapter IS NOT NULL").Scan(&stats.Reading)

	// Unread (无阅读进度的)
	s.db.QueryRow("SELECT COUNT(*) FROM shelf_books WHERE current_chapter = '' OR current_chapter IS NULL").Scan(&stats.Unread)

	return stats, nil
}

// UpdateProgress 更新阅读进度
func (s *Service) UpdateProgress(bookKey string, chapter string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(`
		UPDATE shelf_books SET
			current_chapter = ?, last_read_at = ?, read_count = read_count + 1, updated_at = ?
		WHERE book_key = ?
	`, chapter, now, now, bookKey)
	if err != nil {
		return fmt.Errorf("update progress: %w", err)
	}
	return nil
}

// formatTime formats time.Time for SQLite
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
