package txt

import (
	"regexp"
	"sort"
	"strings"
)

// =============================================================================
// TOC 规则定义
// =============================================================================

// TocRule 目录规则
type TocRule struct {
	// 正则表达式
	Regex *regexp.Regexp
	// 原始正则字符串
	Pattern string
	// 规则名称/描述
	Name string
	// 匹配组数量
	GroupCount int
	// 优先级（用于排序）
	Priority int
}

// TocRuleMatch 单次匹配结果
type TocRuleMatch struct {
	// 匹配的章节标题
	Title string
	// 章节起始位置（字节偏移）
	Position int64
	// 原始匹配文本
	RawText string
	// 匹配到的分组
	Groups []string
}

// =============================================================================
// 默认 TOC 规则集（18 条，参考 legado TextFile.kt）
// =============================================================================

// DefaultTocRules 默认目录规则列表
// 按优先级从高到低排序（更精确的规则在前）
var DefaultTocRules = []*TocRule{
	// 1. 标准章节格式：第 X 章 标题
	{
		Pattern:   `^(?:第\s*(\d+)\s*[章节卷篇部]\s*)(.+)$`,
		Name:      "chapter_standard",
		Priority:  100,
		GroupCount: 2,
	},
	// 2. 第 X 节格式
	{
		Pattern:   `^(?:第\s*(\d+)\s*[节篇])\s*(.+)$`,
		Name:      "section_standard",
		Priority:  95,
		GroupCount: 2,
	},
	// 3. 纯数字 + 标题：1. 标题
	{
		Pattern:   `^(\d+)\s*[.、．]\s*(.+)$`,
		Name:      "number_dot",
		Priority:  90,
		GroupCount: 2,
	},
	// 4. 纯数字 + 标题：1 标题（无标点）
	{
		Pattern:   `^(\d+)\s+(.+)$`,
		Name:      "number_space",
		Priority:  85,
		GroupCount: 2,
	},
	// 5. 罗马数字章节：I. 标题 / II. 标题
	{
		Pattern:   `^([IVX]+)\s*[.、．]\s*(.+)$`,
		Name:      "roman_numeral",
		Priority:  80,
		GroupCount: 2,
	},
	// 6. 中文数字章节：第一章 标题
	{
		Pattern:   `^(第[一二三四五六七八九十百千万]+[章节卷篇部])\s*(.+)$`,
		Name:      "chinese_number",
		Priority:  75,
		GroupCount: 2,
	},
	// 7. 带括号的数字：(1) 标题 / （1）标题
	{
		Pattern:   `^[\(（](\d+)[\)）]\s*(.+)$`,
		Name:      "bracket_number",
		Priority:  70,
		GroupCount: 2,
	},
	// 8. 方括号数字：[1] 标题
	{
		Pattern:   `^\[(\d+)\]\s*(.+)$`,
		Name:      "square_bracket",
		Priority:  65,
		GroupCount: 2,
	},
	// 9. 破折号章节：—— 第一章 标题
	{
		Pattern:   `^(?:—|-|·)\s*(?:第\s*\d+\s*[章节])?\s*(.+)$`,
		Name:      "dash_chapter",
		Priority:  60,
		GroupCount: 1,
	},
	// 10. 仅标题（无编号，但单独成行且长度合理）
	{
		Pattern:   `^([^0-9\n]{4,30})$`,
		Name:      "pure_title",
		Priority:  50,
		GroupCount: 1,
	},
	// 11. 卷/篇标记：第一卷 XXX
	{
		Pattern:   `^(?:第\s*([一二三四五六七八九十\d]+)\s*[卷篇])\s*(.+)$`,
		Name:      "volume",
		Priority:  98,
		GroupCount: 2,
	},
	// 12. 番外/外传标记
	{
		Pattern:   `^(?:番外|外传|特别篇|特别章|尾声|后记|序言|序章|引子)\s*[：:：]?\s*(.+)$`,
		Name:      "special_chapter",
		Priority:  88,
		GroupCount: 1,
	},
	// 13. 带序号的标题：第一章 XXX
	{
		Pattern:   `^(第[一二三四五六七八九十\d]+章)\s*(.+)$`,
		Name:      "chapter_cn_num",
		Priority:  92,
		GroupCount: 2,
	},
	// 14. 英文章节：Chapter 1 标题
	{
		Pattern:   `^(?:Chapter|Section|Part|Book)\s+(\d+)\s*(.+)$`,
		Name:      "english_chapter",
		Priority:  78,
		GroupCount: 2,
	},
	// 15. 带横线的标题：--- 标题 ---
	{
		Pattern:   `^(?:[-—]{3,}\s*)(.+?)(?:\s*[-—]{3,})$`,
		Name:      "decorated_title",
		Priority:  45,
		GroupCount: 1,
	},
	// 16. 短标题（2-20 字符，不含句号/感叹号/问号等标点）
	{
		Pattern:   `^([^0-9\n。，！？…]{2,20})$`,
		Name:      "short_title",
		Priority:  40,
		GroupCount: 1,
	},
	// 17. 带冒号的标题：第一章：标题
	{
		Pattern:   `^(第[一二三四五六七八九十\d]+[章节卷篇部])\s*[：:]\s*(.+)$`,
		Name:      "chapter_colon",
		Priority:  93,
		GroupCount: 2,
	},
	// 18. 纯标题（过滤常见非章节文本）
	{
		Pattern:   `^([^0-9\n。，！？…]{4,50})$`,
		Name:      "filtered_title",
		Priority:  30,
		GroupCount: 1,
	},
}

