package rule

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// JsEngine 基础测试
// ============================================================================

func TestNewJsEngine(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		engine := NewJsEngine(nil)
		defer engine.Close()

		assert.NotNil(t, engine)
		assert.NotNil(t, engine.runtime)

		// 验证扩展对象已注册
		val := engine.runtime.Get("ext")
		assert.NotNil(t, val)
		assert.False(t, goja.IsUndefined(val))

		// 验证便捷方法已注册
		assert.NotNil(t, engine.runtime.Get("ajax"))
		assert.NotNil(t, engine.runtime.Get("base64Encode"))
		assert.NotNil(t, engine.runtime.Get("md5Encode"))
		assert.NotNil(t, engine.runtime.Get("log"))
	})

	t.Run("with custom variables", func(t *testing.T) {
		engine := NewJsEngine(&JsEngineOptions{
			Variables: map[string]interface{}{
				"title": "测试标题",
				"count": 42,
			},
		})
		defer engine.Close()

		val, err := engine.RunString("title + ' - ' + count")
		require.NoError(t, err)
		assert.Equal(t, "测试标题 - 42", val.String())
	})

	t.Run("with custom extensions", func(t *testing.T) {
		customExt := &customTestExtensions{
			cookieValue: "test-cookie-value",
			cacheValue:  "test-cache-value",
		}
		engine := NewJsEngine(&JsEngineOptions{
			Extensions: customExt,
		})
		defer engine.Close()

		val, err := engine.RunString("ext.cookie.get('session')")
		require.NoError(t, err)
		assert.Equal(t, "test-cookie-value", val.String())
	})
}

// ============================================================================
// JsEngine.RunString 测试
// ============================================================================

func TestJsEngine_RunString(t *testing.T) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	t.Run("simple expression", func(t *testing.T) {
		val, err := engine.RunString("2 + 2")
		require.NoError(t, err)
		assert.Equal(t, "4", val.String())
	})

	t.Run("string manipulation", func(t *testing.T) {
		val, err := engine.RunString(`"hello".toUpperCase() + " " + "WORLD".toLowerCase()`)
		require.NoError(t, err)
		assert.Equal(t, "HELLO world", val.String())
	})

	t.Run("array operations", func(t *testing.T) {
		val, err := engine.RunString(`[1,2,3,4,5].filter(x => x % 2 === 0).join(',')`)
		require.NoError(t, err)
		assert.Equal(t, "2,4", val.String())
	})

	t.Run("object creation", func(t *testing.T) {
		val, err := engine.RunString(`({name: "test", value: 123})`)
		require.NoError(t, err)
		assert.Equal(t, `[object Object]`, val.String())

		// 验证对象属性可访问
		obj := val.ToObject(engine.runtime)
		name := obj.Get("name")
		assert.Equal(t, "test", name.String())
	})

	t.Run("function definition and call", func(t *testing.T) {
		script := `
			function add(a, b) {
				return a + b;
			}
			add(10, 20)
		`
		val, err := engine.RunString(script)
		require.NoError(t, err)
		assert.Equal(t, "30", val.String())
	})

	t.Run("arrow function", func(t *testing.T) {
		val, err := engine.RunString(`const double = x => x * 2; double(21)`)
		require.NoError(t, err)
		assert.Equal(t, "42", val.String())
	})

	t.Run("JSON parse/stringify", func(t *testing.T) {
		val, err := engine.RunString(`JSON.stringify(JSON.parse('{"a":1,"b":2}'))`)
		require.NoError(t, err)
		assert.Contains(t, val.String(), `"a":1`)
	})
}

// ============================================================================
// JsEngine.RunInContext 测试
// ============================================================================

