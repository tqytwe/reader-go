// Package rule 提供多种规则解析器（CSS、XPath、JSONPath、JS 等）
package rule

import (
	"fmt"
	"strconv"
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

	// 将 JSONPath 语法转换为 gjson 语法
	gjsonPath, hasSlice, sliceStart, sliceEnd, sliceBasePath, sliceSuffix := convertToGjsonPath(path)

	// 如果包含数组切片，需要特殊处理
	if hasSlice {
		return p.execSliceQuery(jsonStr, sliceBasePath, sliceStart, sliceEnd, sliceSuffix)
	}

	var results []string

	// 使用 gjson 执行查询
	result := gjson.Get(jsonStr, gjsonPath)

	if !result.Exists() {
		return nil, nil
	}

	// 是否是根查询（$ 或 @this）— 根查询需要展开对象/数组的所有值
	isRootQuery := gjsonPath == "@this"

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
	case gjson.Null:
		results = append(results, "null")
	case gjson.JSON:
		if result.IsArray() {
			// 数组 — 遍历元素，递归展平嵌套数组
			p.flattenArray(result, &results)
		} else if isRootQuery {
			// 根对象查询 — 展开所有字段值
			result.ForEach(func(key, value gjson.Result) bool {
				results = append(results, p.extractValue(value))
				return true
			})
		} else {
			// 普通对象 — 返回紧凑 JSON
			results = append(results, compactJSON(result.Raw))
		}
	}

	return results, nil
}

// convertToGjsonPath 将 JSONPath 语法转换为 gjson 语法
// 返回: gjson路径, 是否含切片, 切片起始, 切片结束, 切片前路径, 切片后路径
func convertToGjsonPath(path string) (gjsonPath string, hasSlice bool, sliceStart, sliceEnd int, sliceBasePath, sliceSuffix string) {
	// 先去掉 $ 前缀
	clean := path
	if strings.HasPrefix(clean, "$.") {
		clean = clean[2:]
	} else if clean == "$" {
		return "@this", false, 0, 0, "", ""
	} else if strings.HasPrefix(clean, "$[") {
		clean = clean[1:]
	}

	// 扫描路径，转换 [...] 语法
	var result strings.Builder
	i := 0
	for i < len(clean) {
		if clean[i] == '[' {
			// 找到匹配的 ]
			j := i + 1
			depth := 1
			for j < len(clean) && depth > 0 {
				if clean[j] == '[' {
					depth++
				} else if clean[j] == ']' {
					depth--
				}
				j++
			}
			bracketContent := clean[i+1 : j-1]

			// 检查是否是切片语法 [N:M] 或 [N:]
			if colonIdx := strings.Index(bracketContent, ":"); colonIdx >= 0 {
				// 切片语法
				startStr := strings.TrimSpace(bracketContent[:colonIdx])
				endStr := strings.TrimSpace(bracketContent[colonIdx+1:])

				sStart := 0
				sEnd := -1 // -1 表示到末尾

				if startStr != "" {
					if v, err := strconv.Atoi(startStr); err == nil {
						sStart = v
					}
				}
				if endStr != "" {
					if v, err := strconv.Atoi(endStr); err == nil {
						sEnd = v
					}
				}

				// 收集切片前的路径
				basePath := result.String()
				// 去掉末尾的 .
				basePath = strings.TrimRight(basePath, ".")

				// 收集切片后的路径
				suffix := ""
				if j < len(clean) {
					suffix = clean[j:]
					// 去掉开头的 .
					suffix = strings.TrimLeft(suffix, ".")
				}

				return "", true, sStart, sEnd, basePath, suffix
			}

			// [*] 通配符
			if bracketContent == "*" {
				// 如果 [*] 在路径末尾（后面没有更多内容），直接省略
				// gjson 会返回整个数组，遍历即可
				if j >= len(clean) {
					// 末尾 [*] — 省略
					i = j
					continue
				}
				// 中间 [*] → .# (通配符，用于访问数组元素属性)
				result.WriteString(".#")
				i = j
				continue
			}

			// [N] → .N (数组索引)
			if v, err := strconv.Atoi(bracketContent); err == nil {
				result.WriteString(fmt.Sprintf(".%d", v))
				i = j
				continue
			}

			// 其他 [...] 保持原样（gjson 的条件语法等）
			result.WriteString(".")
			result.WriteString(bracketContent)
			i = j
			continue
		}

		result.WriteByte(clean[i])
		i++
	}

	r := result.String()
	if r == "" {
		// 空路径（如 $[*] 去掉 [*] 后）→ 返回根元素
		return "@this", false, 0, 0, "", ""
	}
	return r, false, 0, 0, "", ""
}

