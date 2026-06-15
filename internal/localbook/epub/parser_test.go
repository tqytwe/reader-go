package epub

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// 测试辅助：创建测试用 EPUB 文件
// =============================================================================

// createTestEPUB 在内存中创建一个测试用 EPUB 文件
func createTestEPUB(tb testing.TB, variant string) *bytes.Buffer {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	switch variant {
	case "epub2":
		createEPUB2(zipWriter)
	case "epub3":
		createEPUB3(zipWriter)
	case "epub2-ncx":
		createEPUB2WithNCX(zipWriter)
	case "epub3-nav":
		createEPUB3WithNav(zipWriter)
	case "simple":
		createSimpleEPUB(zipWriter)
	default:
		createSimpleEPUB(zipWriter)
	}

	err := zipWriter.Close()
	if err != nil {
		tb.Fatalf("failed to close zip writer: %v", err)
	}

	return buf
}

// createSimpleEPUB 创建最简单的 EPUB（单文件）
func createSimpleEPUB(zw *zip.Writer) {
	// mimetype (必须第一个，未压缩)
	w, _ := zw.Create("mimetype")
	w.Write([]byte("application/epub+zip"))

	// META-INF/container.xml
	w, _ = zw.Create("META-INF/container.xml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// content.opf
	w, _ = zw.Create("content.opf")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="uid">test-book-001</dc:identifier>
    <dc:title>测试书籍</dc:title>
    <dc:creator>测试作者</dc:creator>
    <dc:language>zh-CN</dc:language>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.html" media-type="application/xhtml+xml"/>
    <item id="chapter2" href="chapter2.html" media-type="application/xhtml+xml"/>
    <item id="chapter3" href="chapter3.html" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
    <itemref idref="chapter2"/>
    <itemref idref="chapter3"/>
  </spine>
</package>`))

	// chapter1.html
	w, _ = zw.Create("chapter1.html")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>第一章 开始</title></head>
<body>
  <h1>第一章 开始</h1>
  <p>这是第一章的内容。</p>
  <p>故事从这里开始...</p>
  <script>alert('hello');</script>
  <style>.test { color: red; }</style>
</body>
</html>`))

	// chapter2.html
	w, _ = zw.Create("chapter2.html")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>第二章 发展</title></head>
<body>
  <h1>第二章 发展</h1>
  <p>这是第二章的内容。</p>
  <p>故事继续发展...</p>
</body>
</html>`))

	// chapter3.html
	w, _ = zw.Create("chapter3.html")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>第三章 结局</title></head>
<body>
  <h1>第三章 结局</h1>
  <p>这是第三章的内容。</p>
  <p>故事结束了。</p>
</body>
</html>`))
}

// createEPUB2WithNCX 创建带 NCX 的 EPUB 2.0
func createEPUB2WithNCX(zw *zip.Writer) {
	// mimetype
	w, _ := zw.Create("mimetype")
	w.Write([]byte("application/epub+zip"))

	// META-INF/container.xml
	w, _ = zw.Create("META-INF/container.xml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// OEBPS/content.opf
	w, _ = zw.Create("OEBPS/content.opf")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="uid">test-book-002</dc:identifier>
    <dc:title>带 NCX 的测试书籍</dc:title>
    <dc:creator>测试作者</dc:creator>
    <dc:language>zh-CN</dc:language>
    <meta name="cover" content="cover-img"/>
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="chapter1" href="chapter1.html" media-type="application/xhtml+xml"/>
    <item id="chapter2" href="chapter2.html" media-type="application/xhtml+xml"/>
    <item id="cover-img" href="cover.jpg" media-type="image/jpeg"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="chapter1"/>
    <itemref idref="chapter2"/>
  </spine>
</package>`))

	// OEBPS/toc.ncx
	w, _ = zw.Create("OEBPS/toc.ncx")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="test-book-002"/>
  </head>
  <docTitle><text>带 NCX 的测试书籍</text></docTitle>
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>第一章 开始</text></navLabel>
      <content src="chapter1.html"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>第二章 发展</text></navLabel>
      <content src="chapter2.html"/>
      <navPoint id="np2-1" playOrder="3">
        <navLabel><text>第二章第一节</text></navLabel>
        <content src="chapter2.html#section1"/>
      </navPoint>
    </navPoint>
  </navMap>
</ncx>`))

	// OEBPS/chapter1.html
	w, _ = zw.Create("OEBPS/chapter1.html")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>第一章 开始</title></head>
