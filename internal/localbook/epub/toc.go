package epub

import (
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// =============================================================================
// parseTOC - TOC 解析入口
// =============================================================================

// parseTOC 解析 TOC，优先 NCX (EPUB 2)，回退到 EPUB 3 Navigation Document
func parseTOC(zrc *ZipReadCloser, book *Book, opfPath string) ([]*Chapter, error) {
	// 方法 1: 尝试解析 NCX (EPUB 2)
	toc, err := parseNCX(zrc, book, opfPath)
	if err == nil && len(toc) > 0 {
		return toc, nil
	}

	// 方法 2: 尝试解析 EPUB 3 Navigation Document
	toc, err = parseNavigation(zrc, book, opfPath)
	if err == nil && len(toc) > 0 {
		return toc, nil
	}

	// 方法 3: 从 spine 构建基础 TOC
	toc = buildTOCFromSpine(book)
	if len(toc) > 0 {
		return toc, nil
	}

	return nil, ErrNoTOC
}

// =============================================================================
// NCX (EPUB 2 TOC) 解析
// =============================================================================

// parseNCX 解析 NCX 导航文件
func parseNCX(zrc *ZipReadCloser, book *Book, opfPath string) ([]*Chapter, error) {
	ncxPath := findNCX(zrc, book, opfPath)
	if ncxPath == "" {
		return nil, nil
	}

	rc, err := zrc.openFile(ncxPath)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var ncx NCXDocument
	if err := xml.NewDecoder(rc).Decode(&ncx); err != nil {
		return nil, fmt.Errorf("failed to decode NCX: %w", err)
	}

	if ncx.NavMap == nil || len(ncx.NavMap.NavPoints) == 0 {
		return nil, nil
	}

	// 递归解析 navPoint
	return parseNavPoints(ncx.NavMap.NavPoints, book, 0), nil
}

// findNCX 查找 NCX 文件路径
func findNCX(zrc *ZipReadCloser, book *Book, opfPath string) string {
	// 从 spine 的 toc 属性查找
	// 注意：这里需要重新解析 OPF 获取 toc 属性
	// 简化：搜索常见路径

	// 方法 1: 从 OPF 的 spine.toc 属性
	// (需要在 parser.go 中传递 tocID)

	// 方法 2: 搜索常见 NCX 路径
	ncxPaths := []string{
		"toc.ncx",
		"nav.ncx",
		"OEBPS/toc.ncx",
		"OEBPS/nav.ncx",
		"OEBPS/toc.html",
		"toc.html",
		"nav.html",
	}

	for _, p := range ncxPaths {
		rc, err := zrc.openFile(p)
		if err == nil {
			// 验证是否是 NCX
			buf := make([]byte, 500)
			n, _ := rc.Read(buf)
			rc.Close()
			content := string(buf[:n])
			if strings.Contains(content, "<ncx") || strings.Contains(content, "NCX") {
				return p
			}
		}
	}

	// 方法 3: 列出所有 .ncx 文件
	files := zrc.listFiles("")
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f), ".ncx") {
			return f
		}
	}

	return ""
}

// parseNavPoints 递归解析 NCX navPoint
func parseNavPoints(nps []NavPoint, book *Book, depth int) []*Chapter {
	chapters := make([]*Chapter, 0, len(nps))

	for _, np := range nps {
		ch := &Chapter{
			Title:  np.Label.Text,
			Index:  np.PlayOrder,
			Depth:  depth,
			Parent: nil,
		}

		// 解析 content src
		if np.Content.Src != "" {
			ch.Path = resolveRelativeURL(book.opfDir, np.Content.Src)
			// 尝试从 manifest 找到对应项
			if item, ok := book.manifest[ch.ResourceID]; ok {
				ch.Path = item.FullPath
			}
		}

		// 解析子 navPoint
		if len(np.NavPoints) > 0 {
			ch.Children = parseNavPoints(np.NavPoints, book, depth+1)
			// 设置父引用
			for _, child := range ch.Children {
				child.Parent = ch
			}
		}

		chapters = append(chapters, ch)
	}

	return chapters
}

// =============================================================================
// EPUB 3 Navigation Document 解析
// =============================================================================

