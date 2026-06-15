package booksource

import (
	"time"

	"reader-go/internal/rule"
)

// BookSource 书源实体
type BookSource struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`                   // 书源名称
	BaseURL     string    `json:"baseUrl" db:"base_url"`             // 基础URL
	SearchURL   string    `json:"searchUrl" db:"search_url"`         // 搜索URL模板
	BookInfoURL string    `json:"bookInfoUrl" db:"book_info_url"`    // 书籍详情URL模板
	TocURL      string    `json:"tocUrl" db:"toc_url"`               // 目录URL模板
	ContentURL  string    `json:"contentUrl" db:"content_url"`       // 正文URL模板
	// 规则字段 — 存储解析后的规则字符串
	SearchRule       string `json:"searchRule" db:"search_rule"`        // 搜索规则
	BookInfoRule     string `json:"bookInfoRule" db:"book_info_rule"`   // 书籍信息规则
	TocRule          string `json:"tocRule" db:"toc_rule"`              // 目录规则
	ContentRule      string `json:"contentRule" db:"content_rule"`      // 正文规则
	// 解析模式配置
	SearchMode       string `json:"searchMode" db:"search_mode"`        // 搜索解析模式: default|xpath|jsonpath|css|regex
	BookInfoMode     string `json:"bookInfoMode" db:"book_info_mode"`   // 书籍信息解析模式
	TocMode          string `json:"tocMode" db:"toc_mode"`              // 目录解析模式
	ContentMode      string `json:"contentMode" db:"content_mode"`      // 正文解析模式
	// 请求配置
	UserAgent        string `json:"userAgent" db:"user_agent"`          // User-Agent
	Headers          string `json:"headers" db:"headers"`               // 额外请求头(JSON格式)
	Cookie           string `json:"cookie" db:"cookie"`                 // Cookie
	Timeout          int    `json:"timeout" db:"timeout"`               // 请求超时(秒)
	// 其他配置
	Enabled          bool   `json:"enabled" db:"enabled"`               // 是否启用
	Group            string `json:"group" db:"group"`                   // 分组
	Order            int    `json:"order" db:"order"`                   // 排序权重
	// 书海 / 登录
	ExploreURL       string `json:"exploreUrl" db:"explore_url"`
	ExploreRule      string `json:"exploreRule" db:"explore_rule"`
	ExploreMode      string `json:"exploreMode" db:"explore_mode"`
	LoginURL         string `json:"loginUrl" db:"login_url"`
	// 元数据
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time `json:"updatedAt" db:"updated_at"`
}

// BookSearchResult 搜索结果项
type BookSearchResult struct {
	Name  string `json:"name"`
	Author string `json:"author,omitempty"`
	// 唯一标识，用于后续查询
	BookKey string `json:"bookKey"`
	// 来源信息
	SourceID   int64  `json:"sourceId"`
	SourceName string `json:"sourceName"`
	// 封面、简介等
	CoverURL   string `json:"coverUrl,omitempty"`
	Summary    string `json:"summary,omitempty"`
}

// BookInfo 书籍详情
type BookInfo struct {
	Name     string `json:"name"`
	Author   string `json:"author,omitempty"`
	CoverURL string `json:"coverUrl,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Tags     string `json:"tags,omitempty"`
	// 来源信息
	SourceID   int64  `json:"sourceId"`
	SourceName string `json:"sourceName"`
	// 唯一标识
	BookKey string `json:"bookKey"`
}

// Chapter 目录章节
type Chapter struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	SubChapters []Chapter `json:"subChapters,omitempty"`
}

// TocResult 目录结果
type TocResult struct {
	BookKey  string    `json:"bookKey"`
	Chapters []Chapter `json:"chapters"`
}

// ContentResult 正文结果
type ContentResult struct {
	BookKey  string `json:"bookKey"`
	Chapter  string `json:"chapter"`
	Content  string `json:"content"`
}

// ParseRules 解析书源的所有规则字段，返回解析后的规则片段
func (bs *BookSource) ParseRules() (searchSegments, infoSegments, tocSegments, contentSegments []rule.RuleSegment, err error) {
	searchSegments, _ = rule.ParseRule(bs.SearchRule)
	infoSegments, _ = rule.ParseRule(bs.BookInfoRule)
	tocSegments, _ = rule.ParseRule(bs.TocRule)
	contentSegments, _ = rule.ParseRule(bs.ContentRule)
	return
}
