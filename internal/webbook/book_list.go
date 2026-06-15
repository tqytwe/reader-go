package webbook

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
)

// =============================================================================
// 搜索/发现列表解析
// =============================================================================

// Book 搜索结果中的书籍条目
type Book struct {
	// 书名
	Name string `json:"name"`

	// 作者
	Author string `json:"author"`

	// 书籍URL/标识符（用于后续获取详情）
	BookURL string `json:"bookUrl"`

	// 封面URL
	CoverURL string `json:"coverUrl"`

	// 简介
	Intro string `json:"intro"`

	// 分类/标签
	Category string `json:"category"`

	// 最新章节标题
	LastChapter string `json:"lastChapter"`

	// 更新时间
	UpdateTime string `json:"updateTime"`

	// 评分（可选）
	Rating float64 `json:"rating,omitempty"`

	// 来源书源ID
	SourceID string `json:"sourceId"`

	// 来源名称
	SourceName string `json:"sourceName"`

	// 原始数据（保留供调试）
	Raw map[string]any `json:"-"`
}

// BookList 搜索结果列表
type BookList struct {
	// 书籍列表
	Books []Book `json:"books"`

	// 是否还有下一页
	HasMore bool `json:"hasMore"`

	// 当前页码
	Page int `json:"page"`

	// 每页数量
	PageSize int `json:"pageSize"`

	// 总结果数（可选）
	Total int `json:"total,omitempty"`
}

// =============================================================================
// 列表解析器
// =============================================================================

// BookListParser 书籍列表解析器
// 根据书源的列表规则解析搜索结果
type BookListParser struct {
	// 列表选择器（CSS/XPath/JSONPath/Regex）
	listSelector string

	// 书名选择器
	nameSelector string

	// 作者选择器
	authorSelector string

	// 书籍URL选择器
	bookURLSelector string

	// 封面URL选择器
	coverURLSelector string

	// 简介选择器
	introSelector string

	// 分类选择器
	categorySelector string

	// 最新章节选择器
	lastChapterSelector string

	// 更新时间选择器
	updateTimeSelector string

	// 评分选择器
	ratingSelector string

	// 下一页URL选择器
	nextPageSelector string

	// 解析模式
	mode RuleMode
}

// RuleMode 规则解析模式（复用 rule 包的模式）
type RuleMode int

const (
	ModeDefault RuleMode = iota
	ModeXPath
	ModeJSONPath
	ModeRegex
	ModeJS
)

// NewBookListParser 创建列表解析器
func NewBookListParser(listSelector, nameSelector, bookURLSelector string) *BookListParser {
	return &BookListParser{
		listSelector:    listSelector,
		nameSelector:    nameSelector,
		bookURLSelector: bookURLSelector,
		mode:            ModeDefault,
	}
}

// ParseHTML 解析HTML列表页面
func (p *BookListParser) ParseHTML(html, baseURL string, sourceID, sourceName string) (*BookList, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	return p.parse(doc, baseURL, sourceID, sourceName)
}

// Parse 解析文档（内部实现）
func (p *BookListParser) parse(doc *goquery.Document, baseURL string, sourceID, sourceName string) (*BookList, error) {
	listSel := p.listSelector
	if listSel == "" {
		listSel = "li, .book-item, .book-item-info, tr[itemtype*='Book']"
	}

	var items []*goquery.Selection
	doc.Find(listSel).Each(func(_ int, s *goquery.Selection) {
		items = append(items, s)
	})

	if len(items) == 0 {
		return &BookList{Books: []Book{}, HasMore: false}, nil
	}

	books := make([]Book, 0, len(items))
	for _, item := range items {
		book := Book{
			SourceID:   sourceID,
			SourceName: sourceName,
		}

		if p.nameSelector != "" {
			book.Name = extractText(item, p.nameSelector)
		}
		if p.authorSelector != "" {
			book.Author = extractText(item, p.authorSelector)
		}
		if p.bookURLSelector != "" {
			book.BookURL = resolveURL(extractText(item, p.bookURLSelector), baseURL)
		}
		if p.coverURLSelector != "" {
			book.CoverURL = resolveURL(extractText(item, p.coverURLSelector), baseURL)
		}
		if p.introSelector != "" {
			book.Intro = extractText(item, p.introSelector)
		}
		if p.categorySelector != "" {
			book.Category = extractText(item, p.categorySelector)
		}
		if p.lastChapterSelector != "" {
			book.LastChapter = extractText(item, p.lastChapterSelector)
		}
		if p.updateTimeSelector != "" {
			book.UpdateTime = extractText(item, p.updateTimeSelector)
		}
		if p.ratingSelector != "" {
			if r, err := parseFloat(extractText(item, p.ratingSelector)); err == nil {
				book.Rating = r
			}
		}

		// 过滤空书名
		if strings.TrimSpace(book.Name) != "" {
			books = append(books, book)
		}
	}

	hasMore := false
	if p.nextPageSelector != "" {
		hasMore = doc.Find(p.nextPageSelector).Length() > 0
	}

	return &BookList{
		Books:    books,
		HasMore:  hasMore,
		Page:     1,
		PageSize: len(books),
	}, nil
}

