package rss

import "testing"

func TestDiscoverFeedURLFromHTML(t *testing.T) {
	html := `<html><head>
<link rel="alternate" type="application/rss+xml" href="/feed.xml" title="RSS">
</head></html>`
	got := discoverFeedURLFromHTML(html, "https://example.com/blog/")
	if got != "https://example.com/feed.xml" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveFeedFetchURL(t *testing.T) {
	rules := map[string]string{"sortUrl": "https://api.example.com/list"}
	got, err := resolveFeedFetchURL("📖虚拟源", "https://example.com", rules)
	if err != nil || got != "https://api.example.com/list" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestIsHTTPURL(t *testing.T) {
	if !isHTTPURL("https://a.com") || isHTTPURL("📖虚拟源") {
		t.Fatal("isHTTPURL mismatch")
	}
}
