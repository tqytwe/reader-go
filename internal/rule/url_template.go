package rule

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// URLTemplate 表示一个解析后的 URL 模板
// 包含原始模板字符串和解析后的各个组件
type URLTemplate struct {
	// Raw 原始 URL 模板字符串
	Raw string

	// BaseURL 基础 URL（不含选项部分）
	BaseURL string

	// JSInjections JS 注入片段列表 <js>...</js>
	JSInjections []string

	// Placeholders 参数占位符列表 {{key}}
	Placeholders []string

	// PageMultiSelect 页数多选配置 <page1,2,3>
	// 格式: [1,2,3] 表示第1页用1，第2页用2，第3页及以后用3
	PageMultiSelect []int

	// Options URL 请求选项
	Options *URLOptions
}

// URLOptions URL 请求选项（逗号 + JSON 格式）
type URLOptions struct {
	// Method HTTP 方法，默认 GET
	Method string `json:"method,omitempty"`

	// Headers 请求头
	Headers map[string]string `json:"headers,omitempty"`

	// Body 请求体
	Body string `json:"body,omitempty"`

	// Charset 字符集
	Charset string `json:"charset,omitempty"`

	// Retry 重试次数
	Retry int `json:"retry,omitempty"`
}

// ParsedURL 解析后的完整 URL 信息
// 用于执行实际 HTTP 请求
type ParsedURL struct {
	// URL 最终生成的 URL（替换占位符、执行 JS 后）
	URL string

	// Method HTTP 方法
	Method string

	// Headers 请求头
	Headers map[string]string

	// Body 请求体
	Body string

	// Charset 字符集
	Charset string

	// Retry 重试次数
	Retry int

	// Template 原始模板引用
	Template *URLTemplate

	// PageValue 当前页数（用于页数多选）
	PageValue int

	// Params 参数映射（{{key}} 的替换值）
	Params map[string]string
}

// ==================== 正则表达式常量 ====================

var (
	// jsInjectionRe 匹配 <js>...</js>
	jsInjectionRe = regexp.MustCompile(`<js>(.*?)</js>`)

	// placeholderRe 匹配 {{key}}
	placeholderRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

	// pageMultiSelectRe 匹配 <page1,2,3>
	pageMultiSelectRe = regexp.MustCompile(`<page([\d,\s]+)`)

	// urlOptionsRe 匹配 URL 末尾的选项部分
	// 格式: , {"method":"POST", "headers":{...}, ...}
	urlOptionsRe = regexp.MustCompile(`,\s*(\{.*\})\s*$`)
)

// ==================== ParseURLTemplate ====================

// ParseURLTemplate 解析 URL 模板字符串
// 支持以下语法：
//   - <js>...</js> - JS 注入，动态生成 URL
//   - {{key}}, {{page}} - 参数占位符替换
//   - <page1,2,3> - 页数多选（第1页用1，第2页用2，第3页及以后用3）
//   - , {"method":"POST", "headers":{...}, "body":"..."} - URL 选项
//
// 示例:
//   ParseURLTemplate("https://example.com/api?book={{bookId}}&page={{page}}<page1,2,3>, {\"method\":\"POST\", \"headers\":{\"User-Agent\":\"Reader\"}, \"retry\":3}")
func ParseURLTemplate(url string) (*URLTemplate, error) {
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("url template is empty")
	}

	tmpl := &URLTemplate{
		Raw: url,
	}

	// 1. 先提取并移除 URL 选项部分
	urlWithoutOptions, options := parseURLOptions(url)

	// 2. 提取 JS 注入片段
	jsMatches := jsInjectionRe.FindAllStringSubmatch(urlWithoutOptions, -1)
	for _, m := range jsMatches {
		if len(m) >= 2 {
			tmpl.JSInjections = append(tmpl.JSInjections, m[1])
		}
	}

	// 3. 提取参数占位符
	placeMatches := placeholderRe.FindAllStringSubmatch(urlWithoutOptions, -1)
	for _, m := range placeMatches {
		if len(m) >= 2 {
			tmpl.Placeholders = append(tmpl.Placeholders, m[1])
		}
	}

	// 4. 提取页数多选
	pageMatches := pageMultiSelectRe.FindAllStringSubmatch(urlWithoutOptions, -1)
	for _, m := range pageMatches {
		if len(m) >= 2 {
			tmpl.PageMultiSelect = parsePageMultiSelect(m[1])
		}
	}

	// 5. 构建基础 URL（移除所有模板语法标记）
	tmpl.BaseURL = buildBaseURL(urlWithoutOptions, tmpl.JSInjections, tmpl.Placeholders, tmpl.PageMultiSelect)

	// 6. 解析 URL 选项
	if options != "" {
		opts, err := parseURLOptionsJSON(options)
		if err != nil {
			return nil, fmt.Errorf("parse url options: %w", err)
		}
		tmpl.Options = opts
	}

	return tmpl, nil
}

