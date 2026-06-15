package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"reader-go/internal/booksource"
	"reader-go/internal/localbook"
	"reader-go/internal/replace"
	"reader-go/internal/rss"
	"reader-go/internal/shelf"
	"reader-go/internal/utils"
	"reader-go/internal/webbook"

	"github.com/gin-gonic/gin"
)

const maxLocalBookFileSize = 50 * 1024 * 1024 // 50MB

// App 应用上下文，持有所有服务
type App struct {
	BookSourceSvc *booksource.Service
	ShelfSvc      *shelf.Service
	ReplaceSvc    *replace.Service
	WebBook       *webbook.WebBook
	LocalBookSvc  *localbook.Service
	RSSSvc        *rss.Service
}

// ==================== 通用响应结构 ====================

type resp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(200, resp{Code: 0, Message: "ok", Data: data})
}

func okList(c *gin.Context, list interface{}, total ...int64) {
	// 确保 list 不为 nil，转换为空数组
	if list == nil {
		list = []interface{}{}
	}
	// 空 slice 序列化为 [] 而非 null
	switch v := list.(type) {
	case []*booksource.BookSource:
		if v == nil {
			list = []*booksource.BookSource{}
		}
	case []*rss.Feed:
		if v == nil {
			list = []*rss.Feed{}
		}
	case []*shelf.ShelfBook:
		if v == nil {
			list = []*shelf.ShelfBook{}
		}
	case []*replace.ReplaceRule:
		if v == nil {
			list = []*replace.ReplaceRule{}
		}
	case []*rss.FeedItem:
		if v == nil {
			list = []*rss.FeedItem{}
		}
	}
	if len(total) > 0 {
		ok(c, gin.H{"list": list, "total": total[0]})
	} else {
		ok(c, list)
	}
}

func errResp(c *gin.Context, code int, msg string) {
	c.JSON(200, resp{Code: code, Message: msg})
}

func errRespInternal(c *gin.Context, err error) {
	logError(err)
	c.JSON(200, resp{Code: -1, Message: "internal server error"})
}

// ==================== Book Sources ====================

func listBookSources(c *gin.Context) {
	sources, err := app.BookSourceSvc.List()
	if err != nil {
		errRespInternal(c, err)
		return
	}
	okList(c, sources)
}

func createBookSource(c *gin.Context) {
	var bs booksource.BookSource
	if err := c.ShouldBindJSON(&bs); err != nil {
		errResp(c, 400, "invalid request: "+err.Error())
		return
	}
	if bs.Name == "" {
		errResp(c, 400, "name is required")
		return
	}
	if err := app.BookSourceSvc.Create(&bs); err != nil {
		errRespInternal(c, err)
		return
	}
	reloadWebBookSources()
	ok(c, bs)
}

func updateBookSource(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}
	var bs booksource.BookSource
	if err := c.ShouldBindJSON(&bs); err != nil {
		errResp(c, 400, "invalid request: "+err.Error())
		return
	}
	bs.ID = id
	existing, err := app.BookSourceSvc.GetByID(id)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	if existing == nil {
		errResp(c, 404, "book source not found")
		return
	}
	if err := app.BookSourceSvc.Update(&bs); err != nil {
		errRespInternal(c, err)
		return
	}
	reloadWebBookSources()
	ok(c, bs)
}

func deleteBookSource(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}
	if err := app.BookSourceSvc.Delete(id); err != nil {
		errRespInternal(c, err)
		return
	}
	reloadWebBookSources()
	ok(c, gin.H{"deleted": true})
}

func importBookSources(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		errResp(c, 400, "read body: "+err.Error())
		return
	}

	var sources []*booksource.BookSource
	if bytesContains(body, "bookSourceName") || bytesContains(body, "bookSourceUrl") {
		parsed, parseResult := booksource.ParseBookSourceCollection(body)
		sources = parsed
		agentDebugLog("handlers.go:importBookSources", "legado parse", "B", "pre-fix", map[string]interface{}{
			"parsed": len(parsed), "total": parseResult.Total, "failed": parseResult.Failed,
		})
	} else {
		if err := json.Unmarshal(body, &sources); err != nil {
			errResp(c, 400, "invalid request: "+err.Error())
			return
		}
	}

	result := app.BookSourceSvc.ImportWithResult(sources)
	agentDebugLog("handlers.go:importBookSources", "import result", "A", "pre-fix", map[string]interface{}{
		"imported": result.Success, "failed": result.Failed, "total": result.Total,
		"firstError": firstError(result.Errors),
	})
	reloadWebBookSources()
	ok(c, gin.H{
		"imported": result.Success,
		"failed":   result.Failed,
		"total":    result.Total,
		"errors":   result.Errors,
	})
}

