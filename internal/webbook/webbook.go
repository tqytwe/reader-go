package webbook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"reader-go/internal/browser"
	"reader-go/internal/rule"
	"reader-go/internal/utils"
)

// =============================================================================
// WebBook 核心编排类
// =============================================================================

// WebBook 网络书籍流程编排器
// 负责协调多书源并发搜索、书籍信息/目录/正文获取的完整流程
type WebBook struct {
	// 书源集合
	sources []*BookSource

	// HTTP 客户端
	client *http.Client

	// 并发控制器
	concurrent *ConcurrentRecord

	// 默认请求超时
	timeout time.Duration

	// 用户代理
	userAgent string

	// 请求头
	headers map[string]string

	// 结果合并策略
	mergeStrategy MergeStrategy

	// 解析器（可配置）
	listParser    *BookListParser
	infoParser    *BookInfoParser
	chapterParser *ChapterListParser
	contentParser *ContentParser

	// 统一规则执行器
	executor *rule.Executor

	// 事件回调
	onSearchStart  func(source *BookSource, query string)
	onSearchDone   func(source *BookSource, result *BookList, err error)
	onInfoStart    func(source *BookSource, book *Book)
	onInfoDone     func(source *BookSource, info *BookInfo, err error)
	onChapterStart func(source *BookSource, book *BookInfo)
	onChapterDone  func(source *BookSource, list *BookChapterList, err error)
	onContentStart func(source *BookSource, chapter *BookChapter)
	onContentDone  func(source *BookSource, content *BookContent, err error)
}

// BookSource 书源定义（运行时版本，从 booksource.BookSource 转换而来）
// 注意：数据库实体在 internal/booksource/model.go 中定义
type BookSource struct {
	// 书源ID（唯一标识）
	ID string `json:"id"`

	// 书源名称
	Name string `json:"name"`

	// 书源URL（基础URL）
	BaseURL string `json:"baseUrl"`

	// 搜索URL模板（支持 {key} 占位）
	SearchURL string `json:"searchUrl"`

	// 书籍详情URL模板
	BookInfoURL string `json:"bookInfoUrl"`

	// 目录URL模板
	ChapterListURL string `json:"chapterListUrl"`

	// 正文URL模板
	ContentURL string `json:"contentUrl"`

	// 搜索规则
	SearchRule string `json:"searchRule"`

	// 书籍详情规则
	BookInfoRule string `json:"bookInfoRule"`

	// 目录规则
	ChapterListRule string `json:"chapterListRule"`

	// 正文规则
	ContentRule string `json:"contentRule"`

	// 请求头（书源特有）
	Headers map[string]string `json:"headers"`

	// 请求超时（秒）
	TimeoutSec int `json:"timeout"`

	// 频率限制（每秒请求数）
	Rate float64 `json:"rate"`

	// 突发上限
	Burst int `json:"burst"`

	// 是否启用
	Enabled bool `json:"enabled"`

	// 权重（用于结果排序）
	Weight int `json:"weight"`

	// 标签
	Tags []string `json:"tags"`

	// 登录状态（是否需要登录）
	NeedsLogin bool `json:"needsLogin"`

	// 登录URL
	LoginURL string `json:"loginUrl"`

	// 书海
	ExploreURL       string `json:"exploreUrl"`
	ExploreRule      string `json:"exploreRule"`
	ExploreMode      string `json:"exploreMode"`
	ExploreSearchURL string `json:"exploreSearchUrl"`

	// 登录Cookie
	Cookie string `json:"-"`

	// 解析模式
	Mode         rule.RuleMode `json:"mode"`
	SearchMode   string        `json:"searchMode"`
	BookInfoMode string        `json:"bookInfoMode"`
	TocMode      string        `json:"tocMode"`
	ContentMode  string        `json:"contentMode"`
}

// MergeStrategy 结果合并策略
type MergeStrategy int

const (
	MergeFirst  MergeStrategy = iota // 只取第一个有效结果
	MergeAll                         // 合并所有书源结果
	MergeBest                        // 取质量最好的结果（基于权重）
	MergeUnique                      // 去重合并（按书名+作者）
)

