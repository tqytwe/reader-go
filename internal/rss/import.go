package rss

import (
	"encoding/json"
	"fmt"
)

// LegacyRssSource 旧版订阅源格式（来自订阅源合集）
type LegacyRssSource struct {
	SourceName     string `json:"sourceName"`     // 源名称
	SourceURL      string `json:"sourceUrl"`      // 源URL
	SourceIcon     string `json:"sourceIcon"`     // 图标
	SourceGroup    string `json:"sourceGroup"`    // 分组
	EnableJS       bool   `json:"enableJs"`       // 是否启用JS
	LoadWithBaseUrl bool  `json:"loadWithBaseUrl"` // 是否使用基础URL
	Enabled        bool   `json:"enabled"`         // 是否启用
	CustomOrder    int    `json:"customOrder"`     // 排序
	ArticleStyle   int    `json:"articleStyle"`    // 文章样式
	// 解析规则
	RuleArticles   string `json:"ruleArticles"`    // 列表规则
	RuleTitle      string `json:"ruleTitle"`       // 标题规则
	RuleLink       string `json:"ruleLink"`        // 链接规则
	RuleContent    string `json:"ruleContent"`     // 内容规则
	RuleImage      string `json:"ruleImage"`       // 图片规则
	RulePubDate    string `json:"rulePubDate"`     // 时间规则
	RuleDescription string `json:"ruleDescription"` // 描述规则
	RuleAuthor     string `json:"ruleAuthor"`      // 作者 rules
	Header         string `json:"header"`          // 请求头
	SortURL        string `json:"sortUrl"`         // Legado 分类/入口 URL
	LoginURL       string `json:"loginUrl"`        // Legado 登录/初始 URL
}

// LegacyRssCollection 订阅源合集格式
type LegacyRssCollection struct {
	Sources []LegacyRssSource `json:"sources"`
}

// ConvertToFeed 将旧版订阅源转换为标准Feed
func ConvertToFeed(legacy *LegacyRssSource) (*Feed, error) {
	if legacy.SourceName == "" {
		return nil, fmt.Errorf("source name is empty")
	}
	if legacy.SourceURL == "" {
		return nil, fmt.Errorf("source url is empty")
	}

	rules := map[string]string{
		"ruleArticles":    legacy.RuleArticles,
		"ruleTitle":       legacy.RuleTitle,
		"ruleLink":        legacy.RuleLink,
		"ruleContent":     legacy.RuleContent,
		"ruleImage":       legacy.RuleImage,
		"rulePubDate":     legacy.RulePubDate,
		"ruleDescription": legacy.RuleDescription,
		"ruleAuthor":      legacy.RuleAuthor,
		"header":          legacy.Header,
		"sortUrl":         legacy.SortURL,
		"loginUrl":        legacy.LoginURL,
	}
	rulesJSON, _ := json.Marshal(rules)

	feed := &Feed{
		Title:       legacy.SourceName,
		FeedURL:     legacy.SourceURL,
		SiteURL:     legacy.SourceURL,
		IconURL:     legacy.SourceIcon,
		Group:       legacy.SourceGroup,
		Enabled:     legacy.Enabled,
		ParseRules:  string(rulesJSON),
	}

	if legacy.RuleArticles != "" {
		feed.FeedType = FeedTypeRSS2
	}

	return feed, nil
}

// GetRuleArticles 获取列表规则
func (f *Feed) GetRuleArticles() string {
	if f.ParseRules == "" {
		return ""
	}
	var rules map[string]string
	if json.Unmarshal([]byte(f.ParseRules), &rules) == nil {
		return rules["ruleArticles"]
	}
	return ""
}

// RssImportResult 导入结果
type RssImportResult struct {
	Total   int      `json:"total"`   // 总数
	Success int      `json:"success"` // 成功数
	Failed  int      `json:"failed"`  // 失败数
	Errors  []string `json:"errors"`  // 错误信息
}

// ParseRssSourceCollection 解析订阅源合集JSON
func ParseRssSourceCollection(data []byte) ([]*Feed, *RssImportResult) {
	result := &RssImportResult{Errors: []string{}}

	// 尝试多种格式
	var collection LegacyRssCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		// 尝试直接是数组格式
		var sources []LegacyRssSource
		if err := json.Unmarshal(data, &sources); err != nil {
			// 尝试单对象格式
			var single LegacyRssSource
			if err := json.Unmarshal(data, &single); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("JSON解析失败: %v", err))
				return nil, result
			}
			sources = []LegacyRssSource{single}
		} else {
			collection.Sources = sources
		}
	}

	if len(collection.Sources) == 0 {
		result.Errors = append(result.Errors, "未找到任何订阅源")
		return nil, result
	}

	result.Total = len(collection.Sources)
	feeds := make([]*Feed, 0, result.Total)

	for i, legacy := range collection.Sources {
		feed, err := ConvertToFeed(&legacy)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("[%d] %s: %v", i, legacy.SourceName, err))
			continue
		}
		feeds = append(feeds, feed)
		result.Success++
	}

	return feeds, result
}