func bytesContains(b []byte, sub string) bool {
	return strings.Contains(string(b), sub)
}

func firstError(errs []string) string {
	if len(errs) == 0 {
		return ""
	}
	return errs[0]
}

// importBookSourceCollection handles POST /api/bookSources/import/collection - import from 书源合集 URL
func importBookSourceCollection(c *gin.Context) {
	var input struct {
		URL             string `json:"url" binding:"required"`
		EnableOnlyNonJS *bool  `json:"enableOnlyNonJS"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		errResp(c, 400, "url is required")
		return
	}
	enableOnlyNonJS := true
	if input.EnableOnlyNonJS != nil {
		enableOnlyNonJS = *input.EnableOnlyNonJS
	}

	// Fetch the JSON from URL
	data, err := httpGet(input.URL)
	if err != nil {
		errResp(c, 400, "failed to fetch URL: "+err.Error())
		return
	}

	// Parse and convert
	bookSources, result := booksource.ParseBookSourceCollection(data)
	if bookSources == nil || len(bookSources) == 0 {
		errResp(c, 400, "failed to parse book source collection: "+strings.Join(result.Errors, "; "))
		return
	}

	enableStats := booksource.ApplyEnablePolicy(bookSources, enableOnlyNonJS)

	// 合集导入为全量替换
	_ = app.BookSourceSvc.DeleteAll()

	importResult := app.BookSourceSvc.ImportWithResult(bookSources)
	agentDebugLog("handlers.go:importBookSourceCollection", "collection import", "A", "pre-fix", map[string]interface{}{
		"imported": importResult.Success, "failed": importResult.Failed, "total": importResult.Total,
		"enabled": enableStats.Enabled, "jsRequired": enableStats.JSRequired,
		"firstError": firstError(importResult.Errors),
	})
	reloadWebBookSources()

	ok(c, gin.H{
		"imported":     importResult.Success,
		"total":        result.Total,
		"success":      result.Success,
		"failed":       importResult.Failed,
		"errors":       append(result.Errors, importResult.Errors...),
		"enabled":      enableStats.Enabled,
		"disabled":     enableStats.Disabled,
		"jsRequired":   enableStats.JSRequired,
		"nonJs":        enableStats.NonJS,
		"enablePolicy": enableOnlyNonJS,
	})
}

// batchSetBookSourceEnabled handles POST /api/bookSources/batch/enable
func batchSetBookSourceEnabled(c *gin.Context) {
	var input struct {
		Target          string `json:"target" binding:"required"` // js | nonJs | all
		Enabled         *bool  `json:"enabled" binding:"required"`
		EnableOnlyNonJS *bool  `json:"enableOnlyNonJS"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		errResp(c, 400, "target and enabled are required")
		return
	}

	var updated int
	var stats booksource.EnablePolicyResult
	var err error

	if input.EnableOnlyNonJS != nil && *input.EnableOnlyNonJS {
		updated, stats, err = app.BookSourceSvc.ApplyEnablePolicyExisting(true)
	} else {
		switch input.Target {
		case "js", "nonJs", "all":
		default:
			errResp(c, 400, "target must be js, nonJs, or all")
			return
		}
		updated, stats, err = app.BookSourceSvc.BatchSetEnabledByTarget(input.Target, *input.Enabled)
	}
	if err != nil {
		errRespInternal(c, err)
		return
	}
	reloadWebBookSources()
	ok(c, gin.H{
		"updated":    updated,
		"enabled":    stats.Enabled,
		"disabled":   stats.Disabled,
		"jsRequired": stats.JSRequired,
		"nonJs":      stats.NonJS,
	})
}

