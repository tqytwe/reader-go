package webbook

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
)

// =============================================================================
// 目录解析
// =============================================================================

// BookChapter 章节信息
type BookChapter struct {
	// 章节标题
	Title string `json:"title"`

	// 章节URL/标识符
	URL string `json:"url"`

	// 章节序号（从1开始）
	Index int `json:"index"`

	// 更新时间（可选）
	UpdateTime string `json:"updateTime,omitempty"`

	// 是否VIP/付费（可选）
	IsVip bool `json:"isVip,omitempty"`

	// 是否免费试读（可选）
	IsFree bool `json:"isFree,omitempty"`
}

// BookChapterList 章节目录列表
type BookChapterList struct {
	// 书籍名称
	BookName string `json:"bookName"`

	// 书籍URL
	BookURL string `json:"bookUrl"`

	// 章节列表
	Chapters []BookChapter `json:"chapters"`

	// 来源书源ID
	SourceID string `json:"sourceId"`

	// 来源名称
	SourceName string `json:"sourceName"`

	// 是否有倒序（目录从后往前）
	Reverse bool `json:"reverse,omitempty"`

	// 总章节数
	Total int `json:"total"`
}

// =============================================================================
// 目录解析器
// =============================================================================

// ChapterListParser 章节目录解析器
type ChapterListParser struct {
	// 章节列表选择器
	listSelector string

	// 章节标题选择器
	titleSelector string

	// 章节URL选择器
	urlSelector string

	// 章节更新时间选择器
	updateTimeSelector string

	// 是否VIP标识选择器
	vipSelector string

	// 是否免费标识选择器
	freeSelector string

	// 是否倒序
	Reverse bool

	// 解析模式
	mode RuleMode
}

// NewChapterListParser 创建目录解析器
func NewChapterListParser(listSelector, titleSelector, urlSelector string) *ChapterListParser {
	return &ChapterListParser{
		listSelector:  listSelector,
		titleSelector: titleSelector,
		urlSelector:   urlSelector,
		mode:          ModeDefault,
	}
}

// SetSelectors 批量设置选择器
func (p *ChapterListParser) SetSelectors(selectors map[string]string) *ChapterListParser {
	for k, v := range selectors {
		switch k {
		case "list":
			p.listSelector = v
		case "title":
			p.titleSelector = v
		case "url":
			p.urlSelector = v
		case "updateTime":
			p.updateTimeSelector = v
		case "vip":
			p.vipSelector = v
		case "free":
			p.freeSelector = v
		case "reverse":
			p.Reverse = v == "true" || v == "1"
		}
	}
	return p
}

// ParseHTML 解析HTML目录页面
func (p *ChapterListParser) ParseHTML(html, baseURL string, bookName, bookURL string, sourceID, sourceName string) (*BookChapterList, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	listSel := p.listSelector
	if listSel == "" {
		// 默认选择器
		listSel = "li, .chapter-item, .catalog-item, .chapter, tr[itemtype*='Chapter'], .chapter-list a, .directory a"
	}

	titleSel := p.titleSelector
	if titleSel == "" {
		titleSel = "a, .chapter-title, .title, [itemprop~='name']"
	}

	urlSel := p.urlSelector
	if urlSel == "" {
		urlSel = "a"
	}

	var items []*goquery.Selection
	doc.Find(listSel).Each(func(_ int, s *goquery.Selection) {
		items = append(items, s)
	})

	if len(items) == 0 {
		return &BookChapterList{
			BookName:  bookName,
			BookURL:   bookURL,
			Chapters:  []BookChapter{},
			SourceID:  sourceID,
			SourceName: sourceName,
			Total:     0,
		}, nil
	}

	chapters := make([]BookChapter, 0, len(items))
	index := 1

	for _, item := range items {
		chapter := BookChapter{
			Index: index,
		}

		// 提取标题
		if p.titleSelector != "" {
			chapter.Title = extractText(item, p.titleSelector)
		} else {
			// 从链接或自身提取
			linkSel := item.Find(titleSel).First()
			if linkSel.Length() > 0 {
				chapter.Title = strings.TrimSpace(linkSel.Text())
			} else {
				chapter.Title = strings.TrimSpace(item.Text())
			}
		}

		// 提取URL
		if p.urlSelector != "" {
			chapter.URL = resolveURL(extractAttr(item, p.urlSelector, "href"), baseURL)
		} else {
			chapter.URL = resolveURL(extractAttr(item, "a", "href"), baseURL)
			if chapter.URL == "" {
				// 尝试从自身属性获取
				href, exists := item.Attr("href")
				if exists {
					chapter.URL = resolveURL(href, baseURL)
				}
			}
		}

		// 提取更新时间
		if p.updateTimeSelector != "" {
			chapter.UpdateTime = extractText(item, p.updateTimeSelector)
		}

		// 提取VIP标识
		if p.vipSelector != "" {
			chapter.IsVip = item.Find(p.vipSelector).Length() > 0
		}

		// 提取免费标识
		if p.freeSelector != "" {
			chapter.IsFree = item.Find(p.freeSelector).Length() > 0
		}

		// 过滤空标题
		if strings.TrimSpace(chapter.Title) != "" {
			chapters = append(chapters, chapter)
			index++
		}
	}

	// 处理倒序
	if p.Reverse {
		for i, j := 0, len(chapters)-1; i < j; i, j = i+1, j-1 {
			chapters[i], chapters[j] = chapters[j], chapters[i]
			// 重新编号
			chapters[i].Index = i + 1
			chapters[j].Index = j + 1
		}
	}

	return &BookChapterList{
		BookName:   bookName,
		BookURL:    bookURL,
		Chapters:   chapters,
		SourceID:   sourceID,
		SourceName: sourceName,
		Reverse:    p.Reverse,
		Total:      len(chapters),
	}, nil
}

