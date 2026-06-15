package webbook

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
	"reader-go/internal/rule"
)

// RSSArticle Legado 订阅源单条文章解析结果
// 规则说明: https://www.yckceo.com/yuedu/tools/index/id/rss.html
type RSSArticle struct {
	Title       string
	Link        string
	Description string
	Content     string
	Author      string
	Image       string
	PubDate     string
}

// ParseLegadoRSSArticles 按 Legado 订阅源规则解析文章列表
func ParseLegadoRSSArticles(body, baseURL string, rules map[string]string) ([]RSSArticle, error) {
	listRule := strings.TrimSpace(rules["ruleArticles"])
	if listRule == "" {
		return nil, fmt.Errorf("missing ruleArticles")
	}
	trimmed := strings.TrimSpace(body)
	if isJSONPathRule(listRule) || strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return parseLegadoJSONRSS(body, baseURL, rules)
	}
	return parseLegadoHTMLRSS(body, baseURL, rules)
}

func parseLegadoJSONRSS(body, baseURL string, rules map[string]string) ([]RSSArticle, error) {
	listRule := strings.TrimSpace(rules["ruleArticles"])
	listParts := strings.Split(listRule, "&&")
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
		return nil, fmt.Errorf("no articles matched ruleArticles")
	}

	articles := make([]RSSArticle, 0, len(items))
	for _, item := range items {
		itemJSON := item.Raw
		article := RSSArticle{
			Title:   gjsonPathFirst(itemJSON, rules["ruleTitle"]),
			Link:    resolveLegadoURL(gjsonPathFirst(itemJSON, rules["ruleLink"]), baseURL, itemJSON),
			Author:  gjsonPathFirst(itemJSON, rules["ruleAuthor"]),
			Image:   resolveLegadoURL(gjsonPathFirst(itemJSON, rules["ruleImage"]), baseURL, itemJSON),
			PubDate: gjsonPathFirst(itemJSON, rules["rulePubDate"]),
		}
		if r := strings.TrimSpace(rules["ruleDescription"]); r != "" {
			article.Description = gjsonPathFirst(itemJSON, r)
		} else if r := strings.TrimSpace(rules["ruleContent"]); r != "" {
			article.Content = gjsonPathFirst(itemJSON, r)
		}
		if article.Title == "" && article.Link == "" {
			continue
		}
		if article.Title == "" {
			article.Title = article.Link
		}
		articles = append(articles, article)
	}
	if len(articles) == 0 {
		return nil, fmt.Errorf("no valid articles parsed")
	}
	return articles, nil
}

func parseLegadoHTMLRSS(body, baseURL string, rules map[string]string) ([]RSSArticle, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	itemHTMLs, err := extractRSSListHTML(doc, rules["ruleArticles"])
	if err != nil {
		return nil, err
	}
	if len(itemHTMLs) == 0 {
		return nil, fmt.Errorf("no articles matched ruleArticles")
	}

	executor := rule.NewExecutor()
	ctx := context.Background()

	articles := make([]RSSArticle, 0, len(itemHTMLs))
	for _, itemHTML := range itemHTMLs {
		article := RSSArticle{
			Title:   execRSSField(ctx, executor, rules["ruleTitle"], itemHTML, baseURL),
			Link:    resolveLegadoURL(execRSSField(ctx, executor, rules["ruleLink"], itemHTML, baseURL), baseURL, ""),
			Author:  execRSSField(ctx, executor, rules["ruleAuthor"], itemHTML, baseURL),
			Image:   resolveLegadoURL(execRSSField(ctx, executor, rules["ruleImage"], itemHTML, baseURL), baseURL, ""),
			PubDate: execRSSField(ctx, executor, rules["rulePubDate"], itemHTML, baseURL),
		}
		if r := strings.TrimSpace(rules["ruleDescription"]); r != "" {
			article.Description = execRSSField(ctx, executor, r, itemHTML, baseURL)
		} else if r := strings.TrimSpace(rules["ruleContent"]); r != "" {
			article.Content = execRSSField(ctx, executor, r, itemHTML, baseURL)
		}
		if article.Title == "" && article.Link == "" {
			continue
		}
		if article.Title == "" {
			article.Title = article.Link
		}
		articles = append(articles, article)
	}
	if len(articles) == 0 {
		return nil, fmt.Errorf("no valid articles parsed")
	}
	return articles, nil
}

func extractRSSListHTML(doc *goquery.Document, listRule string) ([]string, error) {
	listRule = strings.TrimSpace(listRule)
	if listRule == "" {
		return nil, fmt.Errorf("empty ruleArticles")
	}

	lower := strings.ToLower(listRule)
	switch {
	case strings.HasPrefix(lower, "@css:"):
		idx := strings.Index(listRule, ":")
		selector := strings.TrimSpace(listRule[idx+1:])
		return selectionToOuterHTML(doc.Find(selector)), nil
	case strings.HasPrefix(lower, "@xpath:"), strings.HasPrefix(listRule, "//"):
		html, err := doc.Html()
		if err != nil {
			return nil, err
		}
		xpathRule := listRule
		if strings.HasPrefix(lower, "@xpath:") {
			idx := strings.Index(listRule, ":")
			xpathRule = strings.TrimSpace(listRule[idx+1:])
		}
		results, err := rule.QueryXPath(html, xpathRule)
		if err != nil {
			return nil, err
		}
		return results, nil
	default:
		items, err := evalLegadoListItems(doc.Selection, listRule)
		if err != nil {
			return nil, err
		}
		return selectionsToOuterHTML(items), nil
	}
}

func selectionToOuterHTML(sel *goquery.Selection) []string {
	var out []string
	sel.Each(func(_ int, s *goquery.Selection) {
		if h, err := goquery.OuterHtml(s); err == nil && strings.TrimSpace(h) != "" {
			out = append(out, strings.TrimSpace(h))
		}
	})
	return out
}

func selectionsToOuterHTML(items []*goquery.Selection) []string {
	var out []string
	for _, item := range items {
		if h, err := goquery.OuterHtml(item); err == nil && strings.TrimSpace(h) != "" {
			out = append(out, strings.TrimSpace(h))
		}
	}
	return out
}

func execRSSField(ctx context.Context, executor *rule.Executor, ruleStr, body, baseURL string) string {
	ruleStr = strings.TrimSpace(ruleStr)
	if ruleStr == "" {
		return ""
	}

	// Legado Default 链式语法: class.xxx@tag.a@text
	if isLegadoChainRule(ruleStr) {
		return evalLegadoRuleOnHTML(body, ruleStr)
	}

	mode := rssRuleMode(ruleStr)
	value, err := executor.ExecuteFirst(ctx, mode, ruleStr, body)
	if err != nil || value == "" {
		return ""
	}
	return value
}

func rssRuleMode(ruleStr string) rule.RuleMode {
	lower := strings.ToLower(strings.TrimSpace(ruleStr))
	switch {
	case strings.HasPrefix(lower, "@xpath:"), strings.HasPrefix(ruleStr, "//"):
		return rule.ModeXPath
	case strings.HasPrefix(lower, "@json:"), strings.HasPrefix(lower, "@jsonpath:"), strings.HasPrefix(ruleStr, "$"):
		return rule.ModeJSONPath
	case strings.HasPrefix(lower, "@css:"):
		return rule.ModeCSS
	case strings.HasPrefix(lower, "@regex:"), strings.HasPrefix(ruleStr, ":"):
		return rule.ModeRegex
	case strings.HasPrefix(lower, "@js:"), strings.Contains(ruleStr, "<js>"):
		return rule.ModeJS
	default:
		return rule.ModeDefault
	}
}
