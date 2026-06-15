package webbook

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"strings"

	"reader-go/internal/rule"
	"reader-go/internal/utils"
)

// evalJSURL 执行 @js: URL 模板，返回 URL 字符串
// 超时由 JsEngine 内部通过 JS_TIMEOUT_MS 环境变量控制（默认 5s）
func (wb *WebBook) evalJSURL(source *BookSource, script string, vars map[string]string) (string, error) {
	script = strings.TrimPrefix(strings.TrimSpace(script), "@js:")
	if script == "" {
		return "", fmt.Errorf("empty js url script")
	}

	ext := newWebbookJSExtensions(wb, source)
	jsVars := make(map[string]interface{}, len(vars))
	for k, v := range vars {
		jsVars[k] = v
	}
	engine := rule.NewJsEngine(&rule.JsEngineOptions{
		Variables:  jsVars,
		Extensions: ext,
	})
	v, err := engine.RunString(script)
	if err != nil {
		return "", err
	}
	url := strings.TrimSpace(v.String())
	if url == "" {
		return "", fmt.Errorf("js url returned empty")
	}
	return url, nil
}

// webbookJSExtensions 书源 HTTP 扩展
type webbookJSExtensions struct {
	wb     *WebBook
	source *BookSource
	store  map[string]string
}

func newWebbookJSExtensions(wb *WebBook, source *BookSource) *webbookJSExtensions {
	return &webbookJSExtensions{wb: wb, source: source, store: make(map[string]string)}
}

func (e *webbookJSExtensions) Ajax(url string) (map[string]interface{}, error) {
	if err := utils.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("ajax SSRF check: %w", err)
	}
	resp, err := e.wb.fetch(context.Background(), e.source, url, "GET", "")
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"body":   resp.Body,
		"url":    resp.URL,
		"status": 200,
	}, nil
}

func (e *webbookJSExtensions) Cookie() rule.CookieExtensions {
	return &wbCookieExt{store: e.store}
}

func (e *webbookJSExtensions) Cache() rule.CacheExtensions {
	return &wbCacheExt{store: e.store}
}

func (e *webbookJSExtensions) Base64Encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func (e *webbookJSExtensions) Base64Decode(str string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(str)
	return string(b), err
}

func (e *webbookJSExtensions) Md5Encode(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

func (e *webbookJSExtensions) Log(msg string) {
	fmt.Printf("[js-booksource:%s] %s\n", e.source.Name, msg)
}

type wbCookieExt struct{ store map[string]string }

func (c *wbCookieExt) Get(key string) (string, error) { return c.store[key], nil }
func (c *wbCookieExt) Set(key, value string) error  { c.store[key] = value; return nil }

type wbCacheExt struct{ store map[string]string }

func (c *wbCacheExt) CacheGet(key string) (interface{}, error) { return c.store[key], nil }
func (c *wbCacheExt) CacheSet(key string, value interface{}, _ int64) error {
	c.store[key] = fmt.Sprint(value)
	return nil
}
