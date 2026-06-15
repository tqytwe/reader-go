package webbook

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"

	"reader-go/internal/rule"
)

func (wb *WebBook) modeFor(source *BookSource, phase string) rule.RuleMode {
	var s string
	switch phase {
	case "search":
		s = source.SearchMode
	case "info":
		s = source.BookInfoMode
	case "toc":
		s = source.TocMode
	case "content":
		s = source.ContentMode
	case "explore":
		s = source.ExploreMode
	}
	if s == "" {
		return source.Mode
	}
	return rule.ModeFromString(s)
}

func ruleField(rules map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(rules[k]); v != "" {
			return v
		}
		// 大小写不敏感查找
		lowerK := strings.ToLower(k)
		for rk, rv := range rules {
			if strings.ToLower(rk) == lowerK && strings.TrimSpace(rv) != "" {
				return rv
			}
		}
	}
	return ""
}

func isLegadoChainRule(r string) bool {
	r = strings.TrimSpace(r)
	if r == "" {
		return false
	}
	prefixes := []string{"@js:", "@XPath:", "@xpath:", "@Json:", "@json:", "@JSONPath:", "@jsonpath:",
		"@CSS:", "@css:", "@Regex:", "@regex:", "@Default:", "@default:"}
	for _, p := range prefixes {
		if strings.HasPrefix(r, p) {
			return false
		}
	}
	return strings.Contains(r, "@") ||
		strings.HasPrefix(r, "class.") ||
		strings.HasPrefix(r, "tag.") ||
		strings.HasPrefix(r, "id.")
}

func (wb *WebBook) execField(ctx context.Context, mode rule.RuleMode, fieldRule, body, baseURL, itemJSON string) string {
	fieldRule = strings.TrimSpace(fieldRule)
	if fieldRule == "" {
		return ""
	}

	if itemJSON != "" && (mode == rule.ModeJSONPath || isJSONPathRule(fieldRule)) {
		return gjsonPathFirst(itemJSON, fieldRule)
	}

	if isLegadoChainRule(fieldRule) {
		return evalLegadoRuleOnHTML(body, fieldRule)
	}

	if mode == rule.ModeJSONPath || isJSONPathRule(fieldRule) {
		if v, err := wb.executor.ExecuteFirst(ctx, rule.ModeJSONPath, fieldRule, body); err == nil && v != "" {
			return v
		}
	}

	if v, err := wb.executor.ExecuteFirst(ctx, mode, fieldRule, body); err == nil && v != "" {
		return v
	}

	return evalLegadoRuleOnHTML(body, fieldRule)
}

func (wb *WebBook) parseBookInfoWithRules(ctx context.Context, source *BookSource, body, baseURL, sourceID, sourceName string) (*BookInfo, error) {
	rules := parseLegadoFieldRules(source.BookInfoRule)
	mode := wb.modeFor(source, "info")

	if isJSONPathRule(ruleField(rules, "name", "title")) ||
		strings.HasPrefix(strings.TrimSpace(body), "{") ||
		strings.HasPrefix(strings.TrimSpace(body), "[") {
		return wb.parseBookInfoJSON(body, rules, sourceID, sourceName)
	}

	info := &BookInfo{
		SourceID:   sourceID,
		SourceName: sourceName,
		Raw:        make(map[string]any),
	}
	info.Name = wb.execField(ctx, mode, ruleField(rules, "name", "title"), body, baseURL, "")
	info.Author = wb.execField(ctx, mode, ruleField(rules, "author"), body, baseURL, "")
	info.Intro = wb.execField(ctx, mode, ruleField(rules, "intro", "introduction"), body, baseURL, "")
	info.CoverURL = resolveLegadoURL(wb.execField(ctx, mode, ruleField(rules, "coverUrl", "cover"), body, baseURL, ""), baseURL, "")
	info.Category = wb.execField(ctx, mode, ruleField(rules, "category", "kind"), body, baseURL, "")
	info.LastChapter = wb.execField(ctx, mode, ruleField(rules, "lastChapter", "latestChapter"), body, baseURL, "")
	info.LastChapterURL = resolveLegadoURL(wb.execField(ctx, mode, ruleField(rules, "lastChapterUrl", "latestChapterUrl"), body, baseURL, ""), baseURL, "")
	info.UpdateTime = wb.execField(ctx, mode, ruleField(rules, "updateTime"), body, baseURL, "")
	info.Status = wb.execField(ctx, mode, ruleField(rules, "status"), body, baseURL, "")

	if info.Name == "" {
		fallback, err := NewBookInfoParser("", "", "").ParseHTML(body, baseURL, sourceID, sourceName)
		if err != nil {
			return info, nil
		}
		mergeBookInfo(info, fallback)
	}
	return info, nil
}

