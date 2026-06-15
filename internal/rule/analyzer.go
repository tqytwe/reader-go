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

	// 平衡组感知的切分（已处理 CSS 选择器括号内的 && 拆分）
	segments, operators, err := a.splitBalanced(rule)
	if err != nil {
		result.Error = err
		return result
	}

	// 解析每个段
	for _, seg := range segments {
		raw := seg
		seg = strings.TrimSpace(seg)

		if seg == "" {
			continue
		}

		mode, content := a.detectMode(seg)

		result.Segments = append(result.Segments, RuleSegment{
			Mode:     mode,
			Selector: content,
			Raw:      raw,
		})
	}

	result.Operators = operators

	return result
}

// splitBalanced 平衡组感知的规则切分
// 核心算法：迭代扫描，跟踪平衡组状态
//
// 拆分规则：
//   - 顶层（所有括号闭合）：&&、||、%% 均拆分
//   - CSS 选择器括号内（有前缀的括号）：仅当无顶层 ||/%% 时，&& 也拆分
//   - 纯分组括号内（无前缀的括号）：不拆分（由 countOperatorsWithFlatten 递归处理）
//
// CSS 选择器括号：前面紧接标识符字符的 (，如 div.book-info(...)
// 纯分组括号：没有前缀的 (，如 (a||b)
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
				// 在 operators 中插入子操作符
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

	// 后处理：在 CSS 选择器括号内拆分操作符
	// - || 在 CSS 选择器括号内始终拆分
	// - && 在 CSS 选择器括号内仅当有顶层 && 时拆分
	hasTopLevelAnd := false
	for _, op := range operators {
		if op == "&&" {
			hasTopLevelAnd = true
			break
		}
	}
	segments, operators = splitCSSSelectorParens(segments, operators, hasTopLevelAnd)

	// 验证：操作符数量应该等于段数量-1
	if len(operators) > 0 && len(segments) != len(operators)+1 {
		return nil, nil, &ParseError{Msg: "mismatched operators and segments"}
	}

	return segments, operators, nil
}

// splitCSSSelectorParens 对段中的 CSS 选择器括号内的操作符进行拆分
// hasTopLevelAnd 为 true 时：拆分 CSS 选择器括号内的 && 和 ||
// hasTopLevelAnd 为 false 时：仅拆分 CSS 选择器括号内的 ||
func splitCSSSelectorParens(segments []string, operators []string, hasTopLevelAnd bool) ([]string, []string) {
	var newSegments []string
	var newOperators []string

	for i, seg := range segments {
		splitPoints := findOpsInCSSSelectorParens(seg, hasTopLevelAnd)
		if len(splitPoints) > 0 {
			// 按位置排序（已经是有序的，因为扫描从左到右）
			subSegs := make([]string, 0, len(splitPoints)+1)
			subOps := make([]string, 0, len(splitPoints))
			prev := 0
			for _, sp := range splitPoints {
				subSegs = append(subSegs, seg[prev:sp.pos])
				subOps = append(subOps, sp.op)
				prev = sp.pos + 2 // skip the 2-char operator
			}
			if prev < len(seg) {
				subSegs = append(subSegs, seg[prev:])
			}

			newSegments = append(newSegments, subSegs...)
			newOperators = append(newOperators, subOps...)
		} else {
			newSegments = append(newSegments, seg)
		}

		// 添加该段之后的顶层操作符
		if i < len(operators) {
			newOperators = append(newOperators, operators[i])
		}
	}

	return newSegments, newOperators
}

type splitPoint struct {
	pos int
	op  string
}