// importRssSourceCollection handles POST /api/rss/import/collection - import from 订阅源合集 URL
func importRssSourceCollection(c *gin.Context) {
	var input struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		errResp(c, 400, "url is required")
		return
	}

	// Fetch the JSON from URL
	data, err := httpGet(input.URL)
	if err != nil {
		errResp(c, 400, "failed to fetch URL: "+err.Error())
		return
	}

	// Parse and convert
	feeds, result := rss.ParseRssSourceCollection(data)
	if feeds == nil || len(feeds) == 0 {
		errResp(c, 400, "failed to parse RSS source collection: "+strings.Join(result.Errors, "; "))
		return
	}

	// Import to database (upsert by feed_url)
	var count int
	for _, feed := range feeds {
		if err := app.RSSSvc.UpsertFeed(feed); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", feed.Title, err))
			continue
		}
		count++
	}

	ok(c, gin.H{
		"imported": count,
		"total":    result.Total,
		"success":  result.Success,
		"failed":   result.Failed,
		"errors":   result.Errors,
	})
}

// httpGet fetches content from a URL and returns it as a byte slice.
// Includes SSRF protection to prevent requests to private/internal addresses.
func httpGet(url string) ([]byte, error) {
	if err := utils.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ==================== Search ====================

func searchBooks(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		errResp(c, 400, "query parameter 'q' is required")
		return
	}
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()
	result, err := app.WebBook.SearchBook(ctx, q)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		errRespInternal(c, err)
		return
	}

	books := result.Books
	if len(books) > limit {
		books = books[:limit]
	}

	// #region agent log
	agentDebugLog("handlers.go:searchBooks", "search done", "H1", "pre-fix", map[string]interface{}{
		"query": q, "resultCount": len(result.Books), "sourceCount": len(result.SourceResults),
	})
	// #endregion

	// Convert webbook.Book to booksource.BookSearchResult
	var results []*booksource.BookSearchResult
	for _, b := range books {
		if strings.TrimSpace(b.BookURL) == "" {
			continue
		}
		sourceID, _ := strconv.ParseInt(b.SourceID, 10, 64)
		results = append(results, &booksource.BookSearchResult{
			Name:       b.Name,
			Author:     b.Author,
			BookKey:    encodeBookKey(b.SourceID, b.BookURL),
			SourceID:   sourceID,
			SourceName: b.SourceName,
			CoverURL:   b.CoverURL,
			Summary:    b.Intro,
		})
	}

	ok(c, gin.H{
		"query":   q,
		"results": results,
		"total":   len(results),
	})
}

// ==================== Book Info ====================

func getBookInfo(c *gin.Context) {
	bookKey := c.Query("bookKey")
	if bookKey == "" {
		errResp(c, 400, "parameter 'bookKey' is required")
		return
	}

	sourceID, bookURL, err := decodeBookKey(bookKey)
	if err != nil {
		errResp(c, 400, "invalid bookKey format")
		return
	}
	if strings.TrimSpace(bookURL) == "" {
		errResp(c, 400, "bookKey has empty book URL; re-search and add the book again")
		return
	}

	book := &webbook.Book{
		SourceID: sourceID,
		BookURL:  bookURL,
	}

	ctx := context.Background()
	info, err := app.WebBook.GetBookInfo(ctx, book)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	if info == nil {
		errResp(c, 404, "book info not found")
		return
	}

	ok(c, gin.H{
		"name":       info.Name,
		"author":     info.Author,
		"coverUrl":   info.CoverURL,
		"summary":    info.Intro,
		"tags":       info.Tags,
		"sourceId":   sourceID,
		"sourceName": info.SourceName,
		"bookKey":    bookKey,
	})
}

// ==================== Book TOC ====================

func getBookToc(c *gin.Context) {
	bookKey := c.Query("bookKey")
	if bookKey == "" {
		errResp(c, 400, "parameter 'bookKey' is required")
		return
	}

	sourceID, bookURL, err := decodeBookKey(bookKey)
	if err != nil {
		errResp(c, 400, "invalid bookKey format")
		return
	}
	if strings.TrimSpace(bookURL) == "" {
		errResp(c, 400, "bookKey has empty book URL; re-search and add the book again")
		return
	}

	book := &webbook.Book{
		SourceID: sourceID,
		BookURL:  bookURL,
	}

	ctx := context.Background()
	info, err := app.WebBook.GetBookInfo(ctx, book)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	if info == nil {
		errResp(c, 404, "book info not found")
		return
	}

	chapterList, err := app.WebBook.GetChapterList(ctx, info)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	if chapterList == nil {
		errResp(c, 404, "toc not found")
		return
	}

	// Convert webbook.BookChapter to booksource.Chapter
	var chapters []booksource.Chapter
	for _, ch := range chapterList.Chapters {
		chapters = append(chapters, booksource.Chapter{
			Name: ch.Title,
			URL:  ch.URL,
		})
	}

	ok(c, gin.H{
		"bookKey":  bookKey,
		"chapters": chapters,
	})
}

