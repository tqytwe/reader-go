package webbook

import (
	"testing"
)

func TestResolveExploreURLPlain(t *testing.T) {
	url, tabs, tab, err := ResolveExploreURL("https://example.com/list", "")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://example.com/list" || len(tabs) != 0 || tab != "" {
		t.Fatalf("got url=%q tabs=%v tab=%q", url, tabs, tab)
	}
}

func TestResolveExploreURLTabsSkipEmptyDefault(t *testing.T) {
	exploreURL := `[{"title":"榜单","url":""},{"title":"周点击榜","url":"/paihangbang_weekvisit/{{page}}.html"}]`
	url, tabs, tab, err := ResolveExploreURL(exploreURL, "")
	if err != nil {
		t.Fatal(err)
	}
	if tab != "周点击榜" {
		t.Fatalf("tab = %q, want 周点击榜", tab)
	}
	if url != "/paihangbang_weekvisit/{{page}}.html" {
		t.Fatalf("url = %q", url)
	}
	if len(tabs) != 2 {
		t.Fatalf("tabs len = %d", len(tabs))
	}
}

func TestResolveExploreURLTabsByTitle(t *testing.T) {
	exploreURL := `[{"title":"榜单","url":""},{"title":"月点击榜","url":"/month/{{page}}.html"}]`
	url, _, tab, err := ResolveExploreURL(exploreURL, "月点击榜")
	if err != nil {
		t.Fatal(err)
	}
	if tab != "月点击榜" || url != "/month/{{page}}.html" {
		t.Fatalf("tab=%q url=%q", tab, url)
	}
}

func TestResolveExploreURLTabsEmptySelected(t *testing.T) {
	exploreURL := `[{"title":"榜单","url":""}]`
	_, _, _, err := ResolveExploreURL(exploreURL, "榜单")
	if err == nil {
		t.Fatal("expected error for empty tab URL")
	}
}

func TestBuildExploreURL(t *testing.T) {
	wb := &WebBook{}
	source := &BookSource{
		BaseURL: "https://www.example.com#",
	}
	got, err := wb.buildExploreURL(source, "/paihangbang_weekvisit/{{page}}.html", 1)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://www.example.com/paihangbang_weekvisit/1.html"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
