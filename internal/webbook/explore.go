package webbook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"reader-go/internal/rule"
)

// ExploreTab 书海分类 Tab
type ExploreTab struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ExploreResult 书海发现结果
type ExploreResult struct {
	SourceID   string       `json:"sourceId"`
	SourceName string       `json:"sourceName"`
	Tab        string       `json:"tab"`
	URL        string       `json:"url"`
	Tabs       []ExploreTab `json:"tabs,omitempty"`
	Books      []Book       `json:"books"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	HasMore    bool         `json:"hasMore"`
}

// ResolveExploreURL 解析 exploreUrl（支持 JSON Tab 数组、换行分隔的 分类::URL 格式、纯URL）。
// 返回 resolved URL、全部 tabs、实际选中的 tab 标题。
func ResolveExploreURL(exploreURL, tab string) (resolved string, tabs []ExploreTab, resolvedTab string, err error) {
	exploreURL = strings.TrimSpace(exploreURL)
	if exploreURL == "" {
		return "", nil, "", fmt.Errorf("explore URL empty")
	}

	// 1. 支持JSON数组格式（兼容单引号、双引号）
	if strings.HasPrefix(exploreURL, "[") {
		// 先尝试直接解析（保留原始转义）
		cleaned := strings.ReplaceAll(exploreURL, "\n", "")
		cleaned = strings.ReplaceAll(cleaned, "\t", "")
		cleaned = strings.TrimSpace(cleaned)
		// 将 null 替换为空字符串，避免解析失败
		cleaned = regexp.MustCompile(`:\s*null`).ReplaceAllString(cleaned, `:""`)
		if err := json.Unmarshal([]byte(cleaned), &tabs); err != nil || len(tabs) == 0 {
			// 直接解析失败，尝试单引号替换
			normalized := regexp.MustCompile(`'([^']*)'`).ReplaceAllString(cleaned, `"$1"`)
			if err := json.Unmarshal([]byte(normalized), &tabs); err == nil && len(tabs) > 0 {
				// 成功
			} else {
				// JSON解析失败，返回错误而不是原始字符串
				return "", nil, "", fmt.Errorf("invalid JSON tabs: %v", err)
			}
		}
		if len(tabs) > 0 {
			// 处理URL中的变量占位符
			for i := range tabs {
				tabs[i].URL = strings.ReplaceAll(tabs[i].URL, "&amp;", "&")
				tabs[i].URL = strings.ReplaceAll(tabs[i].URL, "&", "&")
			}
		}
	}

	// 2. 支持换行分隔的 分类名::URL 格式（Legado常用格式）
	if len(tabs) == 0 && strings.Contains(exploreURL, "::") && strings.Contains(exploreURL, "\n") {
		lines := strings.Split(exploreURL, "\n")
		var baseURL string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "::", 2)
			if len(parts) != 2 {
				continue
			}
			title := strings.TrimSpace(parts[0])
			link := strings.TrimSpace(parts[1])
			if title == "" || link == "" {
				continue
			}
			// 第一行如果是http开头，当成baseURL
			if baseURL == "" && strings.HasPrefix(title, "http") {
				baseURL = title
				continue
			}
			// 相对URL自动和baseURL拼接
			if baseURL != "" && !strings.HasPrefix(link, "http") {
				if !strings.HasPrefix(link, "/") {
					link = "/" + link
				}
				// 解析baseURL
				u, err := url.Parse(baseURL)
				if err == nil {
					link = u.Scheme + "://" + u.Host + link
				}
			}
			tabs = append(tabs, ExploreTab{Title: title, URL: link})
		}
	}

	// 3. 纯URL格式
	if len(tabs) == 0 {
		return exploreURL, nil, tab, nil
	}
	if tab != "" {
		tabTrim := strings.TrimSpace(strings.ToLower(tab))
		for _, t := range tabs {
			if strings.TrimSpace(strings.ToLower(t.Title)) == tabTrim {
				if strings.TrimSpace(t.URL) == "" {
					return "", tabs, tab, fmt.Errorf("explore tab %q has empty URL", tab)
				}
				return t.URL, tabs, tab, nil
			}
		}
		// 模糊匹配：包含关键词就算匹配
		for _, t := range tabs {
			if strings.Contains(strings.ToLower(t.Title), tabTrim) {
				if strings.TrimSpace(t.URL) == "" {
					return "", tabs, tab, fmt.Errorf("explore tab %q has empty URL", tab)
				}
				return t.URL, tabs, tab, nil
			}
		}
		// Tab not found - fall back to first valid tab instead of error
		for _, t := range tabs {
			if strings.TrimSpace(t.URL) != "" {
				return t.URL, tabs, t.Title, nil
			}
		}
		return "", tabs, tab, fmt.Errorf("explore tab not found: %s", tab)
	}
	for _, t := range tabs {
		if strings.TrimSpace(t.URL) == "" {
			continue
		}
		return t.URL, tabs, t.Title, nil
	}
	return "", tabs, "", fmt.Errorf("no explore tab with URL configured")
}

