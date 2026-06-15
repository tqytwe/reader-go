// Package rule 规则解析引擎
// JS 引擎集成模块：基于 goja 的 JavaScript 运行时
package rule

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"

	"reader-go/internal/utils"
)

// ============================================================================
// JsExtensions 接口
// 定义暴露给 JavaScript 环境的扩展方法
// ============================================================================

// JsExtensions 是 JS 环境中可用的扩展方法集合
// 实现该接口的类型会被注册到 goja.Runtime 的全局对象中
type JsExtensions interface {
	// Ajax 发起 HTTP GET 请求
	Ajax(url string) (map[string]interface{}, error)

	// Cookie 操作
	Cookie() CookieExtensions

	// Cache 操作
	Cache() CacheExtensions

	// Base64Encode 编码为 Base64
	Base64Encode(str string) string

	// Base64Decode 解码 Base64
	Base64Decode(str string) (string, error)

	// Md5Encode 计算 MD5 哈希
	Md5Encode(str string) string

	// Log 输出调试日志
	Log(msg string)
}

// cookieGetterSetter 内部接口，用于区分 Cookie 和 Cache 的 Get/Set
type cookieGetterSetter interface {
	cookieGet(key string) (string, error)
	cookieSet(key, value string) error
}

type cacheGetterSetter interface {
	cacheGet(key string) (interface{}, error)
	cacheSet(key string, value interface{}, ttl int64) error
}

// CookieExtensions Cookie 操作扩展
type CookieExtensions interface {
	// Get 获取指定 key 的 cookie 值
	Get(key string) (string, error)

	// Set 设置 cookie
	Set(key, value string) error
}

// CacheExtensions 缓存操作扩展
type CacheExtensions interface {
	// CacheGet 获取缓存值
	CacheGet(key string) (interface{}, error)

	// CacheSet 设置缓存值，ttl 单位为秒，0 表示永久
	CacheSet(key string, value interface{}, ttl int64) error
}

// ============================================================================
// JsEngine 结构体
// ============================================================================

// JsEngine JS 引擎，封装 goja.Runtime
type JsEngine struct {
	runtime    *goja.Runtime
	ctx        context.Context
	cancel     context.CancelFunc
	variables  map[string]interface{}
	variableMu sync.RWMutex
	extensions JsExtensions
}

// JsEngineOptions JS 引擎配置选项
type JsEngineOptions struct {
	// Timeout 脚本执行超时时间，0 表示不限制
	Timeout time.Duration
	// Variables 初始变量映射
	Variables map[string]interface{}
	// Extensions 扩展方法实现，nil 则使用默认实现
	Extensions JsExtensions
}

// NewJsEngine 创建新的 JS 引擎实例
func NewJsEngine(opts *JsEngineOptions) *JsEngine {
	if opts == nil {
		opts = &JsEngineOptions{}
	}

	runtime := goja.New()
	ctx, cancel := context.WithCancel(context.Background())

	engine := &JsEngine{
		runtime:   runtime,
		ctx:       ctx,
		cancel:    cancel,
		variables: make(map[string]interface{}),
	}

	// 设置初始变量
	for k, v := range opts.Variables {
		engine.variables[k] = v
		runtime.Set(k, v)
	}

	// 设置超时（使用 context 取消信号）
	if opts.Timeout > 0 {
		go func() {
			select {
			case <-time.After(opts.Timeout):
				engine.runtime.Interrupt("execution timeout")
			case <-ctx.Done():
				engine.runtime.Interrupt("context cancelled")
			}
		}()
	}

	// 注册扩展方法
	ext := opts.Extensions
	if ext == nil {
		ext = newDefaultExtensions(engine)
	}
	engine.extensions = ext
	engine.registerExtensions(ext)

	// 预定义常用全局对象
	engine.registerBuiltins()

	return engine
}