func TestJsEngine_RunInContext(t *testing.T) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	// 设置持久变量
	engine.SetVariable("persistent", "I am persistent")

	t.Run("context vars override", func(t *testing.T) {
		val, err := engine.RunInContext("persistent + ' - ' + tempVar", map[string]interface{}{
			"tempVar": "temporary",
		})
		require.NoError(t, err)
		assert.Equal(t, "I am persistent - temporary", val.String())

		// 验证持久变量未被污染
		val2, err := engine.RunString("persistent")
		require.NoError(t, err)
		assert.Equal(t, "I am persistent", val2.String())
	})

	t.Run("context vars isolated", func(t *testing.T) {
		_, err := engine.RunInContext("nonExistent = 'should not persist'", nil)
		require.NoError(t, err)

		// 验证未污染全局
		val, err := engine.RunString("typeof nonExistent")
		require.NoError(t, err)
		assert.Equal(t, "undefined", val.String())
	})

	t.Run("multiple context vars", func(t *testing.T) {
		val, err := engine.RunInContext("a + b + c", map[string]interface{}{
			"a": 10,
			"b": 20,
			"c": 30,
		})
		require.NoError(t, err)
		assert.Equal(t, "60", val.String())
	})
}

// ============================================================================
// JsEngine.EvaluateExpression 测试
// ============================================================================

func TestJsEngine_EvaluateExpression(t *testing.T) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	t.Run("simple arithmetic", func(t *testing.T) {
		val, err := engine.EvaluateExpression("2 + 3 * 4", nil)
		require.NoError(t, err)
		assert.Equal(t, float64(14), val)
	})

	t.Run("with context variables", func(t *testing.T) {
		val, err := engine.EvaluateExpression("x + y", map[string]interface{}{
			"x": 5,
			"y": 7,
		})
		require.NoError(t, err)
		assert.Equal(t, float64(12), val)
	})

	t.Run("string expression", func(t *testing.T) {
		val, err := engine.EvaluateExpression(`"hello" + " " + "world"`, nil)
		require.NoError(t, err)
		assert.Equal(t, "hello world", val)
	})

	t.Run("array expression", func(t *testing.T) {
		val, err := engine.EvaluateExpression(`[1,2,3].length`, nil)
		require.NoError(t, err)
		assert.Equal(t, float64(3), val)
	})

	t.Run("object expression", func(t *testing.T) {
		val, err := engine.EvaluateExpression(`({a:1, b:2}).a`, nil)
		require.NoError(t, err)
		assert.Equal(t, float64(1), val)
	})

	t.Run("function call expression", func(t *testing.T) {
		val, err := engine.EvaluateExpression(`Math.max(1, 5, 3, 9, 2)`, nil)
		require.NoError(t, err)
		assert.Equal(t, float64(9), val)
	})

	t.Run("empty expression error", func(t *testing.T) {
		_, err := engine.EvaluateExpression("", nil)
		assert.Error(t, err)
	})
}

// ============================================================================
// JsEngine.EvaluateTemplate 测试
// ============================================================================

