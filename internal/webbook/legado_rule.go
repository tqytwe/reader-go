package webbook

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
)

var legadoAttrs = map[string]bool{
	"text": true, "html": true, "href": true, "src": true,
	"content": true, "alt": true, "title": true,
	"textNodes": true, "ownText": true, "allText": true,
}

// legadoIdxAll 表示未指定索引时匹配全部元素（Legado 正文 @p@html 等场景）
const legadoIdxAll = -2

// legadoIdxExcludeBase 排除索引的基础值，!N 对应 legadoIdxExcludeBase - N
const legadoIdxExcludeBase = -1000

var legadoIndexSuffix = regexp.MustCompile(`^(.+)\.(-?\d+)$`)
var legadoExcludeSuffix = regexp.MustCompile(`^(.+)!(\d+)$`)

// parseLegadoFieldRules 解析 Legado 字段规则（JSON 对象或键值字符串）
func parseLegadoFieldRules(rule string) map[string]string {
	out := make(map[string]string)
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return out
	}
	// 1. 尝试解析JSON格式
	if strings.HasPrefix(rule, "{") {
		// 先尝试直接解析（保留原始转义）
		var m map[string]string
		if err := json.Unmarshal([]byte(rule), &m); err == nil {
			return m
		}
		var anyMap map[string]interface{}
		if err := json.Unmarshal([]byte(rule), &anyMap); err == nil {
			for k, v := range anyMap {
				if s, ok := v.(string); ok {
					out[k] = s
				}
			}
			if len(out) > 0 {
				return out
			}
		}
		// 直接解析失败，尝试单引号替换（用于非标准JSON）
		// 注意：这可能破坏包含单引号的JavaScript代码
		normalized := regexp.MustCompile(`'([^']*)'`).ReplaceAllString(rule, `"$1"`)
		if normalized != rule {
			out = make(map[string]string)
			if err := json.Unmarshal([]byte(normalized), &m); err == nil {
				return m
			}
			if err := json.Unmarshal([]byte(normalized), &anyMap); err == nil {
				for k, v := range anyMap {
					if s, ok := v.(string); ok {
						out[k] = s
					}
				}
				if len(out) > 0 {
					return out
				}
			}
		}
	}
	// 2. 尝试解析键值对格式
	kvRule := parseLegadoKVRule(normalizeInfoRule(rule))
	if len(kvRule) > 0 {
		return kvRule
	}
	// 3. 如果是纯选择器字符串，默认当成bookList规则
	out["bookList"] = rule
	return out
}

// parseLegadoSearchRule 兼容别名
func parseLegadoSearchRule(rule string) map[string]string {
	return parseLegadoFieldRules(rule)
}

// parseLegadoKVRule 解析 name:rule&&author:rule 键值格式
func parseLegadoKVRule(rule string) map[string]string {
	selectors := make(map[string]string)
	if rule == "" {
		return selectors
	}
	parts := strings.Split(rule, "&&")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx := strings.Index(part, ":"); idx > 0 {
			key := strings.TrimSpace(part[:idx])
			val := strings.TrimSpace(part[idx+1:])
			if key != "" && val != "" {
				selectors[key] = val
			}
		}
	}
	return selectors
}

func normalizeInfoRule(rule string) string {
	if strings.Contains(rule, "::") {
		rule = strings.ReplaceAll(rule, "::", ":")
	}
	if strings.Contains(rule, "@") && !strings.Contains(rule, "&&") {
		var parts []string
		for _, p := range strings.Split(rule, "@") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if idx := strings.Index(p, ":"); idx > 0 {
				parts = append(parts, p)
			} else if strings.Contains(p, ".") || strings.HasPrefix(p, "//") {
				// 裸规则片段，跳过
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "&&")
		}
	}
	return rule
}

func isJSONPathRule(rule string) bool {
	r := strings.TrimSpace(rule)
	return strings.HasPrefix(r, "$") || strings.Contains(r, "$.")
}

func hasUnsupportedJS(rules map[string]string) bool {
	for _, v := range rules {
		if strings.Contains(v, "<js>") || strings.HasPrefix(strings.TrimSpace(v), "@js:") {
			return true
		}
	}
	return false
}

// parseLegadoSearchResults 按 Legado 规则解析搜索响应
func parseLegadoSearchResults(body, baseURL, sourceID, sourceName string, rules map[string]string) ([]Book, error) {
	if len(rules) == 0 {
		return nil, fmt.Errorf("empty search rule")
	}

	bookListRule := rules["bookList"]
	if bookListRule == "" {
		bookListRule = rules["bookUrl"]
	}
	if isJSONPathRule(bookListRule) || strings.HasPrefix(strings.TrimSpace(body), "{") || strings.HasPrefix(strings.TrimSpace(body), "[") {
		return parseLegadoJSONSearch(body, baseURL, sourceID, sourceName, rules)
	}
	return parseLegadoHTMLSearch(body, baseURL, sourceID, sourceName, rules)
}