// parseNavigation 解析 EPUB 3 Navigation Document (nav 元素)
func parseNavigation(zrc *ZipReadCloser, book *Book, opfPath string) ([]*Chapter, error) {
	navPath := findNavigation(zrc, book, opfPath)
	if navPath == "" {
		return nil, nil
	}

	rc, err := zrc.openFile(navPath)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	// 读取完整内容
	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	// 尝试解析为 XML
	var nav NavigationDocument
	if err := xml.Unmarshal(content, &nav); err == nil && len(nav.NavItems) > 0 {
		return parseNavItems(nav.NavItems, book, 0), nil
	}

	// 回退：使用 HTML 解析
	return parseNavHTML(content, book, navPath), nil
}

// findNavigation 查找 Navigation Document 路径
func findNavigation(zrc *ZipReadCloser, book *Book, opfPath string) string {
	// 方法 1: 从 OPF guide 查找
	// (需要重新解析 OPF)

	// 方法 2: 搜索常见路径
	navPaths := []string{
		"nav.xhtml",
		"nav.html",
		"OEBPS/nav.xhtml",
		"OEBPS/nav.html",
		"toc.xhtml",
		"toc.html",
	}

	for _, p := range navPaths {
		rc, err := zrc.openFile(p)
		if err == nil {
			buf := make([]byte, 500)
			n, _ := rc.Read(buf)
			rc.Close()
			content := string(buf[:n])
			if strings.Contains(content, `<nav`) || strings.Contains(content, `<NAV`) {
				return p
			}
		}
	}

	// 方法 3: 列出所有 nav/toc 相关文件
	files := zrc.listFiles("")
	for _, f := range files {
		base := strings.ToLower(path.Base(f))
		if strings.HasPrefix(base, "nav") || strings.HasPrefix(base, "toc") {
			if strings.HasSuffix(f, ".xhtml") || strings.HasSuffix(f, ".html") {
				return f
			}
		}
	}

	return ""
}

// parseNavItems 解析 XML 格式的导航项
func parseNavItems(items []NavItem, book *Book, depth int) []*Chapter {
	chapters := make([]*Chapter, 0, len(items))

	for _, item := range items {
		ch := &Chapter{
			Index: depth * 100 + len(chapters),
			Depth: depth,
		}

		// 优先使用 span 文本（EPUB 3 常见模式）
		if item.Content != "" {
			ch.Title = item.Content
		} else if item.AnchorText != "" {
			ch.Title = item.AnchorText
		} else {
			ch.Title = fmt.Sprintf("Chapter %d", len(chapters)+1)
		}

		// 解析链接
		if item.Anchor != "" {
			ch.Path = resolveRelativeURL(book.opfDir, item.Anchor)
		}

		// 解析子项
		if len(item.Children) > 0 {
			ch.Children = parseNavItems(item.Children, book, depth+1)
			for _, child := range ch.Children {
				child.Parent = ch
			}
		}

		chapters = append(chapters, ch)
	}

	return chapters
}

// parseNavHTML 使用 HTML 解析器解析导航文档
func parseNavHTML(content []byte, book *Book, navPath string) []*Chapter {
	doc, err := html.Parse(strings.NewReader(string(content)))
	if err != nil {
		return nil
	}

	// 查找 nav 元素
	var navNode *html.Node
	var findNav func(*html.Node)
	findNav = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "nav" {
			navNode = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findNav(c)
			if navNode != nil {
				return
			}
		}
	}
	findNav(doc)

	if navNode == nil {
		return nil
	}

	// 查找 ol/li 结构
	return parseNavList(navNode, book, 0)
}