// execSliceQuery 处理包含数组切片的查询
func (p *JSONPathParser) execSliceQuery(jsonStr string, basePath string, start, end int, suffix string) ([]string, error) {
	// 获取基础数组
	arrayResult := gjson.Get(jsonStr, basePath)
	if !arrayResult.Exists() || !arrayResult.IsArray() {
		return nil, nil
	}

	// 将数组元素收集到切片
	var items []gjson.Result
	arrayResult.ForEach(func(key, value gjson.Result) bool {
		items = append(items, value)
		return true
	})

	// 应用切片
	total := len(items)
	if start < 0 {
		start = 0
	}
	if end < 0 || end > total {
		end = total
	}
	if start >= total || start >= end {
		return nil, nil
	}
	sliced := items[start:end]

	// 对切片后的每个元素应用后缀路径
	var results []string
	for _, item := range sliced {
		if suffix == "" {
			results = append(results, p.extractValue(item))
		} else {
			// 在子元素上执行后缀查询
			subResult := gjson.Parse(item.Raw)
			subQuery := gjson.Get(item.Raw, suffix)
			if subQuery.Exists() {
				if subQuery.IsArray() {
					subQuery.ForEach(func(key, value gjson.Result) bool {
						results = append(results, p.extractValue(value))
						return true
					})
				} else {
					results = append(results, p.extractValue(subQuery))
				}
			}
			_ = subResult
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
	case gjson.Null:
		return "null"
	case gjson.JSON:
		if result.IsArray() || result.IsObject() {
			return compactJSON(result.Raw)
		}
		return result.String()
	default:
		return ""
	}
}

// flattenArray 递归展平嵌套数组，将所有叶子值添加到 results
// 例如 [["A","B"],["C"]] → ["A","B","C"]
func (p *JSONPathParser) flattenArray(result gjson.Result, results *[]string) {
	result.ForEach(func(key, value gjson.Result) bool {
		if value.IsArray() {
			// 递归展平子数组
			p.flattenArray(value, results)
		} else {
			*results = append(*results, p.extractValue(value))
		}
		return true
	})
}

// compactJSON 将 JSON 文本压缩为紧凑格式（去除所有多余空白）
func compactJSON(raw string) string {
	var buf strings.Builder
	inString := false
	escapeNext := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]

		if escapeNext {
			buf.WriteByte(ch)
			escapeNext = false
			continue
		}

		if inString {
			buf.WriteByte(ch)
			if ch == '\\' {
				escapeNext = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			buf.WriteByte(ch)
			continue
		}

		// 跳过所有空白字符
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			continue
		}

		buf.WriteByte(ch)
	}

	return buf.String()
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

// Query 统一查询函数：自动检测输入数据类型，支持 JSON 和 HTML/XML
// 当表达式以 / 或 // 开头时，使用 XPath 解析；否则使用 JSONPath 解析
func Query(data, expression string) ([]string, error) {
	// Auto-detect: if expression looks like XPath, use XPath parser
	if strings.HasPrefix(expression, "/") || strings.HasPrefix(expression, "//") {
		return QueryXPath(data, expression)
	}
	// Default to JSONPath
	parser := NewJSONPathParser()
	return parser.ParseJSONPath(expression, data)
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
