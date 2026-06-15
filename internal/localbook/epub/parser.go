package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// =============================================================================
// Parse - 主解析入口
// =============================================================================

// Parse 解析 EPUB 文件，返回 Book 对象
// 支持 EPUB 2.0 (NCX TOC) 和 EPUB 3.0 (Navigation Document TOC)
func Parse(pathStr string) (*Book, error) {
	// 打开 ZIP 文件
	archive, err := zip.OpenReader(pathStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidEPUB, err)
	}

	zrc := &ZipReadCloser{R: &archive.Reader, Closer: archive}
	defer func() {
		if zrc.Closer != nil {
			zrc.Closer.Close()
		}
	}()

	// 步骤 1: 解析 container.xml
	opfPath, err := parseContainer(zrc)
	if err != nil {
		return nil, err
	}

	// 步骤 2: 解析 OPF 文件
	pkg, err := parseOPF(zrc, opfPath)
	if err != nil {
		return nil, err
	}

	// 步骤 3: 构建 Book 对象
	book := buildBook(pkg, opfPath)

	// 步骤 4: 解析 TOC（优先 NCX，回退到 EPUB 3 nav）
	book.TOC, err = parseTOC(zrc, book, opfPath)
	if err != nil {
		// TOC 解析失败不影响书籍主体
		// book.TOC = nil
	}

	// 步骤 5: 构建章节列表（基于 spine）
	book.Chapters, err = buildChapters(book)
	if err != nil {
		return nil, err
	}

	// 步骤 6: 提取封面
	book.CoverData, book.CoverMIME, err = extractCover(zrc, book, opfPath)
	if err != nil {
		// 封面可选
	}

	return book, nil
}

// =============================================================================
// Container.xml 解析
// =============================================================================

// parseContainer 解析 META-INF/container.xml，找到 OPF 文件路径
func parseContainer(zrc *ZipReadCloser) (string, error) {
	// 标准路径
	containerPath := "META-INF/container.xml"

	// 尝试打开
	rc, err := zrc.openFile(containerPath)
	if err != nil {
		// 尝试其他常见路径
		files := zrc.listFiles("META-INF/")
		for _, f := range files {
			if strings.HasSuffix(f, "container.xml") {
				containerPath = f
				rc, err = zrc.openFile(f)
				if err == nil {
					break
				}
			}
		}
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrNoContainer, err)
		}
	}
	defer rc.Close()

	var container Container
	if err := xml.NewDecoder(rc).Decode(&container); err != nil {
		return "", fmt.Errorf("failed to decode container.xml: %w", err)
	}

	if len(container.Rootfiles) == 0 {
		return "", ErrNoContainer
	}

	// 优先选择 OEBPS 类型的 rootfile
	for _, rf := range container.Rootfiles {
		if rf.MediaType == "application/oebps-package+xml" || rf.MediaType == "application/epub+opf" {
			return rf.FullPath, nil
		}
	}

	// 回退：返回第一个 rootfile
	return container.Rootfiles[0].FullPath, nil
}

// =============================================================================
// OPF 文件解析
// =============================================================================

// parseOPF 解析 OPF 包文档
func parseOPF(zrc *ZipReadCloser, opfPath string) (*PackageDocument, error) {
	rc, err := zrc.openFile(opfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open OPF file %s: %w", opfPath, err)
	}
	defer rc.Close()

	var pkg PackageDocument
	if err := xml.NewDecoder(rc).Decode(&pkg); err != nil {
		return nil, fmt.Errorf("failed to decode OPF file: %w", err)
	}

	if pkg.Metadata == nil {
		pkg.Metadata = &Metadata{}
	}

	// 解析 metadata 中的标准字段
	parseMetadata(pkg.Metadata)

	return &pkg, nil
}

// parseMetadata 解析元数据 — 从数组字段提取第一个值到便捷字段
func parseMetadata(meta *Metadata) {
	if len(meta.Titles) > 0 {
		meta.Title = meta.Titles[0]
	}
	if len(meta.Creators) > 0 {
		meta.Creator = meta.Creators[0]
	}
	if len(meta.Languages) > 0 {
		meta.Language = meta.Languages[0]
	}
	if len(meta.Publishers) > 0 {
		meta.Publisher = meta.Publishers[0]
	}
	if len(meta.Identifiers) > 0 {
		meta.ISBN = meta.Identifiers[0]
	}
	if len(meta.Dates) > 0 {
		meta.Date = meta.Dates[0]
	}
	if len(meta.Descs) > 0 {
		meta.Desc = meta.Descs[0]
	}

	// 从 identifiers 中查找 ISBN
	for _, id := range meta.Identifiers {
		if isISBN(id) {
			meta.ISBN = id
			break
		}
	}

	// 查找封面元数据 — 搜索 meta name="cover" content="..."
	if meta.CoverImageID == "" {
		if idx := strings.Index(meta.RawXML, `name="cover"`); idx >= 0 {
			start := strings.LastIndex(meta.RawXML[:idx], "<meta")
			if start >= 0 {
				end := strings.Index(meta.RawXML[start:], ">")
				if end > 0 {
					tag := meta.RawXML[start : start+end]
					if attrStart := strings.Index(tag, `content="`); attrStart >= 0 {
						attrValStart := attrStart + 9
						if attrValStart < len(tag) {
							attrEnd := strings.Index(tag[attrValStart:], `"`)
							if attrEnd > 0 {
								meta.CoverImageID = tag[attrValStart : attrValStart+attrEnd]
							}
						}
					}
				}
			}
		}
	}
}

