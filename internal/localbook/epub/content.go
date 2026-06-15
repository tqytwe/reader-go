package epub

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// =============================================================================
// ChapterContent 章节内容（含图片嵌入）
// =============================================================================

// ChapterContent 包含清洗后的章节内容和嵌入的图片
type ChapterContent struct {
	// 章节标题
	Title string `json:"title"`

	// 章节索引
	Index int `json:"index"`

	// 清洗后的正文内容（HTML 或纯文本）
	Content string `json:"content"`

	// 纯文本内容（无 HTML 标签）
	PlainText string `json:"plainText"`

	// 字数（中文字符数）
	WordCount int `json:"wordCount"`

	// 嵌入的图片（base64 编码）
	Images map[string]string `json:"images,omitempty"`

	// 图片引用顺序
	ImageOrder []string `json:"imageOrder,omitempty"`

	// 原始 HTML（可选）
	RawHTML string `json:"rawHtml,omitempty"`

	// 章节路径
	Path string `json:"path"`
}

// =============================================================================
// GetChapterContent 获取章节完整内容（含图片嵌入）
// =============================================================================

// GetChapterContent 获取指定章节的完整内容，包括图片嵌入为 base64
func (b *Book) GetChapterContent(chapterIndex int, zrc *ZipReadCloser) (*ChapterContent, error) {
	if chapterIndex < 0 || chapterIndex >= len(b.Chapters) {
		return nil, fmt.Errorf("chapter index %d out of range [0, %d]", chapterIndex, len(b.Chapters)-1)
	}

	ch := b.Chapters[chapterIndex]
	if ch == nil || ch.Path == "" {
		return nil, fmt.Errorf("chapter %d has no path", chapterIndex)
	}

	// 读取章节文件
	rc, err := zrc.openFile(ch.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open chapter %s: %w", ch.Path, err)
	}
	defer rc.Close()

	rawContent, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read chapter %s: %w", ch.Path, err)
	}

	return ProcessChapter(ch, rawContent, b, zrc)
}

// ProcessChapter 处理章节内容（清洗 HTML + 嵌入图片）
func ProcessChapter(ch *Chapter, rawContent []byte, book *Book, zrc *ZipReadCloser) (*ChapterContent, error) {
	content := &ChapterContent{
		Title:       ch.Title,
		Index:       ch.Index,
		Path:        ch.Path,
		Images:      make(map[string]string),
		ImageOrder:  make([]string, 0),
		RawHTML:     string(rawContent),
	}

	// 解析 HTML
	doc, err := html.Parse(strings.NewReader(string(rawContent)))
	if err != nil {
		// 回退：尝试提取纯文本
		content.PlainText = extractTextFromHTML(string(rawContent))
		content.WordCount = countChineseChars(content.PlainText)
		return content, nil
	}

	// 尝试从 HTML <title> 标签提取更准确的标题
	if title := extractTitleFromHTML(doc); title != "" {
		content.Title = title
	}

	// 步骤 1: 移除 script/style/noscript 标签
	removeUnwantedTags(doc)

	// 步骤 2: 处理图片，嵌入为 base64
	processImages(doc, content, book, zrc)

	// 步骤 3: 提取正文内容
	content.Content = renderHTML(doc)

	// 步骤 4: 提取纯文本
	content.PlainText = extractTextFromNode(doc)

	// 步骤 5: 计算字数
	content.WordCount = countChineseChars(content.PlainText)

	// 清理内容（移除多余空白）
	content.Content = cleanContent(content.Content)
	content.PlainText = cleanPlainText(content.PlainText)

	return content, nil
}

// extractTitleFromHTML 从 HTML 文档的 <title> 标签提取标题
func extractTitleFromHTML(doc *html.Node) string {
	var title string
	var findTitle func(*html.Node)
	findTitle = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "title" {
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				title = strings.TrimSpace(n.FirstChild.Data)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTitle(c)
			if title != "" {
				return
			}
		}
	}
	findTitle(doc)
	return title
}

