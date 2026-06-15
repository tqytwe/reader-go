package rule

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"
)

// RegexParser 正则解析器
// 使用 github.com/dlclark/regexp2 作为底层库，支持 .NET 风格正则（包括 lookbehind）
//
// 支持特性:
//   - 链式正则: 多个正则依次执行，前一个的 group(0) 作为后一个的输入
//   - 分组引用: $1, $2, $3 等引用匹配分组
//   - 替换语法: ##find##replace 进行文本替换
//   - .NET 风格正则: 支持 (?<=...) lookbehind 等高级特性
type RegexParser struct {
	// flags 默认正则标志位
	flags regexp2.RegexOptions
}

// NewRegexParser 创建新的正则解析器
func NewRegexParser() *RegexParser {
	return &RegexParser{
		flags: regexp2.None,
	}
}

// WithIgnoreCase 设置忽略大小写
func (p *RegexParser) WithIgnoreCase() *RegexParser {
	p.flags |= regexp2.IgnoreCase
	return p
}

// WithMultiline 设置多行模式
func (p *RegexParser) WithMultiline() *RegexParser {
	p.flags |= regexp2.Multiline
	return p
}

// WithSingleline 设置单行模式（. 匹配换行）
func (p *RegexParser) WithSingleline() *RegexParser {
	p.flags |= regexp2.Singleline
	return p
}

// RegexChain 表示一条链式正则规则
// 包含多个正则步骤，依次执行
type RegexChain struct {
	// Steps 正则步骤序列
	Steps []RegexStep
}

// RegexStep 单个正则步骤
type RegexStep struct {
	// Pattern 正则表达式
	Pattern string
	// Flags 该步骤的独立标志位
	Flags regexp2.RegexOptions
	// ReplacePatterns ##find##replace 替换规则
	ReplacePatterns []ReplacePattern
	// GroupRefs $1, $2 等分组引用
	GroupRefs []int
}

// RegexMatch 单次正则匹配结果
type RegexMatch struct {
	// Input 输入文本
	Input string
	// Matches 所有匹配项（每个匹配项包含分组）
	Matches []MatchResult
}

// MatchResult 单次匹配结果（含分组）
type MatchResult struct {
	// Group0 整个匹配内容 (group 0)
	Group0 string
	// Groups 命名/索引分组 (1-based)
	Groups map[int]string
}

// ParseRegex 解析并执行正则规则
//
// 参数:
//   - rule: 正则规则字符串，支持:
//     * 单个正则: /pattern/ 或 pattern
//     * 链式正则: regex1 && regex2 && regex3
//     * 分组引用: $1 $2 $3
//     * 替换语法: ##find##replace
//   - input: 待匹配的输入文本
//
// 返回:
//   - 所有匹配结果（每个匹配为 group(0) 的字符串）
//   - 错误（如果正则编译失败或执行出错）
//
// 示例:
//   ParseRegex(`(\w+)\s+(\d+)`, "abc 123") → ["abc 123"]
//   ParseRegex(`(\w+) && (\d+)`, "abc 123") → ["123"] (先用第一个匹配，再用第二个)
//   ParseRegex(`(\w+)\s+(\d+) $1-$2`, "abc 123") → ["abc-123"]
//   ParseRegex(`##\s+## `, "a  b   c") → ["a b c"]
func ParseRegex(rule string, input string) ([]string, error) {
	parser := NewRegexParser()
	return parser.Parse(rule, input)
}

// Parse 执行正则解析（实例方法）
func (p *RegexParser) Parse(rule string, input string) ([]string, error) {
	if input == "" {
		return nil, fmt.Errorf("input is empty")
	}

	// Step 1: 使用 RuleAnalyzer 进行平衡组感知的切分
	analyzer := NewRuleAnalyzer()
	result := analyzer.Analyze(rule)
	if result.Error != nil {
		return nil, result.Error
	}

	// Step 2: 解析为 RegexChain
	chain, err := p.parseChain(result)
	if err != nil {
		return nil, err
	}

	// Step 3: 执行链式匹配
	return p.executeChain(chain, input)
}

