package txt

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/text/transform"
)

// =============================================================================
// Book 和 Chapter 数据结构
// =============================================================================

// Book 书籍实体
type Book struct {
	// 书名（从文件名或内容推断）
	Name string `json:"name"`

	// 作者（从内容推断，可选）
	Author string `json:"author,omitempty"`

	// 文件路径
	Path string `json:"-"`

	// 编码类型
	Encoding EncodingType `json:"encoding"`

	// 文件大小（字节）
	Size int64 `json:"size"`

	// 章节列表
	Chapters []Chapter `json:"chapters"`

	// 总章节数
	TotalChapters int `json:"totalChapters"`

	// 总字数
	TotalWords int `json:"totalWords"`

	// 是否已解析目录
	TocParsed bool `json:"tocParsed"`

	// 原始文本（用于流式读取时缓存）
	// 注意：大文件不会加载全部到内存
	rawContent *bytes.Buffer

	// 目录分析器
	tocAnalyzer *TocAnalyzer

	// 最佳 TOC 规则
	bestTocRule *TocRule

	// 解析选项
	options *ParseOptions
}

// Chapter 章节实体
type Chapter struct {
	// 章节标题
	Title string `json:"title"`

	// 章节序号（从 1 开始）
	Index int `json:"index"`

	// 章节起始位置（字节偏移，相对于 UTF-8 内容）
	StartPos int64 `json:"startPos"`

	// 章节结束位置（字节偏移）
	EndPos int64 `json:"endPos"`

	// 章节字数
	WordCount int `json:"wordCount"`

	// 父章节索引（用于嵌套章节）
	ParentIndex int `json:"parentIndex,omitempty"`

	// 子章节列表
	SubChapters []Chapter `json:"subChapters,omitempty"`

	// 是否 VIP/付费章节（TXT 无法判断，保留字段）
	IsVip bool `json:"isVip,omitempty"`

	// 是否免费试读（TXT 无法判断，保留字段）
	IsFree bool `json:"isFree,omitempty"`
}

// =============================================================================
// ParseOptions 解析选项
// =============================================================================

// ParseOptions TXT 解析选项
type ParseOptions struct {
	// 缓冲区大小（字节），默认 512KB
	BufferSize int

	// 最大章节长度（字符数），超过则拆分
	MaxChapterLength int

	// 是否自动拆分长章节
	AutoSplitChapters bool

	// 是否检测编码
	DetectEncoding bool

	// 强制指定编码（如果设置，跳过自动检测）
	ForceEncoding EncodingType

	// TOC 规则（如果为空，使用默认规则）
	TocRules []*TocRule

	// 是否过滤可疑章节标题
	FilterSuspiciousTitles bool

	// 章节标题最小长度
	MinTitleLength int

	// 章节标题最大长度
	MaxTitleLength int

	// 空行分隔符模式（用于检测章节分隔）
	EmptyLinePattern *regexp.Regexp

	// 是否保留原始换行
	KeepLineBreaks bool
}

// DefaultParseOptions 默认解析选项
func DefaultParseOptions() *ParseOptions {
	return &ParseOptions{
		BufferSize:            512 * 1024, // 512KB
		MaxChapterLength:      50000,      // 5 万字
		AutoSplitChapters:     true,
		DetectEncoding:        true,
		ForceEncoding:         EncodingUnknown,
		TocRules:              nil,        // 使用默认规则
		FilterSuspiciousTitles: true,
		MinTitleLength:        2,
		MaxTitleLength:        50,
		KeepLineBreaks:        true,
	}
}

// =============================================================================
// TextFile 主解析器
// =============================================================================

// TextFile TXT 文件解析器
// 参考 legado 的 TextFile.kt 实现
type TextFile struct {
	// 文件路径
	path string

	// 文件对象
	file *os.File

	// 解析选项
	options *ParseOptions

	// 编码检测器
	encodingDetector *EncodingDetector

	// 目录分析器
	tocAnalyzer *TocAnalyzer

	// 当前读取位置
	readPos int64

	// 文件大小
	fileSize int64

	// 已解析的书籍
	book *Book
}

