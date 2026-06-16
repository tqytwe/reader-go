package web

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"reader-go/internal/booksource"
	"reader-go/internal/localbook"
	"reader-go/internal/migrate"
	"reader-go/internal/replace"
	"reader-go/internal/rss"
	"reader-go/internal/rule"
	"reader-go/internal/shelf"
	"reader-go/internal/web/middleware"
	"reader-go/internal/webbook"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var app *App

// resolveLocalBooksDir 解析本地书籍存储目录，与 DATABASE_URL / CACHE_DIR 共用同一 data 根目录。
func resolveLocalBooksDir() string {
	if v := os.Getenv("LOCAL_BOOKS_DIR"); v != "" {
		return v
	}
	if cache := os.Getenv("CACHE_DIR"); cache != "" {
		return filepath.Join(filepath.Dir(cache), "localbooks")
	}
	if dbPath := os.Getenv("DATABASE_URL"); dbPath != "" {
		return filepath.Join(filepath.Dir(dbPath), "localbooks")
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, "data", "localbooks")
}

func init() {
	// 确保 app 在包初始化时已创建
	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		// 默认路径：data/reader.db（相对于工作目录）
		wd, _ := os.Getwd()
		dbPath = filepath.Join(wd, "data", "reader.db")
	}

	// 确保目录存在
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	// 尝试打开数据库，损坏就自动重建
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		log.Printf("WARNING: Database open failed, trying to rebuild: %v", err)
		// 备份损坏的数据库
		backupPath := dbPath + ".corrupt." + time.Now().Format("20060102150405")
		os.Rename(dbPath, backupPath)
		os.Rename(dbPath+"-shm", backupPath+"-shm")
		os.Rename(dbPath+"-wal", backupPath+"-wal")
		// 新建数据库
		db, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
		if err != nil {
			log.Fatalf("Failed to create new database: %v", err)
		}
	}

	// 连接池配置
	db.SetMaxOpenConns(1) // SQLite 单写者
	db.SetMaxIdleConns(1)

	// 测试数据库是否正常，损坏就重建
	var test int
	err = db.QueryRow("SELECT 1").Scan(&test)
	if err != nil {
		log.Printf("WARNING: Database is malformed, rebuilding: %v", err)
		db.Close()
		backupPath := dbPath + ".corrupt." + time.Now().Format("20060102150405")
		os.Rename(dbPath, backupPath)
		os.Rename(dbPath+"-shm", backupPath+"-shm")
		os.Rename(dbPath+"-wal", backupPath+"-wal")
		// 新建数据库
		db, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
		if err != nil {
			log.Fatalf("Failed to create new database: %v", err)
		}
	}

	// 初始化服务（先创建基础表）
	bookSourceSvc := booksource.NewService(db)
	if err := bookSourceSvc.Init(); err != nil {
		log.Printf("WARNING: booksource init failed, trying to rebuild database: %v", err)
		db.Close()
		backupPath := dbPath + ".corrupt." + time.Now().Format("20060102150405")
		os.Rename(dbPath, backupPath)
		os.Rename(dbPath+"-shm", backupPath+"-shm")
		os.Rename(dbPath+"-wal", backupPath+"-wal")
		// 新建数据库
		db, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
		if err != nil {
			log.Fatalf("Failed to create new database: %v", err)
		}
		bookSourceSvc = booksource.NewService(db)
		if err := bookSourceSvc.Init(); err != nil {
			log.Fatalf("Failed to init booksource even after rebuild: %v", err)
		}
	}

	// 迁移（添加新列/新表，依赖基础表已存在）
	if err := migrate.Migrate(db); err != nil {
		log.Printf("WARNING: migrate failed: %v", err)
	}

	shelfSvc := shelf.NewService(db)
	if err := shelfSvc.Init(); err != nil {
		log.Fatalf("Failed to init shelf: %v", err)
	}

	replaceSvc := replace.NewService(db)
	if err := replaceSvc.Init(); err != nil {
		log.Fatalf("Failed to init replace: %v", err)
	}

	localBookSvc := localbook.NewService(db, resolveLocalBooksDir())
	if err := localBookSvc.Init(); err != nil {
		log.Fatalf("Failed to init localbook: %v", err)
	}

	rssSvc := rss.NewService(db)
	if err := rssSvc.Init(); err != nil {
		log.Fatalf("Failed to init rss: %v", err)
	}

	app = &App{
		BookSourceSvc: bookSourceSvc,
		ShelfSvc:      shelfSvc,
		ReplaceSvc:    replaceSvc,
		LocalBookSvc:  localBookSvc,
		RSSSvc:        rssSvc,
	}

	// Initialize WebBook with sources from database
	wb := webbook.NewWebBook()
	sources, _ := bookSourceSvc.List()
	for _, src := range sources {
		wb.AddSource(convertBookSource(src))
	}
	app.WebBook = wb
}

// reloadWebBookSources 从数据库重新加载书源到 WebBook 运行时
func reloadWebBookSources() {
	if app == nil || app.BookSourceSvc == nil || app.WebBook == nil {
		return
	}
	sources, err := app.BookSourceSvc.List()
	if err != nil {
		return
	}
	wbSources := make([]*webbook.BookSource, 0, len(sources))
	for _, src := range sources {
		wbSources = append(wbSources, convertBookSource(src))
	}
	app.WebBook.ReloadSources(wbSources)
}

