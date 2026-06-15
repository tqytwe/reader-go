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
// 支持嵌套 {} 结构（{} 内的空格不作为分隔符）
// 支持 @Prefix: 段落（@XPath:/@Json:/@CSS: 后的内容直到下一个段落标记作为一个 token）
// 支持 @get:/@put: 作为段落边界
func (p *RuleParser) tokenize(rule string) []string {
	tokens := []string{}
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)
	braceDepth := 0

	for i := 0; i < len(rule); i++ {
		ch := rule[i]

		// 引号处理
		if inQuote {
			current.WriteByte(ch)
			if ch == quoteChar {
				inQuote = false
				quoteChar = 0
			}
			continue
		}

		// 顶层 @ 检测（braceDepth == 0）
		// 仅当 @ 后跟已知标记时才视为段落边界
		// 避免将 XPath 属性语法 [@attr] 误判为段落边界
		if ch == '@' && braceDepth == 0 {
			remaining := rule[i:]
			isSegmentMarker := false

			// 检查已知模式前缀 @XPath: / @Json: / @CSS:
			for prefix := range p.prefixes {
				if strings.HasPrefix(remaining, prefix) {
					isSegmentMarker = true
					break
				}
			}

			// 检查 @get: / @put:
			if !isSegmentMarker && (strings.HasPrefix(remaining, "@get:") || strings.HasPrefix(remaining, "@put:")) {
				isSegmentMarker = true
			}

			if isSegmentMarker {
				// 发射当前 token（如果有内容）
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
				// 消费整个段落直到下一个段落标记或字符串结尾
				end := findSegmentEnd(rule, i)
				segment := strings.TrimRightFunc(rule[i:end], func(r rune) bool {
					return r == ' ' || r == '\t' || r == '\n'
				})
				tokens = append(tokens, segment)
				i = end - 1
				continue
			}
		}

		switch {
		case ch == '"' || ch == '\'':
			inQuote = true
			quoteChar = ch
			current.WriteByte(ch)
		case ch == '{':
			braceDepth++
			current.WriteByte(ch)
		case ch == '}' && braceDepth > 0:
			braceDepth--
			current.WriteByte(ch)
		case braceDepth == 0 && (ch == ' ' || ch == '\t' || ch == '\n'):
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

	// 后处理：合并 CSS > 组合子相关的 tokens
	tokens = mergeCSSCombinators(tokens)

	// 后处理：合并 ## 替换模式到前一个 @Prefix: token
	tokens = mergeReplacePatterns(tokens)

	return tokens
}

// findSegmentEnd 从前缀起始位置开始，找到段落结束位置
// 段落边界：顶层（braceDepth=0）的 @get: / @put: / 下一个已知前缀
// 忽略 {} 内部的 @ 标记
func findSegmentEnd(rule string, start int) int {
	knownPrefixes := []string{"@XPath:", "@Json:", "@CSS:"}
	braceDepth := 0

	for i := start + 1; i < len(rule); i++ {
		switch rule[i] {
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '@':
			if braceDepth > 0 {
				continue // 忽略 {} 内部的 @
			}
			remaining := rule[i:]
			// 检查 @get: / @put:
			if strings.HasPrefix(remaining, "@get:") || strings.HasPrefix(remaining, "@put:") {
				return i
			}
			// 检查已知前缀
			for _, prefix := range knownPrefixes {
				if strings.HasPrefix(remaining, prefix) {
					return i
				}
			}
		}
	}
	return len(rule)
}

// mergeCSSCombinators 合并 CSS > 组合子相关的 tokens
// 当 tokens 中出现 ">" 时，将前一个 token、">"、后一个 token 合并为一个
// 并继续吸收后续的 CSS 选择器 token（直到遇到 @ 标记或 ## 模式）
func mergeCSSCombinators(tokens []string) []string {
	result := []string{}

	for i := 0; i < len(tokens); i++ {
		if tokens[i] == ">" && len(result) > 0 && i+1 < len(tokens) {
			// 合并：prev + " " + ">" + " " + next
			prev := result[len(result)-1]
			next := tokens[i+1]
			merged := prev + " > " + next
			result[len(result)-1] = merged
			i++ // 跳过 ">"

			// 继续吸收后续的 CSS 选择器 token
			for i+1 < len(tokens) {
				nextTok := tokens[i+1]
				if strings.HasPrefix(nextTok, "@") || strings.HasPrefix(nextTok, "##") {
					break
				}
				result[len(result)-1] += " " + nextTok
				i++
			}
		} else {
			result = append(result, tokens[i])
		}
	}

	return result
}

// mergeReplacePatterns 合并 ## 替换模式到前一个 @Prefix: token
// 当前一个 token 以 @Prefix: 开头时，将后续的 ## token 合并到该 token
func mergeReplacePatterns(tokens []string) []string {
	result := []string{}

	for _, tok := range tokens {
		if strings.HasPrefix(tok, "##") && len(result) > 0 && hasExplicitPrefix(result[len(result)-1]) {
			result[len(result)-1] += " " + tok
		} else {
			result = append(result, tok)
		}
	}

	return result
}

// hasExplicitPrefix 检查 token 是否以显式模式前缀开头
func hasExplicitPrefix(token string) bool {
	prefixes := []string{"@XPath:", "@Json:", "@CSS:"}
	for _, p := range prefixes {
		if strings.HasPrefix(token, p) {
			return true
		}
	}
	return false
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
		// $ 开头 → JSONPath（去掉 $ 前缀）
		selector := raw[1:]
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

	// 3. 解析 $1 正则分组引用（所有模式都支持）
	groupRefs := parseGroupRefs(raw)
	if len(groupRefs) > 0 {
		seg.GroupRefs = groupRefs
		seg.HasInlinedSyntax = true
	}

	// 4. 解析 ## 正则替换语法
	replacePatterns := parseReplacePatterns(raw)
	if len(replacePatterns) > 0 {
		seg.ReplacePatterns = replacePatterns
		seg.HasInlinedSyntax = true
	}

	return seg
}

// parseVariableBindings 解析 @put:key{rule} 和 @get:key 或 @get:{key}
// 支持嵌套的 {} 结构
func parseVariableBindings(input string) []VariableBinding {
	var bindings []VariableBinding
	i := 0

	for i < len(input) {
		// 查找 @put:
		if i+5 <= len(input) && input[i:i+5] == "@put:" {
			i += 5
			// 提取 key（到 { 为止）
			keyStart := i
			for i < len(input) && input[i] != '{' {
				i++
			}
			if i >= len(input) {
				continue
			}
			key := strings.TrimSpace(input[keyStart:i])

			// 匹配 {} 内容（支持嵌套）
			start := i
			depth := 0
			for i < len(input) {
				if input[i] == '{' {
					depth++
				} else if input[i] == '}' {
					depth--
					if depth == 0 {
						i++
						break
					}
				}
				i++
			}

			if depth == 0 {
				content := input[start+1 : i-1]
				bindings = append(bindings, VariableBinding{
					IsRef: false,
					Key:   key,
				})

				// 扫描内容中的 @get 引用
				scanForGetRefs(content, &bindings)
			}
			continue
		}

		// 查找 @get:
		if i+5 <= len(input) && input[i:i+5] == "@get:" {
			i += 5
			if i < len(input) && input[i] == '{' {
				// @get:{key} 格式
				i++
				keyStart := i
				for i < len(input) && input[i] != '}' {
					i++
				}
				if i < len(input) {
					key := strings.TrimSpace(input[keyStart:i])
					bindings = append(bindings, VariableBinding{
						IsRef: true,
						Key:   key,
					})
					i++ // 跳过 }
				}
			} else {
				// @get:key 格式（无 {}）
				keyStart := i
				for i < len(input) && input[i] != ' ' && input[i] != '\t' && input[i] != '\n' {
					i++
				}
				key := strings.TrimSpace(input[keyStart:i])
				if key != "" {
					bindings = append(bindings, VariableBinding{
						IsRef: true,
						Key:   key,
					})
				}
			}
			continue
		}

		i++
	}

	return bindings
}

// scanForGetRefs 扫描内容中的 @get 引用
func scanForGetRefs(content string, bindings *[]VariableBinding) {
	i := 0
	for i < len(content) {
		if i+5 <= len(content) && content[i:i+5] == "@get:" {
			i += 5
			if i < len(content) && content[i] == '{' {
				i++
				keyStart := i
				for i < len(content) && content[i] != '}' {
					i++
				}
				if i < len(content) {
					key := strings.TrimSpace(content[keyStart:i])
					*bindings = append(*bindings, VariableBinding{
						IsRef: true,
						Key:   key,
					})
					i++
				}
			} else {
				keyStart := i
				for i < len(content) && content[i] != ' ' && content[i] != '\t' && content[i] != '\n' {
					i++
				}
				key := strings.TrimSpace(content[keyStart:i])
				if key != "" {
					*bindings = append(*bindings, VariableBinding{
						IsRef: true,
						Key:   key,
					})
				}
			}
			continue
		}
		i++
	}
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
// 支持多个替换模式: ##find1##replace1 ##find2##replace2
func parseReplacePatterns(input string) []ReplacePattern {
	var patterns []ReplacePattern
	i := 0

	for i < len(input) {
		// 查找下一个 ##
		idx := strings.Index(input[i:], "##")
		if idx == -1 {
			break
		}
		i += idx + 2 // 跳过 ##

		// 查找结束的 ##
		endIdx := strings.Index(input[i:], "##")
		if endIdx == -1 {
			break // 没有结束的 ##，忽略
		}

		find := input[i : i+endIdx]
		i += endIdx + 2 // 跳过结束的 ##

		// 查找替换内容（到下一个 ## 或字符串结尾）
		replaceEnd := len(input)
		if nextIdx := strings.Index(input[i:], "##"); nextIdx != -1 {
			replaceEnd = i + nextIdx
		}

		replace := input[i:replaceEnd]
		// 如果后面还有 ## 模式，trim 尾部空白（空格是分隔符）
		if replaceEnd < len(input) {
			replace = strings.TrimRight(replace, " \t")
		} else if strings.TrimSpace(replace) != "" {
			// 最后一个模式：如果有非空白内容，trim 前导空白（空格是分隔符）
			replace = strings.TrimLeft(replace, " \t")
		}
		// 如果最后一个模式全是空白，保留原样（空白就是替换值）

		patterns = append(patterns, ReplacePattern{
			Find:    strings.TrimSpace(find),
			Replace: replace,
		})

		i = replaceEnd
	}

	return patterns
}

// ExtractSelector 从已解析的 segment 中提取纯 selector（去掉内嵌语法标记）
// 用于实际执行时的选择器
func ExtractSelector(seg *RuleSegment) string {
	result := seg.Selector

	// 移除 @put:key{...}（支持嵌套 {}）
	for {
		idx := strings.Index(result, "@put:")
		if idx == -1 {
			break
		}
		braceStart := strings.Index(result[idx:], "{")
		if braceStart == -1 {
			break
		}
		braceStart += idx

		depth := 0
		end := -1
		for i := braceStart; i < len(result); i++ {
			if result[i] == '{' {
				depth++
			} else if result[i] == '}' {
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
		}
		if end == -1 {
			break
		}
		result = result[:idx] + result[end+1:]
	}

	// 移除 @get:{...} 和 @get:key
	for {
		idx := strings.Index(result, "@get:")
		if idx == -1 {
			break
		}
		afterAtGet := idx + 5
		if afterAtGet < len(result) && result[afterAtGet] == '{' {
			// @get:{key} 格式
			braceEnd := strings.Index(result[afterAtGet:], "}")
			if braceEnd == -1 {
				break
			}
			braceEnd += afterAtGet
			result = result[:idx] + result[braceEnd+1:]
		} else {
			// @get:key 格式
			end := afterAtGet
			for end < len(result) && result[end] != ' ' && result[end] != '\t' && result[end] != '\n' {
				end++
			}
			result = result[:idx] + result[end:]
		}
	}

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
