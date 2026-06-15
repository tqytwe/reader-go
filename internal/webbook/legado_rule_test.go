package webbook

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestEvalLegadoRule_MultiParagraphHTML(t *testing.T) {
	html := `<div class="chaptercontent">
<p>第一段正文。</p>
<p>第二段正文。</p>
<p>第三段正文。</p>
</div>`
	got := evalLegadoRuleOnHTML(html, ".chaptercontent@p@html")
	if got == "" {
		t.Fatal("expected non-empty html")
	}
	for _, want := range []string{"第一段正文", "第二段正文", "第三段正文"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}

func TestEvalLegadoRule_TextSelectorHref(t *testing.T) {
	html := `<div class="pages">
<a href="/chapter/1.html">上一章</a>
<a href="/chapter/3.html">下一页</a>
</div>`
	got := evalLegadoRuleOnHTML(html, "text.下一页@href")
	if got != "/chapter/3.html" {
		t.Fatalf("got %q", got)
	}
}

func TestEvalLegadoRule_TextNodes(t *testing.T) {
	html := `<div id="content"><p>你好</p><p>世界</p></div>`
	got := evalLegadoRuleOnHTML(html, "#content@p@textNodes")
	if got == "" || !strings.Contains(got, "你好") || !strings.Contains(got, "世界") {
		t.Fatalf("got %q", got)
	}
}

func TestLegadoExcludeSyntax(t *testing.T) {
	html := `<div class="directoryArea">
		<p><a href="/chapter/0.html">第一章</a></p>
		<p><a href="/chapter/1.html">第二章</a></p>
		<p><a href="/chapter/2.html">第三章</a></p>
		<p><a href="/chapter/3.html">第四章</a></p>
	</div>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}

	// 测试 !0 排除第一个元素
	items, err := evalLegadoListItems(doc.Selection, ".directoryArea p!0@a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items after excluding first, got %d", len(items))
	}

	// 验证排除的是第一个元素
	for i, item := range items {
		href, _ := item.Attr("href")
		expected := "/chapter/" + string(rune('1'+i)) + ".html"
		if href != expected {
			t.Errorf("item %d: expected href %q, got %q", i, expected, href)
		}
	}
}

func TestLegadoExcludeSyntaxLast(t *testing.T) {
	html := `<div class="list">
		<p>Item 0</p>
		<p>Item 1</p>
		<p>Item 2</p>
	</div>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}

	// 测试 !2 排除最后一个元素
	items, err := evalLegadoListItems(doc.Selection, ".list p!2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items after excluding last, got %d", len(items))
	}

	// 验证排除的是最后一个元素
	for i, item := range items {
		text := strings.TrimSpace(item.Text())
		expected := "Item " + string(rune('0'+i))
		if text != expected {
			t.Errorf("item %d: expected text %q, got %q", i, expected, text)
		}
	}
}