func parseLegadoJSONSearch(body, baseURL, sourceID, sourceName string, rules map[string]string) ([]Book, error) {
	bookListRule := rules["bookList"]
	if bookListRule == "" {
		return nil, fmt.Errorf("missing bookList rule")
	}

	listParts := strings.Split(bookListRule, "&&")
	var items []gjson.Result
	for _, part := range listParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		r := gjson.Get(body, part)
		if r.IsArray() && len(r.Array()) > 0 {
			items = r.Array()
			break
		}
	}
	if len(items) == 0 {
		return nil, nil
	}

	books := make([]Book, 0, len(items))
	for _, item := range items {
		itemJSON := item.Raw
		book := Book{SourceID: sourceID, SourceName: sourceName}
		if r := rules["name"]; r != "" {
			book.Name = gjsonPathFirst(itemJSON, r)
		}
		if r := rules["author"]; r != "" {
			book.Author = gjsonPathFirst(itemJSON, r)
		}
		if r := rules["bookUrl"]; r != "" {
			book.BookURL = resolveLegadoURL(gjsonPathFirst(itemJSON, r), baseURL, itemJSON)
		}
		if r := rules["coverUrl"]; r != "" {
			book.CoverURL = resolveLegadoURL(gjsonPathFirst(itemJSON, r), baseURL, itemJSON)
		}
		if r := rules["intro"]; r != "" {
			book.Intro = gjsonPathFirst(itemJSON, r)
		}
		if strings.TrimSpace(book.Name) != "" {
			books = append(books, book)
		}
	}
	return books, nil
}

func gjsonPathFirst(itemJSON, rule string) string {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return ""
	}
	parts := strings.Split(rule, "&&")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// 模板 URL：book_id={{$.book_id}}
		if strings.Contains(part, "{{") && !strings.HasPrefix(part, "$") {
			return expandLegadoTemplate(part, itemJSON)
		}
		v := gjson.Get(itemJSON, part)
		if v.Exists() {
			return strings.TrimSpace(v.String())
		}
	}
	return ""
}

func expandLegadoTemplate(tmpl, itemJSON string) string {
	out := tmpl
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	out = re.ReplaceAllStringFunc(out, func(m string) string {
		inner := strings.Trim(m, "{}")
		v := gjson.Get(itemJSON, inner)
		if v.Exists() {
			return v.String()
		}
		return m
	})
	return out
}

func parseLegadoHTMLSearch(body, baseURL, sourceID, sourceName string, rules map[string]string) ([]Book, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	bookListRule := rules["bookList"]
	if bookListRule == "" {
		return nil, fmt.Errorf("missing bookList rule")
	}

	items, err := evalLegadoListItems(doc.Selection, bookListRule)
	if err != nil || len(items) == 0 {
		return nil, err
	}

	books := make([]Book, 0, len(items))
	for _, item := range items {
		book := Book{SourceID: sourceID, SourceName: sourceName}
		if r := rules["name"]; r != "" {
			book.Name = evalLegadoRule(item, r)
		}
		if r := rules["author"]; r != "" {
			book.Author = evalLegadoRule(item, r)
		}
		if r := rules["bookUrl"]; r != "" {
			book.BookURL = resolveLegadoURL(evalLegadoRule(item, r), baseURL, "")
		}
		if r := rules["coverUrl"]; r != "" {
			book.CoverURL = resolveLegadoURL(evalLegadoRule(item, r), baseURL, "")
		}
		if r := rules["intro"]; r != "" {
			book.Intro = evalLegadoRule(item, r)
		}
		if strings.TrimSpace(book.Name) != "" {
			books = append(books, book)
		}
	}
	return books, nil
}