// ==================== Book Content ====================

func getBookContent(c *gin.Context) {
	bookKey := c.Query("bookKey")
	chapterURL := c.Query("chapter")
	if bookKey == "" {
		errResp(c, 400, "parameter 'bookKey' is required")
		return
	}

	sourceID, bookURL, err := decodeBookKey(bookKey)
	if err != nil {
		errResp(c, 400, "invalid bookKey format")
		return
	}
	if strings.TrimSpace(bookURL) == "" {
		errResp(c, 400, "bookKey has empty book URL; re-search and add the book again")
		return
	}

	book := &webbook.Book{
		SourceID: sourceID,
		BookURL:  bookURL,
	}

	ctx := context.Background()
	info, err := app.WebBook.GetBookInfo(ctx, book)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	if info == nil {
		errResp(c, 404, "book info not found")
		return
	}

	// If chapter URL is provided, fetch content directly.
	// Otherwise, fetch the chapter list and use the first chapter.
	var chapter *webbook.BookChapter
	if chapterURL != "" {
		chapter = &webbook.BookChapter{
			URL: chapterURL,
		}
	} else {
		chapterList, err := app.WebBook.GetChapterList(ctx, info)
		if err != nil {
			errRespInternal(c, err)
			return
		}
		if chapterList == nil || len(chapterList.Chapters) == 0 {
			errResp(c, 404, "no chapters found")
			return
		}
		chapter = &chapterList.Chapters[0]
	}

	content, err := app.WebBook.GetBookContent(ctx, info, chapter)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	if content == nil || (strings.TrimSpace(content.Content) == "" && len(content.Images) == 0) {
		errResp(c, 404, "content not found")
		return
	}

	// 应用替换规则
	text := applyReplaceRulesScoped(content.Content, replace.ScopeContent)

	ok(c, gin.H{
		"bookKey":    bookKey,
		"chapter":    chapter.URL,
		"content":    text,
		"images":     content.Images,
		"readerMode": content.ReaderMode,
	})
}

// ==================== Shelf ====================

func getShelf(c *gin.Context) {
	books, err := app.ShelfSvc.List()
	if err != nil {
		errRespInternal(c, err)
		return
	}
	stats, _ := app.ShelfSvc.Stats()

	ok(c, gin.H{
		"books": books,
		"stats": stats,
	})
}

func addToShelf(c *gin.Context) {
	var input struct {
		BookKey    string `json:"bookKey"`
		Name       string `json:"name"`
		Author     string `json:"author"`
		CoverURL   string `json:"coverUrl"`
		Summary    string `json:"summary"`
		SourceID   int64  `json:"sourceId"`
		SourceName string `json:"sourceName"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		errResp(c, 400, "invalid request: "+err.Error())
		return
	}
	if input.BookKey == "" {
		errResp(c, 400, "bookKey is required")
		return
	}

	sb := &shelf.ShelfBook{
		BookKey:    input.BookKey,
		Name:       input.Name,
		Author:     input.Author,
		CoverURL:   input.CoverURL,
		Summary:    input.Summary,
		SourceID:   input.SourceID,
		SourceName: input.SourceName,
	}

	if err := app.ShelfSvc.Add(sb); err != nil {
		errRespInternal(c, err)
		return
	}
	// 返回带 id 的完整记录
	saved, _ := app.ShelfSvc.GetByBookKey(input.BookKey)
	if saved != nil {
		ok(c, saved)
		return
	}
	ok(c, sb)
}

func updateShelfBook(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}

	var input shelf.ShelfBook
	if err := c.ShouldBindJSON(&input); err != nil {
		errResp(c, 400, "invalid request: "+err.Error())
		return
	}
	input.ID = id

	existing, err := app.ShelfSvc.GetByID(id)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	if existing == nil {
		errResp(c, 404, "shelf book not found")
		return
	}

	if err := app.ShelfSvc.Update(&input); err != nil {
		errRespInternal(c, err)
		return
	}
	ok(c, input)
}

func removeFromShelf(c *gin.Context) {
	// 支持 ?bookKey= 删除（bookKey 可能含特殊字符）
	if bookKey := c.Query("bookKey"); bookKey != "" {
		if err := app.ShelfSvc.RemoveByBookKey(bookKey); err != nil {
			errRespInternal(c, err)
			return
		}
		ok(c, gin.H{"deleted": true})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}
	if err := app.ShelfSvc.Remove(id); err != nil {
		errRespInternal(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func updateShelfProgress(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}

	var input struct {
		CurrentChapter string `json:"currentChapter"`
		ChapterIndex   int    `json:"chapterIndex"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		errResp(c, 400, "invalid request: "+err.Error())
		return
	}
	if input.CurrentChapter == "" {
		errResp(c, 400, "currentChapter is required")
		return
	}

	book, err := app.ShelfSvc.GetByID(id)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	if book == nil {
		errResp(c, 404, "shelf book not found")
		return
	}

	if err := app.ShelfSvc.UpdateProgress(book.BookKey, input.CurrentChapter); err != nil {
		errRespInternal(c, err)
		return
	}
	ok(c, gin.H{
		"id":             id,
		"bookKey":        book.BookKey,
		"currentChapter": input.CurrentChapter,
		"chapterIndex":   input.ChapterIndex,
	})
}