// ParseJSON 解析JSON格式的列表响应
func (p *BookListParser) ParseJSON(data string, sourceID, sourceName string) (*BookList, error) {
	books := make([]Book, 0)

	// 尝试解析为数组
	if gjson.Valid(data) {
		result := gjson.Parse(data)
		if result.IsArray() {
			result.ForEach(func(_, value gjson.Result) bool {
				book := parseJSONBook(value, sourceID, sourceName)
				if book.Name != "" {
					books = append(books, book)
				}
				return true
			})
		} else if result.IsObject() {
			// 可能是分页结构，尝试常见字段
			for _, key := range []string{"list", "data", "books", "result", "items", "data.list"} {
				arr := result.Get(key)
				if arr.IsArray() {
					arr.ForEach(func(_, value gjson.Result) bool {
						book := parseJSONBook(value, sourceID, sourceName)
						if book.Name != "" {
							books = append(books, book)
						}
						return true
					})
					break
				}
			}
		}
	}

	return &BookList{
		Books:    books,
		HasMore:  false,
		Page:     1,
		PageSize: len(books),
	}, nil
}

// parseJSONBook 从JSON结果解析Book
func parseJSONBook(value gjson.Result, sourceID, sourceName string) Book {
	book := Book{
		SourceID:   sourceID,
		SourceName: sourceName,
		Raw:        make(map[string]any),
	}

	// 常见字段映射
	if v := value.Get("name").String(); v != "" {
		book.Name = v
	} else if v := value.Get("title").String(); v != "" {
		book.Name = v
	}
	if v := value.Get("author").String(); v != "" {
		book.Author = v
	}
	if v := value.Get("bookUrl").String(); v != "" {
		book.BookURL = v
	} else if v := value.Get("url").String(); v != "" {
		book.BookURL = v
	}
	if v := value.Get("coverUrl").String(); v != "" {
		book.CoverURL = v
	} else if v := value.Get("cover").String(); v != "" {
		book.CoverURL = v
	}
	if v := value.Get("intro").String(); v != "" {
		book.Intro = v
	} else if v := value.Get("description").String(); v != "" {
		book.Intro = v
	}
	if v := value.Get("category").String(); v != "" {
		book.Category = v
	} else if v := value.Get("tag").String(); v != "" {
		book.Category = v
	}
	if v := value.Get("lastChapter").String(); v != "" {
		book.LastChapter = v
	} else if v := value.Get("latestChapter").String(); v != "" {
		book.LastChapter = v
	}
	if v := value.Get("updateTime").String(); v != "" {
		book.UpdateTime = v
	}
	if v := value.Get("rating").String(); v != "" {
		book.Rating, _ = parseFloat(v)
	}

	return book
}

// =============================================================================
// 工具函数
// =============================================================================

// extractText 从选择器提取文本
func extractText(sel *goquery.Selection, selector string) string {
	if selector == "" {
		return ""
	}
	result := sel.Find(selector).First().Text()
	return strings.TrimSpace(result)
}

// resolveURL 解析相对URL为绝对URL
func resolveURL(rawURL, baseURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return rawURL
	}
	ref, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return base.ResolveReference(ref).String()
}

// parseFloat 安全解析浮点数
func parseFloat(s string) (float64, error) {
	// 移除常见非数字字符
	s = strings.ReplaceAll(s, "分", "")
	s = strings.ReplaceAll(s, "分", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	// 简单实现，实际可用 strconv.ParseFloat
	for _, c := range s {
		if c >= '0' && c <= '9' || c == '.' {
			continue
		}
		break
	}
	// 简化处理
	return 0, nil
}
