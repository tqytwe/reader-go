package webbook

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
)

// =============================================================================
// 书籍详情解析
// =============================================================================

// BookInfo 书籍详细信息
type BookInfo struct {
	// 书名
	Name string `json:"name"`

	// 作者
	Author string `json:"author"`

	// 书籍URL/标识符
	BookURL string `json:"bookUrl"`

	// 封面URL
	CoverURL string `json:"coverUrl"`

	// 简介
	Intro string `json:"intro"`

	// 分类/标签
	Category string `json:"category"`

	// 最新章节标题
	LastChapter string `json:"lastChapter"`

	// 最新章节URL
	LastChapterURL string `json:"lastChapterUrl"`

	// 更新时间
	UpdateTime string `json:"updateTime"`

	// 评分
	Rating float64 `json:"rating"`

	// 字数
	WordCount int64 `json:"wordCount"`

	// 状态（连载/完结）
	Status string `json:"status"`

	// 出版社
	Publisher string `json:"publisher"`

	// ISBN
	ISBN string `json:"isbn"`

	// 标签/关键词
	Tags []string `json:"tags"`

	// 来源书源ID
	SourceID string `json:"sourceId"`

	// 来源名称
	SourceName string `json:"sourceName"`

	// 原始数据
	Raw map[string]any `json:"-"`
}

// =============================================================================
// 书籍详情解析器
// =============================================================================

// BookInfoParser 书籍详情解析器
type BookInfoParser struct {
	// 书名选择器
	nameSelector string

	// 作者选择器
	authorSelector string

	// 封面URL选择器
	coverURLSelector string

	// 简介选择器
	introSelector string

	// 分类选择器
	categorySelector string

	// 最新章节标题选择器
	lastChapterSelector string

	// 最新章节URL选择器
	lastChapterURLSelector string

	// 更新时间选择器
	updateTimeSelector string

	// 评分选择器
	ratingSelector string

	// 字数选择器
	wordCountSelector string

	// 状态选择器
	statusSelector string

	// 出版社选择器
	publisherSelector string

	// ISBN选择器
	isbnSelector string

	// 标签选择器（多个）
	tagSelectors []string

	// 解析模式
	mode RuleMode
}

// NewBookInfoParser 创建书籍详情解析器
func NewBookInfoParser(nameSelector, authorSelector, bookURL string) *BookInfoParser {
	return &BookInfoParser{
		nameSelector:   nameSelector,
		authorSelector: authorSelector,
		mode:           ModeDefault,
	}
}

// SetSelectors 批量设置选择器
func (p *BookInfoParser) SetSelectors(selectors map[string]string) *BookInfoParser {
	for k, v := range selectors {
		switch k {
		case "name":
			p.nameSelector = v
		case "author":
			p.authorSelector = v
		case "coverUrl":
			p.coverURLSelector = v
		case "intro":
			p.introSelector = v
		case "category":
			p.categorySelector = v
		case "lastChapter":
			p.lastChapterSelector = v
		case "lastChapterUrl":
			p.lastChapterURLSelector = v
		case "updateTime":
			p.updateTimeSelector = v
		case "rating":
			p.ratingSelector = v
		case "wordCount":
			p.wordCountSelector = v
		case "status":
			p.statusSelector = v
		case "publisher":
			p.publisherSelector = v
		case "isbn":
			p.isbnSelector = v
		}
	}
	return p
}

// AddTagSelector 添加标签选择器
func (p *BookInfoParser) AddTagSelector(selector string) *BookInfoParser {
	p.tagSelectors = append(p.tagSelectors, selector)
	return p
}

// ParseHTML 解析HTML书籍详情页面
func (p *BookInfoParser) ParseHTML(html, baseURL string, sourceID, sourceName string) (*BookInfo, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	info := &BookInfo{
		SourceID:   sourceID,
		SourceName: sourceName,
		Raw:        make(map[string]any),
	}

	// 解析各字段
	if p.nameSelector != "" {
		info.Name = extractText(doc.Selection, p.nameSelector)
	} else {
		// 默认尝试 h1, .book-name, [itemprop~="name"]
		info.Name = extractText(doc.Selection, "h1, .book-name, [itemprop~='name'], .title")
	}

	if p.authorSelector != "" {
		info.Author = extractText(doc.Selection, p.authorSelector)
	} else {
		info.Author = extractText(doc.Selection, ".author, [itemprop~='author'], .book-author")
	}

	if p.coverURLSelector != "" {
		info.CoverURL = resolveURL(extractAttr(doc.Selection, p.coverURLSelector, "src"), baseURL)
	} else {
		info.CoverURL = resolveURL(extractAttr(doc.Selection, ".book-cover img, .cover img, [itemprop~='image']", "src"), baseURL)
	}

	if p.introSelector != "" {
		info.Intro = extractText(doc.Selection, p.introSelector)
	} else {
		info.Intro = extractText(doc.Selection, ".intro, .book-intro, [itemprop~='description'], .description")
	}

	if p.categorySelector != "" {
		info.Category = extractText(doc.Selection, p.categorySelector)
	} else {
		info.Category = extractText(doc.Selection, ".category, .book-category, [itemprop~='genre']")
	}

	if p.lastChapterSelector != "" {
		info.LastChapter = extractText(doc.Selection, p.lastChapterSelector)
	} else {
		info.LastChapter = extractText(doc.Selection, ".last-chapter, .latest-chapter")
	}

	if p.lastChapterURLSelector != "" {
		info.LastChapterURL = resolveURL(extractAttr(doc.Selection, p.lastChapterURLSelector, "href"), baseURL)
	} else {
		info.LastChapterURL = resolveURL(extractAttr(doc.Selection, ".last-chapter a, .latest-chapter a", "href"), baseURL)
	}

	if p.updateTimeSelector != "" {
		info.UpdateTime = extractText(doc.Selection, p.updateTimeSelector)
	} else {
		info.UpdateTime = extractText(doc.Selection, ".update-time, .book-update-time, time")
	}

	if p.ratingSelector != "" {
		if r, err := parseFloat(extractText(doc.Selection, p.ratingSelector)); err == nil {
			info.Rating = r
		}
	} else {
		if r, err := parseFloat(extractText(doc.Selection, ".rating, .book-rating, .score")); err == nil {
			info.Rating = r
		}
	}

	if p.wordCountSelector != "" {
		if wc, err := parseWordCount(extractText(doc.Selection, p.wordCountSelector)); err == nil {
			info.WordCount = wc
		}
	}

	if p.statusSelector != "" {
		info.Status = extractText(doc.Selection, p.statusSelector)
	} else {
		info.Status = extractText(doc.Selection, ".status, .book-status")
	}

	if p.publisherSelector != "" {
		info.Publisher = extractText(doc.Selection, p.publisherSelector)
	}

	if p.isbnSelector != "" {
		info.ISBN = extractText(doc.Selection, p.isbnSelector)
	}

	// 解析标签
	tags := make([]string, 0)
	for _, sel := range p.tagSelectors {
		tags = append(tags, extractText(doc.Selection, sel))
	}
	if len(tags) == 0 {
		// 默认尝试
		doc.Find(".tag, .book-tag, .tags a").Each(func(_ int, s *goquery.Selection) {
			t := strings.TrimSpace(s.Text())
			if t != "" {
				tags = append(tags, t)
			}
		})
	}
	info.Tags = tags

	return info, nil
}