func TestJsEngine_EvaluateTemplate(t *testing.T) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	t.Run("no template syntax", func(t *testing.T) {
		result, err := engine.EvaluateTemplate("plain text", nil)
		require.NoError(t, err)
		assert.Equal(t, "plain text", result)
	})

	t.Run("single expression", func(t *testing.T) {
		result, err := engine.EvaluateTemplate("Hello, {{name}}!", map[string]interface{}{
			"name": "World",
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", result)
	})

	t.Run("multiple expressions", func(t *testing.T) {
		result, err := engine.EvaluateTemplate("{{greeting}}, {{name}}! You have {{count}} messages.", map[string]interface{}{
			"greeting": "Hi",
			"name":     "Alice",
			"count":    5,
		})
		require.NoError(t, err)
		assert.Equal(t, "Hi, Alice! You have 5 messages.", result)
	})

	t.Run("expression with method call", func(t *testing.T) {
		result, err := engine.EvaluateTemplate("Length: {{str.length}}", map[string]interface{}{
			"str": "hello",
		})
		require.NoError(t, err)
		assert.Equal(t, "Length: 5", result)
	})

	t.Run("expression with array access", func(t *testing.T) {
		result, err := engine.EvaluateTemplate("First: {{arr[0]}}", map[string]interface{}{
			"arr": []string{"a", "b", "c"},
		})
		require.NoError(t, err)
		assert.Equal(t, "First: a", result)
	})

	t.Run("expression with filter-like operation", func(t *testing.T) {
		result, err := engine.EvaluateTemplate("Sum: {{[1,2,3,4].reduce((a,b)=>a+b,0)}}", nil)
		require.NoError(t, err)
		assert.Equal(t, "Sum: 10", result)
	})

	t.Run("nested braces in expression", func(t *testing.T) {
		result, err := engine.EvaluateTemplate("Result: {{JSON.stringify({a:1})}}", nil)
		require.NoError(t, err)
		assert.Contains(t, result, `"a":1`)
	})
}

// ============================================================================
// JsEngine.RunEmbeddedJS 测试
// ============================================================================

func TestJsEngine_RunEmbeddedJS(t *testing.T) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	t.Run("no js blocks", func(t *testing.T) {
		result, err := engine.RunEmbeddedJS("plain text without js")
		require.NoError(t, err)
		assert.Equal(t, "plain text without js", result.String())
	})

	t.Run("single js block", func(t *testing.T) {
		result, err := engine.RunEmbeddedJS("before <js>2 + 3</js> after")
		require.NoError(t, err)
		assert.Equal(t, "5", result.String())
	})

	t.Run("js block with assignment", func(t *testing.T) {
		result, err := engine.RunEmbeddedJS(`<js>var x = 10; x * 2</js> result`)
		require.NoError(t, err)
		assert.Equal(t, "20", result.String())
	})

	t.Run("multiple js blocks", func(t *testing.T) {
		result, err := engine.RunEmbeddedJS(`<js>1</js> + <js>2</js> + <js>3</js>`)
		require.NoError(t, err)
		assert.Equal(t, "6", result.String())
	})

	t.Run("js block with function", func(t *testing.T) {
		result, err := engine.RunEmbeddedJS(`<js>function add(a,b){return a+b} add(5,7)</js> done`)
		require.NoError(t, err)
		assert.Equal(t, "12", result.String())
	})

	t.Run("case insensitive js tag", func(t *testing.T) {
		result, err := engine.RunEmbeddedJS("before <JS>42</JS> after")
		require.NoError(t, err)
		assert.Equal(t, "42", result.String())
	})

	t.Run("js block with string", func(t *testing.T) {
		result, err := engine.RunEmbeddedJS(`<js>"hello " + "world"</js>`)
		require.NoError(t, err)
		assert.Equal(t, "hello world", result.String())
	})
}

// ============================================================================
// JsExtensions - ajax 测试
// ============================================================================

func TestJsExtensions_Ajax(t *testing.T) {
	t.Run("default implementation returns error for invalid url", func(t *testing.T) {
		engine := NewJsEngine(nil)
		defer engine.Close()

		val, err := engine.RunString(`ajax("http://invalid-url-test-12345.com")`)
		assert.Error(t, err)
		assert.Nil(t, val)
	})

	t.Run("ajax available in ext object", func(t *testing.T) {
		engine := NewJsEngine(nil)
		defer engine.Close()

		// 验证 ext.ajax 存在
		val, err := engine.RunString(`typeof ext.ajax`)
		require.NoError(t, err)
		assert.Equal(t, "function", val.String())
	})
}

// ============================================================================
// JsExtensions - cookie 测试
// ============================================================================

func TestJsExtensions_Cookie(t *testing.T) {
	t.Run("cookie.get returns error for missing key", func(t *testing.T) {
		engine := NewJsEngine(nil)
		defer engine.Close()

		val, err := engine.RunString(`ext.cookie.get('nonexistent')`)
		assert.Error(t, err)
		assert.Nil(t, val)
	})

	t.Run("cookie.set and get work together", func(t *testing.T) {
		// 使用自定义扩展来测试
		ext := &customTestExtensions{
			cookieValue: "test-value",
		}
		engine := NewJsEngine(&JsEngineOptions{
			Extensions: ext,
		})
		defer engine.Close()

		val, err := engine.RunString(`ext.cookie.get('session')`)
		require.NoError(t, err)
		assert.Equal(t, "test-value", val.String())
	})

	t.Run("cookie API available", func(t *testing.T) {
		engine := NewJsEngine(nil)
		defer engine.Close()

		val, err := engine.RunString(`typeof ext.cookie.get + ',' + typeof ext.cookie.set`)
		require.NoError(t, err)
		assert.Equal(t, "function,function", val.String())
	})
}

