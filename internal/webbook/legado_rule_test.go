package webbook

import (
	"strings"
	"testing"
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
