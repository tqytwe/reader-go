package rule

import (
	"fmt"
	"regexp"
	"strings"
)

// RuleParser 规则解析器
// 将规则字符串解析为 RuleSegment 数组
//
// 与 RuleAnalyzer 的关系：
// - RuleAnalyzer: 负责平衡组感知的规则切分（&& || %% 操作符）
// - RuleParser: 负责单个片段内的内嵌语法解析（@put/@get/{{}}/$1/##）
//
// 完整解析流程：
// 规则字符串 → RuleAnalyzer.Split() → []string → RuleParser.ParseSegment() → []RuleSegment
type RuleParser struct {
	// 模式前缀映射
	prefixes map[string]RuleMode
}

// NewRuleParser 创建新的规则解析器
func NewRuleParser() *RuleParser {
	return &RuleParser{
		prefixes: map[string]RuleMode{
			"@XPath:": ModeXPath,
			"@Json:":  ModeJSONPath,
			"@CSS:":   ModeCSS,
		},
	}
}

// Parse 解析规则字符串为 RuleSegment 数组
//
// 解析流程:
// 1. 按空格分割规则字符串为多个片段
// 2. 对每个片段识别解析模式前缀
// 3. 解析片段中的内嵌语法 (@put/@get/{{}}/$\d/##)
// 4. 返回 RuleSegment 数组
//
// 示例:
//   "@XPath://h1 @put:title{@get:title} {{len .}} $1"
//   → [
//       {Mode:XPath, Selector:"//h1"},
//       {Mode:Default, Selector:"@put:title{@get:title}", Bindings:[...]},
//       {Mode:Default, Selector:"{{len .}}", EmbeddedJS:[...]},
//       {Mode:Default, Selector:"$1", GroupRefs:[{GroupIndex:1}]}
//     ]
func (p *RuleParser) Parse(rule string) ([]RuleSegment, error) {
	if strings.TrimSpace(rule) == "" {
		return nil, fmt.Errorf("rule string is empty")
	}

	// 按空格分割（保留连续空格作为分隔）
	rawSegments := p.tokenize(rule)

	var segments []RuleSegment
	for _, raw := range rawSegments {
		seg, err := p.parseSegment(raw)
		if err != nil {
			return nil, fmt.Errorf("parse segment %q: %w", raw, err)
		}
		segments = append(segments, *seg)
	}

	return segments, nil
}