// ==================== Replace Rules ====================

func listReplaceRules(c *gin.Context) {
	rules, err := app.ReplaceSvc.List()
	if err != nil {
		errRespInternal(c, err)
		return
	}
	okList(c, rules)
}

func createReplaceRule(c *gin.Context) {
	var r replace.ReplaceRule
	if err := c.ShouldBindJSON(&r); err != nil {
		errResp(c, 400, "invalid request: "+err.Error())
		return
	}
	if r.Name == "" || r.Pattern == "" {
		errResp(c, 400, "name and pattern are required")
		return
	}
	if err := app.ReplaceSvc.Create(&r); err != nil {
		errRespInternal(c, err)
		return
	}
	ok(c, r)
}

func updateReplaceRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}
	var r replace.ReplaceRule
	if err := c.ShouldBindJSON(&r); err != nil {
		errResp(c, 400, "invalid request: "+err.Error())
		return
	}
	r.ID = id

	existing, err := app.ReplaceSvc.GetByID(id)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	if existing == nil {
		errResp(c, 404, "replace rule not found")
		return
	}

	if err := app.ReplaceSvc.Update(&r); err != nil {
		errRespInternal(c, err)
		return
	}
	ok(c, r)
}

func deleteReplaceRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}
	if err := app.ReplaceSvc.Delete(id); err != nil {
		errRespInternal(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

// applyReplaceRules 应用所有启用的替换规则
func applyReplaceRules(text string) string {
	rules, err := app.ReplaceSvc.ListEnabled()
	if err != nil || len(rules) == 0 {
		return text
	}
	result := text
	for _, r := range rules {
		result = replace.ApplyRule(r, result)
	}
	return result
}

// ==================== Local Books ====================

// uploadLocalBook handles POST /api/localBooks - upload a local book file.
func uploadLocalBook(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		errResp(c, 400, "file is required: "+err.Error())
		return
	}
	defer file.Close()

	// Check file size before processing
	if header.Size > maxLocalBookFileSize {
		errResp(c, 400, fmt.Sprintf("file size exceeds maximum allowed size of %d bytes", maxLocalBookFileSize))
		return
	}

	if app.LocalBookSvc == nil {
		errResp(c, 500, "local book service not initialized")
		return
	}

	book, err := app.LocalBookSvc.Store(header.Filename, file)
	if err != nil {
		errRespInternal(c, err)
		return
	}

	ok(c, book)
}

// listLocalBooks handles GET /api/localBooks - list uploaded local books.
func listLocalBooks(c *gin.Context) {
	if app.LocalBookSvc == nil {
		errResp(c, 500, "local book service not initialized")
		return
	}

	books := app.LocalBookSvc.List()
	okList(c, books)
}

// getLocalBook handles GET /api/localBooks/:id - get local book info.
func getLocalBook(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		errResp(c, 400, "id is required")
		return
	}

	if app.LocalBookSvc == nil {
		errResp(c, 500, "local book service not initialized")
		return
	}

	book, err := app.LocalBookSvc.Get(id)
	if err != nil {
		errResp(c, 404, err.Error())
		return
	}

	ok(c, book)
}