// findOpsInCSSSelectorParens 找出字符串中 CSS 选择器括号（有前缀的括号）内的操作符位置
// hasTopLevelAnd 为 true 时找 && 和 ||，为 false 时只找 ||
func findOpsInCSSSelectorParens(s string, hasTopLevelAnd bool) []splitPoint {
	var points []splitPoint
	n := len(s)

	// 找出所有有前缀的 ( 位置
	prefixedOpenParens := make(map[int]bool)
	inString := false
	escapeNext := false
	for j := 0; j < n; j++ {
		ch := s[j]
		if escapeNext {
			escapeNext = false
			continue
		}
		if inString && ch == '\\' {
			escapeNext = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '(' && j > 0 && isIdentChar(s[j-1]) {
			prefixedOpenParens[j] = true
		}
	}

	if len(prefixedOpenParens) == 0 {
		return nil
	}

	// 扫描，跟踪在有前缀括号内的深度，找出操作符位置
	prefixedParenDepth := 0
	bracketDepth := 0
	inString = false
	escapeNext = false
	for j := 0; j < n; j++ {
		ch := s[j]
		if escapeNext {
			escapeNext = false
			continue
		}
		if inString && ch == '\\' {
			escapeNext = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '[' {
			bracketDepth++
			continue
		}
		if ch == ']' {
			if bracketDepth > 0 {
				bracketDepth--
			}
			continue
		}
		if ch == '(' {
			if prefixedOpenParens[j] {
				prefixedParenDepth++
			}
			continue
		}
		if ch == ')' {
			if prefixedParenDepth > 0 {
				prefixedParenDepth--
			}
			continue
		}
		if prefixedParenDepth > 0 && bracketDepth == 0 && j+1 < n {
			// || 在 CSS 选择器括号内始终拆分
			if ch == '|' && s[j+1] == '|' {
				points = append(points, splitPoint{j, "||"})
			} else if hasTopLevelAnd && ch == '&' && s[j+1] == '&' {
				// && 在 CSS 选择器括号内仅当有顶层 && 时拆分
				points = append(points, splitPoint{j, "&&"})
			}
		}
	}

	return points
}

// isInsidePrefixedParenAt 判断位置 pos 是否在有前缀的括号内
func isInsidePrefixedParenAt(rule string, pos int) bool {
	n := len(rule)

	// 找出所有有前缀的 ( 位置
	prefixedOpenParens := make(map[int]bool)
	inString := false
	escapeNext := false
	for i := 0; i < n && i <= pos; i++ {
		ch := rule[i]
		if escapeNext {
			escapeNext = false
			continue
		}
		if inString && ch == '\\' {
			escapeNext = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '(' && i > 0 && isIdentChar(rule[i-1]) {
			prefixedOpenParens[i] = true
		}
	}

	// 扫描到 pos，跟踪在有前缀括号内的深度
	prefixedParenDepth := 0
	inString = false
	escapeNext = false
	for i := 0; i < pos && i < n; i++ {
		ch := rule[i]
		if escapeNext {
			escapeNext = false
			continue
		}
		if inString && ch == '\\' {
			escapeNext = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '(' {
			if prefixedOpenParens[i] {
				prefixedParenDepth++
			}
		} else if ch == ')' {
			if prefixedParenDepth > 0 {
				prefixedParenDepth--
			}
		}
	}

	return prefixedParenDepth > 0
}

// isIdentChar 判断字符是否为标识符字符（CSS 选择器前缀的组成部分）
// 不包含 ] 和 )，因为 ]( 和 )( 表示分组括号而非 CSS 选择器括号
func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_' || ch == '-' || ch == '.' ||
		ch == '#' || ch == ':'
}

// countOperatorsWithFlatten 递归计算操作符数量，展开纯分组括号
// 纯分组括号：以 ( 开头、以 ) 结尾、括号全程平衡的段
// 对于纯分组括号，去掉外层括号后递归计算内部操作符
func countOperatorsWithFlatten(analyzer *RuleAnalyzer, segments []string, operators []string) int {
	count := len(operators)
	for _, seg := range segments {
		trimmed := strings.TrimSpace(seg)
		if isPureParenGroup(trimmed) {
			inner := trimmed[1 : len(trimmed)-1]
			if inner != "" {
				innerSegs, innerOps, err := analyzer.splitBalanced(inner)
				if err == nil {
					count += countOperatorsWithFlatten(analyzer, innerSegs, innerOps)
				}
			}
		}
	}
	return count
}

// isPureParenGroup 判断字符串是否为纯圆括号分组
// 条件：以 ( 开头，以 ) 结尾，且括号在整个字符串中始终平衡
// 例如: "(a||b)" → true, "((a||b))" → true, "div(a||b)" → false
func isPureParenGroup(s string) bool {
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return false
	}
	depth := 0
	inString := false
	escapeNext := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escapeNext {
			escapeNext = false
			continue
		}
		if inString && ch == '\\' {
			escapeNext = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth < 0 {
				return false
			}
			// 括号在最后一个字符之前归零，说明不是纯分组
			if depth == 0 && i < len(s)-1 {
				return false
			}
		}
	}
	return depth == 0
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

	// CSS 元素选择器：单个单词（无空格、非引号开头）视为 CSS 元素名
	// 例如 footer、div、span 等
	if !strings.Contains(trimmed, " ") && trimmed != "" && trimmed[0] != '"' {
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

// GetOperatorCount 返回规则中的操作符数量（包含纯分组括号内的操作符）
func (a *RuleAnalyzer) GetOperatorCount(rule string) int {
	if rule == "" {
		return 0
	}
	segments, operators, err := a.splitBalanced(rule)
	if err != nil {
		return 0
	}
	return countOperatorsWithFlatten(a, segments, operators)
}

// HasOperator 规则是否包含顶层操作符
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
