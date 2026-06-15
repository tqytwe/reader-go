package rule

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// ==================== 测试 HTML 样本 ====================

const testHTML = `
<!DOCTYPE html>
<html>
<head>
	<title>测试页面</title>
</head>
<body>
	<div id="main" class="content">
		<h1 class="title">主标题</h1>
		<p class="intro">欢迎使用 <strong>阅读器</strong>！</p>
		<ul class="book-list">
			<li class="book" data-id="1"><a href="/book/1">第一本书</a></li>
			<li class="book" data-id="2"><a href="/book/2">第二本书</a></li>
			<li class="book" data-id="3"><a href="/book/3">第三本书</a></li>
			<li class="book" data-id="4"><a href="/book/4">第四本书</a></li>
			<li class="book" data-id="5"><a href="/book/5">第五本书</a></li>
		</ul>
		<div class="sidebar">
			<a href="/link/1">链接一</a>
			<a href="/link/2">链接二</a>
		</div>
		<p>段落1</p>
		<p>段落2</p>
		<p>段落3</p>
	</div>
</body>
</html>
`

func newTestDoc(t *testing.T) *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(testHTML))
	if err != nil {
		t.Fatalf("failed to create test document: %v", err)
	}
	return doc
}

// ==================== 基本 CSS 选择器测试 ====================

func TestParseCSS_BasicSelector(t *testing.T) {
	doc := newTestDoc(t)

	tests := []struct {
		name     string
		selector string
		want     []string
	}{
		{
			name:     "tag selector",
			selector: "p",
			want:     []string{"欢迎使用 阅读器！", "段落1", "段落2", "段落3"},
		},
		{
			name:     "class selector",
			selector: ".book",
			want:     []string{"第一本书", "第二本书", "第三本书", "第四本书", "第五本书"},
		},
		{
			name:     "id selector",
			selector: "#main",
			want:     []string{"主标题\n欢迎使用 阅读器！\n第一本书第二本书第三本书第四本书第五本书\n链接一链接二\n段落1\n段落2\n段落3"},
		},
		{
			name:     "attribute selector",
			selector: "[data-id]",
			want:     []string{"第一本书", "第二本书", "第三本书", "第四本书", "第五本书"},
		},
		{
			name:     "child combinator",
			selector: "div > p",
			want:     []string{"欢迎使用 阅读器！"},
		},
		{
			name:     "descendant selector",
			selector: "div p",
			want:     []string{"欢迎使用 阅读器！", "段落1", "段落2", "段落3"},
		},
		{
			name:     "nested element",
			selector: "li.book a",
			want:     []string{"第一本书", "第二本书", "第三本书", "第四本书", "第五本书"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCSS(tt.selector, doc)
			if err != nil {
				t.Fatalf("ParseCSS() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d, want=%d\ngot=%v\nwant=%v", len(got), len(tt.want), got, tt.want)
			}
			for i, v := range got {
				// 对于包含多个子元素的文本，做模糊匹配
				if !strings.Contains(tt.want[i], strings.TrimSpace(v)) && !strings.Contains(v, strings.TrimSpace(tt.want[i])) {
					// 检查是否只是空白差异
					if !equalText(v, tt.want[i]) {
						t.Errorf("got[%d]=%q, want[%d]=%q", i, v, i, tt.want[i])
					}
				}
			}
		})
	}
}

func equalText(a, b string) bool {
	return strings.Join(strings.Fields(a), " ") == strings.Join(strings.Fields(b), " ")
}

// ==================== 属性提取测试 ====================

