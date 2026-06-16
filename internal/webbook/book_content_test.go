package webbook

import (
	"strings"
	"testing"
)

func TestExtractContent_InlineElements(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		contains []string
	}{
		{
			name:     "span elements",
			html:     `<div class="content"><span>Hello</span> <span>World</span></div>`,
			contains: []string{"Hello", "World"},
		},
		{
			name:     "strong and em elements",
			html:     `<div class="content"><p>This is <strong>bold</strong> and <em>italic</em> text</p></div>`,
			contains: []string{"This is", "bold", "and", "italic", "text"},
		},
		{
			name:     "anchor elements",
			html:     `<div class="content"><p>Visit <a href="#">this link</a> for more</p></div>`,
			contains: []string{"Visit", "this link", "for more"},
		},
		{
			name:     "mixed inline elements",
			html:     `<div class="content"><p>Text with <span>span</span>, <strong>strong</strong>, and <a href="#">link</a></p></div>`,
			contains: []string{"Text with", "span", "strong", "link"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewContentParser(".content")
			content, err := parser.ParseHTML(tt.html, "http://example.com", "Chapter 1", "/ch1", "Book", "/book", "src1", "Source", 1)
			if err != nil {
				t.Fatalf("ParseHTML failed: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(content.Content, expected) {
					t.Errorf("Expected content to contain %q, got: %q", expected, content.Content)
				}
			}
		})
	}
}

func TestCleanTags_NestedNonWhitelisted(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		contains []string
		excludes []string
	}{
		{
			name:     "nested non-whitelisted elements",
			html:     `<div class="content"><section><div><span>Text</span></div></section></div>`,
			contains: []string{"Text"},
			excludes: []string{"<section>", "<div>", "<span>"},
		},
		{
			name:     "non-whitelisted parent with whitelisted child",
			html:     `<div class="content"><section><p>Paragraph</p></section></div>`,
			contains: []string{"Paragraph"},
			excludes: []string{"<section>"},
		},
		{
			name:     "multiple non-whitelisted siblings",
			html:     `<div class="content"><article>First</article><aside>Second</aside></div>`,
			contains: []string{"First", "Second"},
			excludes: []string{"<article>", "<aside>"},
		},
		{
			name:     "whitelisted element with non-whitelisted child",
			html:     `<div class="content"><div>Before <font>middle</font> after</div></div>`,
			contains: []string{"Before", "middle", "after"},
			excludes: []string{"<font>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewContentParser(".content")
			content, err := parser.ParseHTML(tt.html, "http://example.com", "Chapter 1", "/ch1", "Book", "/book", "src1", "Source", 1)
			if err != nil {
				t.Fatalf("ParseHTML failed: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(content.Content, expected) {
					t.Errorf("Expected content to contain %q, got: %q", expected, content.Content)
				}
			}

			for _, excluded := range tt.excludes {
				if strings.Contains(content.RawHTML, excluded) {
					t.Errorf("Expected RawHTML to NOT contain %q, got: %q", excluded, content.RawHTML)
				}
			}
		})
	}
}

func TestExtractContent_ParagraphStructure(t *testing.T) {
	html := `<div class="content">
		<p>First paragraph</p>
		<p>Second paragraph</p>
		<p>Third paragraph</p>
	</div>`

	parser := NewContentParser(".content")
	content, err := parser.ParseHTML(html, "http://example.com", "Chapter 1", "/ch1", "Book", "/book", "src1", "Source", 1)
	if err != nil {
		t.Fatalf("ParseHTML failed: %v", err)
	}

	// Check that paragraphs are preserved as separate blocks
	if !strings.Contains(content.Content, "First paragraph") {
		t.Errorf("Expected 'First paragraph', got: %q", content.Content)
	}
	if !strings.Contains(content.Content, "Second paragraph") {
		t.Errorf("Expected 'Second paragraph', got: %q", content.Content)
	}
	if !strings.Contains(content.Content, "Third paragraph") {
		t.Errorf("Expected 'Third paragraph', got: %q", content.Content)
	}

	// Check that there are line breaks between paragraphs
	lines := strings.Split(content.Content, "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines (paragraphs), got %d lines: %q", len(lines), content.Content)
	}
}

