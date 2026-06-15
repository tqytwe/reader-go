package rule

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Executor 统一规则执行器
type Executor struct {
	js *JsEngine
}

// NewExecutor 创建执行器
func NewExecutor() *Executor {
	return &Executor{
		js: NewJsEngine(&JsEngineOptions{Timeout: 0}),
	}
}

// ModeFromString 解析模式字符串
func ModeFromString(s string) RuleMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "xpath":
		return ModeXPath
	case "json", "jsonpath":
		return ModeJSONPath
	case "css":
		return ModeCSS
	case "regex":
		return ModeRegex
	case "js":
		return ModeJS
	default:
		return ModeDefault
	}
}

// Execute 根据 mode 与 rule 从 body 提取结果
func (e *Executor) Execute(ctx context.Context, mode RuleMode, rule string, body string) ([]string, error) {
	_ = ctx
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return nil, fmt.Errorf("empty rule")
	}

	// 组合规则：取第一段（完整 &&/%% 后续迭代）
	if strings.Contains(rule, "&&") || strings.Contains(rule, "||") || strings.Contains(rule, "%%") {
		analyzer := NewRuleAnalyzer()
		parts, _, err := analyzer.Split(rule)
		if err != nil || len(parts) == 0 {
			return nil, fmt.Errorf("split rule: %w", err)
		}
		rule = parts[0]
	}

	if strings.HasPrefix(rule, "@js:") {
		if e.js == nil {
			return nil, fmt.Errorf("js engine not available")
		}
		script := strings.TrimPrefix(rule, "@js:")
		v, err := e.js.RunString(script)
		if err != nil {
			return nil, err
		}
		return []string{v.String()}, nil
	}

	selector := stripModePrefix(rule)
	switch mode {
	case ModeXPath:
		return QueryXPath(body, selector)
	case ModeJSONPath:
		return Query(body, selector)
	case ModeRegex:
		return ParseRegex(selector, body)
	case ModeJS:
		if e.js == nil {
			return nil, fmt.Errorf("js engine not available")
		}
		v, err := e.js.RunString(selector)
		if err != nil {
			return nil, err
		}
		return []string{v.String()}, nil
	default:
		return ParseCSSWithDoc(selector, body)
	}
}

func stripModePrefix(rule string) string {
	prefixes := []string{"@XPath:", "@xpath:", "@Json:", "@json:", "@JSONPath:", "@jsonpath:", "@CSS:", "@css:", "@Regex:", "@regex:", "@Default:", "@default:"}
	for _, p := range prefixes {
		if strings.HasPrefix(rule, p) {
			return strings.TrimPrefix(rule, p)
		}
	}
	return rule
}

// ExecuteFirst 执行并返回首个结果
func (e *Executor) ExecuteFirst(ctx context.Context, mode RuleMode, rule string, body string) (string, error) {
	results, err := e.Execute(ctx, mode, rule, body)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", nil
	}
	return results[0], nil
}

// ExecuteOnDocument 在 goquery Document 上执行 CSS
func ExecuteOnDocument(doc *goquery.Document, selector string) ([]string, error) {
	return ParseCSS(selector, doc)
}