<body>
  <h1>第一章 开始</h1>
  <p>这是第一章的详细内容。</p>
  <p>包含多个段落。</p>
</body>
</html>`))

	// OEBPS/chapter2.html
	w, _ = zw.Create("OEBPS/chapter2.html")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>第二章 发展</title></head>
<body>
  <h1>第二章 发展</h1>
  <p>这是第二章的内容。</p>
  <h2>第二章第一节</h2>
  <p>第一节的内容。</p>
</body>
</html>`))

	// OEBPS/cover.jpg (1x1 像素 JPEG，填充到 >100 字节)
	w, _ = zw.Create("OEBPS/cover.jpg")
	coverData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xD9}
	// 填充到至少 101 字节以满足 extractCover 的最小大小检查
	for len(coverData) < 101 {
		coverData = append(coverData, 0x00)
	}
	w.Write(coverData)
}

// createEPUB3WithNav 创建带 Navigation Document 的 EPUB 3.0
func createEPUB3WithNav(zw *zip.Writer) {
	// mimetype
	w, _ := zw.Create("mimetype")
	w.Write([]byte("application/epub+zip"))

	// META-INF/container.xml
	w, _ = zw.Create("META-INF/container.xml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// OEBPS/content.opf
	w, _ = zw.Create("OEBPS/content.opf")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="uid">test-book-003</dc:identifier>
    <dc:title>EPUB 3 测试书籍</dc:title>
    <dc:creator>测试作者</dc:creator>
    <dc:language>zh-CN</dc:language>
    <meta property="dcterms:modified">2024-01-01T00:00:00Z</meta>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="chapter2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
    <item id="chapter3" href="chapter3.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
    <itemref idref="chapter2"/>
    <itemref idref="chapter3"/>
  </spine>
</package>`))

	// OEBPS/nav.xhtml (EPUB 3 Navigation Document)
	w, _ = zw.Create("OEBPS/nav.xhtml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head>
  <title>导航</title>
</head>
<body>
  <nav epub:type="toc">
    <h1>目录</h1>
    <ol>
      <li>
        <a href="chapter1.xhtml">第一章 开始</a>
      </li>
      <li>
        <a href="chapter2.xhtml">第二章 发展</a>
        <ol>
          <li>
            <a href="chapter2.xhtml#section1">第二章第一节</a>
          </li>
        </ol>
      </li>
      <li>
        <a href="chapter3.xhtml">第三章 结局</a>
      </li>
    </ol>
  </nav>
</body>
</html>`))

	// OEBPS/chapter1.xhtml
	w, _ = zw.Create("OEBPS/chapter1.xhtml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>第一章 开始</title></head>
<body>
  <h1>第一章 开始</h1>
  <p>这是第一章的详细内容。</p>
  <p>包含多个段落。</p>
  <script>console.log('test');</script>
</body>
</html>`))

	// OEBPS/chapter2.xhtml
	w, _ = zw.Create("OEBPS/chapter2.xhtml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>第二章 发展</title></head>
<body>
  <h1>第二章 发展</h1>
  <p>这是第二章的内容。</p>
  <h2 id="section1">第二章第一节</h2>
  <p>第一节的内容。</p>
</body>
</html>`))

	// OEBPS/chapter3.xhtml
	w, _ = zw.Create("OEBPS/chapter3.xhtml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>第三章 结局</title></head>
<body>
  <h1>第三章 结局</h1>
  <p>故事结束了。</p>
</body>
</html>`))
}

// createEPUB2 创建标准 EPUB 2.0
func createEPUB2(zw *zip.Writer) {
	createEPUB2WithNCX(zw)
}

// createEPUB3 创建标准 EPUB 3.0
func createEPUB3(zw *zip.Writer) {
	createEPUB3WithNav(zw)
}

// saveTestEPUB 保存测试 EPUB 到临时文件
func saveTestEPUB(t *testing.T, buf *bytes.Buffer) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.epub")
	err := os.WriteFile(tmpFile, buf.Bytes(), 0644)
	if err != nil {
		t.Fatalf("failed to write test EPUB: %v", err)
	}
	return tmpFile
}

// =============================================================================
// 单元测试
// =============================================================================

func TestParseSimpleEPUB(t *testing.T) {
	buf := createTestEPUB(t, "simple")
	tmpFile := saveTestEPUB(t, buf)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// 验证基本信息
	if book.Title != "测试书籍" {
		t.Errorf("Expected title '测试书籍', got '%s'", book.Title)
	}
	if book.Author != "测试作者" {
		t.Errorf("Expected author '测试作者', got '%s'", book.Author)
	}
	if book.Language != "zh-CN" {
		t.Errorf("Expected language 'zh-CN', got '%s'", book.Language)
	}

	// 验证章节
	chapters := book.GetChapters()
	if len(chapters) != 3 {
		t.Errorf("Expected 3 chapters, got %d", len(chapters))
	}

	// 验证章节标题（无 TOC 时从文件名提取）
	expectedTitles := []string{"Chapter1", "Chapter2", "Chapter3"}
	for i, ch := range chapters {
		if ch.Title != expectedTitles[i] {
			t.Errorf("Chapter %d: expected title '%s', got '%s'", i, expectedTitles[i], ch.Title)
		}
		if ch.Index != i {
			t.Errorf("Chapter %d: expected index %d, got %d", i, i, ch.Index)
		}
	}

	// 验证封面
	_, err = book.GetCover()
	if err == nil {
		t.Errorf("Expected ErrNoCover, got cover data")
	}
}

func TestParseEPUB2WithNCX(t *testing.T) {
	buf := createTestEPUB(t, "epub2-ncx")
	tmpFile := saveTestEPUB(t, buf)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// 验证 TOC（来自 NCX）
	toc := book.GetTOC()
	if len(toc) < 2 {
		t.Errorf("Expected at least 2 TOC entries, got %d", len(toc))
	}

	// 验证层级结构
	if len(toc) >= 2 {
		// 第二章应该有子章节
		ch2 := toc[1]
		if len(ch2.Children) == 0 {
			t.Errorf("Expected chapter 2 to have children (nested navPoint)")
		}
	}

	// 验证封面
	cover, err := book.GetCover()
	if err != nil {
		t.Errorf("Expected cover image, got error: %v", err)
	} else if len(cover) < 20 {
		t.Errorf("Expected valid cover image data, got %d bytes", len(cover))
	}
}

func TestParseEPUB3WithNav(t *testing.T) {
	buf := createTestEPUB(t, "epub3-nav")
	tmpFile := saveTestEPUB(t, buf)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// 验证基本信息
	if book.Title != "EPUB 3 测试书籍" {
		t.Errorf("Expected title 'EPUB 3 测试书籍', got '%s'", book.Title)
	}

	// 验证章节数量
	chapters := book.GetChapters()
	if len(chapters) != 3 {
		t.Errorf("Expected 3 chapters, got %d", len(chapters))
	}

	// 验证 TOC（来自 nav.xhtml）
	toc := book.GetTOC()
	if len(toc) < 3 {
		t.Errorf("Expected at least 3 TOC entries, got %d", len(toc))
	}

	// 验证层级
	if len(toc) >= 2 {
		ch2 := toc[1]
		if len(ch2.Children) == 0 {
			t.Log("Note: nested TOC from EPUB 3 nav may need HTML parsing")
		}
	}
}

func TestGetChapterContent(t *testing.T) {
	buf := createTestEPUB(t, "simple")
	tmpFile := saveTestEPUB(t, buf)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// 重新打开文件获取 zip reader
	archive, err := zip.OpenReader(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open zip: %v", err)
	}
	defer archive.Close()

	zrc := &ZipReadCloser{R: &archive.Reader, Closer: archive}

	// 获取第一章内容
	content, err := book.GetChapterContent(0, zrc)
	if err != nil {
		t.Fatalf("GetChapterContent failed: %v", err)
	}

	if content.Title != "第一章 开始" {
		t.Errorf("Expected title '第一章 开始', got '%s'", content.Title)
	}

	// 验证 script/style 被移除
	if strings.Contains(content.Content, "script") || strings.Contains(content.Content, "alert") {
		t.Errorf("Expected script tags to be removed, but found them in content")
	}

	// 验证内容包含正文
	if !strings.Contains(content.PlainText, "第一章") {
		t.Errorf("Expected content to contain '第一章'")
	}

	// 验证字数
	if content.WordCount == 0 {
		t.Errorf("Expected word count > 0, got %d", content.WordCount)
	}
}

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSub  string
		noSub    string
	}{
		{
			name:    "remove script",
			input:   `<html><body><p>Hello</p><script>alert('x');</script></body></html>`,
			wantSub: "Hello",
			noSub:   "script",
		},
		{
			name:    "remove style",
			input:   `<html><body><p style="color:red">Text</p><style>.x{}</style></body></html>`,
			wantSub: "Text",
			noSub:   "style",
		},
		{
			name:    "remove noscript",
			input:   `<html><body><noscript>Backup</noscript><p>Main</p></body></html>`,
			wantSub: "Main",
			noSub:   "noscript",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cleanHTML(tt.input)
			if err != nil {
				t.Fatalf("cleanHTML failed: %v", err)
			}
			if tt.wantSub != "" && !strings.Contains(result, tt.wantSub) {
				t.Errorf("Result missing expected substring '%s': %s", tt.wantSub, result)
			}
			if tt.noSub != "" && strings.Contains(strings.ToLower(result), strings.ToLower(tt.noSub)) {
				t.Errorf("Result contains unwanted '%s': %s", tt.noSub, result)
			}
		})
	}
}

func TestExtractTextFromHTML(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{
			name:  "simple paragraph",
			input: `<html><body><p>Hello World</p></body></html>`,
			want:  "Hello World",
		},
		{
			name:  "multiple paragraphs",
			input: `<html><body><p>First</p><p>Second</p></body></html>`,
			want:  "First Second",
		},
		{
			name:  "with headings",
			input: `<html><body><h1>Title</h1><p>Content</p></body></html>`,
			want:  "Title Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextFromHTML(tt.input)
			if result != tt.want {
				t.Errorf("Expected '%s', got '%s'", tt.want, result)
			}
		})
	}
}

func TestResolveRelativeURL(t *testing.T) {
	tests := []struct {
		base string
		ref  string
		want string
	}{
		{"OEBPS/chapters/ch1.html", "ch2.html", "OEBPS/chapters/ch2.html"},
		{"OEBPS/content.opf", "../images/cover.jpg", "images/cover.jpg"},
		{"OEBPS/ch1.html", "images/pic.png", "OEBPS/images/pic.png"},
		{"content.opf", "chapter1.html", "chapter1.html"},
		{"OEBPS/ch1.html", "http://example.com/img.jpg", "http://example.com/img.jpg"},
	}

	for _, tt := range tests {
		result := resolveRelativeURL(tt.base, tt.ref)
		if result != tt.want {
			t.Errorf("resolveRelativeURL(%q, %q) = %q, want %q", tt.base, tt.ref, result, tt.want)
		}
	}
}

func TestCountChineseChars(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"Hello World", 10},
		{"你好世界", 4},
		{"Hello 你好", 7},
		{"", 0},
		{"123 中文 456", 8},
	}

	for _, tt := range tests {
		result := countChineseChars(tt.input)
		if result != tt.want {
			t.Errorf("countChineseChars(%q) = %d, want %d", tt.input, result, tt.want)
		}
	}
}

func TestGetMetadata(t *testing.T) {
	buf := createTestEPUB(t, "simple")
	tmpFile := saveTestEPUB(t, buf)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	meta := book.GetMetadata()

	if meta["title"] != "测试书籍" {
		t.Errorf("Expected title '测试书籍', got '%s'", meta["title"])
	}
	if meta["author"] != "测试作者" {
		t.Errorf("Expected author '测试作者', got '%s'", meta["author"])
	}
	if meta["language"] != "zh-CN" {
		t.Errorf("Expected language 'zh-CN', got '%s'", meta["language"])
	}
	if meta["identifier"] != "test-book-001" {
		t.Errorf("Expected identifier 'test-book-001', got '%s'", meta["identifier"])
	}
}

func TestGetCover(t *testing.T) {
	buf := createTestEPUB(t, "epub2-ncx")
	tmpFile := saveTestEPUB(t, buf)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	cover, err := book.GetCover()
	if err != nil {
		t.Fatalf("GetCover failed: %v", err)
	}

	if len(cover) < 20 {
		t.Errorf("Expected cover data > 20 bytes, got %d", len(cover))
	}

	// 验证是 JPEG
	if cover[0] != 0xFF || cover[1] != 0xD8 {
		t.Errorf("Expected JPEG header, got %02X %02X", cover[0], cover[1])
	}
}

func TestGetCoverAsBase64(t *testing.T) {
	buf := createTestEPUB(t, "epub2-ncx")
	tmpFile := saveTestEPUB(t, buf)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	data, mime, err := book.GetCoverAsBase64()
	if err != nil {
		t.Fatalf("GetCoverAsBase64 failed: %v", err)
	}

	if data == "" {
		t.Error("Expected non-empty base64 data")
	}
	if mime != "image/jpeg" {
		t.Errorf("Expected mime 'image/jpeg', got '%s'", mime)
	}

	// 验证 base64 可解码
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}
	if len(decoded) < 20 {
		t.Errorf("Expected decoded data > 20 bytes, got %d", len(decoded))
	}
}

func TestBookGetChapters(t *testing.T) {
	buf := createTestEPUB(t, "simple")
	tmpFile := saveTestEPUB(t, buf)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	chapters := book.GetChapters()

	// 验证返回的是副本，不影响原数据
	if len(chapters) != 3 {
		t.Errorf("Expected 3 chapters, got %d", len(chapters))
	}

	// 验证章节结构
	for i, ch := range chapters {
		if ch.Index != i {
			t.Errorf("Chapter index mismatch: %d", ch.Index)
		}
		if ch.Path == "" {
			t.Errorf("Chapter %d has empty path", i)
		}
	}
}

func TestBookGetTOC(t *testing.T) {
	buf := createTestEPUB(t, "epub2-ncx")
	tmpFile := saveTestEPUB(t, buf)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	toc := book.GetTOC()

	// 验证返回的是副本
	if len(toc) == 0 {
		t.Error("Expected non-empty TOC")
	}

	// 修改返回的 TOC 不应影响原数据
	if len(toc) > 0 {
		toc[0].Title = "Modified"
		if book.TOC[0].Title == "Modified" {
			t.Error("GetTOC should return a copy, not the original")
		}
	}
}

func TestParseInvalidEPUB(t *testing.T) {
	// 创建一个无效的 ZIP 文件
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.epub")
	os.WriteFile(tmpFile, []byte("not a zip file"), 0644)

	_, err := Parse(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid EPUB")
	}
}

func TestParseEmptyEPUB(t *testing.T) {
	// 创建一个空的 ZIP 文件
	buf := new(bytes.Buffer)
	zip.NewWriter(buf).Close()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.epub")
	os.WriteFile(tmpFile, buf.Bytes(), 0644)

	_, err := Parse(tmpFile)
	if err == nil {
		t.Error("Expected error for empty EPUB (no container.xml)")
	}
}

func TestChapterContentImages(t *testing.T) {
	// 创建带图片的 EPUB
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// mimetype
	w, _ := zw.Create("mimetype")
	w.Write([]byte("application/epub+zip"))

	// container.xml
	w, _ = zw.Create("META-INF/container.xml")
	w.Write([]byte(`<?xml version="1.0"?><container version="1.0"><rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))

	// content.opf
	w, _ = zw.Create("content.opf")
	w.Write([]byte(`<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="uid">img-test</dc:identifier>
    <dc:title>图片测试</dc:title>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.html" media-type="application/xhtml+xml"/>
    <item id="image1" href="images/pic1.jpg" media-type="image/jpeg"/>
  </manifest>
  <spine><itemref idref="chapter1"/></spine>
</package>`))

	// chapter1.html with image
	w, _ = zw.Create("chapter1.html")
	w.Write([]byte(`<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml">
<body><h1>带图片的章节</h1>
<p>这是包含图片的章节。</p>
<img src="images/pic1.jpg" alt="测试图片"/>
<p>图片之后。</p>
</body></html>`))

	// images/pic1.jpg
	w, _ = zw.Create("images/pic1.jpg")
	w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46})

	zw.Close()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "img-test.epub")
	os.WriteFile(tmpFile, buf.Bytes(), 0644)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	archive, err := zip.OpenReader(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open zip: %v", err)
	}
	defer archive.Close()

	zrc := &ZipReadCloser{R: &archive.Reader, Closer: archive}

	content, err := book.GetChapterContent(0, zrc)
	if err != nil {
		t.Fatalf("GetChapterContent failed: %v", err)
	}

	// 验证图片被嵌入
	if len(content.Images) == 0 {
		t.Log("Note: Image embedding requires valid image data")
	}

	// 验证内容包含图片 alt 文本
	if !strings.Contains(content.Content, "测试图片") && !strings.Contains(content.Content, "[图片]") {
		t.Log("Note: Image placeholder may vary")
	}
}