// ParseJSON 解析JSON格式的目录响应
func (p *ChapterListParser) ParseJSON(data, bookName, bookURL string, sourceID, sourceName string) (*BookChapterList, error) {
	if !gjson.Valid(data) {
		return nil, nil
	}

	result := gjson.Parse(data)
	chapters := make([]BookChapter, 0)

	// 尝试解析为数组
	if result.IsArray() {
		result.ForEach(func(i, value gjson.Result) bool {
			chapter := parseJSONChapter(value, int(i.Int())+1)
			if chapter.Title != "" {
				chapters = append(chapters, chapter)
			}
			return true
		})
	} else if result.IsObject() {
		// 可能是分页结构
		for _, key := range []string{"list", "data", "chapters", "toc", "volume.chapters"} {
			arr := result.Get(key)
			if arr.IsArray() {
				arr.ForEach(func(i, value gjson.Result) bool {
					chapter := parseJSONChapter(value, int(i.Int())+1)
					if chapter.Title != "" {
						chapters = append(chapters, chapter)
					}
					return true
				})
				break
			}
		}
	}

	// 处理倒序
	if p.Reverse {
		for i, j := 0, len(chapters)-1; i < j; i, j = i+1, j-1 {
			chapters[i], chapters[j] = chapters[j], chapters[i]
			chapters[i].Index = i + 1
			chapters[j].Index = j + 1
		}
	}

	return &BookChapterList{
		BookName:   bookName,
		BookURL:    bookURL,
		Chapters:   chapters,
		SourceID:   sourceID,
		SourceName: sourceName,
		Reverse:    p.Reverse,
		Total:      len(chapters),
	}, nil
}

// parseJSONChapter 从JSON解析章节
func parseJSONChapter(value gjson.Result, index int) BookChapter {
	chapter := BookChapter{
		Index: index,
	}

	// 字段映射
	if v := value.Get("title").String(); v != "" {
		chapter.Title = v
	} else if v := value.Get("name").String(); v != "" {
		chapter.Title = v
	} else if v := value.Get("chapterTitle").String(); v != "" {
		chapter.Title = v
	}

	if v := value.Get("url").String(); v != "" {
		chapter.URL = v
	} else if v := value.Get("chapterUrl").String(); v != "" {
		chapter.URL = v
	} else if v := value.Get("link").String(); v != "" {
		chapter.URL = v
	}

	if v := value.Get("updateTime").String(); v != "" {
		chapter.UpdateTime = v
	}

	// VIP/免费标识
	if v := value.Get("isVip").String(); v == "true" || v == "1" {
		chapter.IsVip = true
	}
	if v := value.Get("isFree").String(); v == "true" || v == "1" {
		chapter.IsFree = true
	}
	if v := value.Get("vip").String(); v == "true" || v == "1" {
		chapter.IsVip = true
	}
	if v := value.Get("free").String(); v == "true" || v == "1" {
		chapter.IsFree = true
	}

	return chapter
}