// ParseJSON 解析JSON格式的书籍详情响应
func (p *BookInfoParser) ParseJSON(data string, sourceID, sourceName string) (*BookInfo, error) {
	if !gjson.Valid(data) {
		return nil, nil
	}

	result := gjson.Parse(data)
	info := parseJSONBookInfo(result, sourceID, sourceName)

	return info, nil
}

// parseJSONBookInfo 从JSON解析BookInfo
func parseJSONBookInfo(result gjson.Result, sourceID, sourceName string) *BookInfo {
	info := &BookInfo{
		SourceID:   sourceID,
		SourceName: sourceName,
		Raw:        make(map[string]any),
	}

	// 字段映射（支持多种命名）
	setString := func(dest *string, paths ...string) {
		if *dest != "" {
			return
		}
		for _, path := range paths {
			if v := result.Get(path).String(); v != "" {
				*dest = v
				break
			}
		}
	}

	setFloat := func(dest *float64, paths ...string) {
		if *dest != 0 {
			return
		}
		for _, path := range paths {
			if v := result.Get(path).Float(); v != 0 {
				*dest = v
				break
			}
		}
	}

	setInt64 := func(dest *int64, paths ...string) {
		if *dest != 0 {
			return
		}
		for _, path := range paths {
			if v := result.Get(path).Int(); v != 0 {
				*dest = v
				break
			}
		}
	}

	setString(&info.Name, "name", "title", "bookName", "book.name")
	setString(&info.Author, "author", "authorName", "book.author")
	setString(&info.BookURL, "bookUrl", "url", "book.url")
	setString(&info.CoverURL, "coverUrl", "cover", "book.cover")
	setString(&info.Intro, "intro", "description", "introduction", "book.description")
	setString(&info.Category, "category", "tag", "genre", "book.category")
	setString(&info.LastChapter, "lastChapter", "latestChapter", "lastChapterName")
	setString(&info.LastChapterURL, "lastChapterUrl", "latestChapterUrl", "lastChapter.url")
	setString(&info.UpdateTime, "updateTime", "lastUpdateTime", "updateTime")
	setFloat(&info.Rating, "rating", "score", "book.rating")
	setInt64(&info.WordCount, "wordCount", "size", "book.wordCount")
	setString(&info.Status, "status", "bookStatus", "book.status")
	setString(&info.Publisher, "publisher", "publishInfo")
	setString(&info.ISBN, "isbn", "ISBN")

	// 标签
	tags := make([]string, 0)
	if result.Get("tags").IsArray() {
		result.Get("tags").ForEach(func(_, v gjson.Result) bool {
			if t := v.String(); t != "" {
				tags = append(tags, t)
			}
			return true
		})
	} else if result.Get("tag").IsArray() {
		result.Get("tag").ForEach(func(_, v gjson.Result) bool {
			if t := v.String(); t != "" {
				tags = append(tags, t)
			}
			return true
		})
	} else if t := result.Get("tags").String(); t != "" {
		// 可能是逗号分隔的字符串
		for _, tag := range strings.Split(t, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	info.Tags = tags

	return info
}

// =============================================================================
// 工具函数
// =============================================================================

// extractAttr 从选择器提取属性值
func extractAttr(sel *goquery.Selection, selector, attr string) string {
	if selector == "" {
		return ""
	}
	result := sel.Find(selector).First().AttrOr(attr, "")
	return strings.TrimSpace(result)
}

// parseWordCount 解析字数（支持 "123万字", "123456字" 等格式）
func parseWordCount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	var multiplier int64 = 1
	if strings.Contains(s, "万") {
		multiplier = 10000
		s = strings.ReplaceAll(s, "万", "")
	} else if strings.Contains(s, "亿") {
		multiplier = 100000000
		s = strings.ReplaceAll(s, "亿", "")
	}

	// 移除"字"等后缀
	s = strings.ReplaceAll(s, "字", "")
	s = strings.TrimSpace(s)

	result, err := parseFloat(s)
	if err != nil {
		return 0, err
	}
	return int64(result * float64(multiplier)), nil
}