func mergeBookInfo(dst, src *BookInfo) {
	if dst.Name == "" {
		dst.Name = src.Name
	}
	if dst.Author == "" {
		dst.Author = src.Author
	}
	if dst.CoverURL == "" {
		dst.CoverURL = src.CoverURL
	}
	if dst.Intro == "" {
		dst.Intro = src.Intro
	}
	if dst.Category == "" {
		dst.Category = src.Category
	}
	if dst.LastChapter == "" {
		dst.LastChapter = src.LastChapter
	}
	if dst.LastChapterURL == "" {
		dst.LastChapterURL = src.LastChapterURL
	}
	if dst.UpdateTime == "" {
		dst.UpdateTime = src.UpdateTime
	}
	if dst.Status == "" {
		dst.Status = src.Status
	}
}

func (wb *WebBook) parseBookInfoJSON(body string, rules map[string]string, sourceID, sourceName string) (*BookInfo, error) {
	info := &BookInfo{
		SourceID:   sourceID,
		SourceName: sourceName,
		Raw:        make(map[string]any),
	}
	info.Name = gjsonPathFirst(body, ruleField(rules, "name", "title"))
	info.Author = gjsonPathFirst(body, ruleField(rules, "author"))
	info.Intro = gjsonPathFirst(body, ruleField(rules, "intro", "introduction"))
	info.CoverURL = gjsonPathFirst(body, ruleField(rules, "coverUrl", "cover"))
	info.Category = gjsonPathFirst(body, ruleField(rules, "category", "kind"))
	info.LastChapter = gjsonPathFirst(body, ruleField(rules, "lastChapter", "latestChapter"))
	info.LastChapterURL = gjsonPathFirst(body, ruleField(rules, "lastChapterUrl", "latestChapterUrl"))
	info.UpdateTime = gjsonPathFirst(body, ruleField(rules, "updateTime"))
	info.Status = gjsonPathFirst(body, ruleField(rules, "status"))
	return info, nil
}