// =============================================================================
// HTML 清洗
// =============================================================================

// removeUnwantedTags 移除不需要的标签（script, style, noscript, meta, link, head, script）
func removeUnwantedTags(n *html.Node) {
	unwantedTags := map[string]bool{
		"script":   true,
		"style":    true,
		"noscript": true,
		"meta":     true,
		"link":     true,
		"head":     true,
		"template": true,
		"iframe":   true,
		"object":   true,
		"embed":    true,
		"applet":   true,
		"form":     true,
		"input":    true,
		"button":   true,
		"select":   true,
		"textarea": true,
	}

	var remove func(*html.Node)
	remove = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			if unwantedTags[tag] {
				if parent := n.Parent; parent != nil {
					parent.RemoveChild(n)
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			remove(c)
			c = next
		}
	}

	remove(n)
}

// cleanContent 清理 HTML 内容中的多余空白
func cleanContent(htmlStr string) string {
	// 移除多余空白行
	lines := strings.Split(htmlStr, "\n")
	cleaned := make([]string, 0, len(lines))
	emptyCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			emptyCount++
			if emptyCount <= 2 {
				cleaned = append(cleaned, "")
			}
		} else {
			emptyCount = 0
			cleaned = append(cleaned, trimmed)
		}
	}

	// 移除开头和结尾的空行
	for len(cleaned) > 0 && cleaned[0] == "" {
		cleaned = cleaned[1:]
	}
	for len(cleaned) > 0 && cleaned[len(cleaned)-1] == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}

	return strings.Join(cleaned, "\n")
}

// cleanPlainText 清理纯文本
func cleanPlainText(text string) string {
	// 移除多余空白
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 移除多余空白行
	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	emptyCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			emptyCount++
			if emptyCount <= 2 {
				cleaned = append(cleaned, "")
			}
		} else {
			emptyCount = 0
			// 移除行内多余空格
			trimmed = regexp.MustCompile(`\s+`).ReplaceAllString(trimmed, " ")
			cleaned = append(cleaned, trimmed)
		}
	}

	// 移除开头和结尾的空行
	for len(cleaned) > 0 && cleaned[0] == "" {
		cleaned = cleaned[1:]
	}
	for len(cleaned) > 0 && cleaned[len(cleaned)-1] == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}

	return strings.Join(cleaned, "\n")
}

// =============================================================================
// 图片处理
// =============================================================================

