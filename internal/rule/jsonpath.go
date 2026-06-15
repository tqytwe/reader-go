// Package rule 提供多种规则解析器（CSS、XPath、JSONPath、JS 等）
package rule

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// JSONPathParser 使用 gjson 库解析 JSONPath 查询
// 支持基本 JSONPath 语法和链式操作符（&&、||）
type JSONPathParser struct{}

// NewJSONPathParser 创建新的 JSONPath 解析器实例
func NewJSONPathParser() *JSONPathParser {
	return &JSONPathParser{}
}

// ParseJSONPath 解析并执行 JSONPath 查询
// 支持单条查询和复合规则（含 &&、|| 操作符）
//
// 参数:
//   - query:   JSONPath 表达式或复合规则（如 "$.book.title" 或 "$.a && $.b"）
//   - jsonStr: JSON 数据字符串
//
// 返回所有匹配的值（去重后的字符串切片）
func (p *JSONPathParser) ParseJSONPath(query string, jsonStr string) ([]string, error) {
	if query == "" || jsonStr == "" {
		return nil, nil
	}

	// Step 1: 使用 RuleAnalyzer 分析复合规则（处理 &&、|| 操作符）
	analyzer := NewRuleAnalyzer()
	result := analyzer.Analyze(query)
	if result.Error != nil {
		return nil, fmt.Errorf("rule analysis failed: %w", result.Error)
	}

	if len(result.Segments) == 0 {
		return nil, nil
	}

	// Step 2: 执行每个 JSONPath 查询并按操作符合并结果
	// 段和操作符交替：seg[0], op[0], seg[1], op[1], seg[2]...
	var results []string

	// 执行第一条
	firstExpr := strings.TrimSpace(result.Segments[0].Selector)
	if firstExpr != "" {
		matches, err := p.execJSONPath(firstExpr, jsonStr)
		if err != nil {
			return nil, fmt.Errorf("jsonpath query %q failed: %w", firstExpr, err)
		}
		results = append(results, matches...)
	}

	// 依次处理后续段
	for i := 1; i < len(result.Segments); i++ {
		opStr := ""
		if i-1 < len(result.Operators) {
			opStr = result.Operators[i-1]
		}
		op := ParseOperatorType(opStr)

		expr := strings.TrimSpace(result.Segments[i].Selector)
		if expr == "" {
			continue
		}

		matches, err := p.execJSONPath(expr, jsonStr)
		if err != nil {
			return nil, fmt.Errorf("jsonpath query %q failed: %w", expr, err)
		}

		results = p.combineResults(results, matches, op)
	}

	return results, nil
}

// execJSONPath 执行单个 JSONPath 查询
func (p *JSONPathParser) execJSONPath(path string, jsonStr string) ([]string, error) {
	// 验证 JSON 是否有效
	if !gjson.Valid(jsonStr) {
		return nil, fmt.Errorf("invalid JSON")
	}

	// 确保路径以 $ 开头
	if !strings.HasPrefix(path, "$") {
		path = "$" + path
	}

	var results []string

	// 使用 gjson 执行查询
	result := gjson.Get(jsonStr, path)

	if !result.Exists() {
		return nil, nil
	}

	// 根据结果类型处理
	switch result.Type {
	case gjson.String:
		results = append(results, result.String())
	case gjson.Number:
		results = append(results, result.String())
	case gjson.True:
		results = append(results, "true")
	case gjson.False:
		results = append(results, "false")
	case gjson.JSON:
		// 数组或对象 — 需要遍历
		if result.IsArray() {
			result.ForEach(func(key, value gjson.Result) bool {
				results = append(results, p.extractValue(value))
				return true
			})
		} else if result.IsObject() {
			// 对象 — 遍历所有字段值
			result.ForEach(func(key, value gjson.Result) bool {
				results = append(results, p.extractValue(value))
				return true
			})
		} else {
			results = append(results, result.Raw)
		}
	}

	return results, nil
}

// extractValue 从 gjson.Result 中提取字符串值
func (p *JSONPathParser) extractValue(result gjson.Result) string {
	switch result.Type {
	case gjson.String:
		return result.String()
	case gjson.Number:
		return result.String()
	case gjson.True:
		return "true"
	case gjson.False:
		return "false"
	case gjson.JSON:
		if result.IsArray() || result.IsObject() {
			return result.Raw
		}
		return result.String()
	default:
		return ""
	}
}

// combineResults 根据操作符合并两个结果集
func (p *JSONPathParser) combineResults(left, right []string, op OperatorType) []string {
	switch op {
	case OpAnd:
		// && : 交集 — 保留两边都出现的值（按 left 顺序，去重）
		return p.intersection(left, right)
	case OpOr:
		// || : 或操作 — 取第一个非空结果（短路求值语义）
		return p.orShortCircuit(left, right)
	default:
		return right
	}
}

// intersection 返回两个切片中都存在的元素（保持 left 顺序，去重）
func (p *JSONPathParser) intersection(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}

	rightSet := make(map[string]struct{}, len(right))
	for _, v := range right {
		if v != "" {
			rightSet[v] = struct{}{}
		}
	}

	var result []string
	seen := make(map[string]struct{})
	for _, v := range left {
		if v != "" {
			if _, ok := rightSet[v]; ok {
				if _, already := seen[v]; !already {
					seen[v] = struct{}{}
					result = append(result, v)
				}
			}
		}
	}
	return result
}

// orShortCircuit 或操作：取第一个非空结果集
// 如果 left 非空则返回 left，否则返回 right
func (p *JSONPathParser) orShortCircuit(left, right []string) []string {
	if len(left) > 0 {
		// 去重
		seen := make(map[string]struct{})
		var result []string
		for _, v := range left {
			if v != "" {
				if _, ok := seen[v]; !ok {
					seen[v] = struct{}{}
					result = append(result, v)
				}
			}
		}
		return result
	}
	// left 为空，返回 right（同样去重）
	seen := make(map[string]struct{})
	var result []string
	for _, v := range right {
		if v != "" {
			if _, ok := seen[v]; !ok {
				seen[v] = struct{}{}
				result = append(result, v)
			}
		}
	}
	return result
}

// ParseJSON 将字符串解析为 gjson.Result
func (p *JSONPathParser) ParseJSON(jsonStr string) (gjson.Result, error) {
	if !gjson.Valid(jsonStr) {
		return gjson.Result{}, fmt.Errorf("invalid JSON")
	}
	return gjson.Parse(jsonStr), nil
}

// Query 便捷函数：直接对 JSON 字符串执行 JSONPath 查询
func Query(jsonStr, jsonPathExpr string) ([]string, error) {
	parser := NewJSONPathParser()
	return parser.ParseJSONPath(jsonPathExpr, jsonStr)
}

// QueryFirst 返回第一个匹配值（或空字符串）
func QueryFirst(jsonStr, jsonPathExpr string) (string, error) {
	results, err := Query(jsonStr, jsonPathExpr)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", nil
	}
	return results[0], nil
}

// QueryOnResult 在已有的 gjson.Result 上执行子查询
func QueryOnResult(base gjson.Result, jsonPathExpr string) ([]string, error) {
	if !base.Exists() {
		return nil, nil
	}
	parser := NewJSONPathParser()
	return parser.ParseJSONPath(jsonPathExpr, base.Raw)
}
