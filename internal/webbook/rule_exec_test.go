package webbook

import (
	"context"
	"testing"
)

func TestParseLegadoFieldRules_JSON(t *testing.T) {
	rules := parseLegadoFieldRules(`{"name":"$.title","bookList":"$.list"}`)
	if rules["name"] != "$.title" || rules["bookList"] != "$.list" {
		t.Fatalf("got %v", rules)
	}
}

func TestParseLegadoFieldRules_KV(t *testing.T) {
	rules := parseLegadoFieldRules("name:h1&&author:.author")
	if rules["name"] != "h1" || rules["author"] != ".author" {
		t.Fatalf("got %v", rules)
	}
}

func TestParseLegadoKVRule(t *testing.T) {
	rules := parseLegadoKVRule("content:#content&&title:h1")
	if rules["content"] != "#content" {
		t.Fatalf("got %v", rules)
	}
}

func TestIsLegadoChainRule(t *testing.T) {
	if !isLegadoChainRule("class.title@text") {
		t.Fatal("expected legado chain")
	}
	if isLegadoChainRule("@js:return 1") {
		t.Fatal("@js should not be legado chain")
	}
}

func TestWebBook_execField_CSS(t *testing.T) {
	wb := NewWebBook()
	body := `<html><body><h1>书名</h1></body></html>`
	got := wb.execField(context.Background(), wb.modeFor(&BookSource{}, "info"), "h1", body, "http://example.com", "")
	if got != "书名" {
		t.Fatalf("got %q", got)
	}
}
