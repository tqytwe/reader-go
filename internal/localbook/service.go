package localbook

import (
	"archive/zip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"reader-go/internal/localbook/epub"
	"reader-go/internal/localbook/txt"
)

const (
	// MaxFileSize is the maximum allowed file size for uploads (50MB)
	MaxFileSize = 50 * 1024 * 1024
)

// allowedExtensions defines the whitelist of permitted file extensions.
var allowedExtensions = map[string]bool{
	".txt":  true,
	".epub": true,
	".cbz":  true,
	".pdf":  true,
}

// =============================================================================
// Service - Local Book Storage
// =============================================================================

// StoredBook represents a book stored in the service.
type StoredBook struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Author    string         `json:"author"`
	Format    string         `json:"format"`
	FilePath  string         `json:"-"` // internal: stored file path
	FileSize  int64          `json:"fileSize"`
	Chapters  []LocalChapter `json:"chapters"`
	CoverURL  string         `json:"coverUrl,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

// Service manages local book storage.
type Service struct {
	db       *sql.DB
	mu       sync.RWMutex
	books    map[string]*StoredBook // in-memory cache
	dataDir  string                 // directory for storing uploaded files
}

// NewService creates a new local book service.
func NewService(db *sql.DB, dataDir string) *Service {
	if dataDir == "" {
		wd, _ := os.Getwd()
		dataDir = filepath.Join(wd, "data", "localbooks")
	}
	// Ensure dataDir is an absolute path to prevent path traversal
	absDir, err := filepath.Abs(dataDir)
	if err != nil {
		absDir = dataDir
	}
	return &Service{
		db:      db,
		books:   make(map[string]*StoredBook),
		dataDir: absDir,
	}
}

// Init initializes the database table and loads existing books.
func (s *Service) Init() error {
	// Create data directory
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// Create table
	schema := `
	CREATE TABLE IF NOT EXISTS local_books (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		author      TEXT DEFAULT '',
		format      TEXT NOT NULL,
		file_path   TEXT NOT NULL,
		file_size   INTEGER DEFAULT 0,
		chapter_count INTEGER DEFAULT 0,
		cover_url   TEXT DEFAULT '',
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_local_books_name ON local_books(name);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("init local_books table: %w", err)
	}

	// Load existing books into memory
	return s.loadFromDB()
}