// processImages 处理章节中的图片，嵌入为 base64
func processImages(n *html.Node, content *ChapterContent, book *Book, zrc *ZipReadCloser) {
	imageTags := map[string]bool{
		"img": true,
	}

	var process func(*html.Node)
	process = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			if imageTags[tag] {
				// 查找 src 或 data-src 属性
				var src string
				for _, attr := range n.Attr {
					if attr.Key == "src" || attr.Key == "data-src" || attr.Key == "data-lazy-src" {
						src = attr.Val
						break
					}
				}

				if src != "" {
					// 如果是 data URI，直接提取
					if strings.HasPrefix(src, "data:") {
						handleDataURI(src, n, content)
					} else {
						// 解析相对路径
						imgPath := resolveRelativeURL(book.opfDir, src)

						// 读取图片
						imgData, err := readImage(zrc, imgPath)
						if err != nil {
							// 图片不存在，移除 img 标签
							if parent := n.Parent; parent != nil {
								parent.RemoveChild(n)
							}
							return
						}

						// 生成 base64
						mime := guessMIMEByData(imgData)
						base64Data := base64.StdEncoding.EncodeToString(imgData)
						dataURI := fmt.Sprintf("data:%s;base64,%s", mime, base64Data)

						// 更新 src
						for i := range n.Attr {
							if n.Attr[i].Key == "src" {
								n.Attr[i].Val = dataURI
								break
							}
						}

						// 记录图片
						content.Images[src] = dataURI
						content.ImageOrder = append(content.ImageOrder, src)
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			process(c)
		}
	}

	process(n)
}

// handleDataURI 处理 data URI 图片
func handleDataURI(dataURI string, node *html.Node, content *ChapterContent) {
	// 提取 base64 部分
	if idx := strings.Index(dataURI, ","); idx > 0 {
		header := dataURI[:idx]
		body := dataURI[idx+1:]

		// 记录
		content.Images[header] = dataURI
		content.ImageOrder = append(content.ImageOrder, header)

		// 保留原 data URI（可能已经嵌入）
		_ = body // suppress unused variable
	}
}

// readImage 读取图片文件
func readImage(zrc *ZipReadCloser, imgPath string) ([]byte, error) {
	rc, err := zrc.openFile(imgPath)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// =============================================================================
// HTML 渲染
// =============================================================================

// renderHTML 将 HTML 节点渲染为字符串
func renderHTML(n *html.Node) string {
	var buf bytes.Buffer
	renderNode(&buf, n)
	return buf.String()
}

// renderNode 递归渲染节点
func renderNode(w *bytes.Buffer, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		w.WriteString(n.Data)

	case html.ElementNode:
		tag := strings.ToLower(n.Data)

		// 跳过已移除的标签
		unwantedTags := map[string]bool{
			"script": true, "style": true, "noscript": true,
			"meta": true, "link": true, "head": true,
		}
		if unwantedTags[tag] {
			return
		}

		// 块级元素
		blockTags := map[string]bool{
			"p": true, "div": true, "br": true, "hr": true,
			"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
			"ul": true, "ol": true, "li": true, "table": true, "tr": true, "td": true, "th": true,
			"blockquote": true, "pre": true, "address": true,
			"article": true, "aside": true, "footer": true, "header": true, "nav": true, "section": true,
			"figure": true, "figcaption": true, "main": true,
			"details": true, "summary": true, "dialog": true,
		}

		// 块级元素前加换行
		if blockTags[tag] && w.Len() > 0 && w.Bytes()[w.Len()-1] != '\n' {
			w.WriteByte('\n')
		}

		// 渲染标签
		if tag == "br" {
			w.WriteByte('\n')
		} else if tag == "hr" {
			w.WriteString("\n---\n")
		} else if tag == "img" {
			// 渲染图片占位符（已嵌入 base64）
			var alt string
			var src string
			for _, attr := range n.Attr {
				if attr.Key == "alt" {
					alt = attr.Val
				} else if attr.Key == "src" {
					src = attr.Val
				}
			}
			if alt != "" {
				w.WriteString(fmt.Sprintf("\n[%s]\n", alt))
			} else if src != "" && len(src) > 50 {
				// data URI，显示简短提示
				w.WriteString("\n[图片]\n")
			}
		} else {
			// 普通标签
			w.WriteByte('<')
			w.WriteString(tag)
			for _, attr := range n.Attr {
				w.WriteByte(' ')
				w.WriteString(attr.Key)
				w.WriteString(`="`)
				escapeAttr(w, attr.Val)
				w.WriteByte('"')
			}
			w.WriteString(">")
		}

		// 渲染子节点
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderNode(w, c)
		}

		// 闭合标签
		if tag != "br" && tag != "hr" && tag != "img" && !isVoidElement(tag) {
			w.WriteString("</")
			w.WriteString(tag)
			w.WriteByte('>')
		}

		// 块级元素后加换行
		if blockTags[tag] {
			w.WriteByte('\n')
		}
	}
}

// isVoidElement 判断是否为自闭合元素
func isVoidElement(tag string) bool {
	voidElements := map[string]bool{
		"area": true, "base": true, "br": true, "col": true, "embed": true,
		"hr": true, "img": true, "input": true, "link": true, "meta": true,
		"param": true, "source": true, "track": true, "wbr": true,
	}
	return voidElements[tag]
}

