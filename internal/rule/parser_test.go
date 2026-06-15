package rule

import (
	"reflect"
	"testing"
)

// ==================== 模式前缀识别测试 ====================

func TestDetectMode_ExplicitPrefixes(t *testing.T) {
	parser := NewRuleParser()

	tests := []struct {
		input    string
		wantMode RuleMode
		wantSel  string
	}{
		{"@XPath://h1/title", ModeXPath, "//h1/title"},
		{"@XPath://div[@class='content']", ModeXPath, "//div[@class='content']"},
		{"@Json:$.book.title", ModeJSONPath, "$.book.title"},
		{"@Json:$.data.items[*].name", ModeJSONPath, "$.data.items[*].name"},
		{"@CSS:.book-title", ModeCSS, ".book-title"},
		{"@CSS:#main-content p", ModeCSS, "#main-content p"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, sel := parser.detectMode(tt.input)
			if mode != tt.wantMode {
				t.Errorf("detectMode() mode = %v, want %v", mode, tt.wantMode)
			}
			if sel != tt.wantSel {
				t.Errorf("detectMode() selector = %q, want %q", sel, tt.wantSel)
			}
		})
	}
}

func TestDetectMode_AutoPrefixes(t *testing.T) {
	parser := NewRuleParser()

	tests := []struct {
		input    string
		wantMode RuleMode
		wantSel  string
	}{
		{"$.store.book[0].title", ModeJSONPath, ".store.book[0].title"},
		{"$..author", ModeJSONPath, "..author"},
		{"/html/head/title", ModeXPath, "/html/head/title"},
		{"//div[@id='content']", ModeXPath, "//div[@id='content']"},
		{":^(.*?)(?:\\s+|$)", ModeRegex, "^(.*?)(?:\\s+|$)"},
		{":\\d{4}-\\d{2}-\\d{2}", ModeRegex, "\\d{4}-\\d{2}-\\d{2}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, sel := parser.detectMode(tt.input)
			if mode != tt.wantMode {
				t.Errorf("detectMode() mode = %v, want %v", mode, tt.wantMode)
			}
			if sel != tt.wantSel {
				t.Errorf("detectMode() selector = %q, want %q", sel, tt.wantSel)
			}
		})
	}
}

func TestDetectMode_Default(t *testing.T) {
	parser := NewRuleParser()

	tests := []struct {
		input    string
		wantMode RuleMode
		wantSel  string
	}{
		{"div.book > h2", ModeDefault, "div.book > h2"},
		{".chapter-title", ModeDefault, ".chapter-title"},
		{"#content p", ModeDefault, "#content p"},
		{"table tr td", ModeDefault, "table tr td"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, sel := parser.detectMode(tt.input)
			if mode != tt.wantMode {
				t.Errorf("detectMode() mode = %v, want %v", mode, tt.wantMode)
			}
			if sel != tt.wantSel {
				t.Errorf("detectMode() selector = %q, want %q", sel, tt.wantSel)
			}
		})
	}
}

// ==================== 分词测试 ====================

