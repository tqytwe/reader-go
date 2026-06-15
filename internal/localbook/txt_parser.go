package localbook

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// =============================================================================
// TXT Parser - Package-level function
// =============================================================================

// ParseTxt parses a TXT file from an io.Reader with encoding detection and TOC extraction.
// This is the core parsing logic used by TxtParser.Parse.
func ParseTxt(reader io.Reader, filename string) (*LocalBook, error) {
	// Read all data from reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}

	// Detect and convert encoding to UTF-8
	utf8Data, encoding, err := detectAndConvert(data)
	if err != nil {
		return nil, fmt.Errorf("encoding conversion: %w", err)
	}

	// Parse chapters from UTF-8 content
	chapters := parseChapters(utf8Data)

	// Infer book name from filename
	name := inferBookName(filename)

	book := &LocalBook{
		ID:       "", // set by service
		Name:     name,
		Author:   "", // TXT files typically don't have author metadata
		Format:   "txt",
		Chapters: chapters,
		CoverURL: "",
	}

	// If no chapters found, treat entire content as one chapter
	if len(chapters) == 0 {
		book.Chapters = []LocalChapter{
			{
				Title:   name,
				Content: string(utf8Data),
				Index:   0,
			},
		}
	}

	// Store encoding info in Content field for reference
	book.Content = string(utf8Data)

	_ = encoding // may be used for logging
	return book, nil
}

// =============================================================================
// Encoding Detection & Conversion
// =============================================================================

// detectAndConvert detects the encoding of data and converts it to UTF-8.
func detectAndConvert(data []byte) ([]byte, string, error) {
	if len(data) == 0 {
		return data, "utf-8", nil
	}

	// 1. Check BOM
	bomEncoding, bomLen := detectBOM(data)
	if bomEncoding != "" {
		converted, err := convertToUTF8(data[bomLen:], bomEncoding)
		if err != nil {
			return nil, "", err
		}
		return converted, bomEncoding, nil
	}

	// 2. Try UTF-8 first
	if isValidUTF8(data) {
		return data, "utf-8", nil
	}

	// 3. Try GBK/GB2312
	converted, err := convertToUTF8(data, "gbk")
	if err == nil && isValidUTF8(converted) {
		return converted, "gbk", nil
	}

	// 4. Try UTF-16 LE
	converted, err = convertToUTF8(data, "utf-16le")
	if err == nil && isValidUTF8(converted) {
		return converted, "utf-16le", nil
	}

	// 5. Try UTF-16 BE
	converted, err = convertToUTF8(data, "utf-16be")
	if err == nil && isValidUTF8(converted) {
		return converted, "utf-16be", nil
	}

	// Fallback: assume UTF-8
	return data, "utf-8", nil
}

// detectBOM detects Byte Order Mark and returns encoding name + BOM length.
func detectBOM(data []byte) (string, int) {
	if len(data) >= 3 {
		if data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
			return "utf-8-bom", 3
		}
	}
	if len(data) >= 2 {
		if data[0] == 0xFF && data[1] == 0xFE {
			return "utf-16le", 2
		}
		if data[0] == 0xFE && data[1] == 0xFF {
			return "utf-16be", 2
		}
	}
	return "", 0
}

// convertToUTF8 converts data from specified encoding to UTF-8.
func convertToUTF8(data []byte, fromEncoding string) ([]byte, error) {
	switch strings.ToLower(fromEncoding) {
	case "utf-8", "utf-8-bom":
		return data, nil
	case "gbk", "gb2312":
		decoder := simplifiedchinese.GBK.NewDecoder()
		return io.ReadAll(transform.NewReader(bytes.NewReader(data), decoder))
	case "utf-16le":
		decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		return io.ReadAll(transform.NewReader(bytes.NewReader(data), decoder))
	case "utf-16be":
		decoder := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
		return io.ReadAll(transform.NewReader(bytes.NewReader(data), decoder))
	default:
		return data, nil
	}
}

// isValidUTF8 checks if data is valid UTF-8.
func isValidUTF8(data []byte) bool {
	// Simple heuristic: try to decode and check for replacement characters
	decoded := string(data)
	// Check for common invalid sequences
	for i := 0; i < len(data); {
		r, size := utf8DecodeRune(data[i:])
		if r == 0xFFFD && size == 1 {
			// Replacement character indicates invalid UTF-8
			return false
		}
		i += size
	}
	// Also check that decoded string doesn't have too many replacement chars
	if strings.Contains(decoded, "�") {
		return false
	}
	return true
}

// utf8DecodeRune decodes a single UTF-8 rune (simplified version).
func utf8DecodeRune(p []byte) (rune, int) {
	if len(p) == 0 {
		return 0, 0
	}
	b := p[0]
	if b < 0x80 {
		return rune(b), 1
	}
	if b < 0xC0 {
		return 0xFFFD, 1 // invalid continuation byte
	}
	if b < 0xE0 {
		if len(p) < 2 || p[1]&0xC0 != 0x80 {
			return 0xFFFD, 1
		}
		return rune(b&0x1F)<<6 | rune(p[1]&0x3F), 2
	}
	if b < 0xF0 {
		if len(p) < 3 || p[1]&0xC0 != 0x80 || p[2]&0xC0 != 0x80 {
			return 0xFFFD, 1
		}
		return rune(b&0x0F)<<12 | rune(p[1]&0x3F)<<6 | rune(p[2]&0x3F), 3
	}
	if b < 0xF8 {
		if len(p) < 4 || p[1]&0xC0 != 0x80 || p[2]&0xC0 != 0x80 || p[3]&0xC0 != 0x80 {
			return 0xFFFD, 1
		}
		return rune(b&0x07)<<18 | rune(p[1]&0x3F)<<12 | rune(p[2]&0x3F)<<6 | rune(p[3]&0x3F), 4
	}
	return 0xFFFD, 1
}