// ============================================================================
// JsExtensions - cache 测试
// ============================================================================

func TestJsExtensions_Cache(t *testing.T) {
	t.Run("cache.get returns error for missing key", func(t *testing.T) {
		engine := NewJsEngine(nil)
		defer engine.Close()

		val, err := engine.RunString(`ext.cache.get('nonexistent')`)
		assert.Error(t, err)
		assert.Nil(t, val)
	})

	t.Run("cache.set and get work together", func(t *testing.T) {
		engine := NewJsEngine(nil)
		defer engine.Close()

		// 先设置缓存
		_, err := engine.RunString(`ext.cache.set('key1', 'value1', 60)`)
		require.NoError(t, err)

		// 再获取
		val, err := engine.RunString(`ext.cache.get('key1')`)
		require.NoError(t, err)
		assert.Equal(t, "value1", val.String())
	})

	t.Run("cache with ttl expiration", func(t *testing.T) {
		ext := newDefaultExtensions(nil)
		defer ext.engine.Close()

		// 设置一个立即过期的缓存
		ext.CacheSet("expiring", "value", 0)

		// 手动设置过期时间
		ext.cacheMu.Lock()
		if entry, ok := ext.cache["expiring"]; ok {
			entry.expireAt = time.Now().Add(-1 * time.Second)
		}
		ext.cacheMu.Unlock()

		_, err := ext.CacheGet("expiring")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("cache API available", func(t *testing.T) {
		engine := NewJsEngine(nil)
		defer engine.Close()

		val, err := engine.RunString(`typeof ext.cache.get + ',' + typeof ext.cache.set`)
		require.NoError(t, err)
		assert.Equal(t, "function,function", val.String())
	})
}

// ============================================================================
// JsExtensions - base64 测试
// ============================================================================

func TestJsExtensions_Base64(t *testing.T) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	t.Run("base64Encode", func(t *testing.T) {
		val, err := engine.RunString(`base64Encode("Hello, 世界!")`)
		require.NoError(t, err)
		assert.Equal(t, "SGVsbG8sIOS4lueVjCE=", val.String())
	})

	t.Run("base64Decode", func(t *testing.T) {
		val, err := engine.RunString(`base64Decode("SGVsbG8sIOS4lueVjCE=")`)
		require.NoError(t, err)
		assert.Equal(t, "Hello, 世界!", val.String())
	})

	t.Run("base64 round trip", func(t *testing.T) {
		val, err := engine.RunString(`base64Decode(base64Encode("test content 123"))`)
		require.NoError(t, err)
		assert.Equal(t, "test content 123", val.String())
	})

	t.Run("base64Decode invalid input", func(t *testing.T) {
		val, err := engine.RunString(`base64Decode("invalid!!!base64")`)
		assert.Error(t, err)
		assert.Nil(t, val)
	})

	t.Run("ext.base64Encode available", func(t *testing.T) {
		val, err := engine.RunString(`ext.base64Encode("test")`)
		require.NoError(t, err)
		assert.Equal(t, "dGVzdA==", val.String())
	})
}

// ============================================================================
// JsExtensions - md5 测试
// ============================================================================

func TestJsExtensions_Md5(t *testing.T) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	t.Run("md5Encode empty string", func(t *testing.T) {
		val, err := engine.RunString(`md5Encode("")`)
		require.NoError(t, err)
		assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", val.String())
	})

	t.Run("md5Encode hello", func(t *testing.T) {
		val, err := engine.RunString(`md5Encode("hello")`)
		require.NoError(t, err)
		assert.Equal(t, "5d41402abc4b2a76b9719d911017c592", val.String())
	})

	t.Run("md5Encode chinese", func(t *testing.T) {
		val, err := engine.RunString(`md5Encode("你好")`)
		require.NoError(t, err)
		assert.Equal(t, "7eca68990d3381ce9049c66d2266f0a4", val.String())
	})

	t.Run("ext.md5Encode available", func(t *testing.T) {
		val, err := engine.RunString(`ext.md5Encode("test")`)
		require.NoError(t, err)
		assert.Equal(t, "098f6bcd4621d373cade4e832627b4f6", val.String())
	})
}

