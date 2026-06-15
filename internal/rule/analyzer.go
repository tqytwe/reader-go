package rule

import (
	"strings"
)

// RuleAnalyzer 规则分析器
// 负责将复合规则切分为多个原子规则段
// 核心特性：平衡组感知，正确处理 []、()、"" 内部的操作符
type RuleAnalyzer struct{}

// NewRuleAnalyzer 创建新的规则分析器
func NewRuleAnalyzer() *RuleAnalyzer {
	return &RuleAnalyzer{}
}

// Analyze 分析规则字符串，返回解析结果
// 这是对外暴露的主要入口
func (a *RuleAnalyzer) Analyze(rule string) *ParseResult {
	result := &ParseResult{
		Original:  rule,
		Segments:  make([]RuleSegment, 0),
		Operators: make([]string, 0),
	}

	if rule == "" {
		result.Error = &ParseError{Msg: "empty rule"}
		return result
	}

	// 第一步：平衡组感知的切分
	segments, operators, err := a.splitBalanced(rule)
	if err != nil {
		result.Error = err
		return result
	}

	// 第二步：解析每个段
	for _, seg := range segments {
		raw := seg
		seg = strings.TrimSpace(seg)

		if seg == "" {
			continue
		}

		mode, content := a.detectMode(seg)

		result.Segments = append(result.Segments, RuleSegment{
			Mode:    mode,
			Selector: content,
			Raw:     raw,
		})
	}

	result.Operators = operators

	return result
}

// splitBalanced 平衡组感知的规则切分
// 核心算法：迭代扫描，跟踪平衡组状态
// 返回切分后的段、操作符序列、错误
func (a *RuleAnalyzer) splitBalanced(rule string) ([]string, []string, error) {
	var segments []string
	var operators []string

	start := 0
	i := 0
	n := len(rule)

	// 状态跟踪
	bracketDepth := 0   // [] 嵌套深度
	parenDepth := 0     // () 嵌套深度
	inString := false   // 是否在双引号字符串内
	escapeNext := false // 下一个字符是否被转义

	for i < n {
		ch := rule[i]

		// 处理转义字符
		if escapeNext {
			escapeNext = false
			i++
			continue
		}

		// 在字符串内处理转义
		if inString && ch == '\\' {
			escapeNext = true
			i++
			continue
		}

		// 处理字符串开始/结束
		if ch == '"' {
			if !inString {
				inString = true
			} else {
				inString = false
			}
			i++
			continue
		}

		// 在字符串内，不处理其他结构
		if inString {
			i++
			continue
		}

		// 处理方括号
		if ch == '[' {
			bracketDepth++
			i++
			continue
		}
		if ch == ']' {
			if bracketDepth > 0 {
				bracketDepth--
			}
			i++
			continue
		}

		// 处理圆括号
		if ch == '(' {
			parenDepth++
			i++
			continue
		}
		if ch == ')' {
			if parenDepth > 0 {
				parenDepth--
			}
			i++
			continue
		}

		// === 在顶层（所有组闭合，不在字符串内）检查操作符 ===
		if bracketDepth == 0 && parenDepth == 0 {
			// 检查 && (交集)
			if i+1 < n && rule[i] == '&' && rule[i+1] == '&' {
				if i > start {
					segments = append(segments, rule[start:i])
				}
				operators = append(operators, "&&")
				i += 2
				start = i
				continue
			}
			// 检查 || (并集)
			if i+1 < n && rule[i] == '|' && rule[i+1] == '|' {
				if i > start {
					segments = append(segments, rule[start:i])
				}
				operators = append(operators, "||")
				i += 2
				start = i
				continue
			}
			// 检查 %% (交叉合并)
			if i+1 < n && rule[i] == '%' && rule[i+1] == '%' {
				if i > start {
					segments = append(segments, rule[start:i])
				}
				operators = append(operators, "%%")
				i += 2
				start = i
				continue
			}
		}

		i++
	}

	// 添加最后一段
	if start < n {
		segments = append(segments, rule[start:])
	}

	// 对于有未闭合分隔符的段，进行二次扫描，在忽略深度的情况下拆分操作符
	for i := 0; i < len(segments); i++ {
		seg := segments[i]
		if hasUnclosedDelimiters(seg) {
			subSegments, subOperators := splitSegmentFallback(seg)
			if len(subSegments) > 1 {
				// 替换当前段为子段，在操作符列表中插入子操作符
				segments = append(segments[:i], append(subSegments, segments[i+1:]...)...)
				// 在 operators 中插入子操作符（在当前操作符之前）
				// 需要找到正确的插入位置
				newOps := make([]string, 0, len(operators)+len(subOperators))
				for j, op := range operators {
					if j == i {
						newOps = append(newOps, subOperators...)
					}
					newOps = append(newOps, op)
				}
				if i >= len(operators) {
					newOps = append(newOps, subOperators...)
				}
				operators = newOps
			}
		}
	}

	// 验证：操作符数量应该等于段数量-1
	if len(operators) > 0 && len(segments) != len(operators)+1 {
		return nil, nil, &ParseError{Msg: "mismatched operators and segments"}
	}

	return segments, operators, nil
}

// hasUnclosedDelimiters 判断规则是否包含未闭合的分隔符
func hasUnclosedDelimiters(rule string) bool {
	depth := 0
	for _, ch := range rule {
		if ch == '[' || ch == '(' {
			depth++
		} else if ch == ']' || ch == ')' {
			if depth > 0 {
				depth--
			}
		}
	}
	return depth > 0
}