func TestParseCSS_AttributeExtraction(t *testing.T) {
	doc := newTestDoc(t)

	tests := []struct {
		name     string
		selector string
		want     []string
	}{
		{
			name:     "extract text",
			selector: "h1.title@text",
			want:     []string{"主标题"},
		},
		{
			name:     "extract href",
			selector: "li.book a@href",
			want:     []string{"/book/1", "/book/2", "/book/3", "/book/4", "/book/5"},
		},
		{
			name:     "extract data-id",
			selector: "li.book@data-id",
			want:     []string{"1", "2", "3", "4", "5"},
		},
		{
			name:     "extract class",
			selector: "h1@class",
			want:     []string{"title"},
		},
		{
			name:     "extract html",
			selector: "strong@html",
			want:     []string{"阅读器"},
		},
		{
			name:     "extract ownText (direct text, concatenated)",
			selector: "p.intro@ownText",
			want:     []string{"欢迎使用 阅读器！"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCSS(tt.selector, doc)
			if err != nil {
				t.Fatalf("ParseCSS() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d, want=%d\ngot=%v\nwant=%v", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d]=%q, want[%d]=%q", i, got[i], i, tt.want[i])
				}
			}
		})
	}
}

// ==================== 扩展索引语法测试 ====================

func TestParseCSS_IndexSyntax(t *testing.T) {
	doc := newTestDoc(t)

	tests := []struct {
		name     string
		selector string
		want     []string
	}{
		// 单个索引
		{
			name:     "single index :0",
			selector: "p:0",
			want:     []string{"欢迎使用 阅读器！"},
		},
		{
			name:     "single index :2",
			selector: "p:2",
			want:     []string{"段落2"},
		},
		{
			name:     "single index :4 (out of range)",
			selector: "p:4",
			want:     []string{},
		},
		// 最后一个
		{
			name:     "last index :-1",
			selector: "p:-1",
			want:     []string{"段落3"},
		},
		{
			name:     "last index :-2",
			selector: "p:-2",
			want:     []string{"段落2"},
		},
		// 索引列表
		{
			name:     "indices [0,2,4]",
			selector: "p[0,2,4]",
			want:     []string{"欢迎使用 阅读器！", "段落2"},
		},
		{
			name:     "indices [0,1]",
			selector: "li.book[0,1]",
			want:     []string{"第一本书", "第二本书"},
		},
		// 区间选择 [start:end)
		{
			name:     "range [0:3]",
			selector: "li.book[0:3]",
			want:     []string{"第一本书", "第二本书", "第三本书"},
		},
		{
			name:     "range [2:5]",
			selector: "li.book[2:5]",
			want:     []string{"第三本书", "第四本书", "第五本书"},
		},
		{
			name:     "range [1:3]",
			selector: "p[1:3]",
			want:     []string{"段落1", "段落2"},
		},
		// 排除索引 !
		{
			name:     "exclude !0:2",
			selector: "li.book!0:2",
			want:     []string{"第三本书", "第四本书", "第五本书"},
		},
		{
			name:     "exclude !1:4",
			selector: "li.book!1:4",
			want:     []string{"第一本书", "第五本书"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCSS(tt.selector, doc)
			if err != nil {
				t.Fatalf("ParseCSS() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d, want=%d\ngot=%v\nwant=%v", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d]=%q, want[%d]=%q", i, got[i], i, tt.want[i])
				}
			}
		})
	}
}

// ==================== 链式组合 && 测试 ====================

func TestParseCSS_Chain(t *testing.T) {
	doc := newTestDoc(t)

	tests := []struct {
		name     string
		selector string
		wantLen  int
	}{
		{
			name:     "chain two selectors",
			selector: "li.book a@href && div.sidebar a@href",
			wantLen:  7, // 5 book links + 2 sidebar links
		},
		{
			name:     "chain with same tag",
			selector: "p:0 && p:-1",
			wantLen:  2,
		},
		{
			name:     "chain three selectors",
			selector: "h1@text && p.intro@text && p:1@text",
			wantLen:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCSS(tt.selector, doc)
			if err != nil {
				t.Fatalf("ParseCSS() error = %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len(got)=%d, want=%d\ngot=%v", len(got), tt.wantLen, got)
			}
		})
	}
}