// parseChain 将解析结果转换为 RegexChain
func (p *RegexParser) parseChain(result *ParseResult) (*RegexChain, error) {
	chain := &RegexChain{
		Steps: make([]RegexStep, 0, len(result.Segments)),
	}

	for _, seg := range result.Segments {
		step := RegexStep{
			Flags: p.flags,
		}

		// 解析模式：默认模式也作为正则处理
		switch seg.Mode {
		case ModeRegex, ModeDefault:
			step.Pattern = seg.Selector
		default:
			step.Pattern = seg.Selector
		}

		// 解析内嵌语法（从原始内容中提取）
		raw := seg.Raw
		if raw == "" {
			raw = seg.Selector
		}

		// 1. 分组引用 $1 $2 ...
		groupRefs := parseGroupRefs(raw)
		for _, ref := range groupRefs {
			step.GroupRefs = append(step.GroupRefs, ref.GroupIndex)
		}

		// 2. 替换语法 ##find##replace
		replacePatterns := parseReplacePatterns(raw)
		for _, rp := range replacePatterns {
			step.ReplacePatterns = append(step.ReplacePatterns, rp)
		}

		chain.Steps = append(chain.Steps, step)
	}

	return chain, nil
}

// executeChain 执行链式正则匹配
func (p *RegexParser) executeChain(chain *RegexChain, input string) ([]string, error) {
	if len(chain.Steps) == 0 {
		return nil, fmt.Errorf("no regex steps to execute")
	}

	currentInput := input
	var finalResults []string

	for i, step := range chain.Steps {
		isLast := i == len(chain.Steps)-1

		// 编译正则
		re, err := regexp2.Compile(step.Pattern, step.Flags)
		if err != nil {
			return nil, fmt.Errorf("step %d: compile regex %q: %w", i, step.Pattern, err)
		}

		// 执行匹配
		matches, err := p.execMatch(re, currentInput)
		if err != nil {
			return nil, fmt.Errorf("step %d: match %q: %w", i, step.Pattern, err)
		}

		if len(matches) == 0 {
			// 中间步骤无匹配，整个链失败
			if !isLast {
				return nil, fmt.Errorf("step %d: no match for %q", i, step.Pattern)
			}
			// 最后一步无匹配，返回空结果
			return []string{}, nil
		}

		// 处理分组引用和替换
		if len(step.GroupRefs) > 0 || len(step.ReplacePatterns) > 0 {
			// 应用分组引用和替换后作为最终结果
			finalResults = p.applyPostProcessing(matches, step)
		} else if isLast {
			// 最后一步，直接返回 group(0)
			finalResults = make([]string, len(matches))
			for j, m := range matches {
				finalResults[j] = m.Group0
			}
		}

		// 如果不是最后一步，用第一个匹配的 group(0) 作为下一步的输入
		if !isLast && len(matches) > 0 {
			currentInput = matches[0].Group0
		}
	}

	// 兜底：如果没有 finalResults，返回最后一步的所有 group(0)
	if len(finalResults) == 0 && len(chain.Steps) > 0 {
		lastStep := chain.Steps[len(chain.Steps)-1]
		lastRe, err := regexp2.Compile(lastStep.Pattern, lastStep.Flags)
		if err != nil {
			return nil, err
		}
		lastMatches, err := p.execMatch(lastRe, currentInput)
		if err != nil {
			return nil, err
		}
		finalResults = make([]string, len(lastMatches))
		for j, m := range lastMatches {
			finalResults[j] = m.Group0
		}
	}

	return finalResults, nil
}

