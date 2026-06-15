// Package rule 提供多种规则解析器（CSS、XPath、JSONPath、JS 等）
package rule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// CSSRule 解析一条 CSS 规则，返回提取的文本列表
//
// 支持语法:
//   - 标准 CSS 选择器: div, .class, #id, [attr], a[href]
//   - 组合选择器: div > p, div p, div + p, div ~ p
//   - 扩展索引语法:
//     * tag:10     -> 取第 10 个 (0-based)
//     * tag:-1     -> 取最后一个
//     * tag[0,2,4] -> 取指定索引
//     * tag[0:5]   -> 区间选择 [start, end)
//     * tag!0:3    -> 排除索引 [start, end)
//   - 属性提取: @text, @html, @href, @src, @class 等
//   - 链式组合: selector1 && selector2
func ParseCSS(selector string, doc *goquery.Document) ([]string, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}
	if selector == "" {
		return nil, fmt.Errorf("selector is empty")
	}

	// 处理链式组合 &&
	parts := strings.Split(selector, "&&")
	if len(parts) > 1 {
		return parseCSSChain(parts, doc)
	}

	return parseCSSSingle(selector, doc)
}

// parseCSSChain 处理 && 链式组合
func parseCSSChain(parts []string, doc *goquery.Document) ([]string, error) {
	var results []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		res, err := parseCSSSingle(part, doc)
		if err != nil {
			return nil, fmt.Errorf("chain part %q: %w", part, err)
		}
		results = append(results, res...)
	}
	return results, nil
}

// parseCSSSingle 解析单个 CSS 规则
func parseCSSSingle(selector string, doc *goquery.Document) ([]string, error) {
	// 1. 解析属性提取 @attr
	var attr string
	if idx := strings.Index(selector, "@"); idx != -1 {
		attr = selector[idx+1:]
		selector = selector[:idx]
	}

	// 2. 解析扩展索引语法
	indexSpec, cssSelector := parseIndexSyntax(selector)

	// 3. 执行 CSS 选择器
	elems, err := selectElements(cssSelector, doc)
	if err != nil {
		return nil, err
	}

	// 4. 应用索引过滤
	elems = applyIndexSpec(elems, indexSpec)

	// 5. 提取内容
	return extractElements(elems, attr)
}

// IndexSpec 扩展索引规范
type IndexSpec struct {
	// 模式: "single" | "last" | "indices" | "range" | "exclude"
	Mode string
	// 单个索引 (Mode=="single")
	Single int
	// 索引列表 (Mode=="indices")
	Indices []int
	// 区间 [start, end) (Mode=="range" | "exclude")
	Start, End int
}

// 扩展索引正则
var (
	// tag:10 或 tag:-1
	reSingleIndex = regexp.MustCompile(`^(.+?):(-?\d+)$`)
	// tag[0,2,4]
	reIndices     = regexp.MustCompile(`^(.+?)\[(\d+(?:,\d+)*)\]$`)
	// tag[0:5] 区间
	reRange       = regexp.MustCompile(`^(.+?)\[(\d+):(\d+)\]$`)
	// tag!0:3 排除
	reExclude     = regexp.MustCompile(`^(.+?)\!(\d+):(\d+)$`)
)

// parseIndexSyntax 解析扩展索引语法，返回 (IndexSpec, 纯CSS选择器)
func parseIndexSyntax(selector string) (IndexSpec, string) {
	spec := IndexSpec{Mode: "all"} // 默认全部

	// 排除: tag!0:3
	if m := reExclude.FindStringSubmatch(selector); m != nil {
		start, _ := strconv.Atoi(m[2])
		end, _ := strconv.Atoi(m[3])
		return IndexSpec{Mode: "exclude", Start: start, End: end}, m[1]
	}

	// 区间: tag[0:5]
	if m := reRange.FindStringSubmatch(selector); m != nil {
		start, _ := strconv.Atoi(m[2])
		end, _ := strconv.Atoi(m[3])
		return IndexSpec{Mode: "range", Start: start, End: end}, m[1]
	}

	// 索引列表: tag[0,2,4]
	if m := reIndices.FindStringSubmatch(selector); m != nil {
		var indices []int
		for _, v := range strings.Split(m[2], ",") {
			if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				indices = append(indices, i)
			}
		}
		return IndexSpec{Mode: "indices", Indices: indices}, m[1]
	}

	// 单个索引: tag:10 或 tag:-1
	if m := reSingleIndex.FindStringSubmatch(selector); m != nil {
		idx, _ := strconv.Atoi(m[2])
		if idx < 0 {
			return IndexSpec{Mode: "last"}, m[1]
		}
		return IndexSpec{Mode: "single", Single: idx}, m[1]
	}

	return spec, selector
}

// selectElements 执行 CSS 选择器，返回匹配的 goquery 元素集合
func selectElements(selector string, doc *goquery.Document) ([]*goquery.Selection, error) {
	if selector == "" {
		return nil, fmt.Errorf("CSS selector is empty after parsing index syntax")
	}

	var selections []*goquery.Selection

	// 处理伪类 :not, :has, :is, :where 等 — goquery 原生支持
	// 处理属性选择器中的特殊字符

	sel := doc.Find(selector)
	if sel.Length() == 0 {
		return selections, nil
	}

	// 将 Selection 拆分为独立的 Selection 数组（每个元素一个）
	sel.Each(func(_ int, s *goquery.Selection) {
		selections = append(selections, s)
	})

	return selections, nil
}