func TestTokenize(t *testing.T) {
	parser := NewRuleParser()

	tests := []struct {
		input string
		want  []string
	}{
		{"@XPath://h1 @get:title", []string{"@XPath://h1", "@get:title"}},
		{"$.title $.author", []string{"$.title", "$.author"}},
		{"div.content p.text", []string{"div.content", "p.text"}},
		{"@put:name{@XPath://h1} @get:name", []string{"@put:name{@XPath://h1}", "@get:name"}},
		{"  @XPath://title   @get:title  ", []string{"@XPath://title", "@get:title"}},
		{"single", []string{"single"}},
		{"", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parser.tokenize(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tokenize() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ==================== Parse 完整解析测试 ====================

func TestParse_SingleSegment(t *testing.T) {
	parser := NewRuleParser()

	tests := []struct {
		name      string
		rule      string
		wantLen   int
		wantMode  RuleMode
		wantSel   string
		wantInline bool
	}{
		{
			name:     "XPath 单片段",
			rule:     "@XPath://h1/title",
			wantLen:  1,
			wantMode: ModeXPath,
			wantSel:  "//h1/title",
		},
		{
			name:     "JSONPath 单片段",
			rule:     "@Json:$.book.title",
			wantLen:  1,
			wantMode: ModeJSONPath,
			wantSel:  "$.book.title",
		},
		{
			name:     "CSS 单片段",
			rule:     "@CSS:.book-name",
			wantLen:  1,
			wantMode: ModeCSS,
			wantSel:  ".book-name",
		},
		{
			name:     "$ 自动识别 JSONPath",
			rule:     "$.store.book[*].title",
			wantLen:  1,
			wantMode: ModeJSONPath,
			wantSel:  ".store.book[*].title",
		},
		{
			name:     "/ 自动识别 XPath",
			rule:     "//div[@class='chapter']/h2",
			wantLen:  1,
			wantMode: ModeXPath,
			wantSel:  "//div[@class='chapter']/h2",
		},
		{
			name:     ": 自动识别正则",
			rule:     ":^(\\w+)\\s+(\\d+)",
			wantLen:  1,
			wantMode: ModeRegex,
			wantSel:  "^(\\w+)\\s+(\\d+)",
		},
		{
			name:      "默认模式",
			rule:      "div.content > p.intro",
			wantLen:   1,
			wantMode:  ModeDefault,
			wantSel:   "div.content > p.intro",
			wantInline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := parser.Parse(tt.rule)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(segments) != tt.wantLen {
				t.Fatalf("Parse() len = %d, want %d", len(segments), tt.wantLen)
			}
			if segments[0].Mode != tt.wantMode {
				t.Errorf("segments[0].Mode = %v, want %v", segments[0].Mode, tt.wantMode)
			}
			if segments[0].Selector != tt.wantSel {
				t.Errorf("segments[0].Selector = %q, want %q", segments[0].Selector, tt.wantSel)
			}
		})
	}
}

func TestParse_MultiSegment(t *testing.T) {
	parser := NewRuleParser()

	rule := "@XPath://h1 @put:title{@get:title} @get:title"
	segments, err := parser.Parse(rule)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(segments))
	}

	// Segment 1: @XPath://h1
	if segments[0].Mode != ModeXPath || segments[0].Selector != "//h1" {
		t.Errorf("seg[0]: mode=%v sel=%q", segments[0].Mode, segments[0].Selector)
	}

	// Segment 2: @put:title{@get:title}
	if segments[1].Mode != ModeDefault {
		t.Errorf("seg[1] mode = %v, want %v", segments[1].Mode, ModeDefault)
	}
	if len(segments[1].Bindings) != 2 {
		t.Fatalf("seg[1] bindings = %d, want 2", len(segments[1].Bindings))
	}
	// @put:title
	if !segments[1].Bindings[0].IsRef && segments[1].Bindings[0].Key != "title" {
		t.Errorf("seg[1] binding[0]: IsRef=%v Key=%q", segments[1].Bindings[0].IsRef, segments[1].Bindings[0].Key)
	}
	// @get:title
	if segments[1].Bindings[1].IsRef && segments[1].Bindings[1].Key != "title" {
		t.Errorf("seg[1] binding[1]: IsRef=%v Key=%q", segments[1].Bindings[1].IsRef, segments[1].Bindings[1].Key)
	}

	// Segment 3: @get:title
	if !segments[2].Bindings[0].IsRef {
		t.Error("seg[2] should be a @get reference")
	}
}

// ==================== 内嵌语法测试 ====================

func TestParse_VariableBindings(t *testing.T) {
	parser := NewRuleParser()

	tests := []struct {
		name    string
		rule    string
		wantPut []string
		wantGet []string
	}{
		{
			name:    "@put 绑定",
			rule:    "@put:title{@XPath://h1}",
			wantPut: []string{"title"},
			wantGet: nil,
		},
		{
			name:    "@get 引用",
			rule:    "@get:title",
			wantPut: nil,
			wantGet: []string{"title"},
		},
		{
			name:    "@put + @get 组合",
			rule:    "@put:name{@XPath://h1} @get:name",
			wantPut: []string{"name"},
			wantGet: []string{"name"},
		},
		{
			name:    "多个变量",
			rule:    "@put:title{@XPath://h1} @put:author{@XPath://meta[@name='author']} @get:title @get:author",
			wantPut: []string{"title", "author"},
			wantGet: []string{"title", "author"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := parser.Parse(tt.rule)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			var puts, gets []string
			for _, seg := range segments {
				for _, b := range seg.Bindings {
					if b.IsRef {
						gets = append(gets, b.Key)
					} else {
						puts = append(puts, b.Key)
					}
				}
			}

			if !reflect.DeepEqual(puts, tt.wantPut) {
				t.Errorf("puts = %v, want %v", puts, tt.wantPut)
			}
			if !reflect.DeepEqual(gets, tt.wantGet) {
				t.Errorf("gets = %v, want %v", gets, tt.wantGet)
			}
		})
	}
}

func TestParse_EmbeddedJS(t *testing.T) {
	parser := NewRuleParser()

	tests := []struct {
		name     string
		rule     string
		wantExpr []string
	}{
		{
			name:     "len 函数",
			rule:     "{{len .}}",
			wantExpr: []string{"len ."},
		},
		{
			name:     "字符串操作",
			rule:     "{{trim .}}",
			wantExpr: []string{"trim ."},
		},
		{
			name:     "多个 JS 表达式",
			rule:     "{{len .}} {{trim .}}",
			wantExpr: []string{"len .", "trim ."},
		},
		{
			name:     "JS 与其他语法混合",
			rule:     "@XPath://h1 {{len .}}",
			wantExpr: []string{"len ."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := parser.Parse(tt.rule)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			var exprs []string
			for _, seg := range segments {
				for _, e := range seg.EmbeddedJS {
					exprs = append(exprs, e.Expression)
				}
			}

			if !reflect.DeepEqual(exprs, tt.wantExpr) {
				t.Errorf("EmbeddedJS = %v, want %v", exprs, tt.wantExpr)
			}
		})
	}
}

func TestParse_GroupRefs(t *testing.T) {
	parser := NewRuleParser()

	tests := []struct {
		name    string
		rule    string
		wantIdx []int
	}{
		{
			name:    "单个分组引用",
			rule:    "$1",
			wantIdx: []int{1},
		},
		{
			name:    "多个分组引用",
			rule:    "$1 $2 $3",
			wantIdx: []int{1, 2, 3},
		},
		{
			name:    "分组引用在正则中",
			rule:    ":^(\\w+)\\s+(\\d+)$ $1 - $2",
			wantIdx: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := parser.Parse(tt.rule)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			var idxs []int
			for _, seg := range segments {
				for _, ref := range seg.GroupRefs {
					idxs = append(idxs, ref.GroupIndex)
				}
			}

			if !reflect.DeepEqual(idxs, tt.wantIdx) {
				t.Errorf("GroupRefs = %v, want %v", idxs, tt.wantIdx)
			}
		})
	}
}