// registerExtensions 注册扩展方法到 JS 运行时
func (e *JsEngine) registerExtensions(ext JsExtensions) {
	// 创建全局扩展对象
	extObj := e.runtime.NewObject()

	// ajax(url)
	extObj.Set("ajax", func(url string) (*goja.Object, error) {
		result, err := ext.Ajax(url)
		if err != nil {
			return nil, err
		}
		return e.runtime.ToValue(result).ToObject(e.runtime), nil
	})

	// cookie 对象
	cookieObj := e.runtime.NewObject()
	cookieExt := ext.Cookie()
	cookieObj.Set("get", func(key string) (string, error) {
		return cookieExt.Get(key)
	})
	cookieObj.Set("set", func(key, value string) error {
		return cookieExt.Set(key, value)
	})
	extObj.Set("cookie", cookieObj)

	// cache 对象
	cacheObj := e.runtime.NewObject()
	cacheExt := ext.Cache()
	cacheObj.Set("get", func(key string) (interface{}, error) {
		return cacheExt.CacheGet(key)
	})
	cacheObj.Set("set", func(key string, value interface{}, ttl int64) error {
		return cacheExt.CacheSet(key, value, ttl)
	})
	extObj.Set("cache", cacheObj)

	// base64Encode
	extObj.Set("base64Encode", func(str string) string {
		return ext.Base64Encode(str)
	})

	// base64Decode
	extObj.Set("base64Decode", func(str string) (string, error) {
		return ext.Base64Decode(str)
	})

	// md5Encode
	extObj.Set("md5Encode", func(str string) string {
		return ext.Md5Encode(str)
	})

	// log
	extObj.Set("log", func(msg string) {
		ext.Log(msg)
	})

	// 注册到全局
	e.runtime.Set("ext", extObj)

	// Legado java.* 兼容别名
	e.RegisterJavaCompat()

	// 同时提供便捷访问：直接暴露部分方法到全局
	e.runtime.Set("ajax", extObj.Get("ajax"))
	e.runtime.Set("base64Encode", extObj.Get("base64Encode"))
	e.runtime.Set("base64Decode", extObj.Get("base64Decode"))
	e.runtime.Set("md5Encode", extObj.Get("md5Encode"))
	e.runtime.Set("log", extObj.Get("log"))
}

// RegisterJavaCompat 注册 Legado java 全局对象（ajax/getString/put 等）
func (e *JsEngine) RegisterJavaCompat() {
	java := e.runtime.NewObject()
	ext := e.extensions
	if ext == nil {
		return
	}
	java.Set("ajax", func(call goja.FunctionCall) goja.Value {
		url, _ := call.Argument(0).Export().(string)
		res, err := ext.Ajax(url)
		if err != nil {
			panic(e.runtime.NewGoError(err))
		}
		return e.runtime.ToValue(res)
	})
	java.Set("getString", func(key string) string {
		if v, ok := e.variables[key]; ok {
			return fmt.Sprint(v)
		}
		return ""
	})
	java.Set("put", func(key string, val interface{}) {
		e.variableMu.Lock()
		e.variables[key] = val
		e.variableMu.Unlock()
		e.runtime.Set(key, val)
	})
	java.Set("base64Encode", ext.Base64Encode)
	java.Set("base64Decode", ext.Base64Decode)
	java.Set("md5Encode", ext.Md5Encode)
	java.Set("log", ext.Log)
	e.runtime.Set("java", java)
}

// registerBuiltins 注册内置对象和工具函数
func (e *JsEngine) registerBuiltins() {
	// Math 对象（goja 默认已有，但确保一致性）
	// JSON 对象（goja 默认已有）

	// 添加 setTimeout / setInterval 的模拟（基于 context）
	e.runtime.Set("setTimeout", func(fn goja.Callable, delayMs int64) goja.Value {
		go func() {
			select {
			case <-time.After(time.Duration(delayMs) * time.Millisecond):
				_, err := fn(nil)
				if err != nil {
					// TODO: add logging
				}
			case <-e.ctx.Done():
				return
			}
		}()
		return goja.Undefined()
	})

	// 添加 sleep 函数
	e.runtime.Set("sleep", func(delayMs int64) {
		select {
		case <-time.After(time.Duration(delayMs) * time.Millisecond):
		case <-e.ctx.Done():
		}
	})

	// 添加版本信息
	e.runtime.Set("__readerGo__", map[string]interface{}{
		"version": "1.0.0",
		"engine":  "goja",
	})
}