// applyIndexSpec 根据索引规范过滤元素
func applyIndexSpec(elems []*goquery.Selection, spec IndexSpec) []*goquery.Selection {
	if spec.Mode == "all" || len(elems) == 0 {
		return elems
	}

	var result []*goquery.Selection
	n := len(elems)

	switch spec.Mode {
	case "single":
		if spec.Single >= 0 && spec.Single < n {
			result = append(result, elems[spec.Single])
		} else if spec.Single < 0 {
			// 负数索引: -1 表示最后一个
			idx := n + spec.Single
			if idx >= 0 && idx < n {
				result = append(result, elems[idx])
			}
		}

	case "last":
		result = append(result, elems[n-1])

	case "indices":
		for _, idx := range spec.Indices {
			if idx >= 0 && idx < n {
				result = append(result, elems[idx])
			} else if idx < 0 {
				nidx := n + idx
				if nidx >= 0 && nidx < n {
					result = append(result, elems[nidx])
				}
			}
		}

	case "range":
		start := spec.Start
		end := spec.End
		if start < 0 {
			start = 0
		}
		if end > n {
			end = n
		}
		if start < end {
			result = append(result, elems[start:end]...)
		}

	case "exclude":
		excludeStart := spec.Start
		excludeEnd := spec.End
		if excludeStart < 0 {
			excludeStart = 0
		}
		if excludeEnd > n {
			excludeEnd = n
		}
		for i, e := range elems {
			if i < excludeStart || i >= excludeEnd {
				result = append(result, e)
			}
		}
	}

	return result
}

// extractElements 从元素集合中提取文本/HTML/属性
func extractElements(elems []*goquery.Selection, attr string) ([]string, error) {
	if len(elems) == 0 {
		return []string{}, nil
	}

	var results []string

	for _, sel := range elems {
		var text string
		switch attr {
		case "", "text":
			text = strings.TrimSpace(sel.Text())
		case "html":
			html, _ := sel.Html()
			text = strings.TrimSpace(html)
		case "ownText":
			text = strings.TrimSpace(sel.Contents().Text())
		case "allText":
			text = strings.TrimSpace(sel.Text())
		default:
			// 属性提取: @href, @src, @class, @id, @title 等
			val, exists := sel.Attr(attr)
			if exists {
				text = strings.TrimSpace(val)
			}
		}

		if text != "" {
			results = append(results, text)
		}
	}

	return results, nil
}

// ParseCSSWithDoc 从 HTML 字符串创建 Document 并解析 CSS 规则
func ParseCSSWithDoc(selector, html string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return ParseCSS(selector, doc)
}

// ParseCSSFromURL 从 URL 获取页面并解析 CSS 规则
func ParseCSSFromURL(selector, url string) ([]string, error) {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return nil, fmt.Errorf("failed to load URL %q: %w", url, err)
	}
	return ParseCSS(selector, doc)
}

// CSSSelector 链式构建器，用于复杂规则组合
type CSSSelector struct {
	doc       *goquery.Document
	selection *goquery.Selection
	err       error
}

// CSS 创建新的选择器链
func CSS(doc *goquery.Document) *CSSSelector {
	return &CSSSelector{doc: doc, selection: doc.Selection}
}

// Find 向下查找
func (c *CSSSelector) Find(sel string) *CSSSelector {
	if c.err != nil {
		return c
	}
	c.selection = c.selection.Find(sel)
	// 不报错，空结果自然返回空选择集
	return c
}

// Filter 过滤当前集合
func (c *CSSSelector) Filter(sel string) *CSSSelector {
	if c.err != nil {
		return c
	}
	c.selection = c.selection.Filter(sel)
	return c
}

// Eq 取指定索引
func (c *CSSSelector) Eq(idx int) *CSSSelector {
	if c.err != nil {
		return c
	}
	c.selection = c.selection.Eq(idx)
	return c
}

// First 取第一个
func (c *CSSSelector) First() *CSSSelector {
	if c.err != nil {
		return c
	}
	c.selection = c.selection.First()
	return c
}

// Last 取最后一个
func (c *CSSSelector) Last() *CSSSelector {
	if c.err != nil {
		return c
	}
	c.selection = c.selection.Last()
	return c
}

// Each 遍历
func (c *CSSSelector) Each(fn func(int, string)) *CSSSelector {
	if c.err != nil {
		return c
	}
	c.selection.Each(func(i int, s *goquery.Selection) {
		fn(i, s.Text())
	})
	return c
}

// Text 提取文本
func (c *CSSSelector) Text() []string {
	if c.err != nil || c.selection == nil {
		return nil
	}
	var results []string
	c.selection.Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if t != "" {
			results = append(results, t)
		}
	})
	return results
}

// Attr 提取属性
func (c *CSSSelector) Attr(name string) []string {
	if c.err != nil || c.selection == nil {
		return nil
	}
	var results []string
	c.selection.Each(func(_ int, s *goquery.Selection) {
		if v, ok := s.Attr(name); ok {
			results = append(results, strings.TrimSpace(v))
		}
	})
	return results
}

// HTML 提取 HTML
func (c *CSSSelector) HTML() []string {
	if c.err != nil || c.selection == nil {
		return nil
	}
	var results []string
	c.selection.Each(func(_ int, s *goquery.Selection) {
		if h, err := s.Html(); err == nil && h != "" {
			results = append(results, strings.TrimSpace(h))
		}
	})
	return results
}

// Error 返回错误
func (c *CSSSelector) Error() error {
	return c.err
}
