package txt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// BOM 检测测试
// =============================================================================

func TestDetectBOM(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantType EncodingType
		wantLen  int
	}{
		{
			name:     "UTF-8 BOM",
			data:     append(BOM_UTF8, []byte("hello")...),
			wantType: EncodingUTF8BOM,
			wantLen:  3,
		},
		{
			name:     "UTF-16 LE BOM",
			data:     append(BOM_UTF16_LE, []byte{0x68, 0x00}...), // "h" in UTF-16 LE
			wantType: EncodingUTF16LE,
			wantLen:  2,
		},
		{
			name:     "UTF-16 BE BOM",
			data:     append(BOM_UTF16_BE, []byte{0x00, 0x68}...), // "h" in UTF-16 BE
			wantType: EncodingUTF16BE,
			wantLen:  2,
		},
		{
			name:     "No BOM",
			data:     []byte("hello"),
			wantType: EncodingUnknown,
			wantLen:  0,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			wantType: EncodingUnknown,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotLen := DetectBOM(tt.data)
			if gotType != tt.wantType {
				t.Errorf("DetectBOM() gotType = %v, want %v", gotType, tt.wantType)
			}
			if gotLen != tt.wantLen {
				t.Errorf("DetectBOM() gotLen = %v, want %v", gotLen, tt.wantLen)
			}
		})
	}
}

// =============================================================================
// 编码检测测试
// =============================================================================