// ============================================================================
// 执行方法
// ============================================================================

// RunString 执行 JS 代码字符串，返回结果
func (e *JsEngine) RunString(script string) (goja.Value, error) {
	return e.runtime.RunString(script)
}

// RunInContext 在指定上下文变量中执行 JS 代码
// contextVars 会被临时设置到运行时，执行后恢复
func (e *JsEngine) RunInContext(script string, contextVars map[string]interface{}) (goja.Value, error) {
	// 保存旧值
	oldValues := make(map[string]interface{})
	for k, v := range contextVars {
		oldValues[k] = e.runtime.Get(k)
		e.runtime.Set(k, v)
	}

	// 记录当前所有全局 key，用于清理新创建的变量
	existingKeys := make(map[string]bool)
	for _, k := range e.runtime.GlobalObject().Keys() {
		existingKeys[k] = true
	}

	// 执行
	result, err := e.runtime.RunString(script)

	// 删除执行期间新创建的全局变量
	for _, k := range e.runtime.GlobalObject().Keys() {
		if !existingKeys[k] {
			e.runtime.GlobalObject().Delete(k)
		}
	}

	// 恢复旧值
	for k, oldValue := range oldValues {
		if oldValue == nil {
			e.runtime.Set(k, goja.Undefined())
		} else {
			e.runtime.Set(k, oldValue)
		}
	}

	return result, err
}

// RunEmbeddedJS 执行 <js>...</js> 代码块
// 从包含 <js> 标签的字符串中提取并执行 JS 代码
func (e *JsEngine) RunEmbeddedJS(content string) (goja.Value, error) {
	// 提取 <js>...</js> 中的内容
	jsPattern := regexp.MustCompile(`(?i)<js>(.*?)</js>`)
	matches := jsPattern.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		// 没有 JS 块，返回原内容
		return e.runtime.ToValue(content), nil
	}

	// 拼接所有 JS 块并执行
	var scripts []string
	for _, m := range matches {
		scripts = append(scripts, m[1])
	}

	fullScript := strings.Join(scripts, ";\n")
	return e.runtime.RunString(fullScript)
}

// EvaluateExpression 计算 {{expression}} 中的表达式
// 返回表达式计算结果
func (e *JsEngine) EvaluateExpression(expr string, contextVars map[string]interface{}) (interface{}, error) {
	if strings.TrimSpace(expr) == "" {
		return nil, fmt.Errorf("empty expression")
	}

	// 包装为返回表达式的函数
	script := fmt.Sprintf("(function() { return (%s); })()", expr)

	result, err := e.RunInContext(script, contextVars)
	if err != nil {
		return nil, fmt.Errorf("evaluate expression %q: %w", expr, err)
	}

	return exportValue(result), nil
}

// EvaluateTemplate 处理模板字符串，支持 {{expression}} 和 <js>...</js>
// 将模板中的表达式替换为计算结果
func (e *JsEngine) EvaluateTemplate(template string, contextVars map[string]interface{}) (string, error) {
	result, err := e.EvaluateTemplateAsValue(template, contextVars)
	if err != nil {
		return "", err
	}
	return toString(result), nil
}