// NewTextFile 创建新的 TXT 文件解析器
func NewTextFile(path string) *TextFile {
	return &TextFile{
		path:            path,
		encodingDetector: NewEncodingDetector(),
		tocAnalyzer:     NewTocAnalyzer(),
		options:         DefaultParseOptions(),
	}
}

// SetOptions 设置解析选项
func (tf *TextFile) SetOptions(opts *ParseOptions) *TextFile {
	tf.options = opts
	if opts.TocRules != nil {
		tf.tocAnalyzer = NewTocAnalyzerWithRules(opts.TocRules)
	}
	return tf
}

// SetBufferSize 设置缓冲区大小
func (tf *TextFile) SetBufferSize(size int) *TextFile {
	tf.options.BufferSize = size
	return tf
}

// SetEncoding 强制指定编码
func (tf *TextFile) SetEncoding(enc EncodingType) *TextFile {
	tf.options.ForceEncoding = enc
	tf.options.DetectEncoding = false
	return tf
}

// AddTocRule 添加自定义 TOC 规则
func (tf *TextFile) AddTocRule(pattern, name string, priority int) *TextFile {
	tf.options.TocRules = AddCustomRule(tf.options.TocRules, pattern, name, priority)
	tf.tocAnalyzer = NewTocAnalyzerWithRules(tf.options.TocRules)
	return tf
}

// =============================================================================
// Parse - 主解析入口
// =============================================================================

