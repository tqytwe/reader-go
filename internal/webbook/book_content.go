package webbook

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
)

// =============================================================================
// 正文解析
// =============================================================================

// BookContent 章节正文内容
type BookContent struct {
	// 章节标题
	ChapterTitle string `json:"chapterTitle"`

	// 章节URL
	ChapterURL string `json:"chapterUrl"`

	// 书籍名称
	BookName string `json:"bookName"`

	// 书籍URL
	BookURL string `json:"bookUrl"`

	// 正文内容（已清洗）
	Content string `json:"content"`

	// 漫画/图文正文图片列表
	Images []string `json:"images,omitempty"`

	// 阅读模式：text / comic
	ReaderMode string `json:"readerMode,omitempty"`

	// 原始HTML内容（可选，用于调试）
	RawHTML string `json:"rawHtml,omitempty"`

	// 来源书源ID
	SourceID string `json:"sourceId"`

	// 来源名称
	SourceName string `json:"sourceName"`

	// 章节序号
	ChapterIndex int `json:"chapterIndex"`

	// 字数
	WordCount int `json:"wordCount"`

	// 上一页URL（可选）
	PrevURL string `json:"prevUrl,omitempty"`

	// 下一页URL（可选）
	NextURL string `json:"nextUrl,omitempty"`
}

// =============================================================================
// 正文解析器
// =============================================================================

// ContentParser 章节正文解析器
type ContentParser struct {
	// 内容选择器
	contentSelector string

	// 章节标题选择器
	titleSelector string

	// 上一页URL选择器
	prevURLSelector string

	// 下一页URL选择器
	nextURLSelector string

	// 需要排除的选择器（用于过滤页眉页脚等）
	excludeSelectors []string

	// 需要保留的标签（白名单）
	keepTags []string

	// 是否保留换行
	KeepLineBreaks bool

	// 解析模式
	mode RuleMode
}

