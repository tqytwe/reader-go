package rss

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"reader-go/internal/booksource"
	"reader-go/internal/utils"
	"reader-go/internal/webbook"
)

// ParseLegado fetches a page and parses articles using stored Legado rules.
// 规则说明: https://www.yckceo.com/yuedu/tools/index/id/rss.html
func (p *Parser) ParseLegado(feed *Feed) (*ParseResult, error) {
	return p.ParseLegadoCtx(context.Background(), feed)
}

// ParseLegadoCtx fetches a page and parses articles using stored Legado rules with context.
func (p *Parser) ParseLegadoCtx(ctx context.Context, feed *Feed) (*ParseResult, error) {
	rules, err := parseLegadoRules(feed.ParseRules)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(rules["ruleArticles"]) == "" {
		return nil, fmt.Errorf("missing ruleArticles")
	}

	fetchURL, err := resolveFeedFetchURL(feed.FeedURL, feed.SiteURL, rules)
	if err != nil {
		return nil, err
	}

	body, err := p.fetchPage(ctx, fetchURL, rules["header"])
	if err != nil {
		return nil, err
	}

	articles, err := webbook.ParseLegadoRSSArticles(body, fetchURL, rules)
	if err != nil {
		return nil, err
	}

	items := make([]*FeedItem, 0, len(articles))
	for _, article := range articles {
		guid := article.Link
		if guid == "" {
			guid = article.Title
		}
		content := article.Content
		if content == "" {
			content = article.Description
		}
		items = append(items, &FeedItem{
			GUID:        guid,
			Title:       article.Title,
			Link:        article.Link,
			Description: article.Description,
			Content:     content,
			Author:      article.Author,
			PublishedAt: p.parseTime(article.PubDate),
		})
	}

	return &ParseResult{
		Title:    feed.Title,
		Link:     feed.SiteURL,
		FeedType: FeedTypeRSS2,
		Items:    items,
	}, nil
}

func (p *Parser) fetchPage(ctx context.Context, pageURL, headersJSON string) (string, error) {
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
	if headers, err := booksource.ParseHeaders(headersJSON); err == nil {
		for k, v := range headers {
			headers[k] = v
		}
	}
	result, err := utils.FetchText(ctx, utils.FetchTextOptions{
		Method:       http.MethodGet,
		URL:          pageURL,
		Client:       p.client,
		AllowBrowser: true,
		UseHeuristic: true,
		Browser:      browserFetcher,
		Headers:      headers,
	})
	if err != nil {
		return "", err
	}
	return result.Body, nil
}

func parseLegadoRules(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("empty parse rules")
	}
	var rules map[string]string
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil, fmt.Errorf("invalid parse rules: %w", err)
	}
	return rules, nil
}