// Parse 解析 TXT 文件，返回书籍对象
// 这是主要的解析入口函数
func (tf *TextFile) Parse() (*Book, error) {
	// 1. 打开文件
	file, err := os.Open(tf.path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	tf.file = file

	// 2. 获取文件大小
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	tf.fileSize = stat.Size()

	// 3. 检测编码
	encoding, err := tf.detectEncoding(file)
	if err != nil {
		return nil, fmt.Errorf("detect encoding: %w", err)
	}

	// 4. 创建书籍对象
	book := &Book{
		Path:          tf.path,
		Encoding:      encoding,
		Size:          tf.fileSize,
		options:       tf.options,
		tocAnalyzer:   tf.tocAnalyzer,
		rawContent:    &bytes.Buffer{},
	}

	// 5. 从文件名推断书名
	book.Name = tf.inferBookName()

	// 6. 流式读取文件，同时检测目录模式
	if err := tf.streamParse(file, book, encoding); err != nil {
		return nil, fmt.Errorf("stream parse: %w", err)
	}

	// 7. 分析目录
	book.bestTocRule, book.Chapters = tf.analyzeTOC(book)

	// 8. 过滤和整理章节
	tf.filterAndSortChapters(book)

	// 9. 计算总字数
	tf.calculateWordCount(book)

	// 10. 处理长章节拆分
	if book.options.AutoSplitChapters {
		tf.splitLongChapters(book)
	}

	book.TocParsed = true
	book.TotalChapters = len(book.Chapters)

	tf.book = book
	return book, nil
}

// Parse 包级便捷函数：直接解析文件路径
func Parse(path string) (*Book, error) {
	tf := NewTextFile(path)
	return tf.Parse()
}

// =============================================================================
// 编码检测
// =============================================================================

// detectEncoding 检测文件编码
func (tf *TextFile) detectEncoding(file *os.File) (EncodingType, error) {
	if !tf.options.DetectEncoding {
		if tf.options.ForceEncoding != EncodingUnknown {
			return tf.options.ForceEncoding, nil
		}
		return EncodingUTF8, nil
	}

	// 读取文件开头用于检测
	buffer := make([]byte, 8192)
	n, err := io.ReadFull(file, buffer)
	if err != nil && err != io.ErrUnexpectedEOF {
		// 文件太小，尝试读取全部
		buffer = make([]byte, tf.fileSize)
		n, _ = file.Read(buffer)
	}
	buffer = buffer[:n]

	// 先检测 BOM
	bomType, bomLen := DetectBOM(buffer)
	if bomType != EncodingUnknown {
		// 重置文件指针
		file.Seek(int64(bomLen), 0)
		return bomType, nil
	}

	// 使用采样数据检测编码
	encoding, err := tf.encodingDetector.DetectFromBytes(buffer)
	if err != nil {
		return EncodingUTF8, nil // 默认 UTF-8
	}

	// 重置文件指针到开头
	file.Seek(0, 0)

	return encoding, nil
}

// =============================================================================
// 流式解析
// =============================================================================

// streamParse 流式解析文件
// 使用 bufio.Scanner 逐行读取，同时检测目录模式和提取章节
func (tf *TextFile) streamParse(file *os.File, book *Book, encoding EncodingType) error {
	// 跳过 BOM（如果有）
	bomLen := 0
	if encoding == EncodingUTF8BOM {
		bomLen = 3
	} else if encoding == EncodingUTF16LE || encoding == EncodingUTF16BE {
		bomLen = 2
	}

	if bomLen > 0 {
		if _, err := file.Seek(int64(bomLen), 0); err != nil {
			return err
		}
	}

	// 创建缓冲读取器
	bufferSize := tf.options.BufferSize
	if bufferSize <= 0 {
		bufferSize = 512 * 1024
	}
	reader := bufio.NewReaderSize(file, bufferSize)

	// 创建 UTF-8 转换读者（如果需要）
	var baseReader io.Reader = reader
	if encoding != EncodingUTF8 && encoding != EncodingUTF8BOM {
		decoder, err := GetDecoder(encoding)
		if err != nil {
			return err
		}
		baseReader = transform.NewReader(reader, decoder.NewDecoder())
	}

	// 创建扫描器
	scanner := bufio.NewScanner(baseReader)
	maxCapacity := 10 * 1024 * 1024 // 10MB 最大行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	// 当前章节
	var currentChapter *Chapter

	// 用于收集所有文本以进行 TOC 分析
	var allText bytes.Buffer

	// 字节位置跟踪（UTF-8 编码后）
	tf.readPos = int64(bomLen)

	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		line := string(lineBytes)
		lineTrimmed := strings.TrimSpace(line)

		// 记录行起始位置
		lineStartPos := tf.readPos
		lineLen := int64(len(lineBytes))
		tf.readPos += lineLen

		// 累积文本用于 TOC 分析（限制大小）
		if allText.Len() < 1024*1024 { // 最多缓存 1MB 用于分析
			allText.WriteString(line)
			allText.WriteString("\n")
		}

		// 跳过空行
		if lineTrimmed == "" {
			continue
		}

		// 检测是否为章节标题
		isChapter, title, rule := tf.detectChapterTitle(lineTrimmed)
		if isChapter {
			// 保存前一个章节
			if currentChapter != nil {
				currentChapter.EndPos = lineStartPos
				book.Chapters = append(book.Chapters, *currentChapter)
			}

			// 创建新章节
			currentChapter = &Chapter{
				Title:     title,
				Index:     len(book.Chapters) + 1,
				StartPos:  lineStartPos,
				EndPos:    0, // 稍后设置
				WordCount: 0,
			}

			// 记录规则命中
			if rule != nil {
				tf.tocAnalyzer.ruleHits[rule.Name]++
			}
		}
	}

	// 扫描结束
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	// 保存最后一个章节
	if currentChapter != nil {
		currentChapter.EndPos = tf.readPos
		book.Chapters = append(book.Chapters, *currentChapter)
	}

	// 保存累积的文本到 book 中（用于后续 GetContent）
	book.rawContent = &allText

	return nil
}

