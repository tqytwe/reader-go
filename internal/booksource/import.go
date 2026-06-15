package booksource

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// LegacyBookSource 旧版书源格式（来自书源合集）
type LegacyBookSource struct {
	BookSourceName  string `json:"bookSourceName"`
	BookSourceURL   string `json:"bookSourceUrl"`
	Name            string `json:"name"`
	BaseURL         string `json:"baseUrl"`
	BookSourceGroup string `json:"bookSourceGroup"`
	SearchURL       string `json:"searchUrl"`
	Header          string `json:"header"`
	LoginURL        string `json:"loginUrl"`
	EnableJS        bool   `json:"enableJs"`
	Enabled         bool   `json:"enabled"`
	Weight          int    `json:"weight"`
	CustomOrder     int    `json:"customOrder"`
	RuleBookInfo    *LegacyBookInfo `json:"ruleBookInfo"`
	RuleSearch      interface{}     `json:"ruleSearch"`
	RuleToc         interface{}     `json:"ruleToc"`
	RuleContent     interface{}     `json:"ruleContent"`
	ExploreUrl      string          `json:"exploreUrl"`
	RuleExplore     interface{}     `json:"ruleExplore"`
}

// LegacyBookInfo 旧版书籍信息规则（嵌套格式）
type LegacyBookInfo struct {
	TocURL    string `json:"tocUrl"`    // 目录页URL
	Author    string `json:"author"`     // 作者规则
	CoverURL  string `json:"coverUrl"`  // 封面URL规则
	Intro     string `json:"intro"`     // 简介规则
	Name      string `json:"name"`      // 书名规则
	Kind      string `json:"kind"`      // 分类规则
	LastChapter string `json:"lastChapter"` // 最新章节规则
	WordCount string `json:"wordCount"` // 字数规则
	Status    string `json:"status"`    // 状态规则
}

// LegacyRuleItem 旧版规则项（可能是字符串或对象）
type LegacyRuleItem struct {
	Type    string `json:"type"`    // 规则类型: xpath, jsonpath, css, regex, js
	Rule    string `json:"rule"`    // 规则内容
	Extras  string `json:"extras"`  // 额外配置
}

// ImportResult 导入结果
type ImportResult struct {
	Total   int      `json:"total"`   // 总数
	Success int      `json:"success"` // 成功数
	Failed  int      `json:"failed"`  // 失败数
	Errors  []string `json:"errors"`  // 错误信息
}

// ConvertToBookSource 将旧版书源转换为标准书源
func ConvertToBookSource(legacy *LegacyBookSource) (*BookSource, error) {
	name := strings.TrimSpace(legacy.BookSourceName)
	if name == "" {
		name = strings.TrimSpace(legacy.Name)
	}
	baseURL := strings.TrimSpace(legacy.BookSourceURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(legacy.BaseURL)
	}
	if name == "" && baseURL != "" {
		name = deriveBookSourceName(baseURL)
	}
	if name == "" && baseURL == "" {
		return nil, fmt.Errorf("book source name is empty")
	}

	bs := &BookSource{
		Name:         name,
		BaseURL:      baseURL,
		SearchURL:    legacy.SearchURL,
		Enabled:      legacy.Enabled,
		Group:        legacy.BookSourceGroup,
		Order:        legacy.Weight,
		Headers:      legacy.Header,
		BookInfoMode: "default",
		TocMode:      "default",
		ContentMode:  "default",
	}
	if bs.Order == 0 {
		bs.Order = legacy.CustomOrder
	}

	// 转换搜索规则
	bs.SearchRule = convertRuleInterface(legacy.RuleSearch)
	if bs.SearchRule != "" {
		r := strings.TrimSpace(bs.SearchRule)
		lower := strings.ToLower(r)
		switch {
		case strings.HasPrefix(r, "$") || strings.Contains(r, "$.") || strings.HasPrefix(lower, "@json:"):
			bs.SearchMode = "jsonpath"
		case strings.HasPrefix(r, "//") || strings.HasPrefix(lower, "@xpath:"):
			bs.SearchMode = "xpath"
		case strings.HasPrefix(lower, "@css:"):
			bs.SearchMode = "css"
		case strings.HasPrefix(r, ":") || strings.HasPrefix(lower, "@regex:"):
			bs.SearchMode = "regex"
		default:
			bs.SearchMode = "default"
		}
	}

	// 转换嵌套的 BookInfo 规则
	if legacy.RuleBookInfo != nil {
		bs.TocURL = legacy.RuleBookInfo.TocURL
		bs.BookInfoRule = buildBookInfoRule(legacy.RuleBookInfo)
	}

	// 转换目录规则
	bs.TocRule = convertRuleInterface(legacy.RuleToc)

	// 转换正文规则
	bs.ContentRule = convertRuleInterface(legacy.RuleContent)

	bs.ExploreURL = legacy.ExploreUrl
	bs.ExploreRule = convertRuleInterface(legacy.RuleExplore)
	bs.LoginURL = legacy.LoginURL

	return bs, nil
}