func (wb *WebBook) parseChapterListWithRules(ctx context.Context, source *BookSource, body, baseURL, bookName, bookURL, sourceID, sourceName string) (*BookChapterList, error) {
	rules := parseLegadoFieldRules(source.ChapterListRule)
	mode := wb.modeFor(source, "toc")

	listRule := ruleField(rules, "chapterList", "list", "chapterUrl")
	if isJSONPathRule(listRule) ||
		strings.HasPrefix(strings.TrimSpace(body), "{") ||
		strings.HasPrefix(strings.TrimSpace(body), "[") {
		return wb.parseChapterListJSON(body, rules, bookName, bookURL, sourceID, sourceName)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	titleRule := ruleField(rules, "chapterName", "title", "name")
	urlRule := ruleField(rules, "chapterUrl", "url")

	reverse := strings.EqualFold(ruleField(rules, "reverse"), "true") ||
		strings.EqualFold(ruleField(rules, "reverse"), "1")

	if listRule != "" {
		items, err := evalLegadoListItems(doc.Selection, listRule)
		if err == nil && len(items) > 0 {
			chapters := make([]BookChapter, 0, len(items))
			for i, item := range items {
				title := evalLegadoRule(item, titleRule)
				if title == "" && titleRule != "" {
					title = wb.execField(ctx, mode, titleRule, body, baseURL, "")
				}
				chURL := resolveLegadoURL(evalLegadoRule(item, urlRule), baseURL, "")
				if chURL == "" && urlRule != "" {
					chURL = resolveLegadoURL(wb.execField(ctx, mode, urlRule, body, baseURL, ""), baseURL, "")
				}
				if strings.TrimSpace(title) == "" {
					continue
				}
				chapters = append(chapters, BookChapter{
					Title: title,
					URL:   chURL,
					Index: i + 1,
				})
			}
			if reverse {
				reverseChapters(chapters)
			}
			return &BookChapterList{
				BookName:   bookName,
				BookURL:    bookURL,
				Chapters:   chapters,
				SourceID:   sourceID,
				SourceName: sourceName,
				Reverse:    reverse,
				Total:      len(chapters),
			}, nil
		}
	}

	parser := NewChapterListParser("", "", "").SetSelectors(map[string]string{
		"list":    ruleField(rules, "chapterList", "list"),
		"title":   ruleField(rules, "chapterName", "title", "name"),
		"url":     ruleField(rules, "chapterUrl", "url"),
		"reverse": ruleField(rules, "reverse"),
	})
	return parser.ParseHTML(body, baseURL, bookName, bookURL, sourceID, sourceName)
}

func reverseChapters(chapters []BookChapter) {
	for i, j := 0, len(chapters)-1; i < j; i, j = i+1, j-1 {
		chapters[i], chapters[j] = chapters[j], chapters[i]
		chapters[i].Index = i + 1
		chapters[j].Index = j + 1
	}
}

func (wb *WebBook) parseChapterListJSON(body string, rules map[string]string, bookName, bookURL, sourceID, sourceName string) (*BookChapterList, error) {
	listRule := ruleField(rules, "chapterList", "list")
	if listRule == "" {
		return NewChapterListParser("", "", "").ParseJSON(body, bookName, bookURL, sourceID, sourceName)
	}

	listParts := strings.Split(listRule, "&&")
	var items []string
	for _, part := range listParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		r := gjsonPathFirst(body, part)
		if r == "" {
			continue
		}
		// gjsonPathFirst on array path returns first element string; re-fetch array
		from := gjsonGetArrayItems(body, part)
		if len(from) > 0 {
			items = from
			break
		}
	}

	titleRule := ruleField(rules, "chapterName", "title", "name")
	urlRule := ruleField(rules, "chapterUrl", "url")
	chapters := make([]BookChapter, 0, len(items))
	for i, itemJSON := range items {
		title := gjsonPathFirst(itemJSON, titleRule)
		chURL := resolveLegadoURL(gjsonPathFirst(itemJSON, urlRule), bookURL, itemJSON)
		if strings.TrimSpace(title) == "" {
			continue
		}
		chapters = append(chapters, BookChapter{
			Title: title,
			URL:   chURL,
			Index: i + 1,
		})
	}

	return &BookChapterList{
		BookName:   bookName,
		BookURL:    bookURL,
		Chapters:   chapters,
		SourceID:   sourceID,
		SourceName: sourceName,
		Total:      len(chapters),
	}, nil
}

func gjsonGetArrayItems(body, path string) []string {
	r := gjson.Get(body, path)
	if !r.IsArray() {
		return nil
	}
	items := make([]string, 0, len(r.Array()))
	for _, item := range r.Array() {
		items = append(items, item.Raw)
	}
	return items
}

func (wb *WebBook) parseContentWithRules(ctx context.Context, source *BookSource, body, baseURL, chapterTitle, chapterURL, bookName, bookURL, sourceID, sourceName string, chapterIndex int) (*BookContent, error) {
	rules := parseLegadoFieldRules(source.ContentRule)
	mode := wb.modeFor(source, "content")

	contentRule := ruleField(rules, "content")
	titleRule := ruleField(rules, "title", "chapterName")

	raw := wb.execField(ctx, mode, contentRule, body, baseURL, "")
	if raw == "" {
		parser := NewContentParser("").SetSelectors(map[string]string{
			"content": ruleField(rules, "content"),
			"title":   ruleField(rules, "title", "chapterName"),
			"prevUrl": ruleField(rules, "prevUrl", "prev", "prevContentUrl"),
			"nextUrl": ruleField(rules, "nextUrl", "next", "nextContentUrl"),
		})
		return parser.ParseHTML(body, baseURL, chapterTitle, chapterURL, bookName, bookURL, sourceID, sourceName, chapterIndex)
	}

	content := &BookContent{
		ChapterTitle: chapterTitle,
		ChapterURL:   chapterURL,
		BookName:     bookName,
		BookURL:      bookURL,
		SourceID:     sourceID,
		SourceName:   sourceName,
		ChapterIndex: chapterIndex,
	}

	if titleRule != "" {
		if t := wb.execField(ctx, mode, titleRule, body, baseURL, ""); t != "" {
			content.ChapterTitle = t
		}
	}

	if strings.Contains(contentRule, "@html") {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader("<div>" + raw + "</div>"))
		if err == nil {
			parser := NewContentParser("")
			content.Content = parser.extractContent(doc.Find("div").First())
			content.Images = parser.extractImages(doc.Find("div").First(), baseURL)
			content.RawHTML = raw
			if strings.TrimSpace(content.Content) == "" {
				content.Content = cleanText(doc.Find("div").First().Text())
			}
		} else {
			content.Content = cleanText(raw)
		}
	} else {
		content.Content = cleanText(raw)
	}

	if prev := ruleField(rules, "prevUrl", "prev", "prevContentUrl"); prev != "" {
		content.PrevURL = resolveLegadoURL(wb.execField(ctx, mode, prev, body, baseURL, ""), baseURL, "")
	}
	if next := ruleField(rules, "nextContentUrl", "nextUrl", "next"); next != "" {
		content.NextURL = resolveLegadoURL(wb.execField(ctx, mode, next, body, baseURL, ""), baseURL, "")
	}
	if len(content.Images) == 0 {
		for _, key := range []string{"images", "imageList", "imgs", "pics"} {
			if rule := ruleField(rules, key); rule != "" {
				values := gjsonGetArrayItems(body, rule)
				for _, rawItem := range values {
					url := strings.Trim(strings.TrimSpace(rawItem), `"`)
					if url == "" {
						continue
					}
					content.Images = append(content.Images, resolveLegadoURL(url, baseURL, body))
				}
				if len(content.Images) > 0 {
					break
				}
			}
		}
	}
	if len(content.Images) > 0 {
		content.ReaderMode = "comic"
	} else {
		content.ReaderMode = "text"
	}
	content.WordCount = len(strings.ReplaceAll(content.Content, " ", ""))
	return content, nil
}