// ==================== ParseCSSWithDoc 测试 ====================

func TestParseCSSWithDoc(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		html     string
		want     []string
	}{
		{
			name:     "simple div",
			selector: "div.content",
			html:     `<div class="content"><p>Hello</p></div>`,
			want:     []string{"Hello"},
		},
		{
			name:     "attribute from inline HTML",
			selector: "a@href",
			html:     `<a href="https://example.com">Link</a>`,
			want:     []string{"https://example.com"},
		},
		{
			name:     "index on inline HTML",
			selector: "span[0:2]",
			html:     `<span>1</span><span>2</span><span>3</span>`,
			want:     []string{"1", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCSSWithDoc(tt.selector, tt.html)
			if err != nil {
				t.Fatalf("ParseCSSWithDoc() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d, want=%d\ngot=%v\nwant=%v", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d]=%q, want[%d]=%q", i, got[i], i, tt.want[i])
				}
			}
		})
	}
}

// ==================== CSSSelector 链式构建器测试 ====================

func TestCSSSelector_ChainBuilder(t *testing.T) {
	doc := newTestDoc(t)

	t.Run("find and extract", func(t *testing.T) {
		result := CSS(doc).Find("li.book").Find("a").Text()
		if len(result) != 5 {
			t.Fatalf("expected 5 results, got %d: %v", len(result), result)
		}
	})

	t.Run("filter", func(t *testing.T) {
		result := CSS(doc).Find("li.book").Filter("[data-id=\"2\"]").Text()
		if len(result) != 1 || result[0] != "第二本书" {
			t.Fatalf("expected ['第二本书'], got %v", result)
		}
	})

	t.Run("eq", func(t *testing.T) {
		result := CSS(doc).Find("p").Eq(1).Text()
		if len(result) != 1 || result[0] != "段落1" {
			t.Fatalf("expected ['段落1'], got %v", result)
		}
	})

	t.Run("first/last", func(t *testing.T) {
		first := CSS(doc).Find("p").First().Text()
		last := CSS(doc).Find("p").Last().Text()
		if len(first) != 1 || first[0] != "欢迎使用 阅读器！" {
			t.Fatalf("first expected '欢迎使用 阅读器！', got %v", first)
		}
		if len(last) != 1 || last[0] != "段落3" {
			t.Fatalf("last expected '段落3', got %v", last)
		}
	})

	t.Run("attr", func(t *testing.T) {
		result := CSS(doc).Find("li.book a").Attr("href")
		if len(result) != 5 {
			t.Fatalf("expected 5 hrefs, got %d: %v", len(result), result)
		}
	})

	t.Run("html", func(t *testing.T) {
		result := CSS(doc).Find("strong").HTML()
		if len(result) != 1 || result[0] != "阅读器" {
			t.Fatalf("expected ['阅读器'], got %v", result)
		}
	})
}

// ==================== 错误处理测试 ====================

