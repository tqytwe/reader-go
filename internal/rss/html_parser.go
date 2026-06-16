package rss

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// HTMLParser 自适应HTML页面解析器
type HTMLParser struct {
	client *http.Client
}

// NewHTMLParser 创建HTML解析器
func NewHTMLParser(client *http.Client) *HTMLParser {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &HTMLParser{client: client}
}

// ParseHTML 解析HTML页面，自动检测文章列表
func (p *HTMLParser) ParseHTML(ctx context.Context, url string) (*ParseResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// 提取页面标题
	title := strings.TrimSpace(doc.Find("title").First().Text())
	if title == "" {
		title = doc.Find("h1").First().Text()
	}

	// 自动检测文章列表
	articles := p.detectArticles(doc, url)

	return &ParseResult{
		Title:    title,
		Link:     url,
		FeedType: FeedTypeRSS2, // 统一返回RSS2类型
		Items:    articles,
	}, nil
}

// detectArticles 自动检测页面中的文章列表
func (p *HTMLParser) detectArticles(doc *goquery.Document, baseURL string) []*FeedItem {
	// 候选选择器列表（按优先级排序）
	candidateSelectors := []string{
		// 通用文章选择器
		"article",
		".article",
		".post",
		".entry",
		".item",
		".news-item",
		".list-item",
		".card",
		"[itemprop='articleBody']",
		".content-item",
		"main article",
		".main-content article",
		"#content article",
		// Hacker News 特定
		"tr.athing",
		".athing",
		// Reddit 特定
		"[data-testid='post-container']",
		".Post",
		// 通用列表
		".story",
		".news",
		"li.item",
		"div.item",
	}

	var bestItems *goquery.Selection
	maxCount := 0

	// 尝试每个选择器，找到包含最多项目的
	for _, selector := range candidateSelectors {
		items := doc.Find(selector)
		if items.Length() > maxCount {
			maxCount = items.Length()
			bestItems = items
		}
	}

	// 如果没找到，尝试查找包含多个链接的容器
	if bestItems == nil || bestItems.Length() == 0 {
		bestItems = p.findLinkContainers(doc)
	}

	if bestItems == nil || bestItems.Length() == 0 {
		return nil
	}

	// 提取文章信息
	articles := make([]*FeedItem, 0, bestItems.Length())
	bestItems.Each(func(i int, s *goquery.Selection) {
		article := p.extractArticle(s, baseURL)
		if article != nil && article.Title != "" {
			articles = append(articles, article)
		}
	})

	return articles
}

// findLinkContainers 查找包含多个链接的容器
func (p *HTMLParser) findLinkContainers(doc *goquery.Document) *goquery.Selection {
	// 查找所有包含链接的div
	linkCounts := make(map[int]int) // index -> link count
	var containers []*goquery.Selection

	doc.Find("div").Each(func(i int, s *goquery.Selection) {
		links := s.Find("a[href]")
		if links.Length() >= 3 { // 至少3个链接才认为是列表
			linkCounts[i] = links.Length()
			containers = append(containers, s)
		}
	})

	if len(containers) == 0 {
		return nil
	}

	// 找到包含最多链接的容器
	sort.Slice(containers, func(i, j int) bool {
		return linkCounts[i] > linkCounts[j]
	})

	// 返回第一个（链接最多的）容器中的子元素
	best := containers[0]
	// 尝试找到重复的子元素
	children := best.Children()
	if children.Length() >= 3 {
		return children
	}

	return best
}