func exploreBaseURL(base string) string {
	base = strings.TrimSpace(base)
	if i := strings.Index(base, "#"); i >= 0 {
		base = base[:i]
	}
	return strings.TrimRight(base, "/")
}

// buildExploreURL 将 explore 相对路径拼成可请求的绝对 URL，并替换 {{page}} 等模板变量。
// 支持 page 参数（默认 1）。
// 支持 Legado URL 模板语法：<prefix,suffix> 表示 page 前后缀
func (wb *WebBook) buildExploreURL(source *BookSource, template string, page int) (string, error) {
	template = strings.TrimSpace(template)
	if template == "" {
		return "", fmt.Errorf("explore URL empty")
	}

	if strings.HasPrefix(template, "@js:") {
		vars := map[string]string{"page": strconv.Itoa(page)}
		return wb.evalJSURL(source, template, vars)
	}

	// 处理 Legado URL 模板语法：<prefix,suffix> 或 <page1,page2,...>
	template = expandLegadoURLTemplate(template, page)

	if !strings.HasPrefix(template, "http://") && !strings.HasPrefix(template, "https://") {
		base := exploreBaseURL(source.BaseURL)
		if base == "" {
			return "", fmt.Errorf("baseUrl empty for relative explore URL")
		}
		if !strings.HasPrefix(template, "/") {
			template = "/" + template
		}
		template = base + template
	}

	// 有序替换：双括号优先
	pageStr := strconv.Itoa(page)
	template = strings.ReplaceAll(template, "{{page}}", pageStr)
	template = strings.ReplaceAll(template, "{page}", pageStr)
	return template, nil
}

// expandLegadoURLTemplate 展开 Legado URL 模板语法
// 支持：<prefix,suffix> - page前后缀模式
//   - 当 page=1 时，只使用前缀（如果前缀为空则不添加任何内容）
//   - 当 page>1 时，使用前缀+页码+后缀
// 支持：<page1,page2,...> - 页码多选模式（返回对应页码）
func expandLegadoURLTemplate(template string, page int) string {
	// 查找 <...> 模式
	start := strings.Index(template, "<")
	if start < 0 {
		return template
	}
	end := strings.Index(template[start:], ">")
	if end < 0 {
		return template
	}
	end += start

	// 提取 <...> 内容
	content := template[start+1 : end]
	before := template[:start]
	after := template[end+1:]

	// 检查是否是页码多选模式：<page1,page2,...>
	// 如果内容全是数字和逗号，则是页码多选
	isPageList := true
	for _, c := range content {
		if c != ',' && (c < '0' || c > '9') {
			isPageList = false
			break
		}
	}

	if isPageList && strings.Contains(content, ",") {
		// 页码多选模式，使用对应的页码
		pages := strings.Split(content, ",")
		pageIdx := page - 1
		if pageIdx >= 0 && pageIdx < len(pages) {
			return before + pages[pageIdx] + after
		}
		return before + pages[0] + after
	}

	// 前后缀模式：<prefix,suffix>
	if strings.Contains(content, ",") {
		parts := strings.SplitN(content, ",", 2)
		prefix := parts[0]
		suffix := ""
		if len(parts) > 1 {
			suffix = parts[1]
		}
		pageStr := strconv.Itoa(page)

		// 第一页特殊处理：如果前缀为空，不添加任何内容
		if page == 1 && prefix == "" {
			return before + after
		}

		// 后缀中的 {{page}} 也要替换
		suffix = strings.ReplaceAll(suffix, "{{page}}", pageStr)
		suffix = strings.ReplaceAll(suffix, "{page}", pageStr)

		// 如果前缀为空，只添加后缀（不添加页码）
		if prefix == "" {
			return before + suffix + after
		}
		return before + prefix + pageStr + suffix + after
	}

	// 单个值，直接使用
	return before + content + after
}

// ExploreSearchResult 搜索/筛选结果
type ExploreSearchResult struct {
	SourceID   string       `json:"sourceId"`
	SourceName string       `json:"sourceName"`
	Tab        string       `json:"tab"`
	URL        string       `json:"url"`
	Tabs       []ExploreTab `json:"tabs,omitempty"`
	Books      []Book       `json:"books"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	HasMore    bool         `json:"hasMore"`
	Total      int          `json:"total"`
}

// ExploreSearchRequest 搜索请求参数
type ExploreSearchRequest struct {
	SourceID int64  `json:"sourceId"`
	Tab      string `json:"tab"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Search   string `json:"search"`
	Category string `json:"category"`
}

