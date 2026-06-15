// Package rule 规则解析引擎
// 核心功能：平衡组感知的规则切分器，支持 &&、||、%% 操作符
package rule

// =============================================================================
// 统一类型定义
// =============================================================================
// 说明：
// - RuleMode: 统一的解析模式枚举，整合了之前 types.RuleMode 和 segment.ParseMode
// - RuleSegment: 统一的规则片段结构，整合了之前 types.RuleSegment 和 segment.RuleSegment
// - RuleAnalyzer: 负责平衡组感知的规则切分（&& || %%）
// - RuleParser: 负责单个片段内的内嵌语法解析（@put/@get/{{}}/$1/##）
//
// 解析流程：
// 规则字符串 → RuleAnalyzer.Split() → []string → RuleParser.ParseSegment() → []RuleSegment
// =============================================================================

// RuleMode 统一的规则解析模式枚举
type RuleMode int

const (
	// ModeDefault 默认模式：JSoup CSS 选择器
	ModeDefault RuleMode = iota
	// ModeXPath XPath 模式
	ModeXPath
	// ModeJSONPath JSONPath 模式
	ModeJSONPath
	// ModeCSS CSS 选择器模式（与 Default 相同，显式标记）
	ModeCSS
	// ModeRegex 正则模式（AllInOne）
	ModeRegex
	// ModeJS JS 模式
	ModeJS
)

// String 返回 RuleMode 的字符串表示
func (m RuleMode) String() string {
	switch m {
	case ModeDefault:
		return "default"
	case ModeXPath:
		return "xpath"
	case ModeJSONPath:
		return "jsonpath"
	case ModeCSS:
		return "css"
	case ModeRegex:
		return "regex"
	case ModeJS:
		return "js"
	default:
		return "unknown"
	}
}

// =============================================================================
// RuleSegment - 统一的规则片段结构
// =============================================================================

// RuleSegment 表示解析后的单个规则片段
//
// 解析流程：
// 1. RuleAnalyzer 按 && || %% 切分规则字符串 → []string
// 2. RuleParser 对每个片段解析内嵌语法 → RuleSegment
//
// 示例: "@put:title{@XPath://h1} @get:title {{len .}} $1 ##旧##新"
// → 1 个 RuleSegment，包含多种内嵌语法
type RuleSegment struct {
	// Mode 解析模式
	Mode RuleMode

	// Raw 原始片段文本
	Raw string

	// Selector 选择器/规则主体（去掉模式前缀后的内容）
	Selector string

	// Bindings 变量绑定列表（@put 和 @get）
	Bindings []VariableBinding

	// EmbeddedJS 内嵌 JS 表达式（{{}}）
	EmbeddedJS []EmbeddedJS

	// GroupRefs 正则分组引用（$1, $2）
	GroupRefs []GroupRef

	// ReplacePatterns 正则替换模式（##find##replace）
	ReplacePatterns []ReplacePattern

	// HasInlinedSyntax 是否包含内嵌语法
	HasInlinedSyntax bool
}

// VariableBinding 表示一个变量绑定或引用
type VariableBinding struct {
	IsRef bool // true=引用(@get), false=绑定(@put)
	Key   string
}

// EmbeddedJS 表示内嵌的 JS 表达式
type EmbeddedJS struct {
	Expression string
}

// GroupRef 表示正则分组引用
type GroupRef struct {
	GroupIndex int
}

// ReplacePattern 表示正则替换模式
type ReplacePattern struct {
	Find    string
	Replace string
}

// =============================================================================
// ParseResult - 规则解析结果
// =============================================================================

// ParseResult 解析结果
type ParseResult struct {
	Original  string
	Segments  []RuleSegment
	Operators []string // 段之间的操作符，len = len(Segments)-1
	Error     error
}

// IsEmpty 结果是否为空
func (r *ParseResult) IsEmpty() bool {
	return r.Error != nil || len(r.Segments) == 0
}

// HasOperator 是否有操作符
func (r *ParseResult) HasOperator() bool {
	return len(r.Operators) > 0
}

// OperatorType 操作符类型
type OperatorType int

const (
	OpNone OperatorType = iota
	OpAnd    // && 交集
	OpOr     // || 并集
	OpCross  // %% 交叉合并
)

// ParseOperatorType 解析操作符类型
func ParseOperatorType(op string) OperatorType {
	switch op {
	case "&&":
		return OpAnd
	case "||":
		return OpOr
	case "%%":
		return OpCross
	default:
		return OpNone
	}
}

// String 返回操作符类型的字符串
func (t OperatorType) String() string {
	switch t {
	case OpAnd:
		return "&&"
	case OpOr:
		return "||"
	case OpCross:
		return "%%"
	default:
		return ""
	}
}

// ParseError 解析错误
type ParseError struct {
	Msg  string
	Pos  int // 错误位置
	Rule string
}

func (e *ParseError) Error() string {
	return e.Msg
}