// =============================================================================
// 流程结果
// =============================================================================

// SearchResult 搜索结果（多书源合并）
type SearchResult struct {
	// 查询关键词
	Query string `json:"query"`

	// 合并后的书籍列表
	Books []Book `json:"books"`

	// 各书源搜索结果详情
	SourceResults map[string]*SourceSearchResult `json:"sourceResults"`

	// 总耗时
	Duration time.Duration `json:"duration"`
}

// SourceSearchResult 单个书源的搜索结果
type SourceSearchResult struct {
	SourceID   string        `json:"sourceId"`
	SourceName string        `json:"sourceName"`
	Books      []Book        `json:"books"`
	HasMore    bool          `json:"hasMore"`
	Error      error         `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
}

// BookDetailResult 书籍详情获取结果
type BookDetailResult struct {
	// 书籍信息
	Info *BookInfo `json:"info"`

	// 目录
	ChapterList *BookChapterList `json:"chapterList"`

	// 各书源详情（用于对比）
	SourceInfos map[string]*BookInfo `json:"sourceInfos"`

	// 各书源目录
	SourceChapters map[string]*BookChapterList `json:"sourceChapters"`

	// 总耗时
	Duration time.Duration `json:"duration"`
}

// =============================================================================
// WebBook 构造与配置
// =============================================================================

// browserFetcher 浏览器获取器实例
var browserFetcher = browser.NewFetcher()

// NewWebBook 创建新的WebBook编排器
func NewWebBook() *WebBook {
	return &WebBook{
		sources:       make([]*BookSource, 0),
		timeout:       30 * time.Second,
		userAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		headers:       make(map[string]string),
		mergeStrategy: MergeUnique,
		concurrent:    NewConcurrentRecord(5, 1000, 1.0, 3),
		client:        &http.Client{Timeout: 30 * time.Second},
		executor:      rule.NewExecutor(),
	}
}

func (wb *WebBook) timeoutForSource(source *BookSource) time.Duration {
	if source != nil && source.TimeoutSec > 0 {
		return time.Duration(source.TimeoutSec) * time.Second
	}
	return wb.timeout
}

// WithClient 设置自定义HTTP客户端
func (wb *WebBook) WithClient(client *http.Client) *WebBook {
	wb.client = client
	return wb
}

// WithSources 设置书源集合
func (wb *WebBook) WithSources(sources []*BookSource) *WebBook {
	wb.sources = sources
	return wb
}

// WithMergeStrategy 设置结果合并策略
func (wb *WebBook) WithMergeStrategy(strategy MergeStrategy) *WebBook {
	wb.mergeStrategy = strategy
	return wb
}

// WithTimeout 设置请求超时
func (wb *WebBook) WithTimeout(timeout time.Duration) *WebBook {
	wb.timeout = timeout
	return wb
}

// WithUserAgent 设置User-Agent
func (wb *WebBook) WithUserAgent(ua string) *WebBook {
	wb.userAgent = ua
	return wb
}

// AddSource 添加单个书源
func (wb *WebBook) AddSource(source *BookSource) *WebBook {
	wb.sources = append(wb.sources, source)
	return wb
}

// ReloadSources 从数据库书源列表重建运行时书源
func (wb *WebBook) ReloadSources(sources []*BookSource) {
	wb.sources = sources
}

// WithSearchCallbacks 设置搜索回调
func (wb *WebBook) WithSearchCallbacks(onStart func(source *BookSource, query string), onDone func(source *BookSource, result *BookList, err error)) *WebBook {
	wb.onSearchStart = onStart
	wb.onSearchDone = onDone
	return wb
}

// WithInfoCallbacks 设置详情回调
func (wb *WebBook) WithInfoCallbacks(onStart func(source *BookSource, book *Book), onDone func(source *BookSource, info *BookInfo, err error)) *WebBook {
	wb.onInfoStart = onStart
	wb.onInfoDone = onDone
	return wb
}

// WithChapterCallbacks 设置目录回调
func (wb *WebBook) WithChapterCallbacks(onStart func(source *BookSource, book *BookInfo), onDone func(source *BookSource, list *BookChapterList, err error)) *WebBook {
	wb.onChapterStart = onStart
	wb.onChapterDone = onDone
	return wb
}

// WithContentCallbacks 设置正文回调
func (wb *WebBook) WithContentCallbacks(onStart func(source *BookSource, chapter *BookChapter), onDone func(source *BookSource, content *BookContent, err error)) *WebBook {
	wb.onContentStart = onStart
	wb.onContentDone = onDone
	return wb
}

// =============================================================================
// 核心流程
// =============================================================================

// SearchBook 搜索书籍
// 并发遍历所有启用的书源，返回合并后的结果
func (wb *WebBook) SearchBook(ctx context.Context, query string) (*SearchResult, error) {
	if len(wb.sources) == 0 {
		return nil, fmt.Errorf("no book sources configured")
	}

	start := time.Now()
	result := &SearchResult{
		Query:         query,
		Books:         make([]Book, 0),
		SourceResults: make(map[string]*SourceSearchResult),
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	const maxResults = 50

	// 使用滑动窗口并发搜索
	seen := make(map[string]struct{}) // bookKey -> struct{}
	var mu sync.Mutex

	err := wb.concurrentLimitSearch(ctx, wb.sources, query, func(source *BookSource, books []Book, err error) {
		srcResult := &SourceSearchResult{
			SourceID:   source.ID,
			SourceName: source.Name,
			Books:      books,
			Error:      err,
			Duration:   time.Since(start),
		}

		mu.Lock()
		defer mu.Unlock()

		result.SourceResults[source.ID] = srcResult

		if err != nil {
			return
		}

		// 去重合并
		for _, book := range books {
			if strings.TrimSpace(book.BookURL) == "" {
				continue
			}
			key := book.Name + "|" + book.Author
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				result.Books = append(result.Books, book)
			}
			if len(result.Books) >= maxResults {
				cancel()
				break
			}
		}
	})

	if err != nil {
		return result, err
	}

	result.Duration = time.Since(start)
	return result, nil
}

// SearchBookStream 并发搜索并在每个书源完成时回调（用于 SSE 流式输出）
func (wb *WebBook) SearchBookStream(ctx context.Context, query string, onResult func(source *BookSource, books []Book, err error)) error {
	if len(wb.sources) == 0 {
		return fmt.Errorf("no book sources configured")
	}
	return wb.concurrentLimitSearch(ctx, wb.sources, query, onResult)
}

// GetBookInfo 获取书籍详情
func (wb *WebBook) GetBookInfo(ctx context.Context, book *Book) (*BookInfo, error) {
	// 找到对应的书源
	var source *BookSource
	for _, s := range wb.sources {
		if s.ID == book.SourceID {
			source = s
			break
		}
	}
	if source == nil {
		return nil, fmt.Errorf("source not found: %s", book.SourceID)
	}

	// 执行详情获取流程
	return wb.fetchBookInfo(ctx, source, book)
}

// GetChapterList 获取目录
func (wb *WebBook) GetChapterList(ctx context.Context, info *BookInfo) (*BookChapterList, error) {
	// 找到对应的书源
	var source *BookSource
	for _, s := range wb.sources {
		if s.ID == info.SourceID {
			source = s
			break
		}
	}
	if source == nil {
		return nil, fmt.Errorf("source not found: %s", info.SourceID)
	}

	return wb.fetchChapterList(ctx, source, info)
}

// GetBookContent 获取正文
func (wb *WebBook) GetBookContent(ctx context.Context, info *BookInfo, chapter *BookChapter) (*BookContent, error) {
	// 找到对应的书源
	var source *BookSource
	for _, s := range wb.sources {
		if s.ID == info.SourceID {
			source = s
			break
		}
	}
	if source == nil {
		return nil, fmt.Errorf("source not found: %s", info.SourceID)
	}

	return wb.fetchContent(ctx, source, info, chapter)
}

// SearchAndGetDetail 一键完整流程：搜索 → 获取详情 → 获取目录
func (wb *WebBook) SearchAndGetDetail(ctx context.Context, query string) (*BookDetailResult, error) {
	start := time.Now()
	result := &BookDetailResult{
		SourceInfos:    make(map[string]*BookInfo),
		SourceChapters: make(map[string]*BookChapterList),
	}

	// 1. 搜索
	searchResult, err := wb.SearchBook(ctx, query)
	if err != nil {
		return result, err
	}

	if len(searchResult.Books) == 0 {
		return result, fmt.Errorf("no books found")
	}

	// 2. 获取第一个结果的详情
	firstBook := searchResult.Books[0]
	info, err := wb.GetBookInfo(ctx, &firstBook)
	if err != nil {
		return result, err
	}
	result.Info = info

	// 3. 获取目录
	chapters, err := wb.GetChapterList(ctx, info)
	if err != nil {
		return result, err
	}
	result.ChapterList = chapters

	result.Duration = time.Since(start)
	return result, nil
}

// =============================================================================
// 内部方法
// =============================================================================

// concurrentLimitSearch 并发限制搜索
func (wb *WebBook) concurrentLimitSearch(
	ctx context.Context,
	sources []*BookSource,
	query string,
	onResult func(source *BookSource, books []Book, err error),
) error {
	// 过滤启用的书源
	var enabled []*BookSource
	for _, s := range sources {
		if s.Enabled {
			enabled = append(enabled, s)
		}
	}

	if len(enabled) == 0 {
		return fmt.Errorf("no enabled book sources")
	}

	// 滑动窗口并发
	sem := make(chan struct{}, wb.concurrent.maxConcurrent)
	var wg sync.WaitGroup

	for _, source := range enabled {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		default:
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(src *BookSource) {
			defer wg.Done()
			defer func() { <-sem }()

			books, err := wb.searchWithSource(ctx, src, query)
			onResult(src, books, err)
		}(source)
	}

	wg.Wait()
	return nil
}

// searchWithSource 使用单个书源搜索
func (wb *WebBook) searchWithSource(ctx context.Context, source *BookSource, query string) ([]Book, error) {
	if wb.onSearchStart != nil {
		wb.onSearchStart(source, query)
	}

	// 等待书源频率限制
	wb.concurrent.SourceWait(source.ID)

	// 构建搜索URL
	searchURL, searchMethod, searchBody, err := wb.buildSearchURL(source, query)
	if err != nil {
		if wb.onSearchDone != nil {
			wb.onSearchDone(source, nil, err)
		}
		return nil, err
	}

	// 发起请求
	resp, err := wb.fetch(ctx, source, searchURL, searchMethod, searchBody)
	if err != nil {
		if wb.onSearchDone != nil {
			wb.onSearchDone(source, nil, err)
		}
		return nil, err
	}

	// 解析结果
	rules := parseLegadoSearchRule(source.SearchRule)
	books, err := parseLegadoSearchResults(resp.Body, resp.URL, source.ID, source.Name, rules)
	if err != nil || len(books) == 0 {
		books, err = wb.parseSearchWithExecutor(ctx, source, resp.Body, resp.URL)
		if err != nil {
			if wb.onSearchDone != nil {
				wb.onSearchDone(source, nil, err)
			}
			return nil, err
		}
		if len(books) == 0 {
			if wb.onSearchDone != nil {
				wb.onSearchDone(source, &BookList{Books: []Book{}}, nil)
			}
			return []Book{}, nil
		}
	}

	bookList := &BookList{Books: books, Page: 1, PageSize: len(books)}
	if wb.onSearchDone != nil {
		wb.onSearchDone(source, bookList, nil)
	}
	return books, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// buildSearchURL 构建搜索URL，支持 Legado 模板与 POST 配置
// 返回 (url, method, body, error)
func (wb *WebBook) buildSearchURL(source *BookSource, query string) (string, string, string, error) {
	template := strings.TrimSpace(source.SearchURL)
	if template == "" {
		return "", "GET", "", fmt.Errorf("search URL not configured for source %s", source.Name)
	}

	method := "GET"
	body := ""

	// Legado: /path,{'method':'POST','body':'s={{key}}'}
	if idx := strings.Index(template, ",{"); idx > 0 {
		metaJSON := template[idx+1:]
		template = template[:idx]
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

	if strings.HasPrefix(template, "@js:") {
		vars := map[string]string{"key": query, "page": "1"}
		u, err := wb.evalJSURL(source, template, vars)
		if err != nil {
			return "", "GET", "", err
		}
		return u, method, body, nil
	}

	// 相对路径拼接 baseUrl
	if !strings.HasPrefix(template, "http://") && !strings.HasPrefix(template, "https://") {
		base := strings.TrimRight(source.BaseURL, "/")
		if !strings.HasPrefix(template, "/") {
			template = "/" + template
		}
		template = base + template
	}

	encodedQuery := url.QueryEscape(query)
	replacements := map[string]string{
		"{{key}}":   encodedQuery,
		"{{page}}":  "1",
		"{key}":     encodedQuery,
		"{q}":       encodedQuery,
		"{keyword}": encodedQuery,
	}
	for k, v := range replacements {
		template = strings.ReplaceAll(template, k, v)
		body = strings.ReplaceAll(body, k, query)
	}

	return template, method, body, nil
}

// fetch 发起HTTP请求，包含 SSRF 防护
func (wb *WebBook) fetch(ctx context.Context, source *BookSource, rawURL string, method string, reqBody string) (*Response, error) {
	// SSRF 防护：验证目标 URL
	if err := utils.ValidateURL(rawURL); err != nil {
		return nil, fmt.Errorf("SSRF check failed for source %q: %w", source.Name, err)
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if method == "" {
		method = "GET"
	}

	browserAllowed := method == http.MethodGet && reqBody == "" && browserFetcher.IsEnabled()
	timeout := wb.timeoutForSource(source)
	headers := map[string]string{
		"User-Agent": wb.userAgent,
	}
	for k, v := range source.Headers {
		headers[k] = v
	}
	if source.Cookie != "" {
		headers["Cookie"] = source.Cookie
	}
	if reqBody != "" {
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}
	result, err := utils.FetchText(ctx, utils.FetchTextOptions{
		Method:       method,
		URL:          parsedURL.String(),
		Body:         reqBody,
		Headers:      headers,
		Client:       wb.client,
		Timeout:      timeout,
		AllowBrowser: browserAllowed,
		UseHeuristic: browserAllowed,
		Browser:      browserFetcher,
	})
	if err != nil {
		return nil, err
	}

	return &Response{
		URL:    result.URL,
		Body:   result.Body,
		Header: result.Header,
	}, nil
}

// fetchBookInfo 获取书籍详情
func (wb *WebBook) fetchBookInfo(ctx context.Context, source *BookSource, book *Book) (*BookInfo, error) {
	if wb.onInfoStart != nil {
		wb.onInfoStart(source, book)
	}

	// 等待书源频率限制
	wb.concurrent.SourceWait(source.ID)

	// 构建详情URL
	infoURL, infoMethod, infoBody, err := wb.buildBookInfoURL(source, book)
	if err != nil {
		if wb.onInfoDone != nil {
			wb.onInfoDone(source, nil, err)
		}
		return nil, fmt.Errorf("build info URL failed: %w", err)
	}

	// 发起请求
	resp, err := wb.fetch(ctx, source, infoURL, infoMethod, infoBody)
	if err != nil {
		if wb.onInfoDone != nil {
			wb.onInfoDone(source, nil, err)
		}
		return nil, fmt.Errorf("fetch book info failed: %w", err)
	}

	// 解析结果
	var info *BookInfo
	if source.BookInfoRule != "" {
		info, err = wb.parseBookInfoWithRules(ctx, source, resp.Body, resp.URL, source.ID, source.Name)
	} else {
		info, err = NewBookInfoParser("", "", "").ParseHTML(resp.Body, resp.URL, source.ID, source.Name)
	}
	if err != nil {
		if wb.onInfoDone != nil {
			wb.onInfoDone(source, nil, err)
		}
		return nil, fmt.Errorf("parse book info failed: %w", err)
	}

	// 补充基础信息
	info.BookURL = book.BookURL
	if info.Name == "" {
		info.Name = book.Name
	}
	if info.Author == "" {
		info.Author = book.Author
	}
	if info.CoverURL == "" {
		info.CoverURL = book.CoverURL
	}

	if wb.onInfoDone != nil {
		wb.onInfoDone(source, info, nil)
	}
	return info, nil
}

// fetchChapterList 获取目录
func (wb *WebBook) fetchChapterList(ctx context.Context, source *BookSource, info *BookInfo) (*BookChapterList, error) {
	if wb.onChapterStart != nil {
		wb.onChapterStart(source, info)
	}

	// 等待书源频率限制
	wb.concurrent.SourceWait(source.ID)

	// 构建目录URL（tocUrl 可能是 URL 模板，也可能是 Legado 选择器规则）
	chapterListURL, chapterMethod, chapterBody, err := wb.resolveChapterListURL(ctx, source, info)
	if err != nil {
		if wb.onChapterDone != nil {
			wb.onChapterDone(source, nil, err)
		}
		return nil, fmt.Errorf("build chapter list URL failed: %w", err)
	}

	// 发起请求
	resp, err := wb.fetch(ctx, source, chapterListURL, chapterMethod, chapterBody)
	if err != nil {
		if wb.onChapterDone != nil {
			wb.onChapterDone(source, nil, err)
		}
		return nil, fmt.Errorf("fetch chapter list failed: %w", err)
	}

	// 解析结果
	var list *BookChapterList
	if source.ChapterListRule != "" {
		list, err = wb.parseChapterListWithRules(ctx, source, resp.Body, resp.URL, info.Name, info.BookURL, source.ID, source.Name)
	} else {
		list, err = NewChapterListParser("", "", "").ParseHTML(resp.Body, resp.URL, info.Name, info.BookURL, source.ID, source.Name)
	}
	if err != nil {
		if wb.onChapterDone != nil {
			wb.onChapterDone(source, nil, err)
		}
		return nil, fmt.Errorf("parse chapter list failed: %w", err)
	}

	if wb.onChapterDone != nil {
		wb.onChapterDone(source, list, nil)
	}
	return list, nil
}

// fetchContent 获取正文
func (wb *WebBook) fetchContent(ctx context.Context, source *BookSource, info *BookInfo, chapter *BookChapter) (*BookContent, error) {
	if wb.onContentStart != nil {
		wb.onContentStart(source, chapter)
	}

	// 等待书源频率限制
	wb.concurrent.SourceWait(source.ID)

	// 构建正文URL
	contentURL, contentMethod, contentBody, err := wb.buildContentURL(source, info, chapter)
	if err != nil {
		if wb.onContentDone != nil {
			wb.onContentDone(source, nil, err)
		}
		return nil, fmt.Errorf("build content URL failed: %w", err)
	}

	// 发起请求
	resp, err := wb.fetch(ctx, source, contentURL, contentMethod, contentBody)
	if err != nil {
		if wb.onContentDone != nil {
			wb.onContentDone(source, nil, err)
		}
		return nil, fmt.Errorf("fetch content failed: %w", err)
	}

	// 解析结果（支持 nextContentUrl 分页拼接）
	var content *BookContent
	pageBody := resp.Body
	pageURL := resp.URL
	for page := 0; page < 50; page++ {
		var pageContent *BookContent
		if source.ContentRule != "" {
			pageContent, err = wb.parseContentWithRules(ctx, source, pageBody, pageURL,
				chapter.Title, chapter.URL, info.Name, info.BookURL,
				source.ID, source.Name, chapter.Index)
		} else {
			pageContent, err = NewContentParser("").ParseHTML(
				pageBody, pageURL,
				chapter.Title, chapter.URL,
				info.Name, info.BookURL,
				source.ID, source.Name,
				chapter.Index,
			)
		}
		if err != nil {
			if wb.onContentDone != nil {
				wb.onContentDone(source, nil, err)
			}
			return nil, fmt.Errorf("parse content failed: %w", err)
		}
		if pageContent == nil || (strings.TrimSpace(pageContent.Content) == "" && len(pageContent.Images) == 0) {
			break
		}
		if content == nil {
			content = pageContent
		} else {
			content.Content += "\n" + pageContent.Content
			if len(pageContent.Images) > 0 {
				content.Images = append(content.Images, pageContent.Images...)
			}
			content.NextURL = pageContent.NextURL
			content.WordCount = len(strings.ReplaceAll(content.Content, " ", ""))
		}
		nextURL := strings.TrimSpace(pageContent.NextURL)
		if nextURL == "" || nextURL == pageURL {
			break
		}
		pageURL = nextURL
		var nextResp *Response
		nextResp, err = wb.fetch(ctx, source, nextURL, contentMethod, contentBody)
		if err != nil {
			break
		}
		pageBody = nextResp.Body
	}

	if wb.onContentDone != nil {
		wb.onContentDone(source, content, nil)
	}
	if content != nil && len(content.Images) > 0 {
		content.ReaderMode = "comic"
	} else if content != nil {
		content.ReaderMode = "text"
	}
	return content, nil
}

// =============================================================================
// URL 构建
// =============================================================================

// buildBookInfoURL 构建书籍详情 URL
func (wb *WebBook) buildBookInfoURL(source *BookSource, book *Book) (string, string, string, error) {
	return wb.buildSourceURL(source, source.BookInfoURL, book.BookURL, nil)
}

// buildChapterListURL 构建目录 URL
func (wb *WebBook) buildChapterListURL(source *BookSource, info *BookInfo) (string, string, string, error) {
	return wb.buildSourceURL(source, source.ChapterListURL, info.BookURL, nil)
}

// buildContentURL 构建正文 URL
func (wb *WebBook) buildContentURL(source *BookSource, info *BookInfo, chapter *BookChapter) (string, string, string, error) {
	extra := map[string]string{
		"{{chapterUrl}}":  chapter.URL,
		"{{chapter_url}}": chapter.URL,
		"{{chapterURL}}":  chapter.URL,
		"{chapterUrl}":    chapter.URL,
		"{{bookUrl}}":     info.BookURL,
		"{{book_url}}":    info.BookURL,
		"{{bookURL}}":     info.BookURL,
		"{bookUrl}":       info.BookURL,
	}
	return wb.buildSourceURL(source, source.ContentURL, chapter.URL, extra)
}

// replaceURLTemplate 替换URL模板中的占位符
// 支持 {{bookUrl}} / {{chapterUrl}} / {{url}} / {{key}} 等占位符
func replaceURLTemplate(template, value string) string {
	if template == "" {
		return value
	}
	result := template
	// 通用占位符
	result = strings.ReplaceAll(result, "{{url}}", value)
	result = strings.ReplaceAll(result, "{{key}}", value)
	// 书籍相关占位符
	result = strings.ReplaceAll(result, "{{bookUrl}}", value)
	result = strings.ReplaceAll(result, "{{book_url}}", value)
	result = strings.ReplaceAll(result, "{{bookURL}}", value)
	// 章节相关占位符
	result = strings.ReplaceAll(result, "{{chapterUrl}}", value)
	result = strings.ReplaceAll(result, "{{chapter_url}}", value)
	result = strings.ReplaceAll(result, "{{chapterURL}}", value)
	return result
}

// =============================================================================
// 响应类型
// =============================================================================

// Response HTTP 响应
type Response struct {
	URL    string
	Body   string
	Header http.Header
}
