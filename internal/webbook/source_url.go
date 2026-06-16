package webbook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// sanitizeDoubledURL 修复双重URL问题。
// 当Legado规则的正则替换产生类似 "https://a.com/https://b.com/path" 的URL时，
// 提取第一个完整URL（到第二个协议之前）。
func sanitizeDoubledURL(u string) string {
	if u == "" {
		return u
	}
	// 查找第一个 https:// 或 http:// 的位置
	firstHTTPS := strings.Index(u, "https://")
	firstHTTP := strings.Index(u, "http://")
	firstProto := -1
	protoLen := 0
	if firstHTTPS >= 0 {
		firstProto = firstHTTPS
		protoLen = 8 // len("https://")
	} else if firstHTTP >= 0 {
		firstProto = firstHTTP
		protoLen = 7 // len("http://")
	}

	if firstProto < 0 {
		return u
	}

	// 从第一个协议之后开始查找第二个协议
	afterFirst := u[firstProto+protoLen:]
	secondHTTPS := strings.Index(afterFirst, "https://")
	secondHTTP := strings.Index(afterFirst, "http://")
	secondProto := -1
	if secondHTTPS >= 0 && (secondHTTP < 0 || secondHTTPS < secondHTTP) {
		secondProto = secondHTTPS
	} else if secondHTTP >= 0 {
		secondProto = secondHTTP
	}

	// 如果找到第二个协议，截取到第二个协议之前（去掉末尾的斜杠）
	if secondProto >= 0 {
		result := u[:firstProto+protoLen+secondProto]
		// 去掉末尾的斜杠
		result = strings.TrimSuffix(result, "/")
		return result
	}
	return u
}

// isLegadoSelectorURLRule reports whether template is a Legado HTML selector rule
// (e.g. ".list a@href") rather than a URL path template.
func isLegadoSelectorURLRule(template string) bool {
	t := strings.TrimSpace(template)
	if t == "" {
		return false
	}
	if strings.HasPrefix(t, "http://") || strings.HasPrefix(t, "https://") {
		return false
	}
	if strings.HasPrefix(t, "@js:") || strings.HasPrefix(t, "<js>") {
		return false
	}
	// Legado uses @attr selectors in tocUrl/contentUrl when extracting from page HTML.
	return strings.Contains(t, "@")
}

func sourceBaseURL(raw string) string {
	base := strings.TrimSpace(raw)
	if i := strings.Index(base, "#"); i >= 0 {
		base = base[:i]
	}
	return strings.TrimRight(base, "/")
}

// buildSourceURL 根据书源 URL 模板与变量构建请求 URL。
// template 为空时直接使用 value；两者皆空时报错。
func (wb *WebBook) buildSourceURL(source *BookSource, template, value string, extraVars map[string]string) (string, string, string, error) {
	template = strings.TrimSpace(template)
	value = strings.TrimSpace(value)

	if template == "" {
		if value == "" {
			return "", "GET", "", fmt.Errorf("book URL is empty")
		}
		u, err := wb.absolutizeURL(source, value)
		return u, "GET", "", err
	}

	method := "GET"
	body := ""

	// Legado: /path,{'method':'POST','body':'id={{bookUrl}}'}
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

	// 使用有序替换（双括号优先，避免单括号误替换双括号内部）
	type repl struct{ key, val string }
	replacements := []repl{
		{"{{url}}", value},
		{"{{key}}", value},
		{"{{bookUrl}}", value},
		{"{{book_url}}", value},
		{"{{bookURL}}", value},
		{"{url}", value},
		{"{key}", value},
		{"{bookUrl}", value},
	}
	for _, r := range extraVars {
		_ = r
	}
	for k, v := range extraVars {
		replacements = append(replacements, repl{k, v})
	}
	for _, r := range replacements {
		template = strings.ReplaceAll(template, r.key, r.val)
		body = strings.ReplaceAll(body, r.key, r.val)
	}

	if strings.HasPrefix(template, "@js:") {
		jsVars := map[string]string{
			"bookUrl": value,
			"url":     value,
			"key":     value,
		}
		for k, v := range extraVars {
			jsVars[k] = v
		}
		u, err := wb.evalJSURL(source, template, jsVars)
		if err != nil {
			return "", method, body, err
		}
		return u, method, body, nil
	}

	u, err := wb.absolutizeURL(source, template)
	if err != nil {
		return "", method, body, err
	}
	return u, method, body, nil
}

// resolveChapterListURL builds the chapter list request URL.
// When tocUrl is a Legado selector rule, it is applied to the book detail page first.
func (wb *WebBook) resolveChapterListURL(ctx context.Context, source *BookSource, info *BookInfo) (string, string, string, error) {
	template := strings.TrimSpace(source.ChapterListURL)
	if !isLegadoSelectorURLRule(template) {
		return wb.buildChapterListURL(source, info)
	}

	bookPageURL, _, _, err := wb.buildSourceURL(source, "", info.BookURL, nil)
	if err != nil {
		return "", "", "", err
	}

	resp, err := wb.fetch(ctx, source, bookPageURL, "GET", "")
	if err != nil {
		return "", "", "", fmt.Errorf("fetch book page for tocUrl rule: %w", err)
	}

	// 先提取cleanRule（不包含正则替换部分）
	cleanRule := template
	if idx := strings.Index(template, "##"); idx >= 0 {
		cleanRule = template[:idx]
	}

	// 使用cleanRule提取原始URL（不进行正则替换）
	rawURL := evalLegadoRuleOnHTML(resp.Body, cleanRule)

	// 如果原始URL已经是完整URL，直接使用
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL, "GET", "", nil
	}

	// 否则应用完整的规则（包括正则替换）
	tocURL := resolveLegadoURL(evalLegadoRuleOnHTML(resp.Body, template), resp.URL, resp.Body)
	tocURL = sanitizeDoubledURL(tocURL)
	if tocURL == "" {
		return "", "", "", fmt.Errorf("tocUrl rule %q matched no URL on book page", template)
	}
	return tocURL, "GET", "", nil
}

func (wb *WebBook) absolutizeURL(source *BookSource, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("URL is empty")
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw, nil
	}

	base := sourceBaseURL(source.BaseURL)
	if base == "" {
		return "", fmt.Errorf("relative URL %q requires baseUrl on source %q", raw, source.Name)
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	return base + raw, nil
}

