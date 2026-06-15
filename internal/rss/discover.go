package rss

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"reader-go/internal/browser"
	"reader-go/internal/utils"
)

var rssLinkPattern = regexp.MustCompile(`(?i)<link[^>]+rel=["']alternate["'][^>]+type=["']application/(?:rss\+xml|atom\+xml)["'][^>]+href=["']([^"']+)["']`)
var rssLinkPatternAlt = regexp.MustCompile(`(?i)<link[^>]+href=["']([^"']+)["'][^>]+rel=["']alternate["'][^>]+type=["']application/(?:rss\+xml|atom\+xml)["']`)

// browserFetcher 全局浏览器获取器实例
var browserFetcher = browser.NewFetcher()

// resolveFeedFetchURL picks the first usable http(s) URL for fetching a feed page.
func resolveFeedFetchURL(feedURL, siteURL string, rules map[string]string) (string, error) {
	candidates := []string{
		feedURL,
		rules["sortUrl"],
		rules["loginUrl"],
		siteURL,
	}
	for _, u := range candidates {
		u = strings.TrimSpace(u)
		if isHTTPURL(u) {
			return u, nil
		}
	}
	return "", fmt.Errorf("invalid feed URL: no http(s) address (source may require Legado JS engine)")
}

func isHTTPURL(u string) bool {
	u = strings.TrimSpace(u)
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

// discoverFeedURLFromHTML extracts RSS/Atom link from HTML page content.
func discoverFeedURLFromHTML(html, baseURL string) string {
	for _, re := range []*regexp.Regexp{rssLinkPattern, rssLinkPatternAlt} {
		if m := re.FindStringSubmatch(html); len(m) > 1 {
			if abs := resolveAbsoluteURL(strings.TrimSpace(m[1]), baseURL); abs != "" {
				return abs
			}
		}
	}
	return ""
}

func resolveAbsoluteURL(href, baseURL string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if isHTTPURL(href) {
		return href
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}

func (p *Parser) fetchRaw(ctx context.Context, pageURL string) (string, error) {
	result, err := utils.FetchText(ctx, utils.FetchTextOptions{
		Method:       http.MethodGet,
		URL:          pageURL,
		Client:       p.client,
		AllowBrowser: true,
		UseHeuristic: true,
		Browser:      browserFetcher,
		Headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
	})
	if err != nil {
		return "", err
	}
	return result.Body, nil
}

// ParseWithDiscovery fetches and parses RSS/Atom, attempting HTML autodiscovery when needed.
// Uses a shared context so all HTTP calls respect the overall timeout.
func (p *Parser) ParseWithDiscovery(pageURL string) (*ParseResult, string, error) {
	return p.ParseWithDiscoveryCtx(context.Background(), pageURL)
}

// ParseWithDiscoveryCtx is like ParseWithDiscovery but accepts a context for timeout control.
func (p *Parser) ParseWithDiscoveryCtx(ctx context.Context, pageURL string) (*ParseResult, string, error) {
	// 1. Try parsing as feed directly
	result, err := p.ParseWithContext(ctx, pageURL)
	if err == nil {
		return result, pageURL, nil
	}
	if !isUnknownFeedType(err) {
		return nil, pageURL, err
	}

	// 2. Fetch HTML and look for <link rel="alternate"> autodiscovery
	body, fetchErr := p.fetchRaw(ctx, pageURL)
	if fetchErr != nil {
		return nil, pageURL, fmt.Errorf("%w: %v", ErrUnknownFeedType, fetchErr)
	}

	if discovered := discoverFeedURLFromHTML(body, pageURL); discovered != "" {
		result, parseErr := p.ParseWithContext(ctx, discovered)
		if parseErr == nil {
			return result, discovered, nil
		}
		// If discovered URL fails, don't fall through to commonFeedPaths — just return error
		return nil, discovered, parseErr
	}

	// 3. Try only first 3 common feed paths (not all 6) to save time
	remainingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	candidates := commonFeedPaths(pageURL)
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}
	for _, candidate := range candidates {
		result, parseErr := p.ParseWithContext(remainingCtx, candidate)
		if parseErr == nil {
			return result, candidate, nil
		}
	}

	return nil, pageURL, fmt.Errorf("%w: no RSS/Atom feed found at %s", ErrUnknownFeedType, pageURL)
}

func commonFeedPaths(pageURL string) []string {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil
	}
	if base.Path == "" {
		base.Path = "/"
	}
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	suffixes := []string{"feed/", "feed", "rss", "rss.xml", "atom.xml", "index.xml"}
	out := make([]string, 0, len(suffixes))
	for _, s := range suffixes {
		u := *base
		u.Path = base.Path + s
		u.RawQuery = ""
		u.Fragment = ""
		out = append(out, u.String())
	}
	return out
}

func isUnknownFeedType(err error) bool {
	if err == nil {
		return false
	}
	if err == ErrUnknownFeedType {
		return true
	}
	if fe, ok := err.(*FeedTypeError); ok && fe.Message == "unknown feed type" {
		return true
	}
	return strings.Contains(err.Error(), "unknown feed type")
}
