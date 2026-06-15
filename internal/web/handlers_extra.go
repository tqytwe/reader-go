package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"reader-go/internal/booksource"
	"reader-go/internal/replace"
	"reader-go/internal/shelf"
	"reader-go/internal/webbook"

	"github.com/gin-gonic/gin"
)

// GET /api/explore?sourceId=&tab=&page=&search=&category=
func getExplore(c *gin.Context) {
	sourceID, _ := strconv.ParseInt(c.Query("sourceId"), 10, 64)
	tab := c.Query("tab")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "30"))
	search := c.Query("search")
	category := c.Query("category")

	if page <= 0 {
		page = 1
	}
	// pageSize = 0 表示不分页，返回全部
	if pageSize < 0 {
		pageSize = 30
	}
	if pageSize > 100 {
		pageSize = 100
	}

	if sourceID == 0 {
		errResp(c, 400, "sourceId is required")
		return
	}
	src, err := app.BookSourceSvc.GetByID(sourceID)
	if err != nil || src == nil {
		errResp(c, 404, "book source not found")
		return
	}
	if src.ExploreURL == "" {
		ok(c, gin.H{"sourceId": sourceID, "tab": tab, "tabs": []any{}, "items": []any{}, "books": []any{}, "page": page, "pageSize": pageSize, "hasMore": false, "total": 0})
		return
	}

	wbSrc := convertBookSource(src)
	result, err := app.WebBook.ExploreSearch(c.Request.Context(), wbSrc, tab, page, pageSize, search, category)
	if err != nil {
		logError(fmt.Errorf("explore source %d: %w", sourceID, err))
		c.JSON(200, resp{Code: -1, Message: "explore failed: " + err.Error()})
		return
	}

	items := make([]gin.H, 0, len(result.Books))
	for _, b := range result.Books {
		items = append(items, gin.H{
			"name":       b.Name,
			"author":     b.Author,
			"intro":      b.Intro,
			"coverUrl":   b.CoverURL,
			"bookUrl":    b.BookURL,
			"bookKey":    encodeBookKey(b.SourceID, b.BookURL),
			"sourceId":   b.SourceID,
			"sourceName": b.SourceName,
		})
	}

	ok(c, gin.H{
		"sourceId":   sourceID,
		"sourceName": src.Name,
		"tab":        result.Tab,
		"url":        result.URL,
		"tabs":       result.Tabs,
		"books":      items,
		"items":      items,
		"page":       result.Page,
		"pageSize":   result.PageSize,
		"hasMore":    result.HasMore,
		"total":      result.Total,
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GET /api/book/alternates?bookKey=
func getBookAlternates(c *gin.Context) {
	bookKey := c.Query("bookKey")
	if bookKey == "" {
		errResp(c, 400, "bookKey is required")
		return
	}
	currentSourceID, _, err := decodeBookKey(bookKey)
	if err != nil {
		errResp(c, 400, "invalid bookKey")
		return
	}

	shelfBook, _ := app.ShelfSvc.GetByBookKey(bookKey)
	targetName := ""
	targetAuthor := ""
	currentChapter := ""
	if shelfBook != nil {
		targetName = shelfBook.Name
		targetAuthor = shelfBook.Author
		currentChapter = shelfBook.CurrentChapter
	}

	sources, _ := app.BookSourceSvc.ListEnabled()
	ctx := c.Request.Context()
	var candidates []gin.H

	for _, src := range sources {
		srcIDStr := strconv.FormatInt(src.ID, 10)
		if srcIDStr == currentSourceID {
			continue
		}

		query := targetName
		if query == "" {
			continue
		}

		books, searchErr := app.WebBook.SearchSingleSource(ctx, srcIDStr, query)
		if searchErr != nil || len(books) == 0 {
			continue
		}

		var best *struct {
			book  webbook.Book
			score float64
		}
		for _, b := range books {
			score := bookMatchScore(targetName, targetAuthor, b.Name, b.Author)
			if score < 0.45 {
				continue
			}
			if best == nil || score > best.score {
				best = &struct {
					book  webbook.Book
					score float64
				}{book: b, score: score}
			}
		}
		if best == nil {
			continue
		}

		candidateKey := encodeBookKey(best.book.SourceID, best.book.BookURL)
		chapterScore := 0.0
		chapterIndex := -1

		if currentChapter != "" {
			info := &webbook.BookInfo{
					Name:       best.book.Name,
					BookURL:    best.book.BookURL,
					SourceID:   best.book.SourceID,
					SourceName: best.book.SourceName,
				}
				if toc, tocErr := app.WebBook.GetChapterList(ctx, info); tocErr == nil && toc != nil {
					titles := make([]string, 0, len(toc.Chapters))
					for _, ch := range toc.Chapters {
						titles = append(titles, ch.Title)
					}
					chapterScore, chapterIndex = bestChapterMatch(currentChapter, titles)
				}
		}

		candidates = append(candidates, gin.H{
			"sourceId":      src.ID,
			"sourceName":    src.Name,
			"bookKey":       candidateKey,
			"name":          best.book.Name,
			"author":        best.book.Author,
			"matchScore":    best.score,
			"chapterScore":  chapterScore,
			"chapterIndex":  chapterIndex,
			"coverUrl":      best.book.CoverURL,
		})
	}

	ok(c, gin.H{
		"bookKey":    bookKey,
		"name":       targetName,
		"author":     targetAuthor,
		"candidates": candidates,
	})
}

// GET /api/sync/export
func syncExport(c *gin.Context) {
	books, _ := app.ShelfSvc.List()
	sources, _ := app.BookSourceSvc.List()
	rules, _ := app.ReplaceSvc.List()
	ok(c, gin.H{
		"version":    1,
		"exportedAt": time.Now().Format(time.RFC3339),
		"shelf":      books,
		"bookSources": sources,
		"replaceRules": rules,
	})
}

// POST /api/sync/import
func syncImport(c *gin.Context) {
	var bundle struct {
		Shelf        []*shelf.ShelfBook       `json:"shelf"`
		BookSources  []*booksource.BookSource `json:"bookSources"`
		ReplaceRules []*replace.ReplaceRule   `json:"replaceRules"`
	}
	if err := c.ShouldBindJSON(&bundle); err != nil {
		errResp(c, 400, err.Error())
		return
	}
	for _, b := range bundle.Shelf {
		_ = app.ShelfSvc.Add(b)
	}
	if len(bundle.BookSources) > 0 {
		_, _ = app.BookSourceSvc.Import(bundle.BookSources)
		reloadWebBookSources()
	}
	ok(c, gin.H{"imported": true})
}

// GET /api/search/stream?q=
func searchStream(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		errResp(c, 400, "q is required")
		return
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		errResp(c, 500, "streaming not supported")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	const maxStreamResults = 80

	var mu sync.Mutex
	streamed := 0
	seen := make(map[string]struct{})

	err := app.WebBook.SearchBookStream(ctx, q, func(_ *webbook.BookSource, books []webbook.Book, err error) {
		if err != nil || len(books) == 0 {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if streamed >= maxStreamResults {
			cancel()
			return
		}
		for _, book := range books {
			if strings.TrimSpace(book.BookURL) == "" {
				continue
			}
			bookKey := encodeBookKey(book.SourceID, book.BookURL)
			if _, dup := seen[bookKey]; dup {
				continue
			}
			seen[bookKey] = struct{}{}
			payload, _ := json.Marshal(gin.H{
				"name":       book.Name,
				"author":     book.Author,
				"bookUrl":    book.BookURL,
				"bookKey":    encodeBookKey(book.SourceID, book.BookURL),
				"coverUrl":   book.CoverURL,
				"intro":      book.Intro,
				"sourceId":   book.SourceID,
				"sourceName": book.SourceName,
			})
			fmt.Fprintf(c.Writer, "event: result\ndata: %s\n\n", payload)
			streamed++
			if streamed >= maxStreamResults {
				cancel()
				break
			}
		}
		flusher.Flush()
	})
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		fmt.Fprintf(c.Writer, "event: server-error\ndata: %q\n\n", err.Error())
		flusher.Flush()
		return
	}
	fmt.Fprintf(c.Writer, "event: done\ndata: {\"total\":%d}\n\n", streamed)
	flusher.Flush()
}

// applyReplaceRulesScoped 按 scope 应用替换规则
func applyReplaceRulesScoped(text, scope string) string {
	rules, err := app.ReplaceSvc.ListEnabled()
	if err != nil || len(rules) == 0 {
		return text
	}
	result := text
	for _, r := range rules {
		if r.Scope != replace.ScopeAll && r.Scope != scope {
			continue
		}
		result = replace.ApplyRule(r, result)
	}
	return result
}

// GET /api/bookSources/stats
func listSourceStats(c *gin.Context) {
	rows, err := app.BookSourceSvc.DB().Query(`SELECT source_id, last_success, last_error, error_count, success_count FROM source_stats`)
	if err != nil {
		ok(c, []any{})
		return
	}
	defer rows.Close()
	var stats []gin.H
	for rows.Next() {
		var id, ls, le, ec, sc int64
		_ = rows.Scan(&id, &ls, &le, &ec, &sc)
		stats = append(stats, gin.H{"sourceId": id, "lastSuccess": ls, "lastError": le, "errorCount": ec, "successCount": sc})
	}
	ok(c, stats)
}