// buildBookInfoRule 从LegacyBookInfo构建书籍信息规则字符串
func buildBookInfoRule(info *LegacyBookInfo) string {
	var rules []string
	if info.Author != "" {
		rules = append(rules, fmt.Sprintf("author:%s", info.Author))
	}
	if info.CoverURL != "" {
		rules = append(rules, fmt.Sprintf("coverUrl:%s", info.CoverURL))
	}
	if info.Intro != "" {
		rules = append(rules, fmt.Sprintf("intro:%s", info.Intro))
	}
	if info.Name != "" {
		rules = append(rules, fmt.Sprintf("name:%s", info.Name))
	}
	if info.Kind != "" {
		rules = append(rules, fmt.Sprintf("kind:%s", info.Kind))
	}
	if info.LastChapter != "" {
		rules = append(rules, fmt.Sprintf("lastChapter:%s", info.LastChapter))
	}
	if info.WordCount != "" {
		rules = append(rules, fmt.Sprintf("wordCount:%s", info.WordCount))
	}
	if info.Status != "" {
		rules = append(rules, fmt.Sprintf("status:%s", info.Status))
	}

	return strings.Join(rules, "&&")
}

// convertRuleInterface 将可能的多种格式转换为字符串
func convertRuleInterface(rule interface{}) string {
	if rule == nil {
		return ""
	}

	switch v := rule.(type) {
	case string:
		return v
	case []interface{}:
		// 可能是规则数组
		result := ""
		for i, item := range v {
			if i > 0 {
				result += "@"
			}
			if str, ok := item.(string); ok {
				result += str
			} else if m, ok := item.(map[string]interface{}); ok {
				// 可能是对象格式 {type, rule, extras}
				if t, ok := m["type"].(string); ok {
					if r, ok := m["rule"].(string); ok {
						result += fmt.Sprintf("<%s>%s", t, r)
					}
				}
			}
		}
		return result
	case map[string]interface{}:
		// Legado 规则对象（如 ruleSearch）
		if t, ok := v["type"].(string); ok {
			if r, ok := v["rule"].(string); ok {
				return fmt.Sprintf("<%s>%s", t, r)
			}
		}
		b, err := json.Marshal(v)
		if err == nil {
			return string(b)
		}
	}

	return ""
}

// bookSourceCollectionWrapper Legado 合集包装格式
type bookSourceCollectionWrapper struct {
	Sources     []LegacyBookSource `json:"sources"`
	BookSources []LegacyBookSource `json:"bookSources"`
}

// ParseBookSourceCollection 解析书源合集JSON
func ParseBookSourceCollection(data []byte) ([]*BookSource, *ImportResult) {
	result := &ImportResult{Errors: []string{}}

	legacySources, err := parseLegacyBookSources(data)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("JSON解析失败: %v", err))
		return nil, result
	}

	result.Total = len(legacySources)
	bookSources := make([]*BookSource, 0, result.Total)

	for i, legacy := range legacySources {
		if strings.TrimSpace(legacy.BookSourceName) == "" &&
			strings.TrimSpace(legacy.Name) == "" &&
			strings.TrimSpace(legacy.BookSourceURL) == "" &&
			strings.TrimSpace(legacy.BaseURL) == "" {
			continue
		}
		bs, err := ConvertToBookSource(&legacy)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("[%d] %s: %v", i, legacy.BookSourceName, err))
			continue
		}
		bookSources = append(bookSources, bs)
		result.Success++
	}

	return bookSources, result
}

func parseLegacyBookSources(data []byte) ([]LegacyBookSource, error) {
	var legacySources []LegacyBookSource
	if err := json.Unmarshal(data, &legacySources); err == nil && len(legacySources) > 0 {
		if hasValidLegacyBookSource(legacySources) {
			return legacySources, nil
		}
	}

	var wrapped bookSourceCollectionWrapper
	if err := json.Unmarshal(data, &wrapped); err == nil {
		if len(wrapped.Sources) > 0 {
			return wrapped.Sources, nil
		}
		if len(wrapped.BookSources) > 0 {
			return wrapped.BookSources, nil
		}
	}

	var single LegacyBookSource
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, err
	}
	if single.BookSourceName == "" && single.Name == "" && single.BookSourceURL == "" && single.BaseURL == "" {
		return nil, fmt.Errorf("empty book source collection")
	}
	return []LegacyBookSource{single}, nil
}

func hasValidLegacyBookSource(sources []LegacyBookSource) bool {
	for _, s := range sources {
		if strings.TrimSpace(s.BookSourceName) != "" ||
			strings.TrimSpace(s.Name) != "" ||
			strings.TrimSpace(s.BookSourceURL) != "" ||
			strings.TrimSpace(s.BaseURL) != "" {
			return true
		}
	}
	return false
}

func deriveBookSourceName(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "未命名书源"
	}
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		host := strings.TrimPrefix(u.Host, "www.")
		if host != "" {
			return host
		}
	}
	if len(rawURL) > 48 {
		return rawURL[:48] + "..."
	}
	return rawURL
}