// ============================================================================
// JsExtensions - log 测试
// ============================================================================

func TestJsExtensions_Log(t *testing.T) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	// log 主要验证不报错即可
	_, err := engine.RunString(`log("test log message")`)
	assert.NoError(t, err)

	_, err = engine.RunString(`ext.log("ext log message")`)
	assert.NoError(t, err)
}

// ============================================================================
// 综合测试
// ============================================================================

func TestJsEngine_Comprehensive(t *testing.T) {
	engine := NewJsEngine(&JsEngineOptions{
		Variables: map[string]interface{}{
			"pageUrl":  "https://example.com",
			"pageTitle": "示例页面",
		},
	})
	defer engine.Close()

	t.Run("combined template with expressions and extensions", func(t *testing.T) {
		script := `
			var title = pageTitle;
			var encoded = base64Encode(title);
			var md5 = md5Encode(title);
			"Title: " + title + " | Base64: " + encoded + " | MD5: " + md5
		`
		val, err := engine.RunString(script)
		require.NoError(t, err)

		result := val.String()
		assert.Contains(t, result, "Title: 示例页面")
		assert.Contains(t, result, "Base64:")
		assert.Contains(t, result, "MD5:")
	})

	t.Run("array processing with extensions", func(t *testing.T) {
		script := `
			var items = ["apple", "banana", "cherry"];
			var result = items.map(function(item) {
				return item + ":" + md5Encode(item).substring(0, 8);
			});
			result.join("; ")
		`
		val, err := engine.RunString(script)
		require.NoError(t, err)

		result := val.String()
		assert.Contains(t, result, "apple:")
		assert.Contains(t, result, "banana:")
		assert.Contains(t, result, "cherry:")
	})

	t.Run("conditional logic with extensions", func(t *testing.T) {
		script := `
			var input = "test123";
			var encoded = base64Encode(input);
			var decoded = base64Decode(encoded);
			decoded === input ? "OK" : "FAIL"
		`
		val, err := engine.RunString(script)
		require.NoError(t, err)
		assert.Equal(t, "OK", val.String())
	})
}

// ============================================================================
// 变量管理测试
// ============================================================================

func TestJsEngine_Variables(t *testing.T) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	t.Run("set and get single variable", func(t *testing.T) {
		engine.SetVariable("myVar", "myValue")
		val := engine.GetVariable("myVar")
		assert.Equal(t, "myValue", val)
	})

	t.Run("set and get multiple variables", func(t *testing.T) {
		engine.SetVariables(map[string]interface{}{
			"a": 1,
			"b": "two",
			"c": true,
		})

		assert.Equal(t, float64(1), engine.GetVariable("a"))
		assert.Equal(t, "two", engine.GetVariable("b"))
		assert.Equal(t, true, engine.GetVariable("c"))
	})

	t.Run("get non-existent variable", func(t *testing.T) {
		val := engine.GetVariable("nonexistent")
		assert.Nil(t, val)
	})

	t.Run("update existing variable", func(t *testing.T) {
		engine.SetVariable("counter", 1)
		engine.SetVariable("counter", 2)
		val := engine.GetVariable("counter")
		assert.Equal(t, float64(2), val)
	})

	t.Run("get all variables", func(t *testing.T) {
		engine.SetVariables(map[string]interface{}{
			"x": 10,
			"y": 20,
		})
		vars := engine.GetVariables()
		assert.Equal(t, float64(10), vars["x"])
		assert.Equal(t, float64(20), vars["y"])
	})
}

// ============================================================================
// 资源管理测试
// ============================================================================

