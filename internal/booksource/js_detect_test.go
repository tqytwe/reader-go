package booksource

import (
	"encoding/json"
	"testing"
)

func TestContainsJSRule(t *testing.T) {
	cases := []struct {
		rule string
		want bool
	}{
		{"", false},
		{"class.grid@tag.tr", false},
		{"$.data.content", false},
		{"$.id@js:\"https://example.com/book/\"+result", true},
		{"<js>eval(String(source.bookSourceComment))</js>\np.sone", true},
		{"@js:result", true},
	}
	for _, tc := range cases {
		if got := ContainsJSRule(tc.rule); got != tc.want {
			t.Errorf("ContainsJSRule(%q) = %v, want %v", tc.rule, got, tc.want)
		}
	}
}

func TestRequiresJSSearchLegacy(t *testing.T) {
	plain := &LegacyBookSource{
		BookSourceName: "若初文学",
		SearchURL:      "http://search.example?q={{key}}",
		RuleSearch: map[string]interface{}{
			"bookList": "$.data.content",
			"bookUrl":  "$.id@js:\"https://www.ruochu.com/book/\"+result",
		},
	}
	if !RequiresJSSearchLegacy(plain) {
		t.Fatal("expected JS required for bookUrl @js rule")
	}

	css := &LegacyBookSource{
		BookSourceName: "八中文网",
		SearchURL:      "http://www.example.com/search.php",
		RuleSearch: map[string]interface{}{
			"bookList": "class.grid@tag.tr!0",
			"name":     "class.odd.0@text",
		},
	}
	if RequiresJSSearchLegacy(css) {
		t.Fatal("expected non-JS source")
	}

	webview := &LegacyBookSource{
		BookSourceName: "纵横",
		SearchURL:      "https://m.zongheng.com/search?keywords={{key}},{\"webView\": true}",
		RuleSearch:     map[string]interface{}{"bookList": "class.book-li"},
	}
	if !RequiresJSSearchLegacy(webview) {
		t.Fatal("expected webView source to require JS")
	}
}

func TestApplyEnablePolicy(t *testing.T) {
	sources := []*BookSource{
		{Name: "a", Enabled: true, SearchURL: "http://a/search?q={{key}}", SearchRule: `{"bookList":"class.x"}`},
		{Name: "b", Enabled: true, SearchURL: "@js:result", SearchRule: `{}`},
		{Name: "c", Enabled: true, SearchURL: "http://c/search", SearchRule: `{"bookUrl":"@js:result"}`},
	}
	stats := ApplyEnablePolicy(sources, true)
	if stats.NonJS != 1 || stats.JSRequired != 2 {
		t.Fatalf("unexpected counts: %+v", stats)
	}
	if stats.Enabled != 1 || stats.Disabled != 2 {
		t.Fatalf("unexpected enable counts: %+v", stats)
	}
	if !sources[0].Enabled || sources[1].Enabled || sources[2].Enabled {
		t.Fatal("enable flags not applied correctly")
	}

	statsAll := ApplyEnablePolicy([]*BookSource{
		{Name: "a", Enabled: true, SearchURL: "http://a/search?q={{key}}", SearchRule: `{"bookList":"class.x"}`},
		{Name: "b", Enabled: true, SearchURL: "@js:result", SearchRule: `{}`},
		{Name: "c", Enabled: true, SearchURL: "http://c/search", SearchRule: `{"bookUrl":"@js:result"}`},
	}, false)
	if statsAll.Enabled != 3 {
		t.Fatalf("expected all enabled when policy off, got %+v", statsAll)
	}
}

func TestParseCollection1128Sample(t *testing.T) {
	raw := `[{
		"bookSourceName":"若初文学",
		"bookSourceUrl":"http://www.ruochu.com",
		"searchUrl":"http://search.ruochu.com/web/search?queryString={{key}}",
		"enabled":true,
		"ruleSearch":{"bookList":"$.data.content","bookUrl":"$.id@js:\"https://www.ruochu.com/book/\"+result","name":"$.name"}
	},{
		"bookSourceName":"八中文网",
		"bookSourceUrl":"http://www.zwduxs.com",
		"searchUrl":"http://www.zwduxs.com/modules/article/search.php,{\"method\":\"POST\",\"body\":\"searchkey={{key}}\"}",
		"enabled":true,
		"ruleSearch":{"bookList":"class.grid@tag.tr!0","name":"class.odd.0@text"}
	}]`
	var legacy []LegacyBookSource
	if err := json.Unmarshal([]byte(raw), &legacy); err != nil {
		t.Fatal(err)
	}
	sources, result := ParseBookSourceCollection([]byte(raw))
	if result.Success != 2 {
		t.Fatalf("parse failed: %+v", result)
	}
	stats := ApplyEnablePolicy(sources, true)
	if stats.Enabled != 1 || stats.JSRequired != 1 || stats.NonJS != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