// ==================== 内部解析函数 ====================

// parseURLOptions 从 URL 字符串中提取选项部分
// 返回 (去除选项的URL, 选项JSON字符串)
func parseURLOptions(url string) (string, string) {
	matches := urlOptionsRe.FindStringSubmatch(url)
	if len(matches) >= 2 {
		return url[:len(url)-len(matches[1])-1], matches[1] // 减去前面的 ", "
	}
	return url, ""
}

// parseURLOptionsJSON 解析选项 JSON 字符串
func parseURLOptionsJSON(jsonStr string) (*URLOptions, error) {
	var opts URLOptions

	// 先尝试直接反序列化
	if err := json.Unmarshal([]byte(jsonStr), &opts); err != nil {
		// 如果失败，尝试手动解析（处理更宽松的 JSON 格式）
		return parseLooseURLOptions(jsonStr)
	}

	// 设置默认方法
	if opts.Method == "" {
		opts.Method = http.MethodGet
	}

	return &opts, nil
}

// parseLooseURLOptions 解析宽松格式的 URL 选项
// 支持单引号、未加引号的键等
func parseLooseURLOptions(jsonStr string) (*URLOptions, error) {
	opts := &URLOptions{
		Method: http.MethodGet,
	}

	// 移除外层大括号
	content := strings.TrimPrefix(jsonStr, "{")
	content = strings.TrimSuffix(content, "}")

	// 解析 method
	if m := regexp.MustCompile(`["']?method["']?\s*:\s*["']([^"']+)["']`).FindStringSubmatch(content); len(m) >= 2 {
		opts.Method = m[1]
	}

	// 解析 charset
	if m := regexp.MustCompile(`["']?charset["']?\s*:\s*["']([^"']+)["']`).FindStringSubmatch(content); len(m) >= 2 {
		opts.Charset = m[1]
	}

	// 解析 retry
	if m := regexp.MustCompile(`["']?retry["']?\s*:\s*(\d+)`).FindStringSubmatch(content); len(m) >= 2 {
		opts.Retry = 0
		fmt.Sscanf(m[1], "%d", &opts.Retry)
	}

	// 解析 body
	if m := regexp.MustCompile(`["']?body["']?\s*:\s*["']([^"']*)["']`).FindStringSubmatch(content); len(m) >= 2 {
		opts.Body = m[1]
	}

	// 解析 headers
	headersRe := regexp.MustCompile(`["']?headers["']?\s*:\s*\{([^}]*)\}`)
	if m := headersRe.FindStringSubmatch(content); len(m) >= 2 {
		opts.Headers = make(map[string]string)
		headerContent := m[1]
		// 解析每个 header
		headerRe := regexp.MustCompile(`["']([^"']+)["']\s*:\s*["']([^"']*)["']`)
		for _, hm := range headerRe.FindAllStringSubmatch(headerContent, -1) {
			if len(hm) >= 3 {
				opts.Headers[hm[1]] = hm[2]
			}
		}
	}

	return opts, nil
}