// splitSegmentFallback 对包含未闭合分隔符的段进行拆分
func splitSegmentFallback(seg string) ([]string, []string) {
	var segments, operators []string
	start := 0
	n := len(seg)
	i := 0
	for i < n {
		if i+1 < n {
			if seg[i] == '&' && seg[i+1] == '&' {
				if i > start {
					segments = append(segments, seg[start:i])
				}
				operators = append(operators, "&&")
				i += 2
				start = i
				continue
			}
			if seg[i] == '|' && seg[i+1] == '|' {
				if i > start {
					segments = append(segments, seg[start:i])
				}
				operators = append(operators, "||")
				i += 2
				start = i
				continue
			}
		}
		i++
	}
	if start < n {
		segments = append(segments, seg[start:])
	}
	return segments, operators
}

// detectMode 检测规则模式
// 根据规则内容特征自动推断解析模式
func (a *RuleAnalyzer) detectMode(rule string) (RuleMode, string) {
	trimmed := strings.TrimSpace(rule)

	// 模式前缀检测：$XPath:xxx, $Json:xxx, $CSS:xxx, $Regex:xxx, $JS:xxx
	if strings.HasPrefix(trimmed, "$") {
		if idx := strings.IndexByte(trimmed, ':'); idx > 0 && idx < 10 {
			prefix := strings.ToLower(trimmed[1:idx])
			switch prefix {
			case "xpath":
				return ModeXPath, trimmed[idx+1:]
			case "json":
				return ModeJSONPath, trimmed[idx+1:]
			case "css":
				return ModeCSS, trimmed[idx+1:]
			case "regex":
				return ModeRegex, trimmed[idx+1:]
			case "js":
				return ModeJS, trimmed[idx+1:]
			}
		}
	}

	// 模式前缀检测（@ 风格）：@XPath:xxx, @Json:xxx, @CSS:xxx
	if idx := strings.IndexByte(trimmed, ':'); idx > 0 {
		prefix := strings.ToLower(trimmed[:idx])
		switch prefix {
		case "@xpath":
			return ModeXPath, trimmed[idx+1:]
		case "@json":
			return ModeJSONPath, trimmed[idx+1:]
		case "@css":
			return ModeCSS, trimmed[idx+1:]
		}
	}

	// 正则模式：/pattern/ 分隔符
	if strings.HasPrefix(trimmed, "/") && strings.HasSuffix(trimmed, "/") && len(trimmed) > 2 {
		return ModeRegex, trimmed[1 : len(trimmed)-1]
	}

	// 正则模式：: 开头（AllInOne）
	if len(trimmed) >= 1 && trimmed[0] == ':' {
		return ModeRegex, trimmed[1:]
	}

	// CSS ID 选择器：#content
	if strings.HasPrefix(trimmed, "#") {
		return ModeXPath, trimmed
	}

	// JS 模式：包含 JS 关键字
	if strings.Contains(trimmed, "function") ||
		strings.Contains(trimmed, "=>") ||
		strings.Contains(trimmed, "return ") ||
		strings.Contains(trimmed, "var ") ||
		strings.Contains(trimmed, "let ") ||
		strings.Contains(trimmed, "const ") {
		return ModeJS, trimmed
	}

	// JSONPath 模式：包含 $ 开头的路径
	if strings.Contains(trimmed, "$.") {
		return ModeJSONPath, trimmed
	}

	// XPath/CSS 选择器模式：包含选择器特征
	if strings.Contains(trimmed, ".") ||
		strings.Contains(trimmed, "[") ||
		strings.Contains(trimmed, "//") ||
		strings.Contains(trimmed, "@") ||
		strings.Contains(trimmed, "::") {
		return ModeXPath, trimmed
	}

	// 默认模式
	return ModeDefault, trimmed
}

// Split 简化版：仅做切分，不解析模式和变量
// 适用于只需要获取原始段的场景
func (a *RuleAnalyzer) Split(rule string) ([]string, []string, error) {
	if rule == "" {
		return nil, nil, &ParseError{Msg: "empty rule"}
	}
	return a.splitBalanced(rule)
}

// GetOperatorCount 返回规则中的操作符数量
func (a *RuleAnalyzer) GetOperatorCount(rule string) int {
	_, operators, _ := a.splitBalanced(rule)
	return len(operators)
}

// HasOperator 规则是否包含操作符
func (a *RuleAnalyzer) HasOperator(rule string) bool {
	_, operators, _ := a.splitBalanced(rule)
	return len(operators) > 0
}

// AnalyzeMany 批量分析多个规则
func (a *RuleAnalyzer) AnalyzeMany(rules []string) []*ParseResult {
	results := make([]*ParseResult, len(rules))
	for i, rule := range rules {
		results[i] = a.Analyze(rule)
	}
	return results
}

// Join 将段和操作符重新组合为规则字符串
func Join(segments []string, operators []string) string {
	if len(segments) == 0 {
		return ""
	}
	if len(operators) == 0 {
		return segments[0]
	}

	var sb strings.Builder
	sb.WriteString(segments[0])
	for i, op := range operators {
		if i+1 < len(segments) {
			sb.WriteString(op)
			sb.WriteString(segments[i+1])
		}
	}
	return sb.String()
}