func (wb *WebBook) parseSearchWithExecutor(ctx context.Context, source *BookSource, body, baseURL string) ([]Book, error) {
	return wb.parseListPageWithRules(ctx, source, body, baseURL, source.SearchRule, wb.modeFor(source, "search"))
}

func (wb *WebBook) parseListPageWithRules(ctx context.Context, source *BookSource, body, baseURL, ruleText string, mode rule.RuleMode) ([]Book, error) {
	rules := parseLegadoFieldRules(ruleText)
	if len(rules) == 0 {
		// 尝试从搜索规则继承
		rules = parseLegadoFieldRules(source.SearchRule)
		if len(rules) == 0 {
			return nil, fmt.Errorf("missing bookList rule")
		}
	}

	bookListRule := ruleField(rules, "bookList", "bookUrl", "list")
	// 自动检测JSON响应
	if isJSONPathRule(bookListRule) ||
		strings.HasPrefix(strings.TrimSpace(body), "{") ||
		strings.HasPrefix(strings.TrimSpace(body), "[") {
		return parseLegadoJSONSearch(body, baseURL, source.ID, source.Name, rules)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	if bookListRule == "" {
		return nil, fmt.Errorf("missing bookList rule")
	}

	items, err := evalLegadoListItems(doc.Selection, bookListRule)
	if err != nil || len(items) == 0 {
		return nil, err
	}

	books := make([]Book, 0, len(items))
	for _, item := range items {
		book := Book{SourceID: source.ID, SourceName: source.Name}
		nameRule := ruleField(rules, "name", "title")
		if nameRule != "" {
			book.Name = evalLegadoRule(item, nameRule)
			if book.Name == "" {
				book.Name = wb.execField(ctx, mode, nameRule, body, baseURL, "")
			}
		}
		if r := rules["author"]; r != "" {
			book.Author = evalLegadoRule(item, r)
			if book.Author == "" {
				book.Author = wb.execField(ctx, mode, r, body, baseURL, "")
			}
		}
		if r := ruleField(rules, "bookUrl", "bookURL"); r != "" {
			book.BookURL = resolveLegadoURL(evalLegadoRule(item, r), baseURL, "")
			if book.BookURL == "" {
				book.BookURL = resolveLegadoURL(wb.execField(ctx, mode, r, body, baseURL, ""), baseURL, "")
			}
		}
		if r := ruleField(rules, "coverUrl", "cover"); r != "" {
			book.CoverURL = resolveLegadoURL(evalLegadoRule(item, r), baseURL, "")
			if book.CoverURL == "" {
				book.CoverURL = resolveLegadoURL(wb.execField(ctx, mode, r, body, baseURL, ""), baseURL, "")
			}
		}
		if r := rules["intro"]; r != "" {
			book.Intro = evalLegadoRule(item, r)
			if book.Intro == "" {
				book.Intro = wb.execField(ctx, mode, r, body, baseURL, "")
			}
		}
		if strings.TrimSpace(book.Name) != "" {
			books = append(books, book)
		}
	}
	return books, nil
}
