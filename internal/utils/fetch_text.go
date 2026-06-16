package utils

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"reader-go/internal/browser"
)

// FetchTextOptions 描述文本抓取参数。
type FetchTextOptions struct {
	Method       string
	URL          string
	Body         string
	Headers      map[string]string
	Client       *http.Client
	Timeout      time.Duration
	AllowBrowser bool
	UseHeuristic bool
	Browser      *browser.Fetcher
}

// FetchTextResult 是统一文本抓取结果。
type FetchTextResult struct {
	URL        string
	Body       string
	Header     http.Header
	StatusCode int
}

// FetchText 执行 HTTP 文本抓取，并统一处理超时、编码与浏览器兜底。
func FetchText(ctx context.Context, opts FetchTextOptions) (*FetchTextResult, error) {
	method := strings.TrimSpace(opts.Method)
	if method == "" {
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, opts.URL, strings.NewReader(opts.Body))
	if err != nil {
		return nil, err
	}
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	client := cloneHTTPClient(opts.Client, opts.Timeout)
	resp, err := client.Do(req)
	if err != nil {
		if opts.AllowBrowser && opts.Browser != nil && opts.Browser.IsEnabled() {
			browserBody, browserErr := opts.Browser.Fetch(ctx, opts.URL)
			if browserErr == nil && strings.TrimSpace(browserBody) != "" {
				return &FetchTextResult{
					URL:        opts.URL,
					Body:       browserBody,
					Header:     http.Header{},
					StatusCode: 200,
				}, nil
			}
		}
		return nil, err
	}
	defer resp.Body.Close()

	// 处理压缩的响应体
	var reader io.Reader = resp.Body
	contentEncoding := resp.Header.Get("Content-Encoding")
	switch strings.ToLower(contentEncoding) {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
	case "deflate":
		reader = flate.NewReader(resp.Body)
	case "br":
		// Brotli 需要额外的库，这里先不支持
		// 如果没有 brotli 支持，移除 Accept-Encoding 中的 br
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	body := DecodeBodyToUTF8(data, resp.Header.Get("Content-Type"))
	result := &FetchTextResult{
		URL:        resp.Request.URL.String(),
		Body:       body,
		Header:     resp.Header,
		StatusCode: resp.StatusCode,
	}

	if opts.AllowBrowser && opts.UseHeuristic && opts.Browser != nil && opts.Browser.IsEnabled() {
		browserURL := opts.URL
		if result.URL != "" {
			browserURL = result.URL
		}
		if browser.ShouldUseBrowser(browserURL, body) {
			browserBody, browserErr := opts.Browser.Fetch(ctx, browserURL)
			if browserErr == nil && strings.TrimSpace(browserBody) != "" {
				result.URL = browserURL
				result.Body = browserBody
			}
		}
	}

	return result, nil
}

func cloneHTTPClient(client *http.Client, timeout time.Duration) *http.Client {
	if client == nil {
		return &http.Client{Timeout: timeout}
	}
	clone := *client
	if timeout > 0 {
		clone.Timeout = timeout
	}
	return &clone
}