// =============================================================================
// TOC Extraction
// =============================================================================

// tocRule defines a regex pattern for detecting chapter titles.
type tocRule struct {
	pattern  *regexp.Regexp
	name     string
	priority int
}

// defaultTocRules are common chapter title patterns (Chinese novels, etc.).
var defaultTocRules = []tocRule{
	{regexp.MustCompile(`^[\s]*第[一二三四五六七八九十百千万零\d]+[章节卷篇部集][\s]*[：:．.\s]*[\s]*(.*)$`), "chapter_cn", 100},
	{regexp.MustCompile(`^[\s]*第[一二三四五六七八九十百千万零\d]+[回][\s]*[：:．.\s]*[\s]*(.*)$`), "chapter_hui", 95},
	{regexp.MustCompile(`^[\s]*Chapter[\s]+\d+[\s]*[：:．.\s]*[\s]*(.*)$`), "chapter_en", 90},
	{regexp.MustCompile(`^[\s]*CHAPTER[\s]+\d+[\s]*[：:．.\s]*[\s]*(.*)$`), "chapter_en_upper", 90},
	{regexp.MustCompile(`^[\s]*\d+[\s]*[.．、][\s]*(.*)$`), "number_dot", 70},
	{regexp.MustCompile(`^[\s]*[（(]\d+[)）][\s]*[：:．.\s]*[\s]*(.*)$`), "bracket_number", 65},
	{regexp.MustCompile(`^[\s]*[\[【]\d+[\]】][\s]*[：:．.\s]*[\s]*(.*)$`), "square_bracket", 60},
	{regexp.MustCompile(`^[\s]*[序序章引言前言后记尾声附录][\s]*[：:．.\s]*[\s]*(.*)$`), "special", 50},
	{regexp.MustCompile(`^[\s]*[卷篇部集][\s]*[一二三四五六七八九十百千万零\d]+[\s]*[：:．.\s]*[\s]*(.*)$`), "volume", 85},
}

// parseChapters extracts chapters from UTF-8 text content.
func parseChapters(data []byte) []LocalChapter {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Increase max token size for long lines
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	var chapters []LocalChapter
	var currentContent bytes.Buffer
	var currentTitle string
	var chapterIndex int

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines at the beginning
		if trimmed == "" && currentTitle == "" {
			continue
		}

		// Try to match chapter title
		title, isChapter := matchChapterTitle(trimmed)
		if isChapter && title != "" {
			// Save previous chapter
			if currentTitle != "" {
				chapters = append(chapters, LocalChapter{
					Title:   currentTitle,
					Content: strings.TrimSpace(currentContent.String()),
					Index:   chapterIndex,
				})
			}
			currentTitle = title
			currentContent.Reset()
			chapterIndex++
			continue
		}

		// Accumulate content
		if currentTitle != "" {
			if currentContent.Len() > 0 {
				currentContent.WriteString("\n")
			}
			currentContent.WriteString(line)
		}
	}

	// Save last chapter
	if currentTitle != "" {
		chapters = append(chapters, LocalChapter{
			Title:   currentTitle,
			Content: strings.TrimSpace(currentContent.String()),
			Index:   chapterIndex,
		})
	}

	// Renumber chapters starting from 0
	for i := range chapters {
		chapters[i].Index = i
	}

	return chapters
}

// matchChapterTitle checks if a line matches any chapter title pattern.
func matchChapterTitle(line string) (string, bool) {
	if len(line) > 100 || len(line) < 2 {
		return "", false
	}

	for _, rule := range defaultTocRules {
		matches := rule.pattern.FindStringSubmatch(line)
		if matches != nil {
			// Extract title from capture group if available
			title := line
			if len(matches) > 1 && matches[1] != "" {
				title = strings.TrimSpace(matches[1])
				if title == "" {
					title = line
				}
			}
			// Filter out suspicious titles
			if isSuspiciousTitle(title) {
				continue
			}
			return title, true
		}
	}
	return "", false
}

// isSuspiciousTitle filters out lines that look like TOC headers, ads, etc.
func isSuspiciousTitle(title string) bool {
	lower := strings.ToLower(title)
	suspicious := []string{
		"目录", "contents", "content", "index",
		"版权声明", "版权所有", "copyright",
		"简介", "介绍", "summary",
		"作者", "author", "editor",
		"出版", "publisher",
		"推荐", "recommend",
	}
	for _, s := range suspicious {
		if strings.Contains(lower, s) {
			return true
		}
	}
	// Too short or too long
	if len(title) < 2 || len(title) > 80 {
		return true
	}
	return false
}

// =============================================================================
// Helpers
// =============================================================================

// inferBookName extracts a book name from the filename.
func inferBookName(filename string) string {
	base := filepath.Base(filename)
	ext := strings.ToLower(filepath.Ext(base))
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	// Clean common prefixes/suffixes
	base = strings.TrimSpace(base)
	// Remove common tags
	base = strings.ReplaceAll(base, "【全本】", "")
	base = strings.ReplaceAll(base, "【完结】", "")
	base = strings.ReplaceAll(base, "(完结)", "")
	base = strings.ReplaceAll(base, "（完结）", "")
	base = strings.ReplaceAll(base, "【精校】", "")
	return strings.TrimSpace(base)
}