// utf8ReadLine 读取一行并转换为 UTF-8
// 使用 bufio.Scanner 实现
func utf8ReadLine(r io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(r)
	// 设置最大扫描长度（支持大行）
	maxCapacity := 10 * 1024 * 1024 // 10MB
	buf := make([]byte, 0, 1024)
	scanner.Buffer(buf, maxCapacity)

	if scanner.Scan() {
		return scanner.Bytes(), nil
	}
	err := scanner.Err()
	if err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// =============================================================================
// 章节标题检测
// =============================================================================

// detectChapterTitle 检测一行是否为章节标题
// 返回：是否为章节、标题文本、匹配的规则
func (tf *TextFile) detectChapterTitle(line string) (bool, string, *TocRule) {
	if line == "" {
		return false, "", nil
	}

	// 检查长度限制
	if len(line) < tf.options.MinTitleLength {
		return false, "", nil
	}
	if len(line) > tf.options.MaxTitleLength {
		return false, "", nil
	}

	// 使用分析器检测
	rules := tf.options.TocRules
	if rules == nil {
		rules = DefaultTocRules
	}

	for _, rule := range rules {
		matches := rule.Regex.FindStringSubmatch(line)
		if matches != nil {
			// 提取标题
			title := ""
			if len(matches) > 2 && matches[2] != "" {
				title = strings.TrimSpace(matches[2])
			} else if len(matches) > 1 {
				title = strings.TrimSpace(matches[1])
			}

			if title == "" {
				continue
			}

			// 过滤可疑标题
			if tf.options.FilterSuspiciousTitles && isSuspiciousTitle(title) {
				continue
			}

			return true, title, rule
		}
	}

	return false, "", nil
}

// =============================================================================
// TOC 分析
// =============================================================================

// analyzeTOC 分析目录，找出最佳规则并提取章节
func (tf *TextFile) analyzeTOC(book *Book) (*TocRule, []Chapter) {
	// 使用分析器分析累积的文本
	text := book.rawContent.String()
	bestRule, matches := tf.tocAnalyzer.Analyze(text)

	// 过滤匹配结果
	if book.options.FilterSuspiciousTitles {
		matches = FilterMatches(matches)
	}

	// 转换为章节列表
	chapters := make([]Chapter, 0, len(matches))
	for i, m := range matches {
		chapters = append(chapters, Chapter{
			Title:    m.Title,
			Index:    i + 1,
			StartPos: m.Position,
			EndPos:   0,
		})
	}

	return bestRule, chapters
}

// =============================================================================
// 章节过滤和排序
// =============================================================================

// filterAndSortChapters 过滤和排序章节
func (tf *TextFile) filterAndSortChapters(book *Book) {
	if len(book.Chapters) == 0 {
		return
	}

	// 按起始位置排序
	sort.Slice(book.Chapters, func(i, j int) bool {
		return book.Chapters[i].StartPos < book.Chapters[j].StartPos
	})

	// 重新编号
	for i := range book.Chapters {
		book.Chapters[i].Index = i + 1
	}

	// 设置结束位置
	for i := 0; i < len(book.Chapters); i++ {
		if i+1 < len(book.Chapters) {
			book.Chapters[i].EndPos = book.Chapters[i+1].StartPos
		} else {
			book.Chapters[i].EndPos = book.Size
		}
	}
}

// =============================================================================
// 字数计算
// =============================================================================

// calculateWordCount 计算章节字数
func (tf *TextFile) calculateWordCount(book *Book) {
	totalWords := 0

	for i := range book.Chapters {
		chapter := &book.Chapters[i]
		// 简化：估算字数（实际需要从文件读取内容）
		chapterSize := chapter.EndPos - chapter.StartPos
		// 假设 UTF-8 中文字符平均 3 字节
		if book.Encoding == EncodingUTF8 || book.Encoding == EncodingUTF8BOM {
			chapter.WordCount = int(chapterSize) / 3
		} else {
			chapter.WordCount = int(chapterSize) / 2
		}
		totalWords += chapter.WordCount
	}

	book.TotalWords = totalWords
}

// =============================================================================
// 长章节拆分
// =============================================================================

// splitLongChapters 拆分过长的章节
func (tf *TextFile) splitLongChapters(book *Book) {
	maxLen := book.options.MaxChapterLength
	if maxLen <= 0 {
		return
	}

	// 从后往前处理，避免索引变化
	for i := len(book.Chapters) - 1; i >= 0; i-- {
		chapter := book.Chapters[i]

		if chapter.WordCount <= maxLen {
			continue
		}

		// 需要拆分
		splits := tf.splitChapter(book, i, maxLen)
		if splits == nil {
			continue
		}

		// 替换原章节
		book.Chapters = append(book.Chapters[:i], append(splits, book.Chapters[i+1:]...)...)
	}

	// 重新编号
	for i := range book.Chapters {
		book.Chapters[i].Index = i + 1
	}

	book.TotalChapters = len(book.Chapters)
}

// splitChapter 拆分单个章节
func (tf *TextFile) splitChapter(book *Book, chapterIdx int, maxLen int) []Chapter {
	chapter := book.Chapters[chapterIdx]

	// 读取章节内容
	content, err := tf.GetContent(chapter.StartPos, chapter.EndPos)
	if err != nil || content == "" {
		return nil
	}

	// 按段落拆分
	paragraphs := strings.Split(content, "\n\n")
	if len(paragraphs) <= 1 {
		paragraphs = strings.Split(content, "\n")
	}

	if len(paragraphs) <= 1 {
		// 无法拆分，返回原章节
		return nil
	}

	// 计算总段落数用于标题
	totalSplits := (len(content) + maxLen - 1) / maxLen

	// 分组
	var splits []Chapter
	currentLen := 0
	currentStart := chapter.StartPos

	for i, p := range paragraphs {
		pLen := len(strings.TrimSpace(p))
		if currentLen+pLen > maxLen && currentLen > maxLen/2 {
			// 创建新章节
			splitTitle := fmt.Sprintf("%s (%d/%d)", chapter.Title, len(splits)+1, totalSplits)
			splits = append(splits, Chapter{
				Title:     splitTitle,
				Index:     0, // 稍后重新编号
				StartPos:  currentStart,
				EndPos:    0,
				WordCount: currentLen / 3,
			})
			currentStart = chapter.StartPos + int64(currentLen)
			currentLen = 0
		}
		currentLen += pLen
		_ = i // 避免 unused
	}

	// 最后一个分组
	if currentLen > 0 {
		splitTitle := fmt.Sprintf("%s (%d/%d)", chapter.Title, len(splits)+1, totalSplits)
		splits = append(splits, Chapter{
			Title:     splitTitle,
			Index:     0,
			StartPos:  currentStart,
			EndPos:    chapter.EndPos,
			WordCount: currentLen / 3,
		})
	}

	return splits
}

// =============================================================================
// 书名推断
// =============================================================================

// inferBookName 从文件名推断书名
func (tf *TextFile) inferBookName() string {
	// 去掉路径
	name := tf.path
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		name = name[idx+1:]
	}
	if idx := strings.LastIndex(name, "\\"); idx != -1 {
		name = name[idx+1:]
	}

	// 去掉扩展名
	if ext := strings.ToLower(pathExt(name)); ext == ".txt" {
		name = name[:len(name)-4]
	}

	// 清理常见前缀
	prefixes := []string{"小说", "txt", "全本小说", "完整版", "免费"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			name = name[len(prefix):]
			break
		}
	}

	return strings.TrimSpace(name)
}