func TestCleanTags_PreservesWhitelistedStructure(t *testing.T) {
	html := `<div class="content">
		<div>
			<p>Paragraph 1</p>
			<p>Paragraph 2</p>
		</div>
	</div>`

	parser := NewContentParser(".content")
	content, err := parser.ParseHTML(html, "http://example.com", "Chapter 1", "/ch1", "Book", "/book", "src1", "Source", 1)
	if err != nil {
		t.Fatalf("ParseHTML failed: %v", err)
	}

	// Both paragraphs should be present
	if !strings.Contains(content.Content, "Paragraph 1") {
		t.Errorf("Expected 'Paragraph 1', got: %q", content.Content)
	}
	if !strings.Contains(content.Content, "Paragraph 2") {
		t.Errorf("Expected 'Paragraph 2', got: %q", content.Content)
	}

	// RawHTML should still contain p tags (structure preserved)
	if !strings.Contains(content.RawHTML, "<p>") {
		t.Errorf("Expected RawHTML to contain <p> tags, got: %q", content.RawHTML)
	}
}

func TestExtractContent_ListElements(t *testing.T) {
	html := `<div class="content">
		<ul>
			<li>Item 1</li>
			<li>Item 2</li>
			<li>Item 3</li>
		</ul>
	</div>`

	parser := NewContentParser(".content")
	content, err := parser.ParseHTML(html, "http://example.com", "Chapter 1", "/ch1", "Book", "/book", "src1", "Source", 1)
	if err != nil {
		t.Fatalf("ParseHTML failed: %v", err)
	}

	if !strings.Contains(content.Content, "Item 1") {
		t.Errorf("Expected 'Item 1', got: %q", content.Content)
	}
	if !strings.Contains(content.Content, "Item 2") {
		t.Errorf("Expected 'Item 2', got: %q", content.Content)
	}
	if !strings.Contains(content.Content, "Item 3") {
		t.Errorf("Expected 'Item 3', got: %q", content.Content)
	}
}

func TestExtractContent_Headings(t *testing.T) {
	html := `<div class="content">
		<h1>Main Title</h1>
		<p>Content text</p>
		<h2>Subtitle</h2>
		<p>More content</p>
	</div>`

	parser := NewContentParser(".content")
	content, err := parser.ParseHTML(html, "http://example.com", "Chapter 1", "/ch1", "Book", "/book", "src1", "Source", 1)
	if err != nil {
		t.Fatalf("ParseHTML failed: %v", err)
	}

	for _, expected := range []string{"Main Title", "Content text", "Subtitle", "More content"} {
		if !strings.Contains(content.Content, expected) {
			t.Errorf("Expected %q in content, got: %q", expected, content.Content)
		}
	}
}

func TestCleanTags_RemovesScriptAndStyle(t *testing.T) {
	html := `<div class="content">
		<p>Real content</p>
		<script>alert('xss')</script>
		<style>.hidden { display: none; }</style>
		<p>More content</p>
	</div>`

	parser := NewContentParser(".content")
	content, err := parser.ParseHTML(html, "http://example.com", "Chapter 1", "/ch1", "Book", "/book", "src1", "Source", 1)
	if err != nil {
		t.Fatalf("ParseHTML failed: %v", err)
	}

	// Real content should be present
	if !strings.Contains(content.Content, "Real content") {
		t.Errorf("Expected 'Real content', got: %q", content.Content)
	}
	if !strings.Contains(content.Content, "More content") {
		t.Errorf("Expected 'More content', got: %q", content.Content)
	}

	// Script and style content should NOT be in output
	if strings.Contains(content.Content, "alert") {
		t.Errorf("Script content should not be in output, got: %q", content.Content)
	}
	if strings.Contains(content.Content, "display: none") {
		t.Errorf("Style content should not be in output, got: %q", content.Content)
	}
}