// NewContentParser 创建正文解析器
func NewContentParser(contentSelector string) *ContentParser {
	return &ContentParser{
		contentSelector:  contentSelector,
		KeepLineBreaks:   true,
		keepTags:         []string{"p", "br", "div", "span", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li", "strong", "em", "b", "i", "img"},
		mode:             ModeDefault,
	}
}

// SetSelectors 批量设置选择器
func (p *ContentParser) SetSelectors(selectors map[string]string) *ContentParser {
	for k, v := range selectors {
		switch k {
		case "content":
			p.contentSelector = v
		case "title":
			p.titleSelector = v
		case "prevUrl":
			p.prevURLSelector = v
		case "nextUrl":
			p.nextURLSelector = v
		}
	}
	return p
}

// AddExcludeSelector 添加需要排除的选择器
func (p *ContentParser) AddExcludeSelector(selector string) *ContentParser {
	p.excludeSelectors = append(p.excludeSelectors, selector)
	return p
}

// AddKeepTag 添加需要保留的标签
func (p *ContentParser) AddKeepTag(tag string) *ContentParser {
	p.keepTags = append(p.keepTags, tag)
	return p
}

// ParseHTML 解析HTML章节正文页面
func (p *ContentParser) ParseHTML(html, baseURL string, chapterTitle, chapterURL, bookName, bookURL string, sourceID, sourceName string, chapterIndex int) (*BookContent, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	content := &BookContent{
		ChapterTitle:  chapterTitle,
		ChapterURL:    chapterURL,
		BookName:      bookName,
		BookURL:       bookURL,
		SourceID:      sourceID,
		SourceName:    sourceName,
		ChapterIndex:  chapterIndex,
	}

	// 解析章节标题（如果选择器指定）
	if p.titleSelector != "" {
		title := extractText(doc.Selection, p.titleSelector)
		if title != "" {
			content.ChapterTitle = title
		}
	}

	// 解析正文内容
	contentSelector := p.contentSelector
	if contentSelector == "" {
		// 默认选择器
		contentSelector = ".content, #content, .article, .chapter-content, .text, .book-content, [itemprop~='articleBody']"
	}

	var contentSel *goquery.Selection
	doc.Find(contentSelector).Each(func(_ int, s *goquery.Selection) {
		if contentSel == nil || s.Length() > contentSel.Length() {
			contentSel = s
		}
	})

	if contentSel == nil || contentSel.Length() == 0 {
		// 回退：使用整个body
		contentSel = doc.Find("body")
	}

	// 排除不需要的元素
	for _, sel := range p.excludeSelectors {
		contentSel.Find(sel).Remove()
	}

	// 保留白名单标签，移除其他
	if len(p.keepTags) > 0 {
		p.cleanTags(contentSel)
	}

	// 提取文本内容
	content.Content = p.extractContent(contentSel)
	content.Images = p.extractImages(contentSel, baseURL)
	if len(content.Images) > 0 {
		content.ReaderMode = "comic"
	} else {
		content.ReaderMode = "text"
	}
	htmlStr, _ := contentSel.Html()
	content.RawHTML = htmlStr

	// 解析上一页/下一页
	if p.prevURLSelector != "" {
		content.PrevURL = resolveURL(extractAttr(doc.Selection, p.prevURLSelector, "href"), baseURL)
	}
	if p.nextURLSelector != "" {
		content.NextURL = resolveURL(extractAttr(doc.Selection, p.nextURLSelector, "href"), baseURL)
	}

	// 计算字数
	content.WordCount = len(strings.ReplaceAll(content.Content, " ", ""))

	return content, nil
}

// ParseJSON 解析JSON格式的正文响应
func (p *ContentParser) ParseJSON(data, chapterTitle, chapterURL, bookName, bookURL string, sourceID, sourceName string, chapterIndex int) (*BookContent, error) {
	if !gjson.Valid(data) {
		return nil, nil
	}

	result := gjson.Parse(data)
	content := &BookContent{
		ChapterTitle:  chapterTitle,
		ChapterURL:    chapterURL,
		BookName:      bookName,
		BookURL:       bookURL,
		SourceID:      sourceID,
		SourceName:    sourceName,
		ChapterIndex:  chapterIndex,
	}

	// 字段映射
	if v := result.Get("content").String(); v != "" {
		content.Content = v
	} else if v := result.Get("body").String(); v != "" {
		content.Content = v
	} else if v := result.Get("text").String(); v != "" {
		content.Content = v
	}

	for _, key := range []string{"images", "imgs", "imageList", "pics"} {
		arr := result.Get(key)
		if !arr.IsArray() {
			continue
		}
		for _, item := range arr.Array() {
			url := strings.TrimSpace(item.String())
			if url != "" {
				content.Images = append(content.Images, url)
			}
		}
		if len(content.Images) > 0 {
			break
		}
	}

	if v := result.Get("title").String(); v != "" {
		content.ChapterTitle = v
	}

	if v := result.Get("prevUrl").String(); v != "" {
		content.PrevURL = v
	} else if v := result.Get("prev").String(); v != "" {
		content.PrevURL = v
	}

	if v := result.Get("nextUrl").String(); v != "" {
		content.NextURL = v
	} else if v := result.Get("next").String(); v != "" {
		content.NextURL = v
	}

	// 计算字数
	content.WordCount = len(strings.ReplaceAll(content.Content, " ", ""))
	if len(content.Images) > 0 {
		content.ReaderMode = "comic"
	} else {
		content.ReaderMode = "text"
	}

	return content, nil
}

// cleanTags 清理非白名单标签
func (p *ContentParser) cleanTags(sel *goquery.Selection) {
	// 递归处理所有子节点
	sel.Find("*").Each(func(_ int, s *goquery.Selection) {
		tagName := s.Get(0).Data
		isKeep := false
		for _, keep := range p.keepTags {
			if strings.EqualFold(tagName, keep) {
				isKeep = true
				break
			}
		}
		if !isKeep {
			// 提取文本，替换原元素
			text := s.Text()
			if p.KeepLineBreaks {
				// 保留段落间的换行
				text = strings.ReplaceAll(text, "\n", " ")
			}
			s.ReplaceWithHtml(text)
		}
	})
}

// extractContent 提取清洗后的文本内容
func (p *ContentParser) extractContent(sel *goquery.Selection) string {
	if sel == nil {
		return ""
	}

	var texts []string

	sel.Contents().Each(func(_ int, s *goquery.Selection) {
		switch s.Get(0).Data {
		case "#text":
			t := strings.TrimSpace(s.Text())
			if t != "" {
				texts = append(texts, t)
			}
		case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6":
			t := strings.TrimSpace(s.Text())
			if t != "" {
				texts = append(texts, t)
			}
			if p.KeepLineBreaks {
				texts = append(texts, "") // 空行分隔
			}
		case "br":
			if p.KeepLineBreaks {
				texts = append(texts, "")
			}
		case "ul", "ol":
			// 列表内容
			s.Find("li").Each(func(_ int, li *goquery.Selection) {
				t := strings.TrimSpace(li.Text())
				if t != "" {
					texts = append(texts, t)
				}
			})
		case "li":
			t := strings.TrimSpace(s.Text())
			if t != "" {
				texts = append(texts, t)
			}
		case "img":
			// 漫画模式下正文可能只有图片，这里不把 alt 文本混入小说正文。
		}
	})

	// 合并文本，清理多余空白
	result := strings.Join(texts, "\n")
	result = cleanText(result)
	if result == "" && sel != nil {
		result = cleanText(strings.TrimSpace(sel.Text()))
	}

	return result
}

func (p *ContentParser) extractImages(sel *goquery.Selection, baseURL string) []string {
	if sel == nil {
		return nil
	}

	seen := make(map[string]struct{})
	images := make([]string, 0)
	appendImage := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		resolved := resolveURL(raw, baseURL)
		if resolved == "" {
			return
		}
		if _, ok := seen[resolved]; ok {
			return
		}
		seen[resolved] = struct{}{}
		images = append(images, resolved)
	}

	sel.Find("img").Each(func(_ int, s *goquery.Selection) {
		for _, attr := range []string{"src", "data-src", "data-original", "data-url", "data-echo", "data-lazy-src"} {
			if v, ok := s.Attr(attr); ok && strings.TrimSpace(v) != "" {
				appendImage(v)
				return
			}
		}
	})

	return images
}

// cleanText 清洗文本（移除多余空白、统一换行等）
func cleanText(text string) string {
	// 移除多余空白行
	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	emptyCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			emptyCount++
			if emptyCount <= 2 { // 最多保留2个连续空行
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