// EvaluateTemplateAsValue 处理模板，返回 goja.Value
func (e *JsEngine) EvaluateTemplateAsValue(template string, contextVars map[string]interface{}) (goja.Value, error) {
	// 1. 先处理 <js>...</js> 代码块
	jsPattern := regexp.MustCompile(`(?i)<js>(.*?)</js>`)

	var scriptBuilder strings.Builder
	scriptBuilder.WriteString("(function() {\n")

	lastEnd := 0
	matches := jsPattern.FindAllStringSubmatchIndex(template, -1)

	for _, m := range matches {
		// 添加 JS 块前的文本（转义为字符串）
		if m[0] > lastEnd {
			before := template[lastEnd:m[0]]
			scriptBuilder.WriteString(fmt.Sprintf("ext.log('before: %s');\n", escapeJSString(before)))
		}

		// 执行 JS 块
		jsCode := template[m[2]:m[3]]
		jsKey := fmt.Sprintf("__js_block_%d__", m[0])
		scriptBuilder.WriteString(fmt.Sprintf("var %s = (function() { %s })();\n", jsKey, jsCode))

		lastEnd = m[1]
	}

	// 添加剩余的文本
	if lastEnd < len(template) {
		remaining := template[lastEnd:]
		scriptBuilder.WriteString(fmt.Sprintf("ext.log('remaining: %s');\n", escapeJSString(remaining)))
	}

	scriptBuilder.WriteString("})()")

	_, runErr := e.RunInContext(scriptBuilder.String(), contextVars)
	if runErr != nil {
		return nil, fmt.Errorf("execute template JS: %w", runErr)
	}

	// 2. 处理 {{expression}} 表达式（使用大括号计数以支持嵌套）
	finalStr := template

	for _, expr := range findTemplateExpressions(template) {
		val, err := e.EvaluateExpression(expr, contextVars)
		if err != nil {
			return nil, fmt.Errorf("evaluate expression {{%s}}: %w", expr, err)
		}
		finalStr = strings.Replace(finalStr, "{{"+expr+"}}", toString(val), 1)
	}

	return e.runtime.ToValue(finalStr), nil
}

// findTemplateExpressions 使用大括号计数查找 {{...}} 表达式，支持嵌套大括号
func findTemplateExpressions(template string) []string {
	var results []string
	i := 0
	for i < len(template)-1 {
		if template[i] == '{' && template[i+1] == '{' {
			depth := 1
			start := i + 2
			j := i + 2
			for j < len(template) && depth > 0 {
				if template[j] == '{' {
					depth++
					j++
				} else if template[j] == '}' {
					depth--
					j++
				} else {
					j++
				}
			}
			if depth == 0 {
				// 检查是否以 }} 结尾
				if j >= 4 && template[j-2] == '}' && template[j-1] == '}' {
					results = append(results, template[start:j-2])
				}
				i = j
			} else {
				i++
			}
		} else {
			i++
		}
	}
	return results
}

// ============================================================================
// 变量管理
// ============================================================================

// SetVariable 设置变量
func (e *JsEngine) SetVariable(name string, value interface{}) {
	e.variableMu.Lock()
	defer e.variableMu.Unlock()

	e.variables[name] = value
	e.runtime.Set(name, value)
}

// GetVariable 获取变量
func (e *JsEngine) GetVariable(name string) interface{} {
	e.variableMu.RLock()
	defer e.variableMu.RUnlock()

	if v, ok := e.variables[name]; ok {
		return v
	}
	val := e.runtime.Get(name)
	if val != nil && !goja.IsUndefined(val) {
		return exportValue(val)
	}
	return nil
}

// SetVariables 批量设置变量
func (e *JsEngine) SetVariables(vars map[string]interface{}) {
	for k, v := range vars {
		e.SetVariable(k, v)
	}
}

// GetVariables 获取所有变量
func (e *JsEngine) GetVariables() map[string]interface{} {
	e.variableMu.RLock()
	defer e.variableMu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range e.variables {
		result[k] = v
	}
	return result
}

// ============================================================================
// 资源管理
// ============================================================================

// Close 关闭引擎，释放资源
func (e *JsEngine) Close() {
	if e == nil {
		return
	}
	if e.cancel != nil {
		e.cancel()
	}
	// goja 不需要显式清理，但我们可以清除引用
	e.runtime = nil
	e.extensions = nil
}