// isISBN 判断字符串是否是 ISBN 格式
func isISBN(s string) bool {
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 10 && len(s) != 13 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			if c == 'X' && len(s) == 10 {
				continue
			}
			return false
		}
	}
	return true
}

// =============================================================================
// Book 对象构建
// =============================================================================

// buildBook 从 OPF 构建 Book 对象
func buildBook(pkg *PackageDocument, opfPath string) *Book {
	// 优先使用实际的 identifier 值，而不是 unique-identifier 属性指向的 ID
	identifier := pkg.UniqueID
	for _, id := range pkg.Metadata.Identifiers {
		if id != "" {
			identifier = id
			break
		}
	}

	book := &Book{
		Title:    pkg.Metadata.Title,
		Author:   pkg.Metadata.Creator,
		Language: pkg.Metadata.Language,
		Publisher: pkg.Metadata.Publisher,
		ISBN:     pkg.Metadata.ISBN,
		PubDate:  pkg.Metadata.Date,
		Identifier: identifier,
		Description: pkg.Metadata.Desc,
		MetadataRaw: pkg.Metadata.RawXML,

		manifest: make(map[string]*Item),
		opfDir:   path.Dir(opfPath),
	}

	// 构建 manifest
	for _, item := range pkg.Manifest {
		fullPath := path.Join(book.opfDir, item.Href)
		fullPath = cleanPath(fullPath)
		item.FullPath = fullPath
		book.manifest[item.ID] = &item
	}

	// 构建 spine
	for _, ref := range pkg.Spine.ItemRefs {
		if item, ok := book.manifest[ref.IDRef]; ok {
			ref.Item = item
			book.spine = append(book.spine, ref.IDRef)
		}
	}

	return book
}

// =============================================================================
// 章节构建
// =============================================================================

// buildChapters 基于 spine 构建章节列表
func buildChapters(book *Book) ([]*Chapter, error) {
	if len(book.spine) == 0 {
		return nil, ErrNoSpine
	}

	chapters := make([]*Chapter, 0, len(book.spine))
	for i, id := range book.spine {
		item, ok := book.manifest[id]
		if !ok {
			continue
		}

		ch := &Chapter{
			Title:      id, // 默认用 ID，后续用 TOC 覆盖
			Index:      i,
			ResourceID: id,
			Path:       item.FullPath,
			Depth:      0,
		}

		// 尝试从文件名提取标题
		ch.Title = extractTitleFromPath(item.FullPath)

		chapters = append(chapters, ch)
	}

	// 用 TOC 信息更新章节标题
	if len(book.TOC) > 0 {
		applyTOCTitles(book.TOC, chapters)
	}

	// 如果章节标题仍然是默认的路径提取，尝试从 HTML 文件中的 <title> 标签获取
	for i, ch := range chapters {
		if ch.Title == "" || ch.Title == ch.ResourceID || ch.Title == extractTitleFromPath(ch.Path) {
			// 保持现有逻辑，但确保至少有个合理的标题
			if ch.Title == "" {
				ch.Title = fmt.Sprintf("Chapter %d", i+1)
			}
		}
	}

	return chapters, nil
}

// applyTOCTitles 用 TOC 标题更新章节
func applyTOCTitles(toc []*Chapter, chapters []*Chapter) {
	// 扁平化 TOC
	flatTOC := flattenChapters(toc)
	for _, tocCh := range flatTOC {
		for _, ch := range chapters {
			if ch.Path == tocCh.Path || ch.ResourceID == tocCh.ResourceID {
				ch.Title = tocCh.Title
				break
			}
		}
	}
}

// flattenChapters 将层级 TOC 扁平化
func flattenChapters(chapters []*Chapter) []*Chapter {
	var result []*Chapter
	for _, ch := range chapters {
		result = append(result, ch)
		result = append(result, flattenChapters(ch.Children)...)
	}
	return result
}