// pathExt 获取文件扩展名（简化实现）
func pathExt(path string) string {
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}

// =============================================================================
// 公开接口
// =============================================================================

// GetChapters 获取章节列表
// 这是要求的接口之一
func (tf *TextFile) GetChapters() []Chapter {
	if tf.book == nil {
		return nil
	}
	return tf.book.Chapters
}

// GetContent 获取指定位置的章节内容
// 这是要求的接口之一
// start, end 为字节偏移（UTF-8 编码后）
func (tf *TextFile) GetContent(start, end int64) (string, error) {
	if tf.file == nil {
		// 尝试从缓存获取
		if tf.book != nil && tf.book.rawContent != nil {
			content := tf.book.rawContent.String()
			if int(start) < len(content) {
				endIdx := int(end)
				if endIdx > len(content) {
					endIdx = len(content)
				}
				return content[start:endIdx], nil
			}
		}
		return "", fmt.Errorf("file not open")
	}

	// 从文件读取
	if _, err := tf.file.Seek(start, 0); err != nil {
		return "", fmt.Errorf("seek: %w", err)
	}

	size := end - start
	if size <= 0 {
		return "", nil
	}

	buffer := make([]byte, size)
	n, err := tf.file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read: %w", err)
	}

	// 如果文件编码不是 UTF-8，需要转换
	if tf.book != nil && tf.book.Encoding != EncodingUTF8 && tf.book.Encoding != EncodingUTF8BOM {
		decoder, err := GetDecoder(tf.book.Encoding)
		if err != nil {
			return "", err
		}
		converted, err := io.ReadAll(transform.NewReader(bytes.NewReader(buffer[:n]), decoder.NewDecoder()))
		if err != nil {
			return "", err
		}
		return string(converted), nil
	}

	return string(buffer[:n]), nil
}