func (wb *WebBook) modeForExplore(source *BookSource) rule.RuleMode {
	if source.ExploreMode != "" {
		return rule.ModeFromString(source.ExploreMode)
	}
	return wb.modeFor(source, "search")
}

// Explore 按书源 exploreRule 解析书单（单页，默认 page=1）
func (wb *WebBook) Explore(ctx context.Context, source *BookSource, tab string) (*ExploreResult, error) {
	result, err := wb.ExploreSearch(ctx, source, tab, 1, 30, "", "")
	if err != nil {
		return nil, err
	}
	return &ExploreResult{
		SourceID:   result.SourceID,
		SourceName: result.SourceName,
		Tab:        result.Tab,
		URL:        result.URL,
		Tabs:       result.Tabs,
		Books:      result.Books,
		Page:       result.Page,
		PageSize:   result.PageSize,
		HasMore:    result.HasMore,
	}, nil
}

// ExploreSearch 支持分页、搜索、分类筛选的书海查询
// 当 pageSize > 0 时，会自动翻页获取足够的数据
func (wb *WebBook) ExploreSearch(ctx context.Context, source *BookSource, tab string, page, pageSize int, search, category string) (*ExploreSearchResult, error) {
	if source == nil {
		return nil, fmt.Errorf("source is nil")
	}

	// 处理 @js: 前缀：先执行 JavaScript 获取实际的 URL/tabs
	exploreURL := source.ExploreURL
	if strings.HasPrefix(exploreURL, "@js:") {
		jsResult, err := wb.evalJSURL(source, exploreURL, map[string]string{"page": strconv.Itoa(page)})
		if err != nil {
			return nil, fmt.Errorf("evalJSURL failed: %w", err)
		}
		exploreURL = jsResult
	}

	rawURL, tabs, resolvedTab, err := ResolveExploreURL(exploreURL, tab)
	if err != nil {
		return nil, err
	}

	ruleText := strings.TrimSpace(source.ExploreRule)
	// 如果 exploreRule 为空或解析后没有 bookList，回退到 searchRule
	if ruleText == "" || ruleText == "{}" {
		ruleText = source.SearchRule
	} else {
		// 检查解析后是否有 bookList 字段
		rules := parseLegadoFieldRules(ruleText)
		if _, hasBookList := rules["bookList"]; !hasBookList {
			// 没有 bookList，尝试使用 searchRule
			searchRules := parseLegadoFieldRules(source.SearchRule)
			if _, hasSearchBookList := searchRules["bookList"]; hasSearchBookList {
				ruleText = source.SearchRule
			}
		}
	}
	mode := wb.modeForExplore(source)

	var allBooks []Book
	currentPage := page
	maxPages := 10 // 防止无限翻页
	var lastURL string

	// 自动翻页获取足够的数据
	// pageSize <= 0 时获取所有可用数据（最多 maxPages 页）
	for (pageSize <= 0 || len(allBooks) < pageSize) && maxPages > 0 {
		maxPages--

		// 构建 URL（使用实际页码）
		resolvedURL, err := wb.buildExploreURL(source, rawURL, currentPage)
		if err != nil {
			if len(allBooks) > 0 {
				break // 已有数据，停止翻页
			}
			return nil, err
		}
		lastURL = resolvedURL

		// 如果有搜索关键词，尝试用搜索 URL 而不是 explore URL
		fetchMethod := "GET"
		fetchBody := ""
		if strings.TrimSpace(search) != "" && currentPage == page {
			searchURL, method, body, err := wb.buildExploreSearchURL(source, search, currentPage)
			if err == nil && searchURL != "" {
				resolvedURL = searchURL
				lastURL = resolvedURL
				fetchMethod = method
				fetchBody = body
			}
		}

		wb.concurrent.SourceWait(source.ID)
		resp, err := wb.fetch(ctx, source, resolvedURL, fetchMethod, fetchBody)
		if err != nil {
			if len(allBooks) > 0 {
				break // 已有数据，停止翻页
			}
			return nil, err
		}

		// 自动检测JSON响应
		respMode := mode
		if strings.HasPrefix(strings.TrimSpace(resp.Body), "{") || strings.HasPrefix(strings.TrimSpace(resp.Body), "[") {
			respMode = rule.ModeJSONPath
		}
		books, err := wb.parseListPageWithRules(ctx, source, resp.Body, resp.URL, ruleText, respMode)
		if err != nil {
			if len(allBooks) > 0 {
				break // 已有数据，停止翻页
			}
			return nil, err
		}

		if len(books) == 0 {
			break // 没有更多数据
		}

		allBooks = append(allBooks, books...)
		currentPage++
	}

	books := allBooks

	// 应用搜索/分类筛选（如果指定了关键词）
	if strings.TrimSpace(search) != "" {
		searchLower := strings.ToLower(search)
		filtered := make([]Book, 0, len(books))
		for _, b := range books {
			if strings.Contains(strings.ToLower(b.Name), searchLower) ||
				strings.Contains(strings.ToLower(b.Author), searchLower) {
				filtered = append(filtered, b)
			}
		}
		books = filtered
	}

	if strings.TrimSpace(category) != "" {
		filtered := make([]Book, 0, len(books))
		for _, b := range books {
			if strings.Contains(b.Category, category) {
				filtered = append(filtered, b)
			}
		}
		books = filtered
	}

	// 分页逻辑：
	// - pageSize <= 0 时，返回全部书籍（不分页）
	// - pageSize > 0 时，返回前 pageSize 条数据
	total := len(books)
	var hasMore bool
	if pageSize > 0 && len(books) > pageSize {
		books = books[:pageSize]
		hasMore = true // 可能还有更多数据
	} else if pageSize > 0 {
		// 检查是否还有下一页（通过尝试获取下一页）
		hasMore = len(books) >= pageSize && maxPages == 0
	}

	return &ExploreSearchResult{
		SourceID:   source.ID,
		SourceName: source.Name,
		Tab:        resolvedTab,
		URL:        lastURL,
		Tabs:       tabs,
		Books:      books,
		Page:       page,
		PageSize:   pageSize,
		HasMore:    hasMore,
		Total:      total,
	}, nil
}

