package webbook

import "testing"

func TestBuildSourceURL(t *testing.T) {
	wb := NewWebBook()
	source := &BookSource{
		Name:    "test",
		BaseURL: "https://example.com",
	}

	t.Run("empty template uses book url", func(t *testing.T) {
		u, method, body, err := wb.buildSourceURL(source, "", "https://example.com/book/1", nil)
		if err != nil {
			t.Fatal(err)
		}
		if u != "https://example.com/book/1" || method != "GET" || body != "" {
			t.Fatalf("got (%q, %q, %q)", u, method, body)
		}
	})

	t.Run("empty book url", func(t *testing.T) {
		_, _, _, err := wb.buildSourceURL(source, "", "", nil)
		if err == nil {
			t.Fatal("expected error for empty book URL")
		}
	})

	t.Run("template placeholder", func(t *testing.T) {
		u, _, _, err := wb.buildSourceURL(source, "https://example.com/info?id={{bookUrl}}", "42", nil)
		if err != nil {
			t.Fatal(err)
		}
		if u != "https://example.com/info?id=42" {
			t.Fatalf("got %q", u)
		}
	})

	t.Run("relative template", func(t *testing.T) {
		u, _, _, err := wb.buildSourceURL(source, "/book/{{bookUrl}}", "99", nil)
		if err != nil {
			t.Fatal(err)
		}
		if u != "https://example.com/book/99" {
			t.Fatalf("got %q", u)
		}
	})

	t.Run("relative book url", func(t *testing.T) {
		u, _, _, err := wb.buildSourceURL(source, "", "/book/7", nil)
		if err != nil {
			t.Fatal(err)
		}
		if u != "https://example.com/book/7" {
			t.Fatalf("got %q", u)
		}
	})

	t.Run("base url strips hash suffix", func(t *testing.T) {
		src := &BookSource{Name: "m", BaseURL: "http://m.example.com#"}
		u, _, _, err := wb.buildSourceURL(src, "/toc/{{bookUrl}}", "123", nil)
		if err != nil {
			t.Fatal(err)
		}
		if u != "http://m.example.com/toc/123" {
			t.Fatalf("got %q", u)
		}
	})
}

func TestIsLegadoSelectorURLRule(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"/book/{{bookUrl}}", false},
		{"https://example.com/toc", false},
		{".ptm-card-footer@a@href", true},
		{"@js:()", false},
	}
	for _, tc := range cases {
		if got := isLegadoSelectorURLRule(tc.in); got != tc.want {
			t.Fatalf("isLegadoSelectorURLRule(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestTocURLSelectorExtraction(t *testing.T) {
	html := `<html><body><div class="ptm-card-footer"><a href="/book/1/chapters/">目录</a></div></body></html>`
	got := resolveLegadoURL(evalLegadoRuleOnHTML(html, ".ptm-card-footer@a@href"), "http://m.example.com/book/1/", html)
	if got != "http://m.example.com/book/1/chapters/" {
		t.Fatalf("got %q", got)
	}
}

func TestTocURLSelectorNegativeIndex(t *testing.T) {
	html := `<html><body><div class="btn_book"><a href="/old">旧</a><a href="/toc/latest">目录</a></div></body></html>`
	got := resolveLegadoURL(evalLegadoRuleOnHTML(html, ".btn_book@a.-1@href"), "http://m.example.com/book/1/", html)
	if got != "http://m.example.com/toc/latest" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveURLRootRelative(t *testing.T) {
	got := resolveURL("/dir/24254/", "http://m.bbiquge8.net/book/24254/")
	if got != "http://m.bbiquge8.net/dir/24254/" {
		t.Fatalf("got %q", got)
	}
}