// GetBook 获取解析后的书籍对象
func (tf *TextFile) GetBook() *Book {
	return tf.book
}

// GetBestTocRule 获取最佳 TOC 规则
func (tf *TextFile) GetBestTocRule() *TocRule {
	if tf.book != nil {
		return tf.book.bestTocRule
	}
	return nil
}

// GetEncoding 获取检测到的编码
func (tf *TextFile) GetEncoding() EncodingType {
	if tf.book != nil {
		return tf.book.Encoding
	}
	return EncodingUnknown
}

// Close 关闭文件
func (tf *TextFile) Close() error {
	if tf.file != nil {
		return tf.file.Close()
	}
	return nil
}

// =============================================================================
// 包级便捷函数
// =============================================================================

// ParseFile 解析 TXT 文件（便捷函数）
// 等价于 NewTextFile(path).Parse()
func ParseFile(path string) (*Book, error) {
	return Parse(path)
}

// ParseWithEncoding 使用指定编码解析 TXT 文件
func ParseWithEncoding(path string, enc EncodingType) (*Book, error) {
	tf := NewTextFile(path)
	tf.SetEncoding(enc)
	return tf.Parse()
}

// ParseWithOptions 使用选项解析 TXT 文件
func ParseWithOptions(path string, opts *ParseOptions) (*Book, error) {
	tf := NewTextFile(path)
	tf.SetOptions(opts)
	return tf.Parse()
}

// GetChapters 便捷函数：直接获取章节列表
func GetChapters(book *Book) []Chapter {
	if book == nil {
		return nil
	}
	return book.Chapters
}

// GetContent 便捷函数：从书籍对象获取内容
func GetBookContent(book *Book, start, end int64) (string, error) {
	if book == nil {
		return "", fmt.Errorf("book is nil")
	}
	// 简化实现：直接从 rawContent 读取
	if book.rawContent != nil {
		content := book.rawContent.String()
		if int(start) >= len(content) {
			return "", fmt.Errorf("start position out of range")
		}
		endIdx := int(end)
		if endIdx > len(content) {
			endIdx = len(content)
		}
		return content[start:endIdx], nil
	}
	return "", fmt.Errorf("content not available")
}