// parsePageMultiSelect 解析页数多选字符串 "1,2,3" -> [1,2,3]
func parsePageMultiSelect(s string) []int {
	var result []int
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var n int
		fmt.Sscanf(p, "%d", &n)
		if n > 0 {
			result = append(result, n)
		}
	}
	return result
}

// buildBaseURL 构建基础 URL（移除模板语法标记）
func buildBaseURL(url string, jsInjs []string, placeholders []string, pageMulti []int) string {
	result := url

	// 移除 <js>...</js>
	for _, injection := range jsInjs {
		result = strings.Replace(result, "<js>"+injection+"</js>", "", -1)
	}

	// 移除 {{key}}
	for _, ph := range placeholders {
		result = strings.Replace(result, "{{"+ph+"}}", "", -1)
	}

	// 移除 <page...>
	for _, pm := range pageMulti {
		_ = pm // 页数多选在构建基础 URL 时不直接移除，因为需要保留结构
	}
	result = pageMultiSelectRe.ReplaceAllString(result, "")

	return strings.TrimSpace(result)
}

// ==================== Evaluate ====================

// Evaluate 使用给定的参数和页数计算最终 URL
// params: {{key}} 的替换值
// page: 当前页码
func (t *URLTemplate) Evaluate(params map[string]string, page int) (*ParsedURL, error) {
	if t == nil {
		return nil, fmt.Errorf("url template is nil")
	}

	// 计算页数多选的值
	pageValue := calculatePageValue(t.PageMultiSelect, page)

	// 构建 URL
	url := t.BaseURL

	// 替换占位符
	if params == nil {
		params = make(map[string]string)
	}
	params["page"] = strconv.Itoa(page)
	params["pageValue"] = strconv.Itoa(pageValue)

	for key, value := range params {
		url = strings.ReplaceAll(url, "{{"+key+"}}", value)
	}

	// 处理 JS 注入（简化处理：将 <js> 内容作为注释记录，实际 JS 执行需要 JS 引擎）
	// 这里我们记录 JS 注入点，实际执行由调用方处理
	// 对于简单的 JS 注入，我们可以尝试用 Go 的 eval 替代方案

	// 构建 ParsedURL
	result := &ParsedURL{
		URL:       url,
		Method:    t.Options.Method,
		Headers:   t.Options.Headers,
		Body:      t.Options.Body,
		Charset:   t.Options.Charset,
		Retry:     t.Options.Retry,
		Template:  t,
		PageValue: pageValue,
		Params:    params,
	}

	// 设置默认值
	if result.Method == "" {
		result.Method = http.MethodGet
	}

	return result, nil
}

// calculatePageValue 根据页数多选规则计算实际使用的页数
// pageMultiSelect: [1,2,3] 表示第1页用1，第2页用2，第3页及以后用3
func calculatePageValue(pageMultiSelect []int, page int) int {
	if len(pageMultiSelect) == 0 {
		return page
	}

	if page <= 0 {
		return pageMultiSelect[0]
	}

	// page 从 1 开始
	idx := page - 1
	if idx < len(pageMultiSelect) {
		return pageMultiSelect[idx]
	}

	// 超出范围，使用最后一个值
	return pageMultiSelect[len(pageMultiSelect)-1]
}

// GetRequiredParams 获取模板所需的参数列表
func (t *URLTemplate) GetRequiredParams() []string {
	return t.Placeholders
}

// HasJSInjection 是否有 JS 注入
func (t *URLTemplate) HasJSInjection() bool {
	return len(t.JSInjections) > 0
}

// HasPageMultiSelect 是否有页数多选
func (t *URLTemplate) HasPageMultiSelect() bool {
	return len(t.PageMultiSelect) > 0
}

// HasOptions 是否有 URL 选项
func (t *URLTemplate) HasOptions() bool {
	return t.Options != nil
}
