// Package browser 提供简单的内置浏览器渲染支持
// 使用 chromedp 或类似工具，但为了避免依赖问题，
// 这里提供一个接口，可以通过环境变量配置外部浏览器命令
package browser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Fetcher 浏览器获取器
type Fetcher struct {
	enabled bool
	bin     string
	timeout time.Duration
}

// NewFetcher 创建浏览器获取器
// 环境变量 BROWSER_BIN 指定浏览器路径，默认尝试常见路径
func NewFetcher() *Fetcher {
	bin := os.Getenv("BROWSER_BIN")
	if bin == "" {
		// 尝试常见路径
		candidates := []string{
			"/usr/bin/chromium-browser",
			"/usr/bin/chromium",
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chrome",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				bin = c
				break
			}
		}
	}

	timeoutMs := 30000
	if v := os.Getenv("BROWSER_TIMEOUT_MS"); v != "" {
		fmt.Sscanf(v, "%d", &timeoutMs)
	}

	return &Fetcher{
		enabled: bin != "" && os.Getenv("DISABLE_BROWSER") != "true",
		bin:     bin,
		timeout: time.Duration(timeoutMs) * time.Millisecond,
	}
}

// IsEnabled 是否启用了浏览器
func (f *Fetcher) IsEnabled() bool {
	return f.enabled && f.bin != ""
}

// Fetch 使用浏览器获取页面内容
// 使用 headless 模式获取渲染后的 HTML
func (f *Fetcher) Fetch(ctx context.Context, url string) (string, error) {
	if !f.IsEnabled() {
		return "", fmt.Errorf("browser not available")
	}

	// 创建带超时的 context
	if f.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, f.timeout)
		defer cancel()
	}

	// 使用 Chrome/Chromium headless 模式
	// --dump-dom 输出渲染后的 DOM
	// --disable-gpu 禁用 GPU 加速（服务器环境需要）
	// --no-sandbox 禁用沙盒（Docker 环境需要）
	// --disable-dev-shm-usage 禁用 /dev/shm（Docker 环境需要）
	args := []string{
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-setuid-sandbox",
		"--disable-web-security",
		"--disable-features=IsolateOrigins,site-per-process",
		"--disable-blink-features=AutomationControlled",
		"--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"--dump-dom",
		url,
	}

	cmd := exec.CommandContext(ctx, f.bin, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("browser fetch failed: %w, stderr: %s", err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("browser fetch failed: %w", err)
	}

	return string(output), nil
}

// FetchWithHeaders 使用浏览器获取页面内容，并返回设置的请求头信息
func (f *Fetcher) FetchWithHeaders(ctx context.Context, url string, headers map[string]string) (string, error) {
	// 目前简单实现，headers 在 headless chrome 中设置较复杂
	// 后续可以通过 CDP 协议实现
	return f.Fetch(ctx, url)
}

// ShouldUseBrowser 判断是否应该使用浏览器
// 当 URL 包含 webView 标记，或者普通请求返回的内容看起来是 JS 渲染的页面时返回 true
func ShouldUseBrowser(url string, body string) bool {
	// URL 包含 webView 标记
	if strings.Contains(url, "webView") {
		return true
	}

	// 内容为空或包含明显的 JS 渲染标记
	bodyLower := strings.ToLower(body)
	if strings.Contains(bodyLower, "<noscript>") {
		return true
	}

	// 页面包含大量 script 标签且内容很少
	scriptCount := strings.Count(bodyLower, "<script")
	if scriptCount > 5 && len(body) < 5000 {
		return true
	}

	// 页面包含常见的 JS 框架加载标记
	jsFrameworks := []string{
		"react", "vue", "angular", "next.js", "nuxt",
		"window.__initial", "window.__data", "window.__app",
		"#app", "#root", "[data-reactroot]",
	}
	for _, fw := range jsFrameworks {
		if strings.Contains(bodyLower, fw) {
			return true
		}
	}

	return false
}