// execMatch 执行单次正则匹配，返回所有匹配结果
func (p *RegexParser) execMatch(re *regexp2.Regexp, input string) ([]MatchResult, error) {
	var results []MatchResult

	m, err := re.FindStringMatch(input)
	if err != nil {
		return nil, err
	}

	for m != nil {
		result := MatchResult{
			Group0: m.String(),
			Groups: make(map[int]string),
		}

		groups := m.Groups()
		for i := 1; i < len(groups); i++ {
			g := groups[i]
			if len(g.Captures) > 0 {
				result.Groups[i] = g.Captures[0].String()
			}
		}

		results = append(results, result)

		m, err = re.FindNextMatch(m)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// applyPostProcessing 应用分组引用和替换语法
func (p *RegexParser) applyPostProcessing(matches []MatchResult, step RegexStep) []string {
	if len(matches) == 0 {
		return nil
	}

	base := matches[0]

	// 只有分组引用：按顺序拼接
	if len(step.GroupRefs) > 0 && len(step.ReplacePatterns) == 0 {
		var sb strings.Builder
		for _, refIdx := range step.GroupRefs {
			if val, ok := base.Groups[refIdx]; ok {
				sb.WriteString(val)
			} else if refIdx == 0 {
				sb.WriteString(base.Group0)
			}
		}
		return []string{sb.String()}
	}

	// 有替换语法：在 group(0) 上依次应用替换
	result := base.Group0
	for _, rp := range step.ReplacePatterns {
		re, err := regexp2.Compile(rp.Find, step.Flags)
		if err != nil {
			continue
		}
		result, _ = re.Replace(result, rp.Replace, 0, -1)
	}

	// 如果还有分组引用，在替换后的结果上再应用
	if len(step.GroupRefs) > 0 {
		// 重新对原始匹配做分组引用
		var sb strings.Builder
		for _, refIdx := range step.GroupRefs {
			if val, ok := base.Groups[refIdx]; ok {
				sb.WriteString(val)
			} else if refIdx == 0 {
				sb.WriteString(base.Group0)
			}
		}
		// 对分组引用结果再做替换
		final := sb.String()
		for _, rp := range step.ReplacePatterns {
			re, err := regexp2.Compile(rp.Find, step.Flags)
			if err != nil {
				continue
			}
			final, _ = re.Replace(final, rp.Replace, 0, -1)
		}
		return []string{final}
	}

	return []string{result}
}

// Match 便捷函数：单次正则匹配
// 返回第一个匹配的所有分组
func Match(pattern string, input string) (*MatchResult, error) {
	re, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return nil, err
	}

	m, err := re.FindStringMatch(input)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}

	result := &MatchResult{
		Group0: m.String(),
		Groups: make(map[int]string),
	}

	groups := m.Groups()
	for i := 1; i < len(groups); i++ {
		g := groups[i]
		if len(g.Captures) > 0 {
			result.Groups[i] = g.Captures[0].String()
		}
	}

	return result, nil
}

// Replace 便捷函数：正则替换
func Replace(input string, findPattern string, replaceText string) (string, error) {
	re, err := regexp2.Compile(findPattern, regexp2.None)
	if err != nil {
		return "", err
	}
	return re.Replace(input, replaceText, 0, -1)
}

// CompileRegex 编译正则（返回标准库 regexp.Regexp，用于兼容场景）
func CompileRegex(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile(pattern)
}

// IsMatch 便捷函数：判断是否匹配
func IsMatch(pattern string, input string) (bool, error) {
	re, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return false, err
	}
	m, err := re.FindStringMatch(input)
	if err != nil {
		return false, err
	}
	return m != nil, nil
}

// ExtractGroups 便捷函数：提取所有匹配的分組
func ExtractGroups(pattern string, input string) ([]map[int]string, error) {
	re, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return nil, err
	}

	var results []map[int]string
	m, err := re.FindStringMatch(input)
	if err != nil {
		return nil, err
	}

	for m != nil {
		groups := make(map[int]string)
		groups[0] = m.String()

		groupsData := m.Groups()
		for i := 1; i < len(groupsData); i++ {
			g := groupsData[i]
			if len(g.Captures) > 0 {
				groups[i] = g.Captures[0].String()
			}
		}
		results = append(results, groups)

		m, err = re.FindNextMatch(m)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// parseGroupRefs 解析 $1 $2 等分组引用（与 segment.go 共用）
// parseReplacePatterns 解析 ##find##replace 替换语法（与 segment.go 共用）