// extractArticle 从元素中提取文章信息
func (p *HTMLParser) extractArticle(s *goquery.Selection, baseURL string) *FeedItem {
	article := &FeedItem{}

	// 提取标题
	titleSelectors := []string{
		// 通用标题选择器
		"h1", "h2", "h3", "h4",
		".title", ".post-title", ".entry-title",
		"[itemprop='headline']",
		"a.title",
		// Hacker News 特定
		".titleline a",
		".titleline",
		// 通用链接标题
		"a.storylink",
		"a[href]",
	}
	for _, selector := range titleSelectors {
		if title := strings.TrimSpace(s.Find(selector).First().Text()); title != "" && len(title) > 3 {
			article.Title = title
			// 如果标题来自链接，同时提取链接
			if link, exists := s.Find(selector).First().Attr("href"); exists {
				article.Link = p.resolveURL(link, baseURL)
			}
			break
		}
	}

	// 如果没找到标题，尝试找第一个有意义的链接文本
	if article.Title == "" {
		s.Find("a").Each(func(i int, link *goquery.Selection) {
			text := strings.TrimSpace(link.Text())
			href, exists := link.Attr("href")
			if text != "" && len(text) > 5 && exists && !strings.HasPrefix(href, "#") {
				article.Title = text
				article.Link = p.resolveURL(href, baseURL)
				return
			}
		})
	}

	// 提取链接（如果还没有）
	if article.Link == "" {
		// 尝试多个链接选择器
		linkSelectors := []string{
			"a[href]",
			".titleline a",
			"a.title",
			"a.storylink",
		}
		for _, selector := range linkSelectors {
			if link, exists := s.Find(selector).First().Attr("href"); exists {
				article.Link = p.resolveURL(link, baseURL)
				break
			}
		}
	}

	// 提取描述/摘要
	descSelectors := []string{
		"p", ".summary", ".excerpt", ".description",
		"[itemprop='description']",
		".subtext", ".subline",
	}
	for _, selector := range descSelectors {
		if desc := strings.TrimSpace(s.Find(selector).First().Text()); desc != "" && len(desc) > 20 {
			article.Description = desc
			break
		}
	}

	// 提取作者
	authorSelectors := []string{
		".author", ".byline", "[itemprop='author']",
		".post-author", ".entry-author",
		".subline a[href*='user']",
	}
	for _, selector := range authorSelectors {
		if author := strings.TrimSpace(s.Find(selector).First().Text()); author != "" {
			article.Author = author
			break
		}
	}

	// 提取发布时间
	dateSelectors := []string{
		"time", ".date", ".time", ".published",
		"[itemprop='datePublished']",
		".post-date", ".entry-date",
		".age",
	}
	for _, selector := range dateSelectors {
		if dateEl := s.Find(selector).First(); dateEl.Length() > 0 {
			// 尝试从datetime属性获取
			if datetime, exists := dateEl.Attr("datetime"); exists {
				if t := p.parseTime(datetime); !t.IsZero() {
					article.PublishedAt = t
					break
				}
			}
			// 尝试从文本内容获取
			if dateText := strings.TrimSpace(dateEl.Text()); dateText != "" {
				if t := p.parseTime(dateText); !t.IsZero() {
					article.PublishedAt = t
					break
				}
			}
		}
	}

	// 提取图片
	if img, exists := s.Find("img[src]").First().Attr("src"); exists {
		article.Content = fmt.Sprintf(`<img src="%s" />`, p.resolveURL(img, baseURL))
	}

	// 生成GUID
	if article.Link != "" {
		article.GUID = article.Link
	} else if article.Title != "" {
		article.GUID = article.Title
	}

	return article
}

// resolveURL 解析相对URL为绝对URL
func (p *HTMLParser) resolveURL(href, baseURL string) string {
	if href == "" {
		return ""
	}
	// 已经是绝对URL
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	// 协议相对
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	// 根路径
	if strings.HasPrefix(href, "/") {
		// 提取基础域名
		re := regexp.MustCompile(`^(https?://[^/]+)`)
		if m := re.FindStringSubmatch(baseURL); len(m) > 1 {
			return m[1] + href
		}
	}
	// 相对路径
	return baseURL + "/" + strings.TrimPrefix(href, "./")
}

// parseTime 解析各种时间格式
func (p *HTMLParser) parseTime(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	formats := []string{
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
		"Jan 02, 2006",
		"January 02, 2006",
		"2006年01月02日",
		"2006年1月2日",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}