func TestParse_ReplacePatterns(t *testing.T) {
	parser := NewRuleParser()

	tests := []struct {
		name      string
		rule      string
		wantFind  []string
		wantRep   []string
	}{
		{
			name:     "简单替换",
			rule:     "##旧##新",
			wantFind: []string{"旧"},
			wantRep:  []string{"新"},
		},
		{
			name:     "多个替换",
			rule:     "##A##B ##C##D",
			wantFind: []string{"A", "C"},
			wantRep:  []string{"B", "D"},
		},
		{
			name:     "替换为空",
			rule:     "##多余##",
			wantFind: []string{"多余"},
			wantRep:  []string{""},
		},
		{
			name:     "与选择器混合",
			rule:     "@XPath://title ##HTML## ",
			wantFind: []string{"HTML"},
			wantRep:  []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := parser.Parse(tt.rule)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			var finds, reps []string
			for _, seg := range segments {
				for _, p := range seg.ReplacePatterns {
					finds = append(finds, p.Find)
					reps = append(reps, p.Replace)
				}
			}

			if !reflect.DeepEqual(finds, tt.wantFind) {
				t.Errorf("Replace find = %v, want %v", finds, tt.wantFind)
			}
			if !reflect.DeepEqual(reps, tt.wantRep) {
				t.Errorf("Replace rep = %v, want %v", reps, tt.wantRep)
			}
		})
	}
}