// escapeAttr 转义属性值
func escapeAttr(w *bytes.Buffer, s string) {
	for _, c := range s {
		switch c {
		case '&':
			w.WriteString("&amp;")
		case '<':
			w.WriteString("&lt;")
		case '>':
			w.WriteString("&gt;")
		case '"':
			w.WriteString("&quot;")
		default:
			w.WriteRune(c)
		}
	}
}

// =============================================================================
// 文本提取
// =============================================================================

// extractTextFromNode 从节点提取纯文本
func extractTextFromNode(n *html.Node) string {
	var text strings.Builder
	var extract func(*html.Node)

	extract = func(n *html.Node) {
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				// 块级元素之间添加空格
				if text.Len() > 0 && text.String()[text.Len()-1] != ' ' && text.String()[text.Len()-1] != '\n' {
					text.WriteByte(' ')
				}
				text.WriteString(t)
			}
		}

		// 块级元素后添加换行
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			blockTags := map[string]bool{
				"p": true, "div": true, "br": true, "hr": true,
				"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
				"ul": true, "ol": true, "li": true, "table": true, "tr": true,
				"blockquote": true, "pre": true, "address": true,
				"article": true, "aside": true, "footer": true, "header": true, "nav": true, "section": true,
			}

			if blockTags[tag] {
				// 先处理子节点
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					extract(c)
				}
				// 块级元素后加换行
				if text.Len() > 0 && text.String()[text.Len()-1] != '\n' {
					text.WriteByte('\n')
				}
				if tag == "br" {
					text.WriteByte('\n')
				}
				return
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(n)
	return strings.TrimSpace(text.String())
}

// =============================================================================
// 字数统计
// =============================================================================

// countChineseChars 统计字符数（包括中英文、数字、标点，不包括空格）
func countChineseChars(text string) int {
	count := 0
	for _, r := range text {
		// 中文字符范围：一-鿿
		if r >= '一' && r <= '鿿' {
			count++
		} else if r >= '　' && r <= '〿' {
			// CJK 标点符号
			count++
		} else if r >= '＀' && r <= '￯' {
			// 全角字符
			count++
		} else if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			// 英文字母
			count++
		} else if r >= '0' && r <= '9' {
			// 数字
			count++
		} else if r == '.' || r == ',' || r == '!' || r == '?' || r == ';' || r == ':' {
			// 常用标点
			count++
		}
		// 注意：空格和制表符不计入字数
	}
	return count
}

// =============================================================================
// 图片嵌入导出
// =============================================================================

// ExportWithImages 导出章节内容，将图片保存为单独文件
func (b *Book) ExportWithImages(chapterIndex int, zrc *ZipReadCloser, outputDir string) error {
	_, err := b.GetChapterContent(chapterIndex, zrc)
	if err != nil {
		return err
	}

	// TODO: 实现图片导出
	// 1. 创建 outputDir/images 目录
	// 2. 将 base64 图片解码保存
	// 3. 更新 HTML 中的 src 为相对路径

	return fmt.Errorf("ExportWithImages not yet implemented")
}

// GetImageCount 获取书籍中的图片总数
func (b *Book) GetImageCount(zrc *ZipReadCloser) (int, error) {
	total := 0
	for i := range b.Chapters {
		content, err := b.GetChapterContent(i, zrc)
		if err != nil {
			continue
		}
		total += len(content.Images)
	}
	return total, nil
}

// GetCoverAsBase64 获取封面 base64
func (b *Book) GetCoverAsBase64() (string, string, error) {
	if b.CoverData == nil || len(b.CoverData) == 0 {
		return "", "", ErrNoCover
	}
	mime := b.CoverMIME
	if mime == "" {
		mime = guessMIMEByData(b.CoverData)
	}
	return base64.StdEncoding.EncodeToString(b.CoverData), mime, nil
}
