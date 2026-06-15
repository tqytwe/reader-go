package rule

import (
	"reflect"
	"testing"
)

func TestRuleAnalyzer_SplitBalanced(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSegs []string
		wantOps  []string
		wantErr  bool
	}{
		// --- 基本切分 ---
		{
			name:     "single segment no operator",
			input:    ".title",
			wantSegs: []string{".title"},
			wantOps:  nil,
			wantErr:  false,
		},
		{
			name:     "two segments with &&",
			input:    ".title&&.author",
			wantSegs: []string{".title", ".author"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "three segments with && and ||",
			input:    ".title&&.author||.content",
			wantSegs: []string{".title", ".author", ".content"},
			wantOps:  []string{"&&", "||"},
			wantErr:  false,
		},
		{
			name:     "all three operators",
			input:    "a&&b||c%%d",
			wantSegs: []string{"a", "b", "c", "d"},
			wantOps:  []string{"&&", "||", "%%"},
			wantErr:  false,
		},

		// --- 方括号平衡组 ---
		{
			name:     "bracket with operator inside should not split",
			input:    "tag[0:5]&&.title",
			wantSegs: []string{"tag[0:5]", ".title"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "nested brackets",
			input:    "div[data-x][0]&&.title",
			wantSegs: []string{"div[data-x][0]", ".title"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "bracket with && inside should not split",
			input:    "tag[attr=='a&&b']&&.title",
			wantSegs: []string{"tag[attr=='a&&b']", ".title"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "negative index in bracket",
			input:    "li[-1]&&p",
			wantSegs: []string{"li[-1]", "p"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "complex bracket slice",
			input:    "table[0:10:2]&&.name",
			wantSegs: []string{"table[0:10:2]", ".name"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},

		// --- 圆括号平衡组 ---
		{
			name:     "paren with operator inside should not split",
			input:    "(title||subtitle)&&.author",
			wantSegs: []string{"(title||subtitle)", ".author"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "nested parentheses",
			input:    "((a||b))&&c",
			wantSegs: []string{"((a||b))", "c"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "paren with && inside",
			input:    "(a&&b)||c",
			wantSegs: []string{"(a&&b)", "c"},
			wantOps:  []string{"||"},
			wantErr:  false,
		},
		{
			name:     "multiple parens",
			input:    "(a&&b)||(c&&d)",
			wantSegs: []string{"(a&&b)", "(c&&d)"},
			wantOps:  []string{"||"},
			wantErr:  false,
		},

		// --- 引号处理 ---
		{
			name:     "string with operator inside should not split",
			input:    `"a&&b"||"c"`,
			wantSegs: []string{`"a&&b"`, `"c"`},
			wantOps:  []string{"||"},
			wantErr:  false,
		},
		{
			name:     "string with || inside",
			input:    `"x||y"&&"z"`,
			wantSegs: []string{`"x||y"`, `"z"`},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "string with %% inside",
			input:    `"a%%b"&&"c"`,
			wantSegs: []string{`"a%%b"`, `"c"`},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "escaped quote in string",
			input:    `"say \"hello&&world\""||next"`,
			wantSegs: []string{`"say \"hello&&world\""`, `next"`},
			wantOps:  []string{"||"},
			wantErr:  false,
		},

		// --- 混合场景 ---
		{
			name:     "bracket and paren mixed",
			input:    "div[class][0](a||b)&&.title",
			wantSegs: []string{"div[class][0](a||b)", ".title"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "bracket inside paren with operator",
			input:    "(tag[0:5]||tag[1:6])&&.name",
			wantSegs: []string{"(tag[0:5]||tag[1:6])", ".name"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "string inside bracket",
			input:    "tag[attr='a&&b']||.title",
			wantSegs: []string{"tag[attr='a&&b']", ".title"},
			wantOps:  []string{"||"},
			wantErr:  false,
		},
		{
			name:     "complex real-world rule",
			input:    "div.chapter[0]&&h2.title||h3.title&&p.content",
			wantSegs: []string{"div.chapter[0]", "h2.title", "h3.title", "p.content"},
			wantOps:  []string{"&&", "||", "&&"},
			wantErr:  false,
		},

		// --- 空格处理 ---
		{
			name:     "spaces around operators",
			input:    ".title && .author || .content",
			wantSegs: []string{".title", ".author", ".content"},
			wantOps:  []string{"&&", "||"},
			wantErr:  false,
		},
		{
			name:     "spaces in segments preserved in raw",
			input:    "  .title  &&  .author  ",
			wantSegs: []string{".title", ".author"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},

		// --- 边界情况 ---
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:     "only whitespace",
			input:    "   ",
			wantSegs: []string{},
			wantOps:  nil,
			wantErr:  false,
		},
		{
			name:     "unclosed bracket",
			input:    "tag[0&&.title",
			wantSegs: []string{"tag[0", ".title"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:     "unclosed paren",
			input:    "(a&&b||.title",
			wantSegs: []string{"(a", "b", ".title"},
			wantOps:  []string{"&&", "||"},
			wantErr:  false,
		},
		{
			name:     "unclosed string",
			input:    `"hello&&world`,
			wantSegs: []string{`"hello&&world`},
			wantOps:  nil,
			wantErr:  false,
		},

		// --- 真实书源规则 ---
		{
			name:     "book source rule: chapter list",
			input:    "div.chapter-item[0]&&h3.chapter-title||h4.chapter-title",
			wantSegs: []string{"div.chapter-item[0]", "h3.chapter-title", "h4.chapter-title"},
			wantOps:  []string{"&&", "||"},
			wantErr:  false,
		},
		{
			name:     "book source rule: content extraction",
			input:    "div.content&&p.text||div.article&&span.body",
			wantSegs: []string{"div.content", "p.text", "div.article", "span.body"},
			wantOps:  []string{"&&", "||", "&&"},
			wantErr:  false,
		},
		{
			name:     "book source rule: with attributes",
			input:    "a[href^='/chapter'][0]&&span.title||div.chapter-name",
			wantSegs: []string{"a[href^='/chapter'][0]", "span.title", "div.chapter-name"},
			wantOps:  []string{"&&", "||"},
			wantErr:  false,
		},
		{
			name:     "book source rule: nested selectors",
			input:    "div.book-info(div.title&&div.author)||div.summary",
			wantSegs: []string{"div.book-info(div.title&&div.author)", "div.summary"},
			wantOps:  []string{"||"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewRuleAnalyzer()
			result := analyzer.Analyze(tt.input)

			if tt.wantErr {
				if result.Error == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
				return
			}

			// Check segments
			if len(result.Segments) != len(tt.wantSegs) {
				t.Errorf("segments: want %v, got %v", tt.wantSegs, result.Segments)
				return
			}

			for i, wantSeg := range tt.wantSegs {
				if result.Segments[i].Selector != wantSeg {
					t.Errorf("segment[%d]: want %q, got %q", i, wantSeg, result.Segments[i].Selector)
				}
			}

			// Check operators
			if len(result.Operators) != len(tt.wantOps) {
				t.Errorf("operators: want %v, got %v", tt.wantOps, result.Operators)
				return
			}

			for i, wantOp := range tt.wantOps {
				if result.Operators[i] != wantOp {
					t.Errorf("operator[%d]: want %q, got %q", i, wantOp, result.Operators[i])
				}
			}
		})
	}
}

func TestRuleAnalyzer_DetectMode(t *testing.T) {
	analyzer := NewRuleAnalyzer()

	tests := []struct {
		name  string
		input string
		want  RuleMode
	}{
		{"default plain text", "hello world", ModeDefault},
		{"xpath css selector", ".title", ModeXPath},
		{"xpath with attribute", "div[class='main']", ModeXPath},
		{"xpath with id", "#content", ModeXPath},
		{"xpath with href", "a[href]", ModeXPath},
		{"xpath with pseudo", "a::before", ModeXPath},
		{"jsonpath", "$.book.title", ModeJSONPath},
		{"jsonpath root", "$.data.items", ModeJSONPath},
		{"regex pattern", "/\\d+/", ModeRegex},
		{"regex simple", "/hello/", ModeRegex},
		{"js arrow function", "x => x * 2", ModeJS},
		{"js function", "function() { return x; }", ModeJS},
		{"js var", "var x = 1", ModeJS},
		{"js const", "const y = 2", ModeJS},
		{"explicit xpath prefix", "$xpath:.title", ModeXPath},
		{"explicit json prefix", "$json:$.book", ModeJSONPath},
		{"explicit regex prefix", "$regex:/abc/", ModeRegex},
		{"explicit js prefix", "$js:x => x", ModeJS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, _ := analyzer.detectMode(tt.input)
			if mode != tt.want {
				t.Errorf("detectMode(%q) = %v, want %v", tt.input, mode, tt.want)
			}
		})
	}
}

func TestRuleAnalyzer_Split(t *testing.T) {
	analyzer := NewRuleAnalyzer()

	tests := []struct {
		name     string
		input    string
		wantSegs []string
		wantOps  []string
		wantErr  bool
	}{
		{
			name:     "simple split",
			input:    "a&&b||c",
			wantSegs: []string{"a", "b", "c"},
			wantOps:  []string{"&&", "||"},
			wantErr:  false,
		},
		{
			name:     "with brackets",
			input:    "tag[0:5]&&.title",
			wantSegs: []string{"tag[0:5]", ".title"},
			wantOps:  []string{"&&"},
			wantErr:  false,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segs, ops, err := analyzer.Split(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(segs, tt.wantSegs) {
				t.Errorf("segments: want %v, got %v", tt.wantSegs, segs)
			}
			if !reflect.DeepEqual(ops, tt.wantOps) {
				t.Errorf("operators: want %v, got %v", tt.wantOps, ops)
			}
		})
	}
}

func TestRuleAnalyzer_GetOperatorCount(t *testing.T) {
	analyzer := NewRuleAnalyzer()

	tests := []struct {
		input string
		want  int
	}{
		{".title", 0},
		{".title&&.author", 1},
		{".title&&.author||.content", 2},
		{"a&&b||c%%d", 3},
		{"tag[0:5]&&.title", 1},
		{"(a||b)&&c", 2},
		{``, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := analyzer.GetOperatorCount(tt.input)
			if got != tt.want {
				t.Errorf("GetOperatorCount(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestRuleAnalyzer_HasOperator(t *testing.T) {
	analyzer := NewRuleAnalyzer()

	tests := []struct {
		input string
		want  bool
	}{
		{".title", false},
		{".title&&.author", true},
		{".title || .author", true},
		{"a%%b", true},
		{"tag[0:5]", false},
		{"(a||b)", false}, // || inside parens
		{``, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := analyzer.HasOperator(tt.input)
			if got != tt.want {
				t.Errorf("HasOperator(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestRuleAnalyzer_AnalyzeMany(t *testing.T) {
	analyzer := NewRuleAnalyzer()

	rules := []string{
		".title&&.author",
		"div.content||p.text",
		"tag[0]%%tag[1]",
	}

	results := analyzer.AnalyzeMany(rules)

	if len(results) != len(rules) {
		t.Fatalf("expected %d results, got %d", len(rules), len(results))
	}

	// First rule: 2 segments, 1 operator
	if len(results[0].Segments) != 2 || len(results[0].Operators) != 1 {
		t.Errorf("rule 0: expected 2 segments, 1 operator")
	}

	// Second rule: 2 segments, 1 operator
	if len(results[1].Segments) != 2 || len(results[1].Operators) != 1 {
		t.Errorf("rule 1: expected 2 segments, 1 operator")
	}

	// Third rule: 2 segments, 1 operator
	if len(results[2].Segments) != 2 || len(results[2].Operators) != 1 {
		t.Errorf("rule 2: expected 2 segments, 1 operator")
	}
}

func TestJoin(t *testing.T) {
	tests := []struct {
		segments []string
		ops      []string
		want     string
	}{
		{
			segments: []string{".title", ".author"},
			ops:      []string{"&&"},
			want:     ".title&&.author",
		},
		{
			segments: []string{"a", "b", "c"},
			ops:      []string{"&&", "||"},
			want:     "a&&b||c",
		},
		{
			segments: []string{"single"},
			ops:      nil,
			want:     "single",
		},
		{
			segments: []string{},
			ops:      nil,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := Join(tt.segments, tt.ops)
			if got != tt.want {
				t.Errorf("Join(%v, %v) = %q, want %q", tt.segments, tt.ops, got, tt.want)
			}
		})
	}
}

func TestOperatorType(t *testing.T) {
	tests := []struct {
		op  string
		want OperatorType
	}{
		{"&&", OpAnd},
		{"||", OpOr},
		{"%%", OpCross},
		{"", OpNone},
		{"and", OpNone},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			got := ParseOperatorType(tt.op)
			if got != tt.want {
				t.Errorf("ParseOperatorType(%q) = %v, want %v", tt.op, got, tt.want)
			}
		})
	}
}

func TestParseResult_IsEmpty(t *testing.T) {
	t.Run("empty with error", func(t *testing.T) {
		result := &ParseResult{Error: &ParseError{Msg: "test error"}}
		if !result.IsEmpty() {
			t.Error("expected IsEmpty to be true")
		}
	})

	t.Run("empty no segments", func(t *testing.T) {
		result := &ParseResult{Segments: []RuleSegment{}}
		if !result.IsEmpty() {
			t.Error("expected IsEmpty to be true")
		}
	})

	t.Run("not empty", func(t *testing.T) {
		result := &ParseResult{Segments: []RuleSegment{{Selector: "test"}}}
		if result.IsEmpty() {
			t.Error("expected IsEmpty to be false")
		}
	})
}

func TestParseResult_HasOperator(t *testing.T) {
	t.Run("has operator", func(t *testing.T) {
		result := &ParseResult{Operators: []string{"&&"}}
		if !result.HasOperator() {
			t.Error("expected HasOperator to be true")
		}
	})

	t.Run("no operator", func(t *testing.T) {
		result := &ParseResult{Operators: []string{}}
		if result.HasOperator() {
			t.Error("expected HasOperator to be false")
		}
	})
}

// === 集成测试：真实书源规则 ===

func TestRuleAnalyzer_RealBookSourceRules(t *testing.T) {
	analyzer := NewRuleAnalyzer()

	// 模拟真实书源规则
	rules := []struct {
		name       string
		rule       string
		wantSegs   int
		wantOps    int
		wantModes  []RuleMode
	}{
		{
			name:      "chapter list rule",
			rule:      "div.chapter-list&&a.chapter-link||h3.chapter-title",
			wantSegs:  3,
			wantOps:   2,
			wantModes: []RuleMode{ModeXPath, ModeXPath, ModeXPath},
		},
		{
			name:      "book info rule",
			rule:      "div.book-info(div.title&&div.author||div.subtitle)",
			wantSegs:  2,
			wantOps:   1,
			wantModes: []RuleMode{ModeXPath, ModeXPath},
		},
		{
			name:      "content rule with attributes",
			rule:      "div.content[p]&&span.text||p.article",
			wantSegs:  3,
			wantOps:   2,
			wantModes: []RuleMode{ModeXPath, ModeXPath, ModeXPath},
		},
		{
			name:      "search result rule",
			rule:      "ul.search-results&&li.item[0:10]||div.result",
			wantSegs:  3,
			wantOps:   2,
			wantModes: []RuleMode{ModeXPath, ModeXPath, ModeXPath},
		},
		{
			name:      "complex nested rule",
			rule:      "div.main(div.sidebar&&ul.menu||div.content)&&footer",
			wantSegs:  4,
			wantOps:   3,
			wantModes: []RuleMode{ModeXPath, ModeXPath, ModeXPath, ModeXPath},
		},
		{
			name:      "string with operators",
			rule:      `"title||subtitle"&&.author`,
			wantSegs:  2,
			wantOps:   1,
			wantModes: []RuleMode{ModeDefault, ModeXPath},
		},
	}

	for _, tt := range rules {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.Analyze(tt.rule)

			if result.Error != nil {
				t.Fatalf("unexpected error: %v", result.Error)
			}

			if len(result.Segments) != tt.wantSegs {
				t.Errorf("expected %d segments, got %d: %v", tt.wantSegs, len(result.Segments), result.Segments)
			}

			if len(result.Operators) != tt.wantOps {
				t.Errorf("expected %d operators, got %d: %v", tt.wantOps, len(result.Operators), result.Operators)
			}

			if len(tt.wantModes) == len(result.Segments) {
				for i, wantMode := range tt.wantModes {
					if result.Segments[i].Mode != wantMode {
						t.Errorf("segment[%d] mode: want %v, got %v", i, wantMode, result.Segments[i].Mode)
					}
				}
			}
		})
	}
}

// === 性能测试 ===

func BenchmarkRuleAnalyzer_SplitBalanced(b *testing.B) {
	analyzer := NewRuleAnalyzer()
	rule := "div.chapter[0]&&h2.title||h3.title&&p.content[0:10:2]||footer"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.splitBalanced(rule)
	}
}

func BenchmarkRuleAnalyzer_Analyze(b *testing.B) {
	analyzer := NewRuleAnalyzer()
	rule := "div.chapter[0]&&h2.title||h3.title&&p.content[0:10:2]||footer"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(rule)
	}
}

func BenchmarkRuleAnalyzer_ComplexRule(b *testing.B) {
	analyzer := NewRuleAnalyzer()
	// 更复杂的真实规则
	rule := `div.book-list(div.book-item[0:20]&&div.title||h3.book-name)&&div.author||span.writer`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(rule)
	}
}