// 预编译所有正则
func init() {
	for _, rule := range DefaultTocRules {
		rule.Regex = regexp.MustCompile(rule.Pattern)
	}
}

// =============================================================================
// 规则分析器
// =============================================================================

// TocAnalyzer 目录规则分析器
// 负责检测哪种规则最适合当前文件
type TocAnalyzer struct {
	// 规则列表
	Rules []*TocRule
	// 规则命中统计
	ruleHits map[string]int
}

// NewTocAnalyzer 创建目录分析器
func NewTocAnalyzer() *TocAnalyzer {
	return &TocAnalyzer{
		Rules:      DefaultTocRules,
		ruleHits:   make(map[string]int),
	}
}

// NewTocAnalyzerWithRules 使用自定义规则创建分析器
func NewTocAnalyzerWithRules(rules []*TocRule) *TocAnalyzer {
	// 编译正则
	for _, rule := range rules {
		if rule.Regex == nil {
			rule.Regex = regexp.MustCompile(rule.Pattern)
		}
	}
	return &TocAnalyzer{
		Rules:    rules,
		ruleHits: make(map[string]int),
	}
}

// Analyze 分析文本，找出最佳 TOC 规则
// 返回最佳规则和所有匹配结果
func (a *TocAnalyzer) Analyze(text string) (*TocRule, []*TocRuleMatch) {
	// 重置统计
	a.ruleHits = make(map[string]int)

	// 按行分析
	lines := strings.Split(text, "\n")
	allMatches := make([]*TocRuleMatch, 0)

	for lineIdx, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 对每条规则尝试匹配
		for _, rule := range a.Rules {
			matches := rule.Regex.FindStringSubmatch(line)
			if matches != nil {
				a.ruleHits[rule.Name]++

				// 提取章节标题
				title := a.extractTitle(rule, matches)
				if title != "" {
					allMatches = append(allMatches, &TocRuleMatch{
						Title:    title,
						Position: int64(lineIdx), // 简化：用行号代替字节偏移
						RawText:  line,
						Groups:   matches,
					})
				}
			}
		}
	}

	// 找出命中次数最多的规则
	bestRule := a.findBestRule()

	return bestRule, allMatches
}

// extractTitle 从匹配结果中提取章节标题
func (a *TocAnalyzer) extractTitle(rule *TocRule, matches []string) string {
	if len(matches) < 2 {
		return ""
	}

	// 优先使用第二个分组（标题部分）
	if len(matches) > 2 && matches[2] != "" {
		return strings.TrimSpace(matches[2])
	}

	// 否则使用第一个分组
	return strings.TrimSpace(matches[1])
}

