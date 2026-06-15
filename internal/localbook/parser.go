// Package localbook provides a unified interface for parsing local book files.
package localbook

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"reader-go/internal/localbook/cbz"
	"reader-go/internal/localbook/epub"
	"reader-go/internal/localbook/pdf"
)

// =============================================================================
// Unified Parser Interface
// =============================================================================

// LocalBookParser defines the interface for all local book parsers.
type LocalBookParser interface {
	Parse(reader io.Reader, filename string) (*LocalBook, error)
	SupportedExtensions() []string
}

// LocalBook represents a parsed local book.
type LocalBook struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Author   string         `json:"author"`
	Format   string         `json:"format"`   // "txt", "epub", "cbz"
	Content  string         `json:"content,omitempty"` // for txt: full text or chunked
	Chapters []LocalChapter `json:"chapters"`
	CoverURL string         `json:"coverUrl,omitempty"`
}

// LocalChapter represents a chapter in a local book.
type LocalChapter struct {
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
	Index   int    `json:"index"`
}

// =============================================================================
// Parser Registry
// =============================================================================

var parsers = make(map[string]LocalBookParser)

// RegisterParser registers a parser for its supported extensions.
func RegisterParser(p LocalBookParser) {
	for _, ext := range p.SupportedExtensions() {
		parsers[strings.ToLower(ext)] = p
	}
}

// GetParser returns the parser for the given file extension.
func GetParser(ext string) (LocalBookParser, bool) {
	p, ok := parsers[strings.ToLower(ext)]
	return p, ok
}

// SupportedFormats returns all supported format extensions.
func SupportedFormats() []string {
	var formats []string
	seen := make(map[string]bool)
	for _, p := range parsers {
		for _, ext := range p.SupportedExtensions() {
			if !seen[ext] {
				seen[ext] = true
				formats = append(formats, ext)
			}
		}
	}
	return formats
}

// DetectFormat detects the book format from a filename.
func DetectFormat(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return ""
	}
	if _, ok := GetParser(ext); ok {
		return strings.TrimPrefix(ext, ".")
	}
	return ""
}

// =============================================================================
// ParseFile - convenience function
// =============================================================================

// ParseFile parses a local book file at the given path.
// It auto-detects the format from the file extension.
func ParseFile(filePath string) (*LocalBook, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	parser, ok := GetParser(ext)
	if !ok {
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	return parser.Parse(f, filePath)
}

// =============================================================================
// TXT Parser Implementation
// =============================================================================

// TxtParser parses TXT files into LocalBook.
type TxtParser struct{}

// NewTxtParser creates a new TXT parser.
func NewTxtParser() *TxtParser {
	return &TxtParser{}
}

// SupportedExtensions returns the supported file extensions.
func (p *TxtParser) SupportedExtensions() []string {
	return []string{".txt"}
}

// Parse parses a TXT file from an io.Reader with encoding detection and TOC extraction.
func (p *TxtParser) Parse(reader io.Reader, filename string) (*LocalBook, error) {
	return ParseTxt(reader, filename)
}

// =============================================================================
// EPUB Parser Implementation (stub)
// =============================================================================

// EpubParser parses EPUB files into LocalBook.
type EpubParser struct{}

// NewEpubParser creates a new EPUB parser.
func NewEpubParser() *EpubParser {
	return &EpubParser{}
}

// SupportedExtensions returns the supported file extensions.
func (p *EpubParser) SupportedExtensions() []string {
	return []string{".epub"}
}

// Parse parses an EPUB file from an io.Reader.
func (p *EpubParser) Parse(reader io.Reader, filename string) (*LocalBook, error) {
	// Write to temp file because the epub package needs a file path
	tmpFile, err := os.CreateTemp("", "localbook-*.epub")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	// Parse using the existing epub package
	book, err := epub.Parse(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("parse epub: %w", err)
	}

	// Convert to LocalBook
	lb := &LocalBook{
		Name:     book.Title,
		Author:   book.Author,
		Format:   "epub",
		Chapters: make([]LocalChapter, 0, len(book.Chapters)),
	}

	for i, ch := range book.Chapters {
		lb.Chapters = append(lb.Chapters, LocalChapter{
			Title: ch.Title,
			Index: i,
		})
	}

	return lb, nil
}

// =============================================================================
// CBZ Parser Implementation (stub)
// =============================================================================

// CbzParser parses CBZ files into LocalBook.
type CbzParser struct{}

// NewCbzParser creates a new CBZ parser.
func NewCbzParser() *CbzParser {
	return &CbzParser{}
}

// SupportedExtensions returns the supported file extensions.
func (p *CbzParser) SupportedExtensions() []string {
	return []string{".cbz"}
}

// Parse parses a CBZ file from an io.Reader.
func (p *CbzParser) Parse(reader io.Reader, filename string) (*LocalBook, error) {
	// Write to temp file because the cbz package needs a file path
	tmpFile, err := os.CreateTemp("", "localbook-*.cbz")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	// Parse using the existing cbz package
	book, err := cbz.Parse(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("parse cbz: %w", err)
	}

	// Convert to LocalBook
	lb := &LocalBook{
		Name:     book.Name,
		Author:   book.Author,
		Format:   "cbz",
		Chapters: make([]LocalChapter, 0),
	}

	return lb, nil
}

// =============================================================================
// PDF Parser Implementation
// =============================================================================

// PdfParser parses PDF files into LocalBook.
type PdfParser struct{}

// NewPdfParser creates a new PDF parser.
func NewPdfParser() *PdfParser {
	return &PdfParser{}
}

// SupportedExtensions returns the supported file extensions.
func (p *PdfParser) SupportedExtensions() []string {
	return []string{".pdf"}
}

// Parse parses a PDF file from an io.Reader.
func (p *PdfParser) Parse(reader io.Reader, filename string) (*LocalBook, error) {
	// Write to temp file because the pdf package needs a file path
	tmpFile, err := os.CreateTemp("", "localbook-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	// Parse using the pdf package
	pdfParser := pdf.NewParser()
	metadata, err := pdfParser.GetMetadata(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("get pdf metadata: %w", err)
	}

	title := metadata["title"]
	if title == "" {
		title = filepath.Base(filename)
	}

	lb := &LocalBook{
		Name:     title,
		Format:   "pdf",
		Chapters: make([]LocalChapter, 0),
	}

	// Add metadata to book info if available
	if author, ok := metadata["author"]; ok {
		lb.Author = author
	}

	return lb, nil
}

// =============================================================================
// Init - register all parsers
// =============================================================================

func init() {
	RegisterParser(NewTxtParser())
	RegisterParser(NewEpubParser())
	RegisterParser(NewCbzParser())
	RegisterParser(NewPdfParser())
}