// ============================================================================
// 默认扩展实现
// ============================================================================

// defaultExtensions 默认扩展方法实现
type defaultExtensions struct {
	engine *JsEngine
	cache  map[string]*cacheEntry
	cacheMu sync.RWMutex
	cookies map[string]string
	cookieMu sync.RWMutex
	client  *http.Client
}

type cacheEntry struct {
	value      interface{}
	expireAt   time.Time
}

func newDefaultExtensions(engine *JsEngine) *defaultExtensions {
	return &defaultExtensions{
		engine:  engine,
		cache:   make(map[string]*cacheEntry),
		cookies: make(map[string]string),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Ajax 实现 — 包含 SSRF 防护
func (d *defaultExtensions) Ajax(url string) (map[string]interface{}, error) {
	if err := utils.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("ajax SSRF check: %w", err)
	}

	resp, err := d.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ajax GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	result := map[string]interface{}{
		"status":     resp.StatusCode,
		"statusText": resp.Status,
		"body":       string(body),
		"headers":    headersToMap(resp.Header),
	}

	return result, nil
}

// Cookie 实现
func (d *defaultExtensions) Cookie() CookieExtensions {
	return d
}

func (d *defaultExtensions) Get(key string) (string, error) {
	d.cookieMu.RLock()
	defer d.cookieMu.RUnlock()

	val, ok := d.cookies[key]
	if !ok {
		return "", fmt.Errorf("cookie not found: %s", key)
	}
	return val, nil
}

func (d *defaultExtensions) Set(key, value string) error {
	d.cookieMu.Lock()
	defer d.cookieMu.Unlock()

	d.cookies[key] = value
	return nil
}

// Cache 实现
func (d *defaultExtensions) Cache() CacheExtensions {
	return d
}

func (d *defaultExtensions) CacheGet(key string) (interface{}, error) {
	d.cacheMu.RLock()
	defer d.cacheMu.RUnlock()

	entry, ok := d.cache[key]
	if !ok {
		return nil, fmt.Errorf("cache not found: %s", key)
	}

	if time.Now().After(entry.expireAt) {
		return nil, fmt.Errorf("cache expired: %s", key)
	}

	return entry.value, nil
}

func (d *defaultExtensions) CacheSet(key string, value interface{}, ttl int64) error {
	d.cacheMu.Lock()
	defer d.cacheMu.Unlock()

	entry := &cacheEntry{
		value:    value,
		expireAt: time.Now(),
	}

	if ttl > 0 {
		entry.expireAt = entry.expireAt.Add(time.Duration(ttl) * time.Second)
	} else {
		entry.expireAt = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}

	d.cache[key] = entry
	return nil
}

// Base64Encode 实现
func (d *defaultExtensions) Base64Encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// Base64Decode 实现
func (d *defaultExtensions) Base64Decode(str string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	return string(data), nil
}

// Md5Encode 实现
func (d *defaultExtensions) Md5Encode(str string) string {
	hash := md5.Sum([]byte(str))
	return fmt.Sprintf("%x", hash)
}

// Log 实现
func (d *defaultExtensions) Log(msg string) {
	fmt.Printf("[ext] %s\n", msg)
}

// ============================================================================
// 辅助函数
// ============================================================================

func headersToMap(header http.Header) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range header {
		result[k] = strings.Join(v, ", ")
	}
	return result
}

func exportValue(val goja.Value) interface{} {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return nil
	}

	switch v := val.Export().(type) {
	case int64:
		return float64(v)
	case uint64:
		return float64(v)
	case goja.Object:
		// 如果是数组
		if arr, ok := v.Export().([]interface{}); ok {
			return arr
		}
		// 如果是普通对象，返回 map
		if m, ok := v.Export().(map[string]interface{}); ok {
			return m
		}
		return v
	default:
		return v
	}
}

func toString(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case goja.Value:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func escapeJSString(s string) string {
	// 简单的 JS 字符串转义
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