// ==================== 真实书源规则测试 ====================

func TestParse_RealBookSourceRules(t *testing.T) {
	parser := NewRuleParser()

	// 测试用例：真实书源中常见的规则模式
	tests := []struct {
		name        string
		rule        string
		wantSegCount int
		checkFunc   func(t *testing.T, segments []*RuleSegment)
	}{
		{
			name:        "笔趣阁 - 书名",
			rule:        "@XPath://h1",
			wantSegCount: 1,
			checkFunc: func(t *testing.T, segs []*RuleSegment) {
				if segs[0].Mode != ModeXPath {
					t.Errorf("expected XPath mode, got %v", segs[0].Mode)
				}
				if segs[0].Selector != "//h1" {
					t.Errorf("expected //h1, got %q", segs[0].Selector)
				}
			},
		},
		{
			name:        "笔趣阁 - 作者",
			rule:        "@XPath://div[@class='bookinfo']/p[2] text()",
			wantSegCount: 1,
			checkFunc: func(t *testing.T, segs []*RuleSegment) {
				if segs[0].Mode != ModeXPath {
					t.Errorf("expected XPath mode, got %v", segs[0].Mode)
				}
			},
		},
		{
			name:        "变量绑定 - 书名+作者",
			rule:        "@put:title{@XPath://h1} @put:author{@XPath://meta[@name='author']}/@content} @get:title @get:author",
			wantSegCount: 4,
			checkFunc: func(t *testing.T, segs []*RuleSegment) {
				segsCopy := make([]RuleSegment, len(segs))
				for i, s := range segs {
					segsCopy[i] = *s
				}
				puts := GetPutBindings(segsCopy)
				gets := GetGetBindings(segsCopy)
				if len(puts) != 2 {
					t.Errorf("expected 2 @put bindings, got %d", len(puts))
				}
				if len(gets) != 2 {
					t.Errorf("expected 2 @get bindings, got %d", len(gets))
				}
			},
		},
		{
			name:        "自动识别 - JSONPath 书源",
			rule:        "$.data.title $.data.author $.data.intro",
			wantSegCount: 3,
			checkFunc: func(t *testing.T, segs []*RuleSegment) {
				for _, seg := range segs {
					if seg.Mode != ModeJSONPath {
						t.Errorf("expected JSONPath mode, got %v", seg.Mode)
					}
				}
			},
		},
		{
			name:        "自动识别 - XPath 书源",
			rule:        "//div[@class='booklist']/div //div[@class='chapterlist']/a",
			wantSegCount: 2,
			checkFunc: func(t *testing.T, segs []*RuleSegment) {
				for _, seg := range segs {
					if seg.Mode != ModeXPath {
						t.Errorf("expected XPath mode, got %v", seg.Mode)
					}
				}
			},
		},
		{
			name:        "正则替换 - 清理文本",
			rule:        "@XPath://div[@class='content'] ##\\s+## ",
			wantSegCount: 1,
			checkFunc: func(t *testing.T, segs []*RuleSegment) {
				if len(segs[0].ReplacePatterns) != 1 {
					t.Errorf("expected 1 replace pattern, got %d", len(segs[0].ReplacePatterns))
				}
				if segs[0].ReplacePatterns[0].Find != "\\s+" {
					t.Errorf("expected find='\\s+', got %q", segs[0].ReplacePatterns[0].Find)
				}
			},
		},
		{
			name:        "复杂组合 - 变量+正则+JS",
			rule:        "@put:raw{@XPath://div[@class='content']} {{trim .}} ##\\s+##  $1",
			wantSegCount: 1,
			checkFunc: func(t *testing.T, segs []*RuleSegment) {
				hasPut := false
				hasJS := false
				hasReplace := false
				hasGroupRef := false

				for _, seg := range segs {
					if len(seg.Bindings) > 0 && !seg.Bindings[0].IsRef {
						hasPut = true
					}
					if len(seg.EmbeddedJS) > 0 {
						hasJS = true
					}
					if len(seg.ReplacePatterns) > 0 {
						hasReplace = true
					}
					if len(seg.GroupRefs) > 0 {
						hasGroupRef = true
					}
				}

				if !hasPut {
					t.Error("expected @put binding")
				}
				if !hasJS {
					t.Error("expected embedded JS")
				}
				if !hasReplace {
					t.Error("expected replace pattern")
				}
				if !hasGroupRef {
					t.Error("expected group ref")
				}
			},
		},
		{
			name:        "CSS 选择器模式",
			rule:        "@CSS:.book-title @CSS:.book-author",
			wantSegCount: 2,
			checkFunc: func(t *testing.T, segs []*RuleSegment) {
				for _, seg := range segs {
					if seg.Mode != ModeCSS {
						t.Errorf("expected CSS mode, got %v", seg.Mode)
					}
				}
			},
		},
		{
			name:        "默认模式 - JSoup",
			rule:        "div.book-item > h3 a",
			wantSegCount: 1,
			checkFunc: func(t *testing.T, segs []*RuleSegment) {
				if segs[0].Mode != ModeDefault {
					t.Errorf("expected Default mode, got %v", segs[0].Mode)
				}
			},
		},
		{
			name:        "正则模式 - 日期解析",
			rule:        ":\\d{4}年\\d{1,2}月\\d{1,2}日",
			wantSegCount: 1,
			checkFunc: func(t *testing.T, segs []*RuleSegment) {
				if segs[0].Mode != ModeRegex {
					t.Errorf("expected Regex mode, got %v", segs[0].Mode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := parser.Parse(tt.rule)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(segments) != tt.wantSegCount {
				t.Fatalf("expected %d segments, got %d", tt.wantSegCount, len(segments))
			}

			segsPtr := make([]*RuleSegment, len(segments))
			for i := range segments {
				segsPtr[i] = &segments[i]
			}
			tt.checkFunc(t, segsPtr)
		})
	}
}

// ==================== 工具函数测试 ====================

func TestExtractSelector(t *testing.T) {
	seg := &RuleSegment{
		Mode:  ModeDefault,
		Raw:   "@put:title{@XPath://h1} @get:title {{len .}} $1 ##旧##新",
		Selector: "@put:title{@XPath://h1} @get:title {{len .}} $1 ##旧##新",
		Bindings: []VariableBinding{
			{IsRef: false, Key: "title"},
			{IsRef: true, Key: "title"},
		},
		EmbeddedJS: []EmbeddedJS{{Expression: "len ."}},
		GroupRefs:  []GroupRef{{GroupIndex: 1}},
		ReplacePatterns: []ReplacePattern{{Find: "旧", Replace: "新"}},
	}

	result := ExtractSelector(seg)
	if result != "" {
		t.Errorf("ExtractSelector() = %q, want empty string (all inline syntax removed)", result)
	}

	// 测试部分内嵌语法
	seg2 := &RuleSegment{
		Mode:  ModeDefault,
		Raw:   "div.content $1",
		Selector: "div.content $1",
		GroupRefs: []GroupRef{{GroupIndex: 1}},
	}
	result2 := ExtractSelector(seg2)
	if result2 != "div.content" {
		t.Errorf("ExtractSelector() = %q, want 'div.content'", result2)
	}
}

func TestHasVariableBindings(t *testing.T) {
	segs1 := []RuleSegment{
		{Bindings: []VariableBinding{{IsRef: false, Key: "title"}}},
	}
	segs2 := []RuleSegment{
		{Mode: ModeXPath, Selector: "//h1"},
	}

	if !HasVariableBindings(segs1) {
		t.Error("expected true for segments with bindings")
	}
	if HasVariableBindings(segs2) {
		t.Error("expected false for segments without bindings")
	}
}

func TestModeSummary(t *testing.T) {
	segs := []RuleSegment{
		{Mode: ModeXPath},
		{Mode: ModeJSONPath},
		{Mode: ModeXPath},
		{Mode: ModeDefault},
	}
	summary := ModeSummary(segs)
	expectedModes := []RuleMode{ModeXPath, ModeJSONPath, ModeDefault}
	if !reflect.DeepEqual(summary, expectedModes) {
		t.Errorf("ModeSummary() = %v, want %v", summary, expectedModes)
	}
}

func TestIsSingleSegment(t *testing.T) {
	tests := []struct {
		rule string
		want bool
	}{
		{"@XPath://h1", true},
		{"$.title", true},
		{"div.content", true},
		{"@XPath://h1 @get:title", false},
		{"$.title $.author", false},
		{"", true},
	}

	for _, tt := range tests {
		if got := IsSingleSegment(tt.rule); got != tt.want {
			t.Errorf("IsSingleSegment(%q) = %v, want %v", tt.rule, got, tt.want)
		}
	}
}

// ==================== 便捷函数测试 ====================

func TestParseRule(t *testing.T) {
	segments, err := ParseRule("@XPath://h1")
	if err != nil {
		t.Fatalf("ParseRule() error = %v", err)
	}
	if len(segments) != 1 || segments[0].Mode != ModeXPath {
		t.Errorf("ParseRule() = %v", segments)
	}
}

func TestMustParse(t *testing.T) {
	segments := MustParse("@Json:$.book.title")
	if len(segments) != 1 || segments[0].Mode != ModeJSONPath {
		t.Errorf("MustParse() = %v", segments)
	}

	// Test panic on invalid rule
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse() with empty rule should panic")
		}
	}()
	MustParse("")
}