func TestJsEngine_Close(t *testing.T) {
	engine := NewJsEngine(nil)
	engine.Close()

	// 关闭后引擎应不再可用
	// (实际测试中我们主要验证不 panic)
	assert.Nil(t, engine.runtime)
}

// ============================================================================
// 自定义扩展实现测试
// ============================================================================

type customTestExtensions struct {
	cookieValue string
	cacheValue  string
}

func (c *customTestExtensions) Ajax(url string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"status": 200,
		"body":   fmt.Sprintf("custom ajax response for %s", url),
	}, nil
}

func (c *customTestExtensions) Cookie() CookieExtensions {
	return c
}

func (c *customTestExtensions) Get(key string) (string, error) {
	if key == "session" {
		return c.cookieValue, nil
	}
	return "", fmt.Errorf("cookie not found: %s", key)
}

func (c *customTestExtensions) Set(key, value string) error {
	c.cookieValue = value
	return nil
}

func (c *customTestExtensions) Cache() CacheExtensions {
	return c
}

func (c *customTestExtensions) CacheGet(key string) (interface{}, error) {
	if key == "data" {
		return c.cacheValue, nil
	}
	return nil, fmt.Errorf("cache not found: %s", key)
}

func (c *customTestExtensions) CacheSet(key string, value interface{}, ttl int64) error {
	c.cacheValue = fmt.Sprintf("%v", value)
	return nil
}

func (c *customTestExtensions) Base64Encode(str string) string {
	return "custom-" + base64.StdEncoding.EncodeToString([]byte(str))
}

func (c *customTestExtensions) Base64Decode(str string) (string, error) {
	if str == "custom-dGVzdA==" {
		return "test", nil
	}
	return "", fmt.Errorf("custom decode error")
}

func (c *customTestExtensions) Md5Encode(str string) string {
	return "custom-" + fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

func (c *customTestExtensions) Log(msg string) {
	// 自定义日志实现
}

func TestCustomExtensions(t *testing.T) {
	customExt := &customTestExtensions{
		cookieValue: "custom-cookie",
		cacheValue:  "custom-cache",
	}

	engine := NewJsEngine(&JsEngineOptions{
		Extensions: customExt,
	})
	defer engine.Close()

	t.Run("custom ajax", func(t *testing.T) {
		val, err := engine.RunString(`ajax("http://custom.test")`)
		require.NoError(t, err)
		assert.Contains(t, val.String(), "custom ajax response")
	})

	t.Run("custom cookie", func(t *testing.T) {
		val, err := engine.RunString(`ext.cookie.get('session')`)
		require.NoError(t, err)
		assert.Equal(t, "custom-cookie", val.String())
	})

	t.Run("custom base64", func(t *testing.T) {
		val, err := engine.RunString(`base64Encode("test")`)
		require.NoError(t, err)
		assert.Equal(t, "custom-dGVzdA==", val.String())
	})

	t.Run("custom md5", func(t *testing.T) {
		val, err := engine.RunString(`md5Encode("test")`)
		require.NoError(t, err)
		assert.Contains(t, val.String(), "custom-")
	})
}

// ============================================================================
// 性能测试
// ============================================================================

func BenchmarkJsEngine_RunString(b *testing.B) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.RunString(`
			function fib(n) {
				if (n <= 1) return n;
				return fib(n - 1) + fib(n - 2);
			}
			fib(10)
		`)
	}
}

func BenchmarkJsEngine_EvaluateExpression(b *testing.B) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.EvaluateExpression("a * b + c", map[string]interface{}{
			"a": 10,
			"b": 20,
			"c": 30,
		})
	}
}

func BenchmarkJsEngine_EvaluateTemplate(b *testing.B) {
	engine := NewJsEngine(nil)
	defer engine.Close()

	template := "Hello, {{name}}! Your balance is {{balance}}. MD5: {{md5Encode(email)}}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.EvaluateTemplate(template, map[string]interface{}{
			"name":   "Test User",
			"balance": 1234.56,
			"email":  "test@example.com",
		})
	}
}