// getLocalBookContent handles GET /api/localBooks/:id/content - get local book content.
func getLocalBookContent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		errResp(c, 400, "id is required")
		return
	}

	if app.LocalBookSvc == nil {
		errResp(c, 500, "local book service not initialized")
		return
	}

	chapterStr := c.Query("chapter")
	if chapterStr == "" {
		// Return full text if no chapter specified (for txt only)
		content, err := app.LocalBookSvc.GetFullText(id)
		if err != nil {
			errRespInternal(c, err)
			return
		}
		ok(c, gin.H{
			"id":      id,
			"content": content,
		})
		return
	}

	chapterIndex, err := strconv.Atoi(chapterStr)
	if err != nil || chapterIndex < 0 {
		errResp(c, 400, "invalid chapter index")
		return
	}

	content, err := app.LocalBookSvc.GetContent(id, chapterIndex)
	if err != nil {
		errRespInternal(c, err)
		return
	}

	ok(c, gin.H{
		"id":      id,
		"chapter": chapterIndex,
		"content": content,
	})
}

// deleteLocalBook handles DELETE /api/localBooks/:id - remove a local book.
func deleteLocalBook(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		errResp(c, 400, "id is required")
		return
	}

	if app.LocalBookSvc == nil {
		errResp(c, 500, "local book service not initialized")
		return
	}

	if err := app.LocalBookSvc.Remove(id); err != nil {
		errRespInternal(c, err)
		return
	}

	ok(c, gin.H{"deleted": true})
}

// ==================== RSS ====================

// listRSSFeeds handles GET /api/rss/feeds - list all RSS feeds
func listRSSFeeds(c *gin.Context) {
	feeds, err := app.RSSSvc.ListFeeds()
	if err != nil {
		errRespInternal(c, err)
		return
	}
	okList(c, feeds)
}