// loadFromDB loads all books from the database into memory cache.
func (s *Service) loadFromDB() error {
	rows, err := s.db.Query(`
		SELECT id, name, author, format, file_path, file_size, cover_url, created_at
		FROM local_books ORDER BY created_at DESC
	`)
	if err != nil {
		return fmt.Errorf("load local books: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sb StoredBook
		var createdAtStr string
		if err := rows.Scan(&sb.ID, &sb.Name, &sb.Author, &sb.Format,
			&sb.FilePath, &sb.FileSize, &sb.CoverURL, &createdAtStr); err != nil {
			continue
		}
		sb.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		// Load chapters from stored file if available
		sb.Chapters = s.loadChaptersMeta(sb.ID)
		s.books[sb.ID] = &sb
	}
	return rows.Err()
}

// loadChaptersMeta loads chapter metadata for a book.
func (s *Service) loadChaptersMeta(bookID string) []LocalChapter {
	// For simplicity, we re-parse the stored file to get chapters
	// In production, chapters could be stored in a separate table
	var filePath string
	err := s.db.QueryRow("SELECT file_path FROM local_books WHERE id = ?", bookID).Scan(&filePath)
	if err != nil {
		return nil
	}

	book, err := ParseFile(filePath)
	if err != nil {
		return nil
	}
	return book.Chapters
}

// =============================================================================
// CRUD Operations
// =============================================================================

// GenerateID generates a unique ID from file content.
func GenerateID(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

// Store saves an uploaded book file, parses it, and stores metadata.
func (s *Service) Store(filename string, reader io.Reader) (*StoredBook, error) {
	// Wrap reader with LimitedReader to prevent unbounded memory allocation
	limitedReader := io.LimitReader(reader, MaxFileSize+1)

	// Read all content to compute hash and save to file
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}

	// Check if data exceeds max size
	if int64(len(data)) > MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", MaxFileSize)
	}

	// Generate ID from content
	h := sha256.New()
	h.Write(data)
	id := hex.EncodeToString(h.Sum(nil))[:16]

	// Check if already exists
	s.mu.RLock()
	if existing, ok := s.books[id]; ok {
		s.mu.RUnlock()
		return existing, nil
	}
	s.mu.RUnlock()

	// Detect format
	format := DetectFormat(filename)
	if format == "" {
		return nil, fmt.Errorf("unsupported file format: %s", filename)
	}

	// Save to file
	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExtensions[ext] {
		return nil, errors.New("unsupported file type")
	}
	storedName := id + ext
	filePath := filepath.Join(s.dataDir, storedName)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	// Parse the book
	book, err := ParseFile(filePath)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("parse book: %w", err)
	}

	// Override name if filename is more descriptive
	baseName := strings.TrimSuffix(filepath.Base(filename), ext)
	if book.Name == "" || book.Name == baseName {
		book.Name = baseName
	}
	if book.Name == "" {
		book.Name = "Untitled"
	}

	// Create stored book
	sb := &StoredBook{
		ID:        id,
		Name:      book.Name,
		Author:    book.Author,
		Format:    book.Format,
		FilePath:  filePath,
		FileSize:  int64(len(data)),
		Chapters:  book.Chapters,
		CoverURL:  book.CoverURL,
		CreatedAt: time.Now(),
	}

	// Save to DB
	_, err = s.db.Exec(`
		INSERT INTO local_books (id, name, author, format, file_path, file_size, chapter_count, cover_url, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sb.ID, sb.Name, sb.Author, sb.Format, sb.FilePath, sb.FileSize, len(sb.Chapters), sb.CoverURL,
		sb.CreatedAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("save to db: %w", err)
	}

	// Update cache
	s.mu.Lock()
	s.books[id] = sb
	s.mu.Unlock()

	return sb, nil
}

// List returns all stored books.
func (s *Service) List() []*StoredBook {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*StoredBook, 0, len(s.books))
	for _, b := range s.books {
		result = append(result, b)
	}
	return result
}

// Get returns a book by ID.
func (s *Service) Get(id string) (*StoredBook, error) {
	s.mu.RLock()
	b, ok := s.books[id]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("book not found")
	}
	return b, nil
}

// GetContent returns the content of a specific chapter.
// For TXT files, it reads the file and extracts the chapter content.
func (s *Service) GetContent(id string, chapterIndex int) (string, error) {
	b, err := s.Get(id)
	if err != nil {
		return "", err
	}

	if chapterIndex < 0 || chapterIndex >= len(b.Chapters) {
		return "", fmt.Errorf("chapter index out of range")
	}

	switch b.Format {
	case "txt":
		return s.getTxtChapterContent(b, chapterIndex)
	case "epub":
		return s.getEpubChapterContent(b, chapterIndex)
	case "cbz":
		return "", fmt.Errorf("cbz content not yet supported")
	default:
		return "", fmt.Errorf("unsupported format: %s", b.Format)
	}
}

// getTxtChapterContent extracts a chapter's content from a TXT file.
func (s *Service) getTxtChapterContent(b *StoredBook, chapterIndex int) (string, error) {
	// Re-parse the file to get positions
	tf := txt.NewTextFile(b.FilePath)
	book, err := tf.Parse()
	if err != nil {
		return "", fmt.Errorf("re-parse txt: %w", err)
	}

	if chapterIndex >= len(book.Chapters) {
		return "", fmt.Errorf("chapter index out of range")
	}

	ch := book.Chapters[chapterIndex]
	content, err := tf.GetContent(ch.StartPos, ch.EndPos)
	if err != nil {
		return "", fmt.Errorf("get content: %w", err)
	}

	return content, nil
}

// getEpubChapterContent extracts a chapter's content from an EPUB file.
func (s *Service) getEpubChapterContent(b *StoredBook, chapterIndex int) (string, error) {
	if chapterIndex < 0 || chapterIndex >= len(b.Chapters) {
		return "", fmt.Errorf("chapter index out of range")
	}

	// 打开 EPUB 文件
	book, err := epub.Parse(b.FilePath)
	if err != nil {
		return "", fmt.Errorf("parse epub: %w", err)
	}

	// 获取章节内容
	archive, err := zip.OpenReader(b.FilePath)
	if err != nil {
		return "", fmt.Errorf("open epub archive: %w", err)
	}
	defer archive.Close()

	zrc := &epub.ZipReadCloser{R: &archive.Reader, Closer: archive}
	content, err := book.GetChapterContent(chapterIndex, zrc)
	if err != nil {
		return "", fmt.Errorf("get chapter content: %w", err)
	}

	if content != nil {
		return content.PlainText, nil
	}
	return "", fmt.Errorf("no content available")
}

// Remove deletes a book by ID.
func (s *Service) Remove(id string) error {
	s.mu.Lock()
	b, ok := s.books[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("book not found")
	}
	delete(s.books, id)
	s.mu.Unlock()

	// Delete from DB
	_, err := s.db.Exec("DELETE FROM local_books WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete from db: %w", err)
	}

	// Delete file
	if b.FilePath != "" {
		os.Remove(b.FilePath)
	}

	return nil
}

// GetFullText returns the entire book content (for TXT files).
func (s *Service) GetFullText(id string) (string, error) {
	b, err := s.Get(id)
	if err != nil {
		return "", err
	}

	if b.Format != "txt" {
		return "", fmt.Errorf("full text only supported for txt format")
	}

	data, err := os.ReadFile(b.FilePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	// Detect and convert encoding
	enc, _ := txt.DetectBOM(data)
	if enc == txt.EncodingUnknown {
		detector := txt.NewEncodingDetector()
		enc, _ = detector.DetectFromBytes(data)
	}

	if enc != txt.EncodingUTF8 && enc != txt.EncodingUnknown {
		converted, err := txt.ConvertToUTF8(data, enc)
		if err != nil {
			return string(data), nil // fallback to raw
		}
		return string(converted), nil
	}

	return string(data), nil
}