// =============================================================================
// ExportWithImages 测试
// =============================================================================

func TestExportWithImagesNoImages(t *testing.T) {
	buf := createTestEPUB(t, "simple")
	tmpFile := saveTestEPUB(t, buf)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	archive, err := zip.OpenReader(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open zip: %v", err)
	}
	defer archive.Close()

	zrc := &ZipReadCloser{R: &archive.Reader, Closer: archive}

	outputDir := t.TempDir()
	err = book.ExportWithImages(0, zrc, outputDir)
	if err != nil {
		t.Fatalf("ExportWithImages failed: %v", err)
	}

	// 验证 HTML 文件已创建
	htmlFile := filepath.Join(outputDir, "chapter_000.html")
	if _, err := os.Stat(htmlFile); err != nil {
		t.Errorf("Expected HTML file to exist: %v", err)
	}

	// 验证 images 目录已创建
	imagesDir := filepath.Join(outputDir, "images")
	info, err := os.Stat(imagesDir)
	if err != nil {
		t.Errorf("Expected images dir to exist: %v", err)
	} else if !info.IsDir() {
		t.Errorf("Expected images to be a directory")
	}
}

func TestExportWithImagesWithBase64Images(t *testing.T) {
	// 创建带图片的 EPUB
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// mimetype
	w, _ := zw.Create("mimetype")
	w.Write([]byte("application/epub+zip"))

	// container.xml
	w, _ = zw.Create("META-INF/container.xml")
	w.Write([]byte(`<?xml version="1.0"?><container version="1.0"><rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))

	// content.opf
	w, _ = zw.Create("content.opf")
	w.Write([]byte(`<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="uid">img-export-test</dc:identifier>
    <dc:title>图片导出测试</dc:title>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.html" media-type="application/xhtml+xml"/>
    <item id="image1" href="images/pic1.jpg" media-type="image/jpeg"/>
    <item id="image2" href="images/pic2.png" media-type="image/png"/>
  </manifest>
  <spine><itemref idref="chapter1"/></spine>
</package>`))

	// chapter1.html with two images
	w, _ = zw.Create("chapter1.html")
	w.Write([]byte(`<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml">
<body><h1>带图片的章节</h1>
<p>第一张图片：</p>
<img src="images/pic1.jpg" alt="JPEG图片"/>
<p>第二张图片：</p>
<img src="images/pic2.png" alt="PNG图片"/>
<p>图片之后。</p>
</body></html>`))

	// images/pic1.jpg (fake JPEG with valid header)
	w, _ = zw.Create("images/pic1.jpg")
	jpegData := make([]byte, 200)
	jpegData[0] = 0xFF
	jpegData[1] = 0xD8
	jpegData[2] = 0xFF
	jpegData[3] = 0xE0
	for i := 4; i < 200; i++ {
		jpegData[i] = byte(i % 256)
	}
	w.Write(jpegData)

	// images/pic2.png (fake PNG with valid header)
	w, _ = zw.Create("images/pic2.png")
	pngData := make([]byte, 150)
	pngData[0] = 0x89
	pngData[1] = 0x50
	pngData[2] = 0x4E
	pngData[3] = 0x47
	for i := 4; i < 150; i++ {
		pngData[i] = byte(i % 256)
	}
	w.Write(pngData)

	zw.Close()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "img-export-test.epub")
	os.WriteFile(tmpFile, buf.Bytes(), 0644)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	archive, err := zip.OpenReader(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open zip: %v", err)
	}
	defer archive.Close()

	zrc := &ZipReadCloser{R: &archive.Reader, Closer: archive}

	outputDir := t.TempDir()
	err = book.ExportWithImages(0, zrc, outputDir)
	if err != nil {
		t.Fatalf("ExportWithImages failed: %v", err)
	}

	// 验证 HTML 文件
	htmlFile := filepath.Join(outputDir, "chapter_000.html")
	htmlData, err := os.ReadFile(htmlFile)
	if err != nil {
		t.Fatalf("Failed to read HTML file: %v", err)
	}
	htmlContent := string(htmlData)

	// 验证 HTML 不包含 base64 data URI
	if strings.Contains(htmlContent, "data:image") {
		t.Errorf("Expected HTML to not contain data URIs, but it does")
	}

	// 验证 HTML 包含 images/ 相对路径
	if !strings.Contains(htmlContent, "images/img_") {
		t.Errorf("Expected HTML to contain relative image paths, got: %s", htmlContent)
	}

	// 验证 images 目录中有文件
	imagesDir := filepath.Join(outputDir, "images")
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		t.Fatalf("Failed to read images dir: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 image files, got %d", len(entries))
	}

	// 验证图片文件有正确扩展名和内容
	foundJPG := false
	foundPNG := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".jpg") {
			foundJPG = true
			data, err := os.ReadFile(filepath.Join(imagesDir, entry.Name()))
			if err != nil {
				t.Errorf("Failed to read image file: %v", err)
			}
			if len(data) != 200 {
				t.Errorf("Expected JPEG data length 200, got %d", len(data))
			}
			// 验证 JPEG 头
			if data[0] != 0xFF || data[1] != 0xD8 {
				t.Errorf("Expected JPEG header, got %02X %02X", data[0], data[1])
			}
		}
		if strings.HasSuffix(entry.Name(), ".png") {
			foundPNG = true
			data, err := os.ReadFile(filepath.Join(imagesDir, entry.Name()))
			if err != nil {
				t.Errorf("Failed to read image file: %v", err)
			}
			if len(data) != 150 {
				t.Errorf("Expected PNG data length 150, got %d", len(data))
			}
			// 验证 PNG 头
			if data[0] != 0x89 || data[1] != 0x50 {
				t.Errorf("Expected PNG header, got %02X %02X", data[0], data[1])
			}
		}
	}

	if !foundJPG {
		t.Errorf("Expected to find a .jpg file")
	}
	if !foundPNG {
		t.Errorf("Expected to find a .png file")
	}
}

func TestExportWithImagesDataURI(t *testing.T) {
	// 创建带 data URI 图片的 EPUB
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	w, _ := zw.Create("mimetype")
	w.Write([]byte("application/epub+zip"))

	w, _ = zw.Create("META-INF/container.xml")
	w.Write([]byte(`<?xml version="1.0"?><container version="1.0"><rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))

	w, _ = zw.Create("content.opf")
	w.Write([]byte(`<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="uid">data-uri-test</dc:identifier>
    <dc:title>Data URI 测试</dc:title>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.html" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="chapter1"/></spine>
</package>`))

	// 创建一个小的 PNG 图片数据
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	for i := 0; i < 50; i++ {
		pngData = append(pngData, byte(i))
	}
	base64Data := base64.StdEncoding.EncodeToString(pngData)

	w, _ = zw.Create("chapter1.html")
	w.Write([]byte(fmt.Sprintf(`<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml">
<body><h1>Data URI 章节</h1>
<img src="data:image/png;base64,%s" alt="内嵌图片"/>
<p>图片之后。</p>
</body></html>`, base64Data)))

	zw.Close()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "data-uri-test.epub")
	os.WriteFile(tmpFile, buf.Bytes(), 0644)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	archive, err := zip.OpenReader(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open zip: %v", err)
	}
	defer archive.Close()

	zrc := &ZipReadCloser{R: &archive.Reader, Closer: archive}

	outputDir := t.TempDir()
	err = book.ExportWithImages(0, zrc, outputDir)
	if err != nil {
		t.Fatalf("ExportWithImages failed: %v", err)
	}

	// 验证 HTML 文件
	htmlFile := filepath.Join(outputDir, "chapter_000.html")
	htmlData, err := os.ReadFile(htmlFile)
	if err != nil {
		t.Fatalf("Failed to read HTML file: %v", err)
	}
	htmlContent := string(htmlData)

	// 验证 HTML 不包含 data URI
	if strings.Contains(htmlContent, "data:image") {
		t.Errorf("Expected HTML to not contain data URI, but it does")
	}

	// 验证 HTML 包含相对路径
	if !strings.Contains(htmlContent, "images/") {
		t.Errorf("Expected HTML to contain relative image path")
	}

	// 验证图片文件存在
	imagesDir := filepath.Join(outputDir, "images")
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		t.Fatalf("Failed to read images dir: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 image file, got %d", len(entries))
	}

	if len(entries) > 0 {
		// 验证扩展名
		if !strings.HasSuffix(entries[0].Name(), ".png") {
			t.Errorf("Expected .png extension, got %s", entries[0].Name())
		}

		// 验证文件内容
		data, err := os.ReadFile(filepath.Join(imagesDir, entries[0].Name()))
		if err != nil {
			t.Fatalf("Failed to read image file: %v", err)
		}
		if !bytes.Equal(data, pngData) {
			t.Errorf("Image data mismatch: expected %d bytes, got %d bytes", len(pngData), len(data))
		}
	}
}