// createRSSFeed handles POST /api/rss/feeds - add a new RSS feed
func createRSSFeed(c *gin.Context) {
	var input struct {
		FeedURL string `json:"feedUrl"`
		URL     string `json:"url"`
		Group   string `json:"group"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		errResp(c, 400, "invalid request: "+err.Error())
		return
	}
	feedURL := input.FeedURL
	if feedURL == "" {
		feedURL = input.URL
	}
	if feedURL == "" {
		errResp(c, 400, "feedUrl is required")
		return
	}

	// Parse the feed first to get metadata (with HTML autodiscovery fallback)
	parser := rss.NewParser()
	result, resolvedURL, err := parser.ParseWithDiscoveryCtx(c.Request.Context(), feedURL)
	if err != nil {
		errResp(c, 400, "failed to parse feed: "+err.Error())
		return
	}
	if resolvedURL != "" {
		feedURL = resolvedURL
	}

	feed := &rss.Feed{
		Title:       result.Title,
		Link:        result.Link,
		Description: result.Description,
		FeedURL:     feedURL,
		SiteURL:     result.Link,
		FeedType:    result.FeedType,
		Group:       input.Group,
		Enabled:     true,
	}

	if err := app.RSSSvc.AddFeed(feed); err != nil {
		errRespInternal(c, err)
		return
	}
	ok(c, feed)
}

// deleteRSSFeed handles DELETE /api/rss/feeds/:id - delete an RSS feed
func deleteRSSFeed(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}
	if err := app.RSSSvc.DeleteFeed(id); err != nil {
		errRespInternal(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

// getRSSFeedItems handles GET /api/rss/feeds/:id/items - get items from a feed
func getRSSFeedItems(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", c.DefaultQuery("limit", "20")))

	items, total, err := app.RSSSvc.GetItems(id, page, pageSize)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	ok(c, gin.H{
		"items":    items,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
		"hasMore":  page*pageSize < total,
	})
}

// fetchRSSFeed handles POST /api/rss/feeds/:id/fetch - manually refresh a feed
func fetchRSSFeed(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}

	feed, err := app.RSSSvc.GetFeed(id)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	if feed == nil {
		errResp(c, 404, "feed not found")
		return
	}

	newItems, err := app.RSSSvc.FetchFeedCtx(c.Request.Context(), feed)
	if err != nil {
		logError(fmt.Errorf("fetch feed %d (%s): %w", id, feed.Title, err))
		c.JSON(200, resp{Code: -1, Message: "fetch failed: " + err.Error()})
		return
	}

	ok(c, gin.H{
		"feed":     feed,
		"newItems": len(newItems),
	})
}

// previewRSSFeed handles POST /api/rss/feeds/:id/preview - preview parsed items without writing DB items
func previewRSSFeed(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}

	feed, err := app.RSSSvc.GetFeed(id)
	if err != nil {
		errRespInternal(c, err)
		return
	}
	if feed == nil {
		errResp(c, 404, "feed not found")
		return
	}

	var input struct {
		ParseRules string `json:"parseRules"`
		FeedURL    string `json:"feedUrl"`
		SiteURL    string `json:"siteUrl"`
		Limit      int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&input); err != nil && err.Error() != "EOF" {
		errResp(c, 400, "invalid request: "+err.Error())
		return
	}

	preview := *feed
	if strings.TrimSpace(input.ParseRules) != "" {
		preview.ParseRules = input.ParseRules
	}
	if strings.TrimSpace(input.FeedURL) != "" {
		preview.FeedURL = input.FeedURL
	}
	if strings.TrimSpace(input.SiteURL) != "" {
		preview.SiteURL = input.SiteURL
	}

	result, resolvedURL, err := app.RSSSvc.PreviewFeed(c.Request.Context(), &preview)
	if err != nil {
		logError(fmt.Errorf("preview feed %d (%s): %w", id, feed.Title, err))
		c.JSON(200, resp{Code: -1, Message: "preview failed: " + err.Error()})
		return
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > len(result.Items) {
		limit = len(result.Items)
	}

	ok(c, gin.H{
		"feed":        preview,
		"resolvedUrl": resolvedURL,
		"total":       len(result.Items),
		"items":       result.Items[:limit],
	})
}

// markRSSItemRead handles PUT /api/rss/items/:id/read - mark item as read
func markRSSItemRead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}
	if err := app.RSSSvc.MarkAsRead(id); err != nil {
		errRespInternal(c, err)
		return
	}
	ok(c, gin.H{"marked": true})
}

// toggleRSSItemStar handles PUT /api/rss/items/:id/star - toggle starred status
func toggleRSSItemStar(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid id")
		return
	}
	if err := app.RSSSvc.ToggleStar(id); err != nil {
		errRespInternal(c, err)
		return
	}
	ok(c, gin.H{"toggled": true})
}

// ==================== Helpers ====================

// encodeBookKey encodes sourceID and bookURL into a bookKey string.
// Format: "sourceID::bookURL" (double colon as separator).
func encodeBookKey(sourceID, bookURL string) string {
	return sourceID + "::" + bookURL
}

// decodeBookKey decodes a bookKey into sourceID and bookURL.
func decodeBookKey(bookKey string) (sourceID, bookURL string, err error) {
	parts := strings.SplitN(bookKey, "::", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid bookKey format")
	}
	return parts[0], parts[1], nil
}

// ==================== Debug Book Source (SSE) ====================

// debugEvent represents a single SSE event sent to the client.
type debugEvent struct {
	Step    string      `json:"step"`
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// sendDebugEvent writes a single SSE event to the response writer.
func sendDebugEvent(c *gin.Context, event debugEvent) {
	data, _ := json.Marshal(event)
	c.Writer.WriteString("data: " + string(data) + "\n\n")
	c.Writer.Flush()
}

// debugBookSource streams debug information for a book source via SSE.
func debugBookSource(c *gin.Context) {
	sourceID, err := strconv.ParseInt(c.Query("sourceId"), 10, 64)
	if err != nil {
		errResp(c, 400, "invalid sourceId")
		return
	}

	debugType := c.Query("type")
	if debugType == "" {
		errResp(c, 400, "parameter 'type' is required")
		return
	}

	src, err := app.BookSourceSvc.GetByID(sourceID)
	if err != nil || src == nil {
		errResp(c, 404, "book source not found")
		return
	}

	wbs := convertBookSource(src)

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(200)

	// Create a temporary WebBook with debug callbacks
	wb := webbook.NewWebBook().AddSource(wbs)

	ctx := context.Background()

	switch debugType {
	case "search":
		query := c.Query("query")
		if query == "" {
			sendDebugEvent(c, debugEvent{Step: "search", Status: "error", Message: "query parameter is required"})
			return
		}

		wb.WithSearchCallbacks(
			func(source *webbook.BookSource, q string) {
				sendDebugEvent(c, debugEvent{Step: "search", Status: "start", Message: "开始搜索...", Data: gin.H{"source": source.Name, "query": q}})
			},
			func(source *webbook.BookSource, result *webbook.BookList, err error) {
				if err != nil {
					sendDebugEvent(c, debugEvent{Step: "search", Status: "error", Message: "搜索失败: " + err.Error()})
					return
				}
				sendDebugEvent(c, debugEvent{Step: "search", Status: "done", Message: fmt.Sprintf("搜索完成，找到 %d 本书", len(result.Books)), Data: result})
			},
		)

		books, err := wb.SearchBook(ctx, query)
		if err != nil {
			sendDebugEvent(c, debugEvent{Step: "search", Status: "error", Message: "搜索请求失败: " + err.Error()})
			return
		}
		_ = books

	case "info":
		bookUrl := c.Query("bookUrl")
		if bookUrl == "" {
			sendDebugEvent(c, debugEvent{Step: "info", Status: "error", Message: "bookUrl parameter is required"})
			return
		}

		book := &webbook.Book{BookURL: bookUrl, SourceID: wbs.ID}

		wb.WithInfoCallbacks(
			func(source *webbook.BookSource, b *webbook.Book) {
				sendDebugEvent(c, debugEvent{Step: "info", Status: "start", Message: "开始获取书籍详情...", Data: gin.H{"source": source.Name, "bookUrl": b.BookURL}})
			},
			func(source *webbook.BookSource, info *webbook.BookInfo, err error) {
				if err != nil {
					sendDebugEvent(c, debugEvent{Step: "info", Status: "error", Message: "获取详情失败: " + err.Error()})
					return
				}
				sendDebugEvent(c, debugEvent{Step: "info", Status: "done", Message: "获取详情完成", Data: info})
			},
		)

		info, err := wb.GetBookInfo(ctx, book)
		if err != nil {
			sendDebugEvent(c, debugEvent{Step: "info", Status: "error", Message: "获取详情请求失败: " + err.Error()})
			return
		}
		_ = info

	case "toc":
		bookUrl := c.Query("bookUrl")
		if bookUrl == "" {
			sendDebugEvent(c, debugEvent{Step: "toc", Status: "error", Message: "bookUrl parameter is required"})
			return
		}

		info := &webbook.BookInfo{BookURL: bookUrl, SourceID: wbs.ID}

		wb.WithChapterCallbacks(
			func(source *webbook.BookSource, b *webbook.BookInfo) {
				sendDebugEvent(c, debugEvent{Step: "toc", Status: "start", Message: "开始获取目录...", Data: gin.H{"source": source.Name, "bookUrl": b.BookURL}})
			},
			func(source *webbook.BookSource, list *webbook.BookChapterList, err error) {
				if err != nil {
					sendDebugEvent(c, debugEvent{Step: "toc", Status: "error", Message: "获取目录失败: " + err.Error()})
					return
				}
				sendDebugEvent(c, debugEvent{Step: "toc", Status: "done", Message: fmt.Sprintf("获取目录完成，共 %d 章", len(list.Chapters)), Data: list})
			},
		)

		list, err := wb.GetChapterList(ctx, info)
		if err != nil {
			sendDebugEvent(c, debugEvent{Step: "toc", Status: "error", Message: "获取目录请求失败: " + err.Error()})
			return
		}
		_ = list

	case "content":
		bookUrl := c.Query("bookUrl")
		chapterUrl := c.Query("chapterUrl")
		if bookUrl == "" || chapterUrl == "" {
			sendDebugEvent(c, debugEvent{Step: "content", Status: "error", Message: "bookUrl and chapterUrl parameters are required"})
			return
		}

		info := &webbook.BookInfo{BookURL: bookUrl, SourceID: wbs.ID}
		chapter := &webbook.BookChapter{URL: chapterUrl}

		wb.WithContentCallbacks(
			func(source *webbook.BookSource, ch *webbook.BookChapter) {
				sendDebugEvent(c, debugEvent{Step: "content", Status: "start", Message: "开始获取正文...", Data: gin.H{"source": source.Name, "chapterUrl": ch.URL}})
			},
			func(source *webbook.BookSource, content *webbook.BookContent, err error) {
				if err != nil {
					sendDebugEvent(c, debugEvent{Step: "content", Status: "error", Message: "获取正文失败: " + err.Error()})
					return
				}
				sendDebugEvent(c, debugEvent{Step: "content", Status: "done", Message: "获取正文完成", Data: content})
			},
		)

		content, err := wb.GetBookContent(ctx, info, chapter)
		if err != nil {
			sendDebugEvent(c, debugEvent{Step: "content", Status: "error", Message: "获取正文请求失败: " + err.Error()})
			return
		}
		_ = content

	default:
		sendDebugEvent(c, debugEvent{Step: "validate", Status: "error", Message: "unknown debug type: " + debugType})
	}
}