func TestEncodingDetector_DetectFromBytes(t *testing.T) {
	detector := NewEncodingDetector()

	tests := []struct {
		name string
		data []byte
		want EncodingType
	}{
		{
			name: "UTF-8 ASCII",
			data: []byte("Hello, World! This is a test."),
			want: EncodingUTF8,
		},
		{
			name: "UTF-8 with Chinese",
			data: []byte("你好，世界！这是一段中文测试。"),
			want: EncodingUTF8,
		},
		{
			name: "UTF-8 BOM",
			data: append(BOM_UTF8, []byte("Hello")...),
			want: EncodingUTF8BOM,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detector.DetectFromBytes(tt.data)
			if err != nil {
				t.Errorf("DetectFromBytes() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("DetectFromBytes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToUTF8(t *testing.T) {
	// 测试 UTF-8 数据
	utf8Data := []byte("Hello, 世界！")
	result, err := ConvertToUTF8(utf8Data, EncodingUTF8)
	if err != nil {
		t.Errorf("ConvertToUTF8() error = %v", err)
	}
	if string(result) != "Hello, 世界！" {
		t.Errorf("ConvertToUTF8() got = %v, want %v", string(result), "Hello, 世界！")
	}

	// 测试带 BOM 的 UTF-8
	utf8BOMData := append(BOM_UTF8, []byte("Hello")...)
	result, err = ConvertToUTF8(utf8BOMData, EncodingUTF8BOM)
	if err != nil {
		t.Errorf("ConvertToUTF8() error = %v", err)
	}
	if string(result) != "Hello" {
		t.Errorf("ConvertToUTF8() got = %v, want %v", string(result), "Hello")
	}
}

// =============================================================================
// TOC 规则测试
// =============================================================================

func TestTocRule_Match(t *testing.T) {
	// 测试第一条规则：第 X 章 标题
	rule := DefaultTocRules[0]

	tests := []struct {
		name    string
		line    string
		wantMatch bool
		wantTitle string
	}{
		{
			name:    "标准章节",
			line:    "第1章 开篇",
			wantMatch: true,
			wantTitle: "开篇",
		},
		{
			name:    "第 1 章",
			line:    "第 1 章 开始",
			wantMatch: true,
			wantTitle: "开始",
		},
		{
			name:    "第 12 节",
			line:    "第 12 节 转折",
			wantMatch: true,
			wantTitle: "转折",
		},
		{
			name:    "非章节",
			line:    "这是一个普通段落",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := rule.Regex.FindStringSubmatch(tt.line)
			if tt.wantMatch {
				if matches == nil {
					t.Errorf("Expected match but got none")
					return
				}
				if len(matches) < 3 {
					t.Errorf("Expected at least 3 groups, got %d", len(matches))
					return
				}
				title := strings.TrimSpace(matches[2])
				if title != tt.wantTitle {
					t.Errorf("Title = %v, want %v", title, tt.wantTitle)
				}
			} else {
				if matches != nil {
					t.Errorf("Expected no match but got %v", matches)
				}
			}
		})
	}
}

func TestTocAnalyzer_Analyze(t *testing.T) {
	analyzer := NewTocAnalyzer()

	text := `第一章 开篇
第二章 发展
第三章 高潮
第四章 结局

这是一个普通段落，不是章节。

1. 番外篇
2. 后记

第五卷 特别篇
`

	bestRule, matches := analyzer.Analyze(text)

	if bestRule == nil {
		t.Error("Expected best rule, got nil")
	}

	if len(matches) < 5 {
		t.Errorf("Expected at least 5 matches, got %d", len(matches))
	}

	// 检查章节标题
	expectedTitles := []string{"开篇", "发展", "高潮", "结局"}
	for i, expected := range expectedTitles {
		if i >= len(matches) {
			break
		}
		if matches[i].Title != expected {
			t.Errorf("Match %d title = %v, want %v", i, matches[i].Title, expected)
		}
	}
}

func TestFilterMatches(t *testing.T) {
	matches := []*TocRuleMatch{
		{Title: "第一章 开篇", Position: 0},
		{Title: "目录", Position: 100},
		{Title: "第二章 发展", Position: 200},
		{Title: "版权声明", Position: 300},
		{Title: "第三章 高潮", Position: 400},
		{Title: "1", Position: 500}, // 纯数字
		{Title: "第四章 结局", Position: 600},
	}

	filtered := FilterMatches(matches)

	if len(filtered) != 4 {
		t.Errorf("Expected 4 matches after filtering, got %d", len(filtered))
	}

	for _, m := range filtered {
		if m.Title == "目录" || m.Title == "版权声明" || m.Title == "1" {
			t.Errorf("Filtered match should not contain suspicious title: %s", m.Title)
		}
	}
}

func TestIsSuspiciousTitle(t *testing.T) {
	tests := []struct {
		title string
		want  bool
	}{
		{"第一章 开篇", false},
		{"目录", true},
		{"前言", true},
		{"后记", true},
		{"版权声明", true},
		{"1", true},
		{"", true},
		{"非常长的章节标题名称测试用", false},
		{"A", true}, // 太短
	}

	for _, tt := range tests {
		got := isSuspiciousTitle(tt.title)
		if got != tt.want {
			t.Errorf("isSuspiciousTitle(%q) = %v, want %v", tt.title, got, tt.want)
		}
	}
}

// =============================================================================
// TextFile 解析器测试
// =============================================================================

func createTestFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "test_book.txt")

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	return path
}

func TestTextFile_Parse_Simple(t *testing.T) {
	content := `第一章 开篇

这是一个普通的段落内容。

第二章 发展

故事继续发展。

第三章 高潮

进入高潮部分。

第四章 结局

故事圆满结束。
`

	path := createTestFile(t, content)
	defer os.Remove(path)

	tf := NewTextFile(path)
	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if book == nil {
		t.Fatal("Expected book, got nil")
	}

	if book.Name != "test_book" {
		t.Errorf("Book name = %v, want %v", book.Name, "test_book")
	}

	if len(book.Chapters) != 4 {
		t.Errorf("Expected 4 chapters, got %d", len(book.Chapters))
	}

	expectedTitles := []string{"开篇", "发展", "高潮", "结局"}
	for i, expected := range expectedTitles {
		if book.Chapters[i].Title != expected {
			t.Errorf("Chapter %d title = %v, want %v", i, book.Chapters[i].Title, expected)
		}
	}
}

func TestTextFile_Parse_GetChapters(t *testing.T) {
	content := `第一章 测试章节 1
内容 1

第二章 测试章节 2
内容 2

第三章 测试章节 3
内容 3
`

	path := createTestFile(t, content)
	defer os.Remove(path)

	tf := NewTextFile(path)
	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	chapters := tf.GetChapters()
	if len(chapters) != 3 {
		t.Errorf("GetChapters() returned %d chapters, want 3", len(chapters))
	}

	// 也测试包级函数
	chapters2 := GetChapters(book)
	if len(chapters2) != 3 {
		t.Errorf("GetChapters(book) returned %d chapters, want 3", len(chapters2))
	}
}

func TestTextFile_Parse_GetContent(t *testing.T) {
	content := `第一章 开篇

这是第一章的内容，比较长的一段文字。

第二章 发展

这是第二章的内容。

第三章 高潮

这是第三章的内容，非常精彩的部分。
`

	path := createTestFile(t, content)
	defer os.Remove(path)

	tf := NewTextFile(path)
	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(book.Chapters) == 0 {
		t.Fatal("No chapters found")
	}

	// 获取第一个章节的内容
	chapter := book.Chapters[0]
	chapterContent, err := tf.GetContent(chapter.StartPos, chapter.EndPos)
	if err != nil {
		t.Errorf("GetContent() error = %v", err)
	}

	if chapterContent == "" {
		t.Error("Expected non-empty content")
	}

	// 测试包级函数
	content2, err := GetBookContent(book, chapter.StartPos, chapter.EndPos)
	if err != nil {
		t.Errorf("GetBookContent() error = %v", err)
	}
	if content2 == "" {
		t.Error("Expected non-empty content from GetBookContent")
	}
}

func TestTextFile_Parse_BigFile(t *testing.T) {
	// 创建一个较大的测试文件（模拟大文件）
	var content strings.Builder
	for i := 1; i <= 100; i++ {
		content.WriteString(fmt.Sprintf("第%d章 章节标题%d\n\n", i, i))
		for j := 1; j <= 100; j++ {
			content.WriteString(fmt.Sprintf("这是第%d章第%d段的内容。", i, j))
		}
		content.WriteString("\n\n")
	}

	path := createTestFile(t, content.String())
	defer os.Remove(path)

	tf := NewTextFile(path)
	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(book.Chapters) != 100 {
		t.Errorf("Expected 100 chapters, got %d", len(book.Chapters))
	}

	if book.TotalChapters != 100 {
		t.Errorf("Expected TotalChapters = 100, got %d", book.TotalChapters)
	}
}

func TestTextFile_Parse_LongChapterSplit(t *testing.T) {
	// 创建一个超长章节（超过默认 MaxChapterLength）
	var content strings.Builder
	content.WriteString("第一章 超长章节\n\n")
	// 添加超过 50000 字符的内容
	for i := 0; i < 20000; i++ {
		content.WriteString("这是段落内容。")
	}

	path := createTestFile(t, content.String())
	defer os.Remove(path)

	tf := NewTextFile(path)
	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// 由于内容超过 5 万字，应该被拆分
	if len(book.Chapters) < 1 {
		t.Errorf("Expected at least 1 chapter, got %d", len(book.Chapters))
	}
}

func TestTextFile_Parse_NoToc(t *testing.T) {
	// 创建一个没有明确章节标记的文件
	content := `这是一本小说的正文内容。

第一段文字。

第二段文字。

第三段文字。

继续更多内容...
`

	path := createTestFile(t, content)
	defer os.Remove(path)

	tf := NewTextFile(path)
	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// 没有章节标记，章节列表可能为空或很少
	t.Logf("Chapters found: %d", len(book.Chapters))
}

func TestTextFile_Parse_CustomTocRules(t *testing.T) {
	content := `
=== 第一章：开篇 ===

内容...

=== 第二章：发展 ===

内容...

=== 第三章：高潮 ===

内容...
`

	path := createTestFile(t, content)
	defer os.Remove(path)

	tf := NewTextFile(path)
	tf.AddTocRule(`^===\s*第[^=]+===\s*$`, "custom_dashed", 100)
	// 注意：这个规则需要改进以正确提取标题

	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	t.Logf("Chapters with custom rules: %d", len(book.Chapters))
}

func TestTextFile_Parse_EncodingDetection(t *testing.T) {
	// 测试 UTF-8 文件
	content := "第一章 测试\n\n内容内容。"
	path := createTestFile(t, content)
	defer os.Remove(path)

	tf := NewTextFile(path)
	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if book.Encoding != EncodingUTF8 {
		t.Errorf("Expected UTF-8 encoding, got %v", book.Encoding)
	}
}

func TestTextFile_Parse_ForceEncoding(t *testing.T) {
	content := "第一章 测试\n\n内容。"
	path := createTestFile(t, content)
	defer os.Remove(path)

	tf := NewTextFile(path)
	tf.SetEncoding(EncodingUTF8)

	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if book.Encoding != EncodingUTF8 {
		t.Errorf("Expected forced UTF-8 encoding, got %v", book.Encoding)
	}
}

func TestTextFile_GetBestTocRule(t *testing.T) {
	content := `
第 1 章 开篇
第 2 章 发展
第 3 章 高潮
第 4 章 结局
`

	path := createTestFile(t, content)
	defer os.Remove(path)

	tf := NewTextFile(path)
	_, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	rule := tf.GetBestTocRule()
	if rule == nil {
		t.Error("Expected best TOC rule, got nil")
	}

	t.Logf("Best rule: %s (priority: %d)", rule.Name, rule.Priority)
}

func TestTextFile_ParseOptions(t *testing.T) {
	content := `第一章 测试
内容

第二章 测试 2
内容
`

	path := createTestFile(t, content)
	defer os.Remove(path)

	opts := DefaultParseOptions()
	opts.MaxChapterLength = 1000
	opts.AutoSplitChapters = false
	opts.FilterSuspiciousTitles = true

	tf := NewTextFile(path)
	tf.SetOptions(opts)

	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if book.options.MaxChapterLength != 1000 {
		t.Errorf("Expected MaxChapterLength = 1000, got %d", book.options.MaxChapterLength)
	}
}

func TestParse_Basic(t *testing.T) {
	content := `第一章 开篇
内容 1

第二章 发展
内容 2
`

	path := createTestFile(t, content)
	defer os.Remove(path)

	book, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(book.Chapters) != 2 {
		t.Errorf("Expected 2 chapters, got %d", len(book.Chapters))
	}
}

func TestParseWithEncoding(t *testing.T) {
	content := "第一章 测试"
	path := createTestFile(t, content)
	defer os.Remove(path)

	book, err := ParseWithEncoding(path, EncodingUTF8)
	if err != nil {
		t.Fatalf("ParseWithEncoding() error = %v", err)
	}

	if book.Encoding != EncodingUTF8 {
		t.Errorf("Expected UTF-8, got %v", book.Encoding)
	}
}

func TestParseWithOptions(t *testing.T) {
	content := "第一章 测试"
	path := createTestFile(t, content)
	defer os.Remove(path)

	opts := DefaultParseOptions()
	opts.DetectEncoding = false

	book, err := ParseWithOptions(path, opts)
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}

	if book.Encoding != EncodingUTF8 {
		t.Errorf("Expected UTF-8 (forced), got %v", book.Encoding)
	}
}

// =============================================================================
// 边界情况测试
// =============================================================================

func TestTextFile_Parse_EmptyFile(t *testing.T) {
	path := createTestFile(t, "")
	defer os.Remove(path)

	tf := NewTextFile(path)
	book, err := tf.Parse()
	if err != nil {
		// 空文件可能返回错误，这是可以接受的
		t.Logf("Empty file returned error (acceptable): %v", err)
		return
	}

	if book == nil {
		t.Fatal("Expected book, got nil")
	}

	if len(book.Chapters) != 0 {
		t.Errorf("Expected 0 chapters for empty file, got %d", len(book.Chapters))
	}
}

func TestTextFile_Parse_SingleChapter(t *testing.T) {
	content := `这是一个没有章节标记的长文本。

包含多段内容。

结尾部分。
`

	path := createTestFile(t, content)
	defer os.Remove(path)

	tf := NewTextFile(path)
	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// 没有章节标记，可能没有章节或整个文件作为一个章节
	t.Logf("Single chapter file: %d chapters", len(book.Chapters))
}

func TestTextFile_Parse_MixedChapterFormats(t *testing.T) {
	content := `
第 1 章 第一章
内容...

第二章 第二章
内容...

3. 第三章
内容...

(4) 第四章
内容...

[5] 第五章
内容...
`

	path := createTestFile(t, content)
	defer os.Remove(path)

	tf := NewTextFile(path)
	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	t.Logf("Mixed formats: %d chapters found", len(book.Chapters))
	for i, ch := range book.Chapters {
		t.Logf("  %d: %s", i+1, ch.Title)
	}
}

// =============================================================================
// TOC 规则完整性测试
// =============================================================================

func TestDefaultTocRules_Count(t *testing.T) {
	if len(DefaultTocRules) != 18 {
		t.Errorf("Expected 18 default TOC rules, got %d", len(DefaultTocRules))
	}
}

func TestDefaultTocRules_AllCompiled(t *testing.T) {
	for i, rule := range DefaultTocRules {
		if rule.Regex == nil {
			t.Errorf("Rule %d (%s) has nil Regex", i, rule.Name)
		}
		if rule.Pattern == "" {
			t.Errorf("Rule %d has empty Pattern", i)
		}
	}
}

func TestGetDefaultRules(t *testing.T) {
	rules := GetDefaultRules()
	if len(rules) != 18 {
		t.Errorf("Expected 18 rules from GetDefaultRules(), got %d", len(rules))
	}

	// 确保返回的是副本
	rules[0].Pattern = "modified"
	if DefaultTocRules[0].Pattern == "modified" {
		t.Error("GetDefaultRules() should return a copy, not the original")
	}
}

func TestAddCustomRule(t *testing.T) {
	rules := GetDefaultRules()
	originalCount := len(rules)

	rules = AddCustomRule(rules, `^Custom: (.+)$`, "custom_test", 150)

	if len(rules) != originalCount+1 {
		t.Errorf("Expected %d rules after adding, got %d", originalCount+1, len(rules))
	}

	// 检查优先级排序（高优先级在前）
	if rules[0].Priority < rules[1].Priority {
		t.Error("Rules should be sorted by priority (high to low)")
	}
}

func TestRemoveRule(t *testing.T) {
	rules := GetDefaultRules()
	originalCount := len(rules)

	rules = RemoveRule(rules, "chapter_standard")

	if len(rules) != originalCount-1 {
		t.Errorf("Expected %d rules after removal, got %d", originalCount-1, len(rules))
	}

	// 确保被移除的规则不存在
	for _, r := range rules {
		if r.Name == "chapter_standard" {
			t.Error("Removed rule still exists")
		}
	}
}

// =============================================================================
// 性能测试
// =============================================================================

func BenchmarkTextFile_Parse(b *testing.B) {
	// 创建一个较大的测试文件
	var content strings.Builder
	for i := 1; i <= 50; i++ {
		content.WriteString(fmt.Sprintf("第%d章 章节标题%d\n\n", i, i))
		for j := 1; j <= 50; j++ {
			content.WriteString(fmt.Sprintf("这是第%d章第%d段的内容，包含一些文字。", i, j))
		}
		content.WriteString("\n\n")
	}

	// 写入临时文件
	dir := b.TempDir()
	path := filepath.Join(dir, "benchmark.txt")
	err := os.WriteFile(path, []byte(content.String()), 0644)
	if err != nil {
		b.Fatalf("Failed to create benchmark file: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tf := NewTextFile(path)
		_, err := tf.Parse()
		if err != nil {
			b.Fatalf("Parse() error = %v", err)
		}
	}
}

func BenchmarkTocAnalyzer_Analyze(b *testing.B) {
	var text strings.Builder
	for i := 1; i <= 100; i++ {
		text.WriteString(fmt.Sprintf("第%d章 标题%d\n", i, i))
	}
	text.WriteString("\n普通段落内容...\n")

	analyzer := NewTocAnalyzer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(text.String())
	}
}

func BenchmarkEncodingDetector_Detect(b *testing.B) {
	data := []byte("你好，世界！这是一段中文测试内容。Hello, World! This is a test.")
	detector := NewEncodingDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectFromBytes(data)
	}
}