func TestParseCSS_ErrorHandling(t *testing.T) {
	doc := newTestDoc(t)

	tests := []struct {
		name     string
		selector string
		wantErr  bool
	}{
		{
			name:     "nil document",
			selector: "p",
			wantErr:  true,
		},
		{
			name:     "empty selector",
			selector: "",
			wantErr:  true,
		},
		{
			name:     "invalid selector",
			selector: "div > > p",
			wantErr:  false, // goquery handles gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *goquery.Document
			if tt.selector != "" {
				d = doc
			}
			_, err := ParseCSS(tt.selector, d)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseCSS() error = %v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

// ==================== 真实书源规则测试 ====================

func TestParseCSS_RealWorldBookSource(t *testing.T) {
	// 模拟一个常见的小说网站书源规则
	html := `
	<html>
	<body>
		<div class="book-item" data-id="101">
			<h3 class="book-name"><a href="/book/101">诡秘之主</a></h3>
			<span class="author">爱潜水的乌贼</span>
			<span class="status">已完结</span>
			<div class="intro">在诡秘的世界里...</div>
		</div>
		<div class="book-item" data-id="102">
			<h3 class="book-name"><a href="/book/102">间客</a></h3>
			<span class="author">猫腻</span>
			<span class="status">已完结</span>
			<div class="intro">穿越到未来的故事...</div>
		</div>
		<div class="book-item" data-id="103">
			<h3 class="book-name"><a href="/book/103">庆余年</a></h3>
			<span class="author">猫腻</span>
			<span class="status">已完结</span>
			<div class="intro">一个关于庆国的故事...</div>
		</div>
	</body>
	</html>
	`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to create doc: %v", err)
	}

	tests := []struct {
		name     string
		selector string
		want     []string
	}{
		{
			name:     "书名列表",
			selector: ".book-item .book-name a@text",
			want:     []string{"诡秘之主", "间客", "庆余年"},
		},
		{
			name:     "书名链接",
			selector: ".book-item .book-name a@href",
			want:     []string{"/book/101", "/book/102", "/book/103"},
		},
		{
			name:     "作者列表",
			selector: ".book-item .author@text",
			want:     []string{"爱潜水的乌贼", "猫腻", "猫腻"},
		},
		{
			name:     "只取前两本书",
			selector: ".book-item[0:2] .book-name a@text",
			want:     []string{"诡秘之主", "间客"},
		},
		{
			name:     "取最后一本书",
			selector: ".book-item:-1 .book-name a@text",
			want:     []string{"庆余年"},
		},
		{
			name:     "取第一本和第三本",
			selector: ".book-item[0,2] .book-name a@text",
			want:     []string{"诡秘之主", "庆余年"},
		},
		{
			name:     "排除第一本",
			selector: ".book-item!0:1 .author@text",
			want:     []string{"猫腻", "猫腻"},
		},
		{
			name:     "链式组合: 书名 && 作者",
			selector: ".book-item .book-name a@text && .book-item .author@text",
			want:     []string{"诡秘之主", "猫腻", "间客", "猫腻", "庆余年", "猫腻"},
		},
		{
			name:     "提取 book-item 的 data-id",
			selector: ".book-item@data-id",
			want:     []string{"101", "102", "103"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCSS(tt.selector, doc)
			if err != nil {
				t.Fatalf("ParseCSS() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d, want=%d\ngot=%v\nwant=%v", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d]=%q, want[%d]=%q", i, got[i], i, tt.want[i])
				}
			}
		})
	}
}

// ==================== 边界情况测试 ====================

func TestParseCSS_EdgeCases(t *testing.T) {
	doc := newTestDoc(t)

	tests := []struct {
		name     string
		selector string
		want     []string
	}{
		{
			name:     "index out of range returns empty",
			selector: "p:99",
			want:     []string{},
		},
		{
			name:     "negative index beyond length",
			selector: "p:-99",
			want:     []string{},
		},
		{
			name:     "empty range",
			selector: "p[5:3]",
			want:     []string{},
		},
		{
			name:     "range beyond length",
			selector: "p[2:99]",
			want:     []string{"段落2", "段落3"},
		},
		{
			name:     "exclude all",
			selector: "p!0:10",
			want:     []string{},
		},
		{
			name:     "indices with negative",
			selector: "p[0,-1]",
			want:     []string{"欢迎使用 阅读器！", "段落3"},
		},
		{
			name:     "attribute not exists",
			selector: "p@nonexistent",
			want:     []string{},
		},
		{
			name:     "empty result from no match",
			selector: "div.nonexistent",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCSS(tt.selector, doc)
			if err != nil {
				t.Fatalf("ParseCSS() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d, want=%d\ngot=%v\nwant=%v", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d]=%q, want[%d]=%q", i, got[i], i, tt.want[i])
				}
			}
		})
	}
}