// ==================== RuleMode String 测试 ====================

func TestRuleMode_String(t *testing.T) {
	tests := []struct {
		mode RuleMode
		want string
	}{
		{ModeDefault, "default"},
		{ModeXPath, "xpath"},
		{ModeJSONPath, "jsonpath"},
		{ModeCSS, "css"},
		{ModeRegex, "regex"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("RuleMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

// ==================== 边界和错误测试 ====================

func TestParse_EmptyRule(t *testing.T) {
	parser := NewRuleParser()
	segments, err := parser.Parse("")
	if err == nil {
		t.Fatal("Parse(\"\") should return error")
	}
	if segments != nil {
		t.Errorf("Parse(\"\") returned segments = %v, want nil", segments)
	}
}

func TestParse_WhitespaceOnly(t *testing.T) {
	parser := NewRuleParser()
	segments, err := parser.Parse("   \t\n  ")
	if err == nil {
		t.Fatal("Parse(whitespace only) should return error")
	}
	if segments != nil {
		t.Errorf("Parse(whitespace only) returned segments = %v, want nil", segments)
	}
}

func TestParse_NestedBraces(t *testing.T) {
	parser := NewRuleParser()
	// 测试嵌套的 {} 结构
	rule := "@put:data{@XPath://div[@class='info']}"
	segments, err := parser.Parse(rule)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segments))
	}
	if len(segments[0].Bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(segments[0].Bindings))
	}
}

func TestParse_MixedInlineSyntax(t *testing.T) {
	parser := NewRuleParser()
	// 同一片段中包含多种内嵌语法
	rule := "@put:content{@XPath://div} {{len .}} $1 ##\\s+## "
	segments, err := parser.Parse(rule)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segments))
	}
	seg := segments[0]
	if len(seg.Bindings) == 0 {
		t.Error("expected @put binding")
	}
	if len(seg.EmbeddedJS) == 0 {
		t.Error("expected embedded JS")
	}
	if len(seg.GroupRefs) == 0 {
		t.Error("expected group ref")
	}
	if len(seg.ReplacePatterns) == 0 {
		t.Error("expected replace pattern")
	}
}
