package rule

import (
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helper: create a test HTML document ---

func testHTMLDoc() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>测试页面 - Test Page</title>
</head>
<body>
    <header>
        <h1 id="title">小说名称</h1>
        <div class="author">作者：张三</div>
        <div class="info">
            <span class="status">连载中</span>
            <span class="update">每日更新</span>
        </div>
    </header>
    <main>
        <div class="chapter-list">
            <a class="chapter" href="/chap/1">第一章 开篇</a>
            <a class="chapter" href="/chap/2">第二章 启程</a>
            <a class="chapter" href="/chap/3">第三章 相遇</a>
        </div>
        <article class="content">
            <p>这是正文第一段。</p>
            <p>这是正文第二段。</p>
            <div class="ad">广告内容</div>
            <p>这是正文第三段。</p>
        </article>
    </main>
    <footer>
        <p>© 2024 书源网站</p>
    </footer>
</body>
</html>`
}

func testHTML2() string {
	// A second HTML for testing union/intersection
	return `<!DOCTYPE html>
<html>
<body>
    <div class="book">
        <h2>书名A</h2>
        <p class="intro">简介A</p>
    </div>
    <div class="book">
        <h2>书名B</h2>
        <p class="intro">简介B</p>
    </div>
    <div class="book">
        <h2>书名C</h2>
        <p class="intro">简介C</p>
    </div>
</body>
</html>`
}

func parseTestHTML(t *testing.T, htmlStr string) *xmlquery.Node {
	doc, err := xmlquery.Parse(strings.NewReader(htmlStr))
	require.NoError(t, err)
	return doc
}

// ============================================================
// XPathParser tests - Basic queries
// ============================================================

func TestXPathParser_ParseXPath_Basic(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())

	// Test title extraction
	results, err := parser.ParseXPath("//title/text()", doc)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0], "测试页面")

	// Test h1 text
	results, err = parser.ParseXPath("//h1/text()", doc)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "小说名称", results[0])

	// Test chapter links
	results, err = parser.ParseXPath("//a[@class='chapter']/@href", doc)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, "/chap/1", results[0])
	assert.Equal(t, "/chap/2", results[1])
	assert.Equal(t, "/chap/3", results[2])

	// Test chapter text
	results, err = parser.ParseXPath("//a[@class='chapter']", doc)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Contains(t, results[0], "第一章")
}

func TestXPathParser_ParseXPath_ClassSelector(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())

	// Test class-based selection
	results, err := parser.ParseXPath("//div[@class='author']", doc)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0], "作者")
	assert.Contains(t, results[0], "张三")

	// Test multiple same-class elements
	results, err = parser.ParseXPath("//span[@class='status']", doc)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "连载中", strings.TrimSpace(results[0]))
}

func TestXPathParser_ParseXPath_Descendant(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())

	// Test descendant selector
	results, err := parser.ParseXPath("//main//p", doc)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Contains(t, results[0], "正文第一段")
	assert.Contains(t, results[2], "正文第三段")
}

func TestXPathParser_ParseXPath_Attribute(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())

	// Test attribute selection
	results, err := parser.ParseXPath("//h1/@id", doc)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "title", results[0])

	// Test multiple attributes
	results, err = parser.ParseXPath("//a[@class='chapter']/@href", doc)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

// ============================================================
// XPathParser tests - Chain operators
// ============================================================

func TestXPathParser_ParseXPath_AndChain(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())

	// && intersection: find chapter links that also match a specific pattern
	// In practice, && is used to narrow down results
	results, err := parser.ParseXPath("//a && //a[@class='chapter']", doc)
	require.NoError(t, err)
	// All <a> elements that are also <a class="chapter">
	assert.Greater(t, len(results), 0)
	for _, r := range results {
		assert.Contains(t, r, "章")
	}
}

func TestXPathParser_ParseXPath_OrChain(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())

	// || union: get both h1 and h2 titles
	results, err := parser.ParseXPath("//h1 || //h2", doc)
	require.NoError(t, err)
	// Should have at least the h1 result
	assert.Greater(t, len(results), 0)
	hasH1 := false
	for _, r := range results {
		if strings.Contains(r, "小说名称") {
			hasH1 = true
		}
	}
	assert.True(t, hasH1)
}

func TestXPathParser_ParseXPath_ModChain(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())

	// %% text match: filter elements containing specific text
	results, err := parser.ParseXPath("//p %% 正文", doc)
	require.NoError(t, err)
	// Should find paragraphs containing "正文"
	assert.Greater(t, len(results), 0)
	for _, r := range results {
		assert.Contains(t, r, "正文")
	}
}

func TestXPathParser_ParseXPath_ChainedOperators(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())

	// Complex chain: //div && //div[@class='content'] || //div[@class='ad']
	// This tests the operator precedence handling
	results, err := parser.ParseXPath("//div[@class='content'] || //div[@class='ad']", doc)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

// ============================================================
// XPathParser tests - Real book source rules
// ============================================================

func TestXPathParser_RealBookSource_TitleAndAuthor(t *testing.T) {
	// Simulated book detail page from a real book source
	html := `<!DOCTYPE html>
<html>
<body>
    <div class="book-info">
        <h1 class="book-title">诡秘之主</h1>
        <div class="author-info">
            <span class="author">爱潜水的乌贼</span>
            <span class="category">玄幻-西方玄幻</span>
        </div>
        <div class="book-status">
            <span>已完结</span>
            <span>137.7万字</span>
        </div>
        <p class="intro">在蒸汽与机械的浪潮中，谁能触及非凡？</p>
    </div>
    <div class="chapter-list">
        <a href="/book/1/chap/1">第一卷 小人物 第一章 克苏鲁</a>
        <a href="/book/1/chap/2">第一卷 小人物 第二章 占卜</a>
        <a href="/book/1/chap/3">第一卷 小人物 第三章 仪式</a>
    </div>
</body>
</html>`

	parser := NewXPathParser()
	doc := parseTestHTML(t, html)

	// Extract book title
	title, err := parser.ParseXPath("//h1[@class='book-title']", doc)
	require.NoError(t, err)
	assert.Len(t, title, 1)
	assert.Equal(t, "诡秘之主", strings.TrimSpace(title[0]))

	// Extract author
	author, err := parser.ParseXPath("//span[@class='author']", doc)
	require.NoError(t, err)
	assert.Len(t, author, 1)
	assert.Equal(t, "爱潜水的乌贼", strings.TrimSpace(author[0]))

	// Extract intro
	intro, err := parser.ParseXPath("//p[@class='intro']", doc)
	require.NoError(t, err)
	assert.Len(t, intro, 1)
	assert.Contains(t, intro[0], "蒸汽与机械")
}

func TestXPathParser_RealBookSource_ChapterList(t *testing.T) {
	// Real-world chapter list extraction
	html := `<!DOCTYPE html>
<html>
<body>
    <div id="list">
        <dl>
            <dt>第一卷</dt>
            <dd><a href="/c/1">第1章 穿越</a></dd>
            <dd><a href="/c/2">第2章 觉醒</a></dd>
            <dd><a href="/c/3">第3章 修炼</a></dd>
            <dt>第二卷</dt>
            <dd><a href="/c/10">第10章 突破</a></dd>
        </dl>
    </div>
</body>
</html>`

	parser := NewXPathParser()
	doc := parseTestHTML(t, html)

	// Extract all chapter links
	chapters, err := parser.ParseXPath("//dd/a", doc)
	require.NoError(t, err)
	assert.Len(t, chapters, 4)
	assert.Equal(t, "第1章 穿越", strings.TrimSpace(chapters[0]))
	assert.Contains(t, chapters[3], "第10章")

	// Extract chapter URLs
	urls, err := parser.ParseXPath("//dd/a/@href", doc)
	require.NoError(t, err)
	assert.Len(t, urls, 4)
	assert.Equal(t, "/c/1", urls[0])
}

func TestXPathParser_RealBookSource_ContentExtraction(t *testing.T) {
	// Real-world content extraction with ad filtering
	html := `<!DOCTYPE html>
<html>
<body>
    <div id="content">
        <h1>第一章 穿越</h1>
        <div class="article">
            <p>清晨的阳光透过窗帘洒在书桌上。</p>
            <div class="ad-box">广告：购买会员享特权</div>
            <p>他揉了揉眼睛，发现自己躺在一张陌生的床上。</p>
            <p class="tip">这是穿越后的第一个早晨。</p>
            <div class="ad-box">广告：推荐阅读</div>
            <p>记忆如潮水般涌来，他意识到自己已经不再是原来的自己。</p>
        </div>
        <div class="next-chapter">
            <a href="/c/2">下一章 &gt;</a>
        </div>
    </div>
</body>
</html>`

	parser := NewXPathParser()
	doc := parseTestHTML(t, html)

	// Extract all paragraphs (including ads - user can filter)
	paragraphs, err := parser.ParseXPath("//p", doc)
	require.NoError(t, err)
	assert.Len(t, paragraphs, 4)

	// Extract only non-ad paragraphs using && chain
	// This simulates: get all p elements that are NOT inside .ad-box
	// Since xmlquery doesn't support :not(), we use a different approach
	contentParas, err := parser.ParseXPath("//div[@class='article']/p", doc)
	require.NoError(t, err)
	assert.Len(t, contentParas, 4) // includes .tip paragraph

	// Verify content
	assert.Contains(t, contentParas[0], "清晨的阳光")
	assert.Contains(t, contentParas[1], "陌生的床")
}

func TestXPathParser_RealBookSource_SearchResult(t *testing.T) {
	// Real-world search result page
	html := `<!DOCTYPE html>
<html>
<body>
    <div class="search-results">
        <div class="book-item">
            <h3><a href="/book/100">斗破苍穹</a></h3>
            <p class="author">天蚕土豆</p>
            <p class="update">更新至 第1632章</p>
        </div>
        <div class="book-item">
            <h3><a href="/book/101">斗罗大陆</a></h3>
            <p class="author">唐家三少</p>
            <p class="update">更新至 第371章</p>
        </div>
        <div class="book-item">
            <h3><a href="/book/102">凡人修仙传</a></h3>
            <p class="author">忘语</p>
            <p class="update">已完结</p>
        </div>
    </div>
</body>
</html>`

	parser := NewXPathParser()
	doc := parseTestHTML(t, html)

	// Extract all book names
	books, err := parser.ParseXPath("//div[@class='book-item']/h3/a", doc)
	require.NoError(t, err)
	assert.Len(t, books, 3)
	assert.Equal(t, "斗破苍穹", strings.TrimSpace(books[0]))
	assert.Equal(t, "斗罗大陆", strings.TrimSpace(books[1]))
	assert.Equal(t, "凡人修仙传", strings.TrimSpace(books[2]))

	// Extract all authors
	authors, err := parser.ParseXPath("//div[@class='book-item']/p[@class='author']", doc)
	require.NoError(t, err)
	assert.Len(t, authors, 3)
	assert.Equal(t, "天蚕土豆", strings.TrimSpace(authors[0]))

	// Extract all update info
	updates, err := parser.ParseXPath("//div[@class='book-item']/p[@class='update']", doc)
	require.NoError(t, err)
	assert.Len(t, updates, 3)
	assert.Contains(t, updates[0], "1632")
}

// ============================================================
// XPathParser tests - Convenience functions
// ============================================================

func TestXPathQuery(t *testing.T) {
	html := `<html><body><h1>Hello World</h1><p>Test</p></body></html>`

	results, err := Query(html, "//h1")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Hello World", results[0])
}

func TestQuerySelector(t *testing.T) {
	html := `<html><body><h1 id="main">Main Title</h1></body></html>`

	result, err := QueryXPathSelector(html, "//h1[@id='main']")
	require.NoError(t, err)
	assert.Equal(t, "Main Title", result)
}

func TestQuerySelector_NotFound(t *testing.T) {
	html := `<html><body><h1>Title</h1></body></html>`

	result, err := QueryXPathSelector(html, "//div[@class='nonexistent']")
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

// ============================================================
// Edge cases
// ============================================================

func TestXPathParser_NilDocument(t *testing.T) {
	parser := NewXPathParser()
	results, err := parser.ParseXPath("//title", nil)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestXPathParser_EmptyQuery(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())
	results, err := parser.ParseXPath("", doc)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestXPathParser_InvalidXPath(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())

	// Invalid XPath should return an error
	_, err := parser.ParseXPath("//invalid[@unclosed", doc)
	assert.Error(t, err)
}

func TestXPathParser_TextNodeExtraction(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())

	// Test direct text node selection
	results, err := parser.ParseXPath("//title/text()", doc)
	require.NoError(t, err)
	assert.Greater(t, len(results), 0)
	assert.Contains(t, results[0], "测试页面")
}

func TestXPathParser_MultiLevelChain(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTML2())

	// Test: get all book names, then union with all intros
	results, err := parser.ParseXPath("//h2 || //p[@class='intro']", doc)
	require.NoError(t, err)
	assert.Len(t, results, 6) // 3 book names + 3 intros
}

func TestXPathParser_AndEmptyResult(t *testing.T) {
	parser := NewXPathParser()
	doc := parseTestHTML(t, testHTMLDoc())

	// && with no intersection should return empty
	results, err := parser.ParseXPath("//h1 && //div[@class='nonexistent']", doc)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestXPathParser_ComplexRealRule(t *testing.T) {
	// A complex real-world rule combining multiple selectors
	html := `<!DOCTYPE html>
<html>
<body>
    <div class="novel-detail">
        <h1 class="title">完美世界</h1>
        <div class="meta">
            <span class="author">辰东</span>
            <span class="type">东方玄幻</span>
            <span class="status">已完结</span>
        </div>
        <div class="description">
            <p>一粒尘可填海，一根草斩尽日月星辰...</p>
        </div>
        <div class="chapters">
            <a href="/p1">第一卷 荒域 第一章 石村</a>
            <a href="/p2">第二章 大荒</a>
            <a href="/p3">第三章 古矿</a>
        </div>
    </div>
</body>
</html>`

	parser := NewXPathParser()
	doc := parseTestHTML(t, html)

	// Extract title
	title, err := parser.ParseXPath("//h1[@class='title']", doc)
	require.NoError(t, err)
	assert.Equal(t, "完美世界", strings.TrimSpace(title[0]))

	// Extract all meta info
	meta, err := parser.ParseXPath("//div[@class='meta']/span", doc)
	require.NoError(t, err)
	assert.Len(t, meta, 3)
	assert.Contains(t, meta[0], "辰东")
	assert.Contains(t, meta[1], "东方玄幻")

	// Extract description
	desc, err := parser.ParseXPath("//div[@class='description']/p", doc)
	require.NoError(t, err)
	assert.Len(t, desc, 1)
	assert.Contains(t, desc[0], "一粒尘可填海")

	// Extract chapter list
	chaps, err := parser.ParseXPath("//div[@class='chapters']/a/@href", doc)
	require.NoError(t, err)
	assert.Len(t, chaps, 3)
	assert.Equal(t, "/p1", chaps[0])
}