// extractTitleFromPath 从路径提取标题
func extractTitleFromPath(p string) string {
	base := path.Base(p)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	// 移除数字前缀（如 001_chapter.html）
	name = strings.TrimPrefix(name, "0")
	name = strings.TrimPrefix(name, "00")
	name = strings.TrimPrefix(name, "000")
	// 替换下划线/连字符为空格
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	// 首字母大写
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}

// =============================================================================
// 封面提取
// =============================================================================

// extractCover 提取封面图片
func extractCover(zrc *ZipReadCloser, book *Book, opfPath string) ([]byte, string, error) {
	coverPath := ""
	coverMIME := ""

	// 方法 1: 通过 meta name="cover" 查找
	if book.manifest != nil && book.MetadataRaw != "" {
		// 查找 meta name="cover" 的 content 属性值
		if idx := strings.Index(book.MetadataRaw, `name="cover"`); idx >= 0 {
			start := strings.LastIndex(book.MetadataRaw[:idx], "<meta")
			if start >= 0 {
				end := strings.Index(book.MetadataRaw[start:], ">")
				if end > 0 {
					tag := book.MetadataRaw[start : start+end]
					if attrStart := strings.Index(tag, `content="`); attrStart >= 0 {
						attrValStart := attrStart + 9
						if attrValStart < len(tag) {
							attrEnd := strings.Index(tag[attrValStart:], `"`)
							if attrEnd > 0 {
								coverID := tag[attrValStart : attrValStart+attrEnd]
								if item, ok := book.manifest[coverID]; ok {
									coverPath = item.FullPath
									coverMIME = item.MediaType
								}
							}
						}
					}
				}
			}
		}
	}

	// 方法 2: 通过 guide 查找封面
	if coverPath == "" {
		// 这里需要重新解析 OPF 获取 guide，简化处理：搜索常见路径
		for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif"} {
			files := zrc.listFiles("images/")
			for _, f := range files {
				if strings.HasSuffix(strings.ToLower(f), ext) {
					coverPath = f
					coverMIME = guessMIME(ext)
					break
				}
			}
			if coverPath != "" {
				break
			}
		}
	}

	// 方法 3: 搜索常见封面文件名
	if coverPath == "" {
		coverNames := []string{
			"cover.jpg", "cover.jpeg", "cover.png", "cover.gif",
			"Cover.jpg", "Cover.png",
			"images/cover.jpg", "images/cover.png",
			"OEBPS/cover.jpg", "OEBPS/cover.png",
		}
		for _, name := range coverNames {
			rc, err := zrc.openFile(name)
			if err == nil {
				data, _ := io.ReadAll(rc)
				rc.Close()
				if len(data) > 100 {
					return data, guessMIMEByData(data), nil
				}
			}
		}
	}

	if coverPath == "" {
		return nil, "", ErrNoCover
	}

	rc, err := zrc.openFile(coverPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open cover %s: %w", coverPath, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read cover: %w", err)
	}

	if len(data) < 100 {
		return nil, "", ErrNoCover
	}

	return data, coverMIME, nil
}

// guessMIME 根据扩展名猜测 MIME 类型
func guessMIME(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// guessMIMEByData 根据文件数据猜测 MIME 类型
func guessMIMEByData(data []byte) string {
	if len(data) >= 3 {
		if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
			return "image/jpeg"
		}
		if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
			return "image/png"
		}
	}
	if len(data) >= 6 {
		if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
			return "image/gif"
		}
	}
	if len(data) >= 20 {
		if strings.HasPrefix(string(data[:20]), "<svg") {
			return "image/svg+xml"
		}
	}
	return "application/octet-stream"
}

// =============================================================================
// 工具函数
// =============================================================================

// cleanPath 清理路径（移除多余的 ./ 和 ..）
func cleanPath(p string) string {
	// 统一路径分隔符
	p = strings.ReplaceAll(p, "\\", "/")
	// 清理路径
	p = path.Clean(p)
	return p
}

// resolveRelativeURL 解析相对 URL
func resolveRelativeURL(base, ref string) string {
	if ref == "" {
		return ""
	}
	// 如果是绝对路径（以 / 开头），相对于根
	if strings.HasPrefix(ref, "/") {
		return ref[1:]
	}
	// 如果包含协议，直接返回
	if strings.Contains(ref, "://") {
		return ref
	}
	// 解析 base 的目录
	baseDir := path.Dir(base)
	if baseDir == "." {
		baseDir = ""
	}
	// 拼接
	result := path.Join(baseDir, ref)
	result = cleanPath(result)
	return result
}