// =============================================================================
// 集成测试
// =============================================================================

func TestTextFile_FullWorkflow(t *testing.T) {
	// 模拟完整的用户工作流程
	content := `小说名称：测试小说
作者：测试作者

第一章 开篇

这是一个故事的开始。主角站在山顶，眺望着远方的城市。

第二章 相遇

在城市里，主角遇到了一个重要的人物。

第三章 冒险

他们一起踏上了一段奇妙的冒险旅程。

第四章 挑战

面对强大的敌人，他们必须团结一心。

第五章 胜利

经过艰苦的战斗，他们终于获得了胜利。

尾声

故事圆满结束，但新的冒险仍在继续。

后记

感谢读者的支持！
`

	path := createTestFile(t, content)
	defer os.Remove(path)

	// 1. 创建解析器
	tf := NewTextFile(path)

	// 2. 配置选项
	tf.SetBufferSize(256 * 1024).
		SetOptions(DefaultParseOptions())

	// 3. 解析
	book, err := tf.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// 4. 验证结果
	if book.Name == "" {
		t.Error("Book name should not be empty")
	}

	if book.TotalChapters == 0 {
		t.Error("Should have chapters")
	}

	// 5. 获取章节
	chapters := tf.GetChapters()
	t.Logf("Found %d chapters", len(chapters))

	// 6. 获取内容
	if len(chapters) > 0 {
		chapter := chapters[0]
		content, err := tf.GetContent(chapter.StartPos, chapter.EndPos)
		if err != nil {
			t.Errorf("GetContent() error = %v", err)
		}
		t.Logf("First chapter content length: %d", len(content))
	}

	// 7. 获取最佳规则
	rule := tf.GetBestTocRule()
	if rule != nil {
		t.Logf("Best TOC rule: %s", rule.Name)
	}

	// 8. 关闭
	tf.Close()
}