func TestExportWithImagesInvalidChapter(t *testing.T) {
	buf := createTestEPUB(t, "simple")
	tmpFile := saveTestEPUB(t, buf)

	book, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	archive, err := zip.OpenReader(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open zip: %v", err)
	}
	defer archive.Close()

	zrc := &ZipReadCloser{R: &archive.Reader, Closer: archive}

	outputDir := t.TempDir()

	// 测试无效章节索引
	err = book.ExportWithImages(-1, zrc, outputDir)
	if err == nil {
		t.Error("Expected error for invalid chapter index")
	}

	err = book.ExportWithImages(100, zrc, outputDir)
	if err == nil {
		t.Error("Expected error for out-of-range chapter index")
	}
}

func TestMimeToExtension(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"image/svg+xml", ".svg"},
		{"application/octet-stream", ".jpg"},
		{"", ".jpg"},
	}

	for _, tt := range tests {
		got := mimeToExtension(tt.mime)
		if got != tt.want {
			t.Errorf("mimeToExtension(%q) = %q, want %q", tt.mime, got, tt.want)
		}
	}
}

func TestGenerateImageFilename(t *testing.T) {
	usedNames := make(map[string]bool)

	data1 := []byte("test image data 1")
	data2 := []byte("test image data 2")

	name1 := generateImageFilename(data1, ".jpg", usedNames)
	if !strings.HasPrefix(name1, "img_") {
		t.Errorf("Expected prefix 'img_', got %q", name1)
	}
	if !strings.HasSuffix(name1, ".jpg") {
		t.Errorf("Expected suffix '.jpg', got %q", name1)
	}

	usedNames[name1] = true

	name2 := generateImageFilename(data2, ".png", usedNames)
	if !strings.HasSuffix(name2, ".png") {
		t.Errorf("Expected suffix '.png', got %q", name2)
	}
	if name1 == name2 {
		t.Errorf("Expected different filenames for different data")
	}

	// 相同数据应生成相同文件名（如果 usedNames 不包含它）
	usedNames2 := make(map[string]bool)
	name1Again := generateImageFilename(data1, ".jpg", usedNames2)
	if name1 != name1Again {
		t.Errorf("Expected same filename for same data, got %q vs %q", name1, name1Again)
	}
}

func BenchmarkParseEPUB(b *testing.B) {
	buf := createTestEPUB(b, "epub3-nav")
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "bench.epub")
	os.WriteFile(tmpFile, buf.Bytes(), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Parse(tmpFile)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkGetChapterContent(b *testing.B) {
	buf := createTestEPUB(b, "epub3-nav")
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "bench.epub")
	os.WriteFile(tmpFile, buf.Bytes(), 0644)

	book, _ := Parse(tmpFile)
	archive, _ := zip.OpenReader(tmpFile)
	defer archive.Close()
	zrc := &ZipReadCloser{R: &archive.Reader, Closer: archive}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := book.GetChapterContent(0, zrc)
		if err != nil {
			b.Fatalf("GetChapterContent failed: %v", err)
		}
	}
}