// tokenize 将规则字符串按空格分词
// 支持引号包裹的片段（引号内的空格不作为分隔符）
func (p *RuleParser) tokenize(rule string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(rule); i++ {
		ch := rule[i]

		switch {
		case !inQuote && (ch == '"' || ch == '\''):
			inQuote = true
			quoteChar = ch
		case inQuote && ch == quoteChar:
			inQuote = false
			quoteChar = 0
		case !inQuote && (ch == ' ' || ch == '\t' || ch == '\n'):
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// parseSegment 解析单个规则片段
func (p *RuleParser) parseSegment(raw string) (*RuleSegment, error) {
	// 1. 识别模式前缀
	mode, selector := p.detectMode(raw)

	// 2. 解析内嵌语法
	seg := p.parseInlineSyntax(raw, selector, mode)

	return seg, nil
}

// detectMode 识别解析模式前缀，返回 (模式, 去掉前缀后的选择器)
//
// 模式识别优先级（从高到低）:
//   1. @XPath: → XPath 模式
//   2. @Json:  → JSONPath 模式
//   3. @CSS:   → CSS 模式
//   4. $ 开头  → JSONPath 模式（自动识别）
//   5. / 开头  → XPath 模式（自动识别）
//   6. : 开头  → AllInOne 正则模式
//   7. 无前缀  → Default (JSoup) 模式
func (p *RuleParser) detectMode(raw string) (RuleMode, string) {
	// 检查显式前缀
	for prefix, mode := range p.prefixes {
		if strings.HasPrefix(raw, prefix) {
			selector := strings.TrimPrefix(raw, prefix)
			return mode, selector
		}
	}

	// 自动识别前缀
	if strings.HasPrefix(raw, "$") {
		// $ 开头 → JSONPath
		selector := raw
		return ModeJSONPath, selector
	}

	if strings.HasPrefix(raw, "/") {
		// / 开头 → XPath
		return ModeXPath, raw
	}

	if strings.HasPrefix(raw, ":") {
		// : 开头 → AllInOne 正则
		selector := raw[1:]
		if selector == "" {
			selector = ":"
		}
		return ModeRegex, selector
	}

	// 无前缀 → Default (JSoup)
	return ModeDefault, raw
}

// parseInlineSyntax 解析片段中的内嵌语法
func (p *RuleParser) parseInlineSyntax(raw, selector string, mode RuleMode) *RuleSegment {
	seg := &RuleSegment{
		Mode:             mode,
		Raw:              raw,
		Selector:         selector,
		HasInlinedSyntax: false,
	}

	// 1. 解析 @put:{key:rule} 和 @get:{key}
	bindings := parseVariableBindings(raw)
	if len(bindings) > 0 {
		seg.Bindings = bindings
		seg.HasInlinedSyntax = true
	}

	// 2. 解析 {{js}} 内嵌 JS 表达式
	jsExprs := parseEmbeddedJS(raw)
	if len(jsExprs) > 0 {
		seg.EmbeddedJS = jsExprs
		seg.HasInlinedSyntax = true
	}

	// 3. 解析 $1 正则分组引用
	if mode == ModeRegex || mode == ModeDefault {
		groupRefs := parseGroupRefs(raw)
		if len(groupRefs) > 0 {
			seg.GroupRefs = groupRefs
			seg.HasInlinedSyntax = true
		}
	}

	// 4. 解析 ## 正则替换语法
	replacePatterns := parseReplacePatterns(raw)
	if len(replacePatterns) > 0 {
		seg.ReplacePatterns = replacePatterns
		seg.HasInlinedSyntax = true
	}

	return seg
}

// parseVariableBindings 解析 @put:{key:rule} 和 @get:{key}
// 支持嵌套的 {} 结构
func parseVariableBindings(input string) []VariableBinding {
	var bindings []VariableBinding

	// @put:{key:rule}
	putRe := regexp.MustCompile(`@put:\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`)
	// @get:{key}
	getRe := regexp.MustCompile(`@get:\{([^}]+)\}`)

	// 先找 @put
	putMatches := putRe.FindAllStringSubmatch(input, -1)
	for _, m := range putMatches {
		content := m[1]
		// 分离 key 和 rule（第一个 : 作为分隔符）
		parts := splitByFirstColon(content)
		if len(parts) == 2 {
			bindings = append(bindings, VariableBinding{
				IsRef: false,
				Key:   strings.TrimSpace(parts[0]),
			})
		}
	}

	// 再找 @get
	getMatches := getRe.FindAllStringSubmatch(input, -1)
	for _, m := range getMatches {
		bindings = append(bindings, VariableBinding{
			IsRef: true,
			Key:   strings.TrimSpace(m[1]),
		})
	}

	return bindings
}

// splitByFirstColon 按第一个冒号分割字符串
func splitByFirstColon(s string) []string {
	idx := strings.Index(s, ":")
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}

// parseEmbeddedJS 解析 {{js}} 内嵌表达式
func parseEmbeddedJS(input string) []EmbeddedJS {
	re := regexp.MustCompile(`\{\{([^{}]+)\}\}`)
	matches := re.FindAllStringSubmatch(input, -1)

	var exprs []EmbeddedJS
	for _, m := range matches {
		exprs = append(exprs, EmbeddedJS{
			Expression: m[1],
		})
	}
	return exprs
}

// parseGroupRefs 解析 $1 正则分组引用
func parseGroupRefs(input string) []GroupRef {
	re := regexp.MustCompile(`\$(\d+)`)
	matches := re.FindAllStringSubmatch(input, -1)

	var refs []GroupRef
	for _, m := range matches {
		idx := 0
		fmt.Sscanf(m[1], "%d", &idx)
		refs = append(refs, GroupRef{GroupIndex: idx})
	}
	return refs
}

// parseReplacePatterns 解析 ## 正则替换语法
// 格式: ##find##replace 或 ##find##
func parseReplacePatterns(input string) []ReplacePattern {
	// 匹配 ##...##... 模式
	re := regexp.MustCompile(`##([^#]+)##([^#]*)`)
	matches := re.FindAllStringSubmatch(input, -1)

	var patterns []ReplacePattern
	for _, m := range matches {
		patterns = append(patterns, ReplacePattern{
			Find:    m[1],
			Replace: m[2],
		})
	}
	return patterns
}

// ExtractSelector 从已解析的 segment 中提取纯 selector（去掉内嵌语法标记）
// 用于实际执行时的选择器
func ExtractSelector(seg *RuleSegment) string {
	result := seg.Selector

	// 移除 @put:{...} 和 @get:{...}
	putRe := regexp.MustCompile(`@put:\{[^}]*\}`)
	getRe := regexp.MustCompile(`@get:\{[^}]*\}`)
	result = putRe.ReplaceAllString(result, "")
	result = getRe.ReplaceAllString(result, "")

	// 移除 {{...}}
	jsRe := regexp.MustCompile(`\{\{[^}]+\}\}`)
	result = jsRe.ReplaceAllString(result, "")

	// 移除 $1
	groupRe := regexp.MustCompile(`\$\d+`)
	result = groupRe.ReplaceAllString(result, "")

	// 移除 ##...##...
	replaceRe := regexp.MustCompile(`##[^#]+##[^#]*`)
	result = replaceRe.ReplaceAllString(result, "")

	return strings.TrimSpace(result)
}

// ParseRule 便捷函数：使用默认解析器解析规则字符串
func ParseRule(rule string) ([]RuleSegment, error) {
	parser := NewRuleParser()
	return parser.Parse(rule)
}

// MustParse 便捷函数：解析规则，失败则 panic
func MustParse(rule string) []RuleSegment {
	segments, err := ParseRule(rule)
	if err != nil {
		panic(fmt.Sprintf("ParseRule(%q): %v", rule, err))
	}
	return segments
}

// HasVariableBindings 检查规则是否包含变量绑定
func HasVariableBindings(segments []RuleSegment) bool {
	for _, seg := range segments {
		if len(seg.Bindings) > 0 {
			return true
		}
	}
	return false
}

// GetPutBindings 获取所有 @put 绑定
func GetPutBindings(segments []RuleSegment) []VariableBinding {
	var result []VariableBinding
	for _, seg := range segments {
		for _, b := range seg.Bindings {
			if !b.IsRef {
				result = append(result, b)
			}
		}
	}
	return result
}

// GetGetBindings 获取所有 @get 引用
func GetGetBindings(segments []RuleSegment) []VariableBinding {
	var result []VariableBinding
	for _, seg := range segments {
		for _, b := range seg.Bindings {
			if b.IsRef {
				result = append(result, b)
			}
		}
	}
	return result
}

// HasEmbeddedJS 检查规则是否包含内嵌 JS
func HasEmbeddedJS(segments []RuleSegment) bool {
	for _, seg := range segments {
		if len(seg.EmbeddedJS) > 0 {
			return true
		}
	}
	return false
}

// HasGroupRefs 检查规则是否包含正则分组引用
func HasGroupRefs(segments []RuleSegment) bool {
	for _, seg := range segments {
		if len(seg.GroupRefs) > 0 {
			return true
		}
	}
	return false
}

// HasReplacePatterns 检查规则是否包含正则替换
func HasReplacePatterns(segments []RuleSegment) bool {
	for _, seg := range segments {
		if len(seg.ReplacePatterns) > 0 {
			return true
		}
	}
	return false
}

// ModeSummary 返回所有片段使用的模式摘要
func ModeSummary(segments []RuleSegment) []RuleMode {
	seen := make(map[RuleMode]bool)
	var result []RuleMode
	for _, seg := range segments {
		if !seen[seg.Mode] {
			seen[seg.Mode] = true
			result = append(result, seg.Mode)
		}
	}
	return result
}

// IsSingleSegment 判断是否为单片段规则（无空格分隔）
func IsSingleSegment(rule string) bool {
	return !strings.Contains(rule, " ") && !strings.Contains(rule, "\t") && !strings.Contains(rule, "\n")
}