// buildExploreSearchURL 构建探索搜索 URL（支持 {{search}} 模板）
func (wb *WebBook) buildExploreSearchURL(source *BookSource, search string, page int) (string, string, string, error) {
	searchRule := source.ExploreSearchURL
	if searchRule == "" {
		return "", "", "", fmt.Errorf("no explore search URL configured")
	}

	method := "GET"
	body := ""

	// 支持 JSON 配置
	if idx := strings.Index(searchRule, ",{"); idx > 0 {
		metaJSON := searchRule[idx+1:]
		searchRule = searchRule[:idx]
		metaJSON = strings.ReplaceAll(metaJSON, "'", "\"")
		var meta struct {
			Method string `json:"method"`
			Body   string `json:"body"`
		}
		if err := json.Unmarshal([]byte(metaJSON), &meta); err == nil {
			if meta.Method != "" {
				method = strings.ToUpper(meta.Method)
			}
			body = meta.Body
		}
	}

	if strings.HasPrefix(searchRule, "@js:") {
		vars := map[string]string{"search": search, "page": strconv.Itoa(page)}
		u, err := wb.evalJSURL(source, searchRule, vars)
		if err != nil {
			return "", "", "", err
		}
		return u, method, body, nil
	}

	if !strings.HasPrefix(searchRule, "http://") && !strings.HasPrefix(searchRule, "https://") {
		base := exploreBaseURL(source.BaseURL)
		if base == "" {
			return "", "", "", fmt.Errorf("baseUrl empty for relative explore search URL")
		}
		if !strings.HasPrefix(searchRule, "/") {
			searchRule = "/" + searchRule
		}
		searchRule = base + searchRule
	}

	// 替换模板变量
	encodedSearch := url.QueryEscape(search)
	replacements := map[string]string{
		"{{search}}": encodedSearch,
		"{{key}}":    encodedSearch,
		"{{page}}":   strconv.Itoa(page),
		"{search}":   encodedSearch,
		"{key}":      encodedSearch,
		"{page}":     strconv.Itoa(page),
	}
	for k, v := range replacements {
		searchRule = strings.ReplaceAll(searchRule, k, v)
		body = strings.ReplaceAll(body, k, search)
	}

	return searchRule, method, body, nil
}

// SearchSingleSource 在单个书源内搜索（换源候选）
func (wb *WebBook) SearchSingleSource(ctx context.Context, sourceID, query string) ([]Book, error) {
	for _, s := range wb.sources {
		if s.ID == sourceID && s.Enabled {
			return wb.searchWithSource(ctx, s, query)
		}
	}
	return nil, fmt.Errorf("source %s not found or disabled", sourceID)
}

// GetSourceByID 按 ID 取运行时书源
func (wb *WebBook) GetSourceByID(sourceID string) *BookSource {
	for _, s := range wb.sources {
		if s.ID == sourceID {
			return s
		}
	}
	return nil
}