// convertBookSource converts a database booksource.BookSource to a webbook.BookSource.
func convertBookSource(src *booksource.BookSource) *webbook.BookSource {
	ws := &webbook.BookSource{
		ID:              strconv.FormatInt(src.ID, 10),
		Name:            src.Name,
		BaseURL:         src.BaseURL,
		SearchURL:       src.SearchURL,
		BookInfoURL:     src.BookInfoURL,
		ChapterListURL:  src.TocURL,
		ContentURL:      src.ContentURL,
		SearchRule:      src.SearchRule,
		BookInfoRule:    src.BookInfoRule,
		ChapterListRule: src.TocRule,
		ContentRule:     src.ContentRule,
		Enabled:         src.Enabled,
		Cookie:          src.Cookie,
		TimeoutSec:      src.Timeout,
	}

	// Convert headers JSON string to map (best-effort)
	if src.Headers != "" {
		ws.Headers = parseHeaders(src.Headers)
	}

	ws.SearchMode = src.SearchMode
	ws.BookInfoMode = src.BookInfoMode
	ws.TocMode = src.TocMode
	ws.ContentMode = src.ContentMode
	ws.LoginURL = src.LoginURL
	ws.ExploreURL = src.ExploreURL
	ws.ExploreRule = src.ExploreRule
	ws.ExploreMode = src.ExploreMode
	ws.Mode = parseRuleMode(src.SearchMode)

	return ws
}

// parseHeaders attempts to parse a JSON headers string into a map.
// Falls back to treating the string as a single header value if parsing fails.
func parseHeaders(headersJSON string) map[string]string {
	m, err := booksource.ParseHeaders(headersJSON)
	if err != nil || m == nil {
		return make(map[string]string)
	}
	return m
}

// parseRuleMode converts a mode string to rule.RuleMode.
func parseRuleMode(mode string) rule.RuleMode {
	switch mode {
	case "xpath":
		return rule.ModeXPath
	case "jsonpath":
		return rule.ModeJSONPath
	case "css":
		return rule.ModeCSS
	case "regex":
		return rule.ModeRegex
	case "js":
		return rule.ModeJS
	default:
		return rule.ModeDefault
	}
}

type Server struct {
	router *gin.Engine
}

func NewServer() (*Server, error) {
	r := gin.Default()

	// CORS
	r.Use(middleware.CORS())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := r.Group("/api")
	{
		// Book sources
		bookSources := api.Group("/bookSources")
		{
			bookSources.GET("", listBookSources)
			bookSources.POST("", createBookSource)
			bookSources.PUT("/:id", updateBookSource)
			bookSources.DELETE("/:id", deleteBookSource)
			bookSources.POST("/import", importBookSources)
			bookSources.POST("/import/collection", importBookSourceCollection)
			bookSources.POST("/batch/enable", batchSetBookSourceEnabled)
			bookSources.POST("/batch/delete", batchDeleteBookSources)
			bookSources.GET("/stats", listSourceStats)
		}

		api.GET("/explore", getExplore)
		api.GET("/sync/export", syncExport)
		api.POST("/sync/import", syncImport)

		// Search
		api.GET("/search", searchBooks)
		api.GET("/search/stream", searchStream)

		// Book
		api.GET("/book/alternates", getBookAlternates)
		api.GET("/book/info", getBookInfo)
		api.GET("/book/toc", getBookToc)
		api.GET("/book/content", getBookContent)

		// Shelf
		api.GET("/shelf", getShelf)
		api.POST("/shelf", addToShelf)
		api.PUT("/shelf/:id", updateShelfBook)
		api.PUT("/shelf/:id/progress", updateShelfProgress)
		api.DELETE("/shelf/:id", removeFromShelf)

		// Replace rules
		api.GET("/replaceRules", listReplaceRules)
		api.POST("/replaceRules", createReplaceRule)
		api.PUT("/replaceRules/:id", updateReplaceRule)
		api.DELETE("/replaceRules/:id", deleteReplaceRule)

		// Local books
		api.POST("/localBooks", uploadLocalBook)
		api.GET("/localBooks", listLocalBooks)
		api.GET("/localBooks/:id", getLocalBook)
		api.GET("/localBooks/:id/content", getLocalBookContent)
		api.DELETE("/localBooks/:id", deleteLocalBook)

		// Debug book source (SSE)
		api.GET("/bookSources/debug", debugBookSource)

		// RSS feeds
		rss := api.Group("/rss")
		{
			rss.GET("/feeds", listRSSFeeds)
			rss.POST("/feeds", createRSSFeed)
			rss.DELETE("/feeds/:id", deleteRSSFeed)
			rss.GET("/feeds/:id/items", getRSSFeedItems)
			rss.POST("/feeds/:id/preview", previewRSSFeed)
			rss.POST("/feeds/:id/fetch", fetchRSSFeed)
			rss.PUT("/items/:id/read", markRSSItemRead)
			rss.PUT("/items/:id/star", toggleRSSItemStar)
			rss.POST("/import/collection", importRssSourceCollection)
		}
	}

	// 提供前端静态文件（生产环境）
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		// 默认使用web/dist目录
		wd, _ := os.Getwd()
		staticDir = filepath.Join(wd, "web", "dist")
	}
	if _, err := os.Stat(staticDir); err == nil {
		r.Static("/assets", filepath.Join(staticDir, "assets"))
		r.StaticFile("/", filepath.Join(staticDir, "index.html"))
		r.StaticFile("/favicon.ico", filepath.Join(staticDir, "favicon.ico"))
		// 所有未匹配的路由都返回index.html（支持前端路由）
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(staticDir, "index.html"))
		})
	}

	return &Server{router: r}, nil
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