// parseNavList 递归解析导航列表
func parseNavList(node *html.Node, book *Book, depth int) []*Chapter {
	var chapters []*Chapter

	// 查找所有 li 元素
	var findLI func(*html.Node, func(*html.Node))
	findLI = func(n *html.Node, fn func(*html.Node)) {
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "li" {
			fn(n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findLI(c, fn)
		}
	}

	findLI(node, func(li *html.Node) {
		ch := &Chapter{
			Index: len(chapters),
			Depth: depth,
		}

		// 查找 a 元素
		var anchor *html.Node
		var findA func(*html.Node)
		findA = func(n *html.Node) {
			if n.Type == html.ElementNode && strings.ToLower(n.Data) == "a" {
				anchor = n
				return
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				findA(c)
				if anchor != nil {
					return
				}
			}
		}
		findA(li)

		if anchor != nil {
			// 提取标题
			ch.Title = tocExtractText(anchor)

			// 提取 href
			for _, attr := range anchor.Attr {
				if attr.Key == "href" {
					ch.Path = resolveRelativeURL(book.opfDir, attr.Val)
					break
				}
			}
		} else {
			// 没有 a 标签，用 span 或文本
			ch.Title = extractTextFromNode(li)
		}

		// 查找嵌套的 ol/li
		var nestedOL *html.Node
		for c := li.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && strings.ToLower(c.Data) == "ol" {
				nestedOL = c
				break
			}
		}

		if nestedOL != nil {
			ch.Children = parseNavList(nestedOL, book, depth+1)
			for _, child := range ch.Children {
				child.Parent = ch
			}
		}

		if ch.Title != "" {
			chapters = append(chapters, ch)
		}
	})

	return chapters
}

// tocExtractText 从节点提取文本（TOC专用）
func tocExtractText(n *html.Node) string {
	var text strings.Builder
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				text.WriteString(t)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(n)
	return strings.TrimSpace(text.String())
}

// =============================================================================
// 从 Spine 构建基础 TOC
// =============================================================================

// buildTOCFromSpine 当没有 NCX 或 nav 时，从 spine 构建基础 TOC
func buildTOCFromSpine(book *Book) []*Chapter {
	if len(book.spine) == 0 {
		return nil
	}

	chapters := make([]*Chapter, 0, len(book.spine))
	for i, id := range book.spine {
		item, ok := book.manifest[id]
		if !ok {
			continue
		}

		ch := &Chapter{
			Title:      extractTitleFromPath(item.FullPath),
			Index:      i,
			ResourceID: id,
			Path:       item.FullPath,
			Depth:      0,
		}

		chapters = append(chapters, ch)
	}

	return chapters
}

// =============================================================================
// 章节内容提取辅助
// =============================================================================

// ExtractChapter 提取章节内容（需要活跃的 ZIP 句柄）
// 注意：此函数需要 Book.zipReader 保持打开状态
func (b *Book) ExtractChapter(chapterIndex int, zrc *ZipReadCloser) (string, error) {
	if chapterIndex < 0 || chapterIndex >= len(b.Chapters) {
		return "", fmt.Errorf("chapter index %d out of range [0, %d]", chapterIndex, len(b.Chapters)-1)
	}

	ch := b.Chapters[chapterIndex]
	if ch == nil || ch.Path == "" {
		return "", fmt.Errorf("chapter %d has no path", chapterIndex)
	}

	// 读取章节文件
	rc, err := zrc.openFile(ch.Path)
	if err != nil {
		return "", fmt.Errorf("failed to open chapter %s: %w", ch.Path, err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read chapter %s: %w", ch.Path, err)
	}

	// 清洗 HTML
	cleaned, err := cleanHTML(string(content))
	if err != nil {
		// 回退：返回原始文本
		return extractTextFromHTML(string(content)), nil
	}

	return cleaned, nil
}

// ExtractChapterRaw 提取章节原始 HTML
func (b *Book) ExtractChapterRaw(chapterIndex int, zrc *ZipReadCloser) (string, error) {
	if chapterIndex < 0 || chapterIndex >= len(b.Chapters) {
		return "", fmt.Errorf("chapter index %d out of range [0, %d]", chapterIndex, len(b.Chapters)-1)
	}

	ch := b.Chapters[chapterIndex]
	if ch == nil || ch.Path == "" {
		return "", fmt.Errorf("chapter %d has no path", chapterIndex)
	}

	rc, err := zrc.openFile(ch.Path)
	if err != nil {
		return "", fmt.Errorf("failed to open chapter %s: %w", ch.Path, err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read chapter %s: %w", ch.Path, err)
	}

	return string(content), nil
}