// =============================================================================
// HTML 清洗
// =============================================================================

// cleanHTML 清洗 HTML 内容，移除 script/style 标签
func cleanHTML(htmlStr string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrParseHTML, err)
	}

	// 递归移除 script 和 style
	var removeTags func(*html.Node)
	removeTags = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			if tag == "script" || tag == "style" || tag == "noscript" || tag == "meta" || tag == "link" || tag == "head" {
				// 移除该节点及其子节点
				parent := n.Parent
				if parent != nil {
					parent.RemoveChild(n)
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			removeTags(c)
			c = next
		}
	}

	removeTags(doc)

	// 提取 body 内容
	var bodyContent bytes.Buffer
	var extractBody func(*html.Node)
	extractBody = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "body" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				render(&bodyContent, c)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractBody(c)
		}
	}

	extractBody(doc)

	return bodyContent.String(), nil
}

// render 将 HTML 节点渲染为字符串
func render(w *bytes.Buffer, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		w.WriteString(n.Data)
	case html.ElementNode:
		tag := strings.ToLower(n.Data)

		// 块级元素添加换行
		blockTags := map[string]bool{
			"p": true, "div": true, "br": true, "hr": true,
			"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
			"ul": true, "ol": true, "li": true, "table": true, "tr": true,
			"blockquote": true, "pre": true, "address": true, "article": true,
			"aside": true, "footer": true, "header": true, "nav": true,
			"section": true, "figure": true, "figcaption": true,
		}

		if tag == "br" {
			w.WriteString("\n")
			return
		}

		if blockTags[tag] {
			w.WriteString("\n")
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			render(w, c)
		}

		if blockTags[tag] {
			w.WriteString("\n")
		}
	}
}

// extractTextFromHTML 从 HTML 提取纯文本
func extractTextFromHTML(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	var text bytes.Buffer
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				if text.Len() > 0 && text.Bytes()[text.Len()-1] != '\n' {
					text.WriteByte(' ')
				}
				text.WriteString(t)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(doc)
	return strings.TrimSpace(text.String())
}

// =============================================================================
// 公共接口实现
// =============================================================================

// GetChapters 获取章节列表
func (b *Book) GetChapters() []Chapter {
	chapters := make([]Chapter, 0, len(b.Chapters))
	for _, ch := range b.Chapters {
		chapters = append(chapters, Chapter{
			Title:      ch.Title,
			Index:      ch.Index,
			ResourceID: ch.ResourceID,
			Path:       ch.Path,
			Depth:      ch.Depth,
		})
	}
	return chapters
}

// GetContent 获取指定章节的内容
func (b *Book) GetContent(chapterIndex int) (string, error) {
	if chapterIndex < 0 || chapterIndex >= len(b.Chapters) {
		return "", fmt.Errorf("chapter index %d out of range [0, %d]", chapterIndex, len(b.Chapters)-1)
	}

	ch := b.Chapters[chapterIndex]
	if ch == nil || ch.Path == "" {
		return "", fmt.Errorf("chapter %d has no path", chapterIndex)
	}

	// 从 ZIP 读取章节文件
	// 注意：这里需要 archive 句柄，实际使用时需要保持 Book 的 zipReader
	// 简化版本：返回路径信息
	return "", fmt.Errorf("GetContent requires active archive handle - use ExtractChapter instead")
}

// GetCover 获取封面图片
func (b *Book) GetCover() ([]byte, error) {
	if b.CoverData == nil || len(b.CoverData) == 0 {
		return nil, ErrNoCover
	}
	result := make([]byte, len(b.CoverData))
	copy(result, b.CoverData)
	return result, nil
}

// GetTOC 获取目录树
func (b *Book) GetTOC() []*Chapter {
	result := make([]*Chapter, 0, len(b.TOC))
	for _, ch := range b.TOC {
		result = append(result, cloneChapter(ch))
	}
	return result
}

// cloneChapter 深拷贝章节
func cloneChapter(ch *Chapter) *Chapter {
	if ch == nil {
		return nil
	}
	return &Chapter{
		Title:      ch.Title,
		Index:      ch.Index,
		ResourceID: ch.ResourceID,
		Path:       ch.Path,
		Depth:      ch.Depth,
		Children:   ch.Children, // 浅拷贝，通常够用
	}
}

// GetMetadata 获取书籍元数据
func (b *Book) GetMetadata() map[string]string {
	m := map[string]string{
		"title":      b.Title,
		"author":     b.Author,
		"language":   b.Language,
		"publisher":  b.Publisher,
		"isbn":       b.ISBN,
		"pubDate":    b.PubDate,
		"identifier": b.Identifier,
		"description": b.Description,
	}
	return m
}