// findBestRule 找出命中次数最多的规则
func (a *TocAnalyzer) findBestRule() *TocRule {
	var bestRule *TocRule
	maxHits := 0

	for _, rule := range a.Rules {
		hits := a.ruleHits[rule.Name]
		if hits > maxHits {
			maxHits = hits
			bestRule = rule
		} else if hits == maxHits && bestRule != nil && rule.Priority > bestRule.Priority {
			bestRule = rule
		} else if hits == maxHits && bestRule == nil {
			bestRule = rule
		}
	}

	return bestRule
}

// GetRuleHits 获取所有规则的命中统计
func (a *TocAnalyzer) GetRuleHits() map[string]int {
	return a.ruleHits
}

// FilterMatches 过滤匹配结果，移除可疑的章节标题
func FilterMatches(matches []*TocRuleMatch) []*TocRuleMatch {
	if len(matches) == 0 {
		return matches
	}

	// 过滤规则
	filtered := make([]*TocRuleMatch, 0, len(matches))

	for _, m := range matches {
		if isSuspiciousTitle(m.Title) {
			continue
		}
		filtered = append(filtered, m)
	}

	return filtered
}

// 可疑标题关键词（通常不是真正的章节标题）
var suspiciousKeywords = []string{
	"目录", "contents", "content", "index", "前言", "序言", "序", "引言",
	"后记", "尾声", "附录", "appendix", "参考文献", "参考书目", " bibliography",
	"版权声明", "版权", "copyright", "出版说明", "作者简介", "author",
	"编辑推荐", "推荐语", "媒体评价", "媒体", "书评", "简介", "介绍",
	"内容介绍", "内容简介", "故事梗概", "plot", "summary",
	// 注意：移除了 "第一"/"第二" 等关键词，因为正常章节标题如 "第一章 开篇" 会包含它们
}

// isSuspiciousTitle 判断标题是否可疑（可能不是真正的章节标题）
func isSuspiciousTitle(title string) bool {
	titleLower := strings.ToLower(title)

	for _, kw := range suspiciousKeywords {
		if strings.Contains(titleLower, kw) {
			return true
		}
	}

	// 标题太短（少于 2 个字符）或太长（超过 50 个字符）
	if len(title) < 2 || len(title) > 50 {
		return true
	}

	// 纯数字
	if _, err := ParseInt(title); err == nil {
		return true
	}

	return false
}

// ParseInt 尝试解析整数（简化实现）
func ParseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, &strconvError{"empty"}
	}

	negative := false
	if s[0] == '-' {
		negative = true
		s = s[1:]
	}

	if s == "" {
		return 0, &strconvError{"no digits"}
	}

	result := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &strconvError{"invalid digit"}
		}
		result = result*10 + int(c-'0')
	}

	if negative {
		result = -result
	}

	return result, nil
}

type strconvError struct {
	msg string
}

func (e *strconvError) Error() string {
	return "parsing integer: " + e.msg
}

// =============================================================================
// 规则管理
// =============================================================================

// GetDefaultRules 获取默认规则列表（返回副本）
func GetDefaultRules() []*TocRule {
	rules := make([]*TocRule, len(DefaultTocRules))
	for i, r := range DefaultTocRules {
		rules[i] = &TocRule{
			Pattern:    r.Pattern,
			Name:       r.Name,
			Priority:   r.Priority,
			GroupCount: r.GroupCount,
		}
	}
	return rules
}

// AddCustomRule 添加自定义规则
func AddCustomRule(rules []*TocRule, pattern, name string, priority int) []*TocRule {
	rule := &TocRule{
		Pattern:    pattern,
		Name:       name,
		Priority:   priority,
		GroupCount: 2, // 默认 2 个分组
	}
	rule.Regex = regexp.MustCompile(pattern)

	rules = append(rules, rule)

	// 按优先级排序（高优先级在前）
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	return rules
}

// RemoveRule 移除指定名称的规则
func RemoveRule(rules []*TocRule, name string) []*TocRule {
	result := make([]*TocRule, 0, len(rules))
	for _, r := range rules {
		if r.Name != name {
			result = append(result, r)
		}
	}
	return result
}