func evalLegadoListItems(root *goquery.Selection, rule string) ([]*goquery.Selection, error) {
	_, _, cleanRule := parseLegadoReplaceRule(rule)
	parts := strings.Split(cleanRule, "@")
	if len(parts) == 0 {
		return nil, nil
	}
	sel := root
	for _, part := range parts[:len(parts)-1] {
		next, err := applyLegadoStep(sel, part)
		if err != nil {
			return nil, err
		}
		sel = next
		if sel.Length() == 0 {
			return nil, nil
		}
	}
	last := parts[len(parts)-1]

	// 支持 || OR 操作符：尝试每个候选 CSS 选择器
	alts := strings.Split(last, "||")
	var out []*goquery.Selection
	for _, alt := range alts {
		alt = strings.TrimSpace(alt)
		if alt == "" {
			continue
		}
		css, idx := legadoStepToCSS(alt)
		if css == "" {
			continue
		}
		found := sel.Find(css)
		if found.Length() == 0 {
			continue
		}
		if idx >= 0 {
			if idx < found.Length() {
				return []*goquery.Selection{found.Eq(idx)}, nil
			}
			continue
		}
		// 排除语法 !N
		if idx <= legadoIdxExcludeBase {
			excludeIdx := legadoIdxExcludeBase - idx
			found.Each(func(i int, s *goquery.Selection) {
				if i != excludeIdx {
					out = append(out, s)
				}
			})
			continue
		}
		found.Each(func(_ int, s *goquery.Selection) {
			out = append(out, s)
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// evalLegadoRule 解析 Legado 规则字符串，支持 @ 链式选择器、|| 备选和 ## 替换规则。
// 规则格式：selector1@selector2@attr##replacePattern##replacement
// 支持 || 备选：rule1||rule2 表示先尝试 rule1，为空则尝试 rule2
// 其中 ## 后的部分为替换规则（支持正则替换）。
func evalLegadoRule(sel *goquery.Selection, rule string) string {
	if sel == nil || sel.Length() == 0 {
		return ""
	}
	// 分离替换规则
	replacePattern, replaceWith, cleanRule := parseLegadoReplaceRule(rule)

	// 支持 || 备选操作符：尝试每个备选规则，返回第一个非空结果
	if strings.Contains(cleanRule, "||") {
		alts := strings.Split(cleanRule, "||")
		for _, alt := range alts {
			alt = strings.TrimSpace(alt)
			if alt == "" {
				continue
			}
			result := evalLegadoRuleSingle(sel, alt)
			if result != "" {
				// 应用替换规则
				if replacePattern != "" {
					if re, err := regexp.Compile(replacePattern); err == nil {
						result = re.ReplaceAllString(result, replaceWith)
					}
				}
				return result
			}
		}
		return ""
	}

	result := evalLegadoRuleSingle(sel, cleanRule)

	// 应用替换规则
	if replacePattern != "" {
		if re, err := regexp.Compile(replacePattern); err == nil {
			result = re.ReplaceAllString(result, replaceWith)
		}
	}
	return result
}

// evalLegadoRuleSingle 解析单个 Legado 规则（不含 || 备选）
func evalLegadoRuleSingle(sel *goquery.Selection, cleanRule string) string {
	parts := strings.Split(cleanRule, "@")
	current := sel
	var attr string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if legadoAttrs[part] {
			attr = part
			continue
		}
		next, err := applyLegadoStep(current, part)
		if err != nil || next.Length() == 0 {
			return ""
		}
		current = next
	}
	return extractLegadoAttr(current, attr)
}

// parseLegadoReplaceRule 从规则字符串中解析 ## 替换规则。
// 返回：替换正则、替换内容、去除替换规则后的纯选择器规则。
// 格式：rule##pattern##replacement 或 rule##pattern（替换为空字符串）
func parseLegadoReplaceRule(rule string) (pattern, replacement, cleanRule string) {
	// 查找 ## 分隔符
	idx := strings.Index(rule, "##")
	if idx < 0 {
		return "", "", rule
	}
	cleanRule = strings.TrimSpace(rule[:idx])
	rest := rule[idx+2:]
	// 检查是否有第二个 ##
	idx2 := strings.Index(rest, "##")
	if idx2 >= 0 {
		pattern = strings.TrimSpace(rest[:idx2])
		replacement = rest[idx2+2:]
	} else {
		pattern = strings.TrimSpace(rest)
		replacement = ""
	}
	return pattern, replacement, cleanRule
}

func evalLegadoRuleOnHTML(html, rule string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	return evalLegadoRule(doc.Selection, rule)
}

func applyLegadoStep(sel *goquery.Selection, step string) (*goquery.Selection, error) {
	step = strings.TrimSpace(step)
	if step == "" {
		return sel, nil
	}
	if legadoAttrs[step] {
		return sel, nil
	}
	if strings.HasPrefix(step, "text.") {
		return findLegadoByText(sel, strings.TrimPrefix(step, "text.")), nil
	}
	css, idx := legadoStepToCSS(step)
	if css == "" {
		return sel, fmt.Errorf("empty css for step %q", step)
	}
	found := sel.Find(css)
	if found.Length() == 0 {
		return found, nil
	}
	if idx >= 0 {
		if idx < found.Length() {
			return found.Eq(idx), nil
		}
		return found.First(), nil
	}
	if idx == legadoIdxAll {
		return found, nil
	}
	// 排除语法 !N：idx <= legadoIdxExcludeBase
	if idx <= legadoIdxExcludeBase {
		excludeIdx := legadoIdxExcludeBase - idx
		if excludeIdx < found.Length() {
			// 排除第 excludeIdx 个元素，返回其他所有元素
			result := found.FilterFunction(func(i int, s *goquery.Selection) bool {
				return i != excludeIdx
			})
			return result, nil
		}
		return found, nil
	}
	// 显式负索引：a.-1 表示最后一个匹配元素
	absIdx := found.Length() + idx
	if absIdx >= 0 && absIdx < found.Length() {
		return found.Eq(absIdx), nil
	}
	if found.Length() > 0 {
		return found.Last(), nil
	}
	return found, nil
}

func findLegadoByText(sel *goquery.Selection, needle string) *goquery.Selection {
	needle = strings.TrimSpace(needle)
	if needle == "" || sel == nil {
		return sel
	}
	var matched *goquery.Selection
	sel.Find("a").Each(func(_ int, s *goquery.Selection) {
		if matched != nil {
			return
		}
		if strings.Contains(strings.TrimSpace(s.Text()), needle) {
			matched = s
		}
	})
	if matched != nil {
		return matched
	}
	sel.Find("*").Each(func(_ int, s *goquery.Selection) {
		if matched != nil {
			return
		}
		if strings.Contains(strings.TrimSpace(s.Text()), needle) {
			matched = s
		}
	})
	if matched != nil {
		return matched
	}
	return sel.FilterFunction(func(_ int, _ *goquery.Selection) bool { return false })
}

func legadoStepToCSS(step string) (string, int) {
	idx := legadoIdxAll
	// 先检查排除语法 !N
	if m := legadoExcludeSuffix.FindStringSubmatch(step); m != nil {
		step = m[1]
		excludeIdx, _ := strconv.Atoi(m[2])
		idx = legadoIdxExcludeBase - excludeIdx
	} else if m := legadoIndexSuffix.FindStringSubmatch(step); m != nil {
		// 再检查普通索引 .N
		step = m[1]
		idx, _ = strconv.Atoi(m[2])
	}
	switch {
	case strings.HasPrefix(step, "class."):
		return "." + strings.TrimPrefix(step, "class."), idx
	case strings.HasPrefix(step, "tag."):
		return strings.TrimPrefix(step, "tag."), idx
	case strings.HasPrefix(step, "id."):
		return "#" + strings.TrimPrefix(step, "id."), idx
	default:
		return step, idx
	}
}

func extractLegadoAttr(sel *goquery.Selection, attr string) string {
	if sel == nil || sel.Length() == 0 {
		return ""
	}
	switch attr {
	case "html":
		return joinLegadoAttr(sel, "html")
	case "href", "src", "alt", "title", "content":
		v, _ := sel.First().Attr(attr)
		return strings.TrimSpace(v)
	case "textNodes":
		return joinLegadoTextNodes(sel)
	case "ownText":
		var parts []string
		sel.Each(func(_ int, s *goquery.Selection) {
			if t := strings.TrimSpace(s.Contents().Not("script,style").Text()); t != "" {
				parts = append(parts, t)
			}
		})
		return cleanText(strings.Join(parts, "\n"))
	default:
		return joinLegadoAttr(sel, "text")
	}
}

func joinLegadoAttr(sel *goquery.Selection, attr string) string {
	var parts []string
	sel.Each(func(_ int, s *goquery.Selection) {
		var v string
		switch attr {
		case "html":
			v, _ = s.Html()
		default:
			v = s.Text()
		}
		v = strings.TrimSpace(v)
		if v != "" {
			parts = append(parts, v)
		}
	})
	return strings.Join(parts, "\n")
}

func joinLegadoTextNodes(sel *goquery.Selection) string {
	var parts []string
	sel.Each(func(_ int, s *goquery.Selection) {
		var nodeParts []string
		s.Contents().Each(func(_ int, c *goquery.Selection) {
			if goquery.NodeName(c) == "#text" {
				if t := strings.TrimSpace(c.Text()); t != "" {
					nodeParts = append(nodeParts, t)
				}
			}
		})
		if len(nodeParts) == 0 {
			if t := strings.TrimSpace(s.Text()); t != "" {
				nodeParts = append(nodeParts, t)
			}
		}
		parts = append(parts, nodeParts...)
	})
	return cleanText(strings.Join(parts, "\n"))
}

func resolveLegadoURL(u, baseURL, _ string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	return resolveURL(u, baseURL)
}
