// Package epub 实现 EPUB 电子书解析器
// 支持 EPUB 2.0 (NCX) 和 EPUB 3.0 (Navigation Document)
package epub

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"path"
	"strings"
)

// =============================================================================
// 错误定义
// =============================================================================

var (
	ErrNotFound    = errors.New("file not found")
	ErrInvalidEPUB = errors.New("invalid EPUB file")
	ErrNoContainer = errors.New("no container.xml found")
	ErrNoOPF       = errors.New("no OPF file found")
	ErrNoSpine     = errors.New("no spine entries")
	ErrNoTOC       = errors.New("no TOC available")
	ErrNoCover     = errors.New("no cover image found")
	ErrParseHTML   = errors.New("failed to parse HTML content")
)

// =============================================================================
// 公共数据结构
// =============================================================================

// Chapter 章节信息
type Chapter struct {
	// 章节标题
	Title string `json:"title"`

	// 章节在 spine 中的索引
	Index int `json:"index"`

	// 章节对应的 OPF 资源 ID（manifest item id）
	ResourceID string `json:"resourceId"`

	// 章节对应的文件路径（相对于 OPF 目录）
	Path string `json:"path"`

	// 父章节（用于层级 TOC）
	Parent *Chapter `json:"-"`

	// 子章节
	Children []*Chapter `json:"children,omitempty"`

	// 深度（0=根章节）
	Depth int `json:"depth"`
}

// Book 解析后的书籍信息
type Book struct {
	// 书名
	Title string `json:"title"`

	// 作者
	Author string `json:"author"`

	// 语言
	Language string `json:"language"`

	// 出版社
	Publisher string `json:"publisher"`

	// ISBN
	ISBN string `json:"isbn"`

	// 出版日期
	PubDate string `json:"pubDate"`

	// 唯一标识符（通常是 UUID 或 ISBN）
	Identifier string `json:"identifier"`

	// 描述/简介
	Description string `json:"description"`

	// 封面图片数据（JPEG/PNG）
	CoverData []byte `json:"-"`

	// 封面 MIME 类型
	CoverMIME string `json:"-"`

	// 章节列表（按阅读顺序）
	Chapters []*Chapter `json:"chapters"`

	// TOC 目录树（可能有多级）
	TOC []*Chapter `json:"toc,omitempty"`

	// manifest: id -> Item
	manifest map[string]*Item

	// spine: 阅读顺序（itemref 列表）
	spine []string // itemref idref 列表

	// OPF 文件所在目录（相对于 ZIP 根）
	opfDir string

	// ZIP 归档句柄
	zipReader *ZipReadCloser

	// 元数据（保留原始 XML）
	MetadataRaw string `json:"-"`
}

// Item manifest 中的资源项
type Item struct {
	// 唯一标识符
	ID string `xml:"id,attr"`

	// 文件路径（相对于 OPF 目录）
	Href string `xml:"href,attr"`

	// MIME 类型
	MediaType string `xml:"media-type,attr"`

	// 可选：媒体 Overlay ID（EPUB 3 有声书）
	MediaOverlay string `xml:"media-overlay,attr"`

	// 可选：属性（EPUB 3）
	Properties []string `xml:"properties,attr"`

	// 完整路径（OPF 目录 + Href）
	FullPath string `json:"-"`
}

// ItemRef spine 中的章节引用
type ItemRef struct {
	// 指向 manifest item 的 idref
	IDRef string `xml:"idref,attr"`

	// 是否线性（linear="yes" 表示正文，"no" 表示附录/索引等）
	Linear string `xml:"linear,attr"`

	// 对应 manifest item
	Item *Item `json:"-"`
}

// =============================================================================
// OPF 包文档结构 (OPF 2.0 / 3.0)
// =============================================================================

// PackageDocument OPF 包文档根元素
type PackageDocument struct {
	XMLName  xml.Name  `xml:"package"`
	Version  string    `xml:"version,attr"`
	UniqueID string    `xml:"unique-identifier,attr"`
	Lang     string    `xml:"xml:lang,attr"`

	Metadata *Metadata `xml:"metadata"`

	Manifest []Item `xml:"manifest>item"`

	Spine struct {
		Toc      string    `xml:"toc,attr"` // NCX ID
		ItemRefs []ItemRef `xml:"itemref"`
	} `xml:"spine"`

	// EPUB 3 指南
	Guide []GuideReference `xml:"guide>reference"`
}

// GuideReference EPUB 指南引用（用于定位封面、TOC 等）
type GuideReference struct {
	Type  string `xml:"type,attr"` // "cover", "toc", "text" 等
	Title string `xml:"title,attr"`
	Href  string `xml:"href,attr"`
}

// Metadata 书籍元数据
type Metadata struct {
	// RawXML 保留原始 XML 用于封面 meta 等特殊解析
	RawXML string `xml:",innerxml"`

	// 标准字段 — 使用 slice 接收多个同名元素
	// 注意: Go xml 解码器对命名空间前缀的处理：
	// 标签中的 "dc:title" 会被解析为 Local="title", Space="http://purl.org/dc/elements/1.1/"
	// 使用 > 语法可以匹配任意命名空间前缀
	Titles      []string `xml:"title"`
	Creators    []string `xml:"creator"`
	Languages   []string `xml:"language"`
	Publishers  []string `xml:"publisher"`
	Identifiers []string `xml:"identifier"`
	Dates       []string `xml:"date"`
	Descs       []string `xml:"description"`

	// 便捷访问：从数组提取的第一个值
	Title    string `xml:"-"`
	Creator  string `xml:"-"`
	Language string `xml:"-"`
	Publisher string `xml:"-"`
	ISBN     string `xml:"-"`
	Date     string `xml:"-"`
	Desc     string `xml:"-"`

	// 封面元数据（从 RawXML 解析）
	CoverImageID string `xml:"-"`
}

// =============================================================================
// Container.xml 结构
// =============================================================================

// Container 根容器
type Container struct {
	XMLName   xml.Name    `xml:"container"`
	Version   string      `xml:"version,attr"`
	Rootfiles []Rootfile  `xml:"rootfiles>rootfile"`
}

// Rootfile 根文件（指向 OPF）
type Rootfile struct {
	MediaType string `xml:"media-type,attr"`
	FullPath  string `xml:"full-path,attr"`
}

// =============================================================================
// NCX (EPUB 2 TOC) 结构
// =============================================================================

// NCXDocument NCX 导航文件
type NCXDocument struct {
	XMLName xml.Name `xml:"ncx"`
	Version string   `xml:"version,attr"`
	Lang    string   `xml:"xml:lang,attr"`

	Head  *NCXHead `xml:"head"`
	Title string   `xml:"docTitle>text"`

	NavMap *NavMap `xml:"navMap"`
}

// NCXHead NCX 头部
type NCXHead struct {
	Meta []NCXMeta `xml:"meta"`
}

// NCXMeta NCX 元数据
type NCXMeta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

// NavMap NCX 导航地图
type NavMap struct {
	NavPoints []NavPoint `xml:"navPoint"`
}

// NavPoint NCX 导航点（可嵌套）
type NavPoint struct {
	ID        string      `xml:"id,attr"`
	Label     NCXLabel    `xml:"navLabel"`
	Content   NCXContent  `xml:"content"`
	NavPoints []NavPoint  `xml:"navPoint"`
	PlayOrder int         `xml:"playOrder,attr"`
}

// NCXLabel 导航点标签（标题）
type NCXLabel struct {
	Text string `xml:"text"`
}

// NCXContent 导航点内容（指向章节文件）
type NCXContent struct {
	Src string `xml:"src,attr"`
}

// =============================================================================
// EPUB 3 Navigation Document 结构
// =============================================================================

// NavigationDocument EPUB 3 导航文档（nav 元素）
type NavigationDocument struct {
	XMLName xml.Name `xml:"nav"`
	Lang    string   `xml:"xml:lang,attr"`
	Type    string   `xml:"epub:type,attr"` // "toc", "page-list", "landmarks"

	// 标题（用于 landmarks）
	Title string `xml:"h1"`

	// EPUB 3 列表导航
	NavList *NavList `xml:"ol>li"`

	// 普通 HTML 列表（备用解析）
	NavItems []NavItem `xml:"li"`
}

// NavList 导航列表项（递归结构）
type NavList struct {
	Anchor   *NavAnchor  `xml:"a"`
	Content  *NavContent `xml:"span"`
	Children []*NavList  `xml:"ol>li"`
}

// NavAnchor 导航锚点（带链接）
type NavAnchor struct {
	Href string `xml:"href,attr"`
	Text string `xml:",chardata"`
}

// NavContent 导航内容（无链接的标题）
type NavContent struct {
	Text     string     `xml:",chardata"`
	Children []*NavList `xml:"ol>li"`
}

// NavItem 简化导航项
type NavItem struct {
	Anchor     string    `xml:"a>href,attr"`
	AnchorText string    `xml:"a>text"`
	Content    string    `xml:"span>text"`
	Children   []NavItem `xml:"li"`
}

// =============================================================================
// ZIP 读取器封装
// =============================================================================

// ZipReadCloser 封装 zip.Reader，支持按需读取文件
type ZipReadCloser struct {
	R *zip.Reader
	io.Closer
}

// openFile 从 ZIP 中打开文件，返回 io.ReadCloser
func (z *ZipReadCloser) openFile(name string) (io.ReadCloser, error) {
	for _, f := range z.R.File {
		if f.Name == name || f.Name == path.Join(z.R.File[0].Name, name) {
			return f.Open()
		}
	}
	// 尝试相对路径
	for _, f := range z.R.File {
		relPath := strings.TrimPrefix(f.Name, z.R.File[0].Name)
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath == name {
			return f.Open()
		}
	}
	return nil, &PathError{Op: "open", Path: name, Err: ErrNotFound}
}

// listFiles 列出匹配前缀的文件
func (z *ZipReadCloser) listFiles(prefix string) []string {
	var files []string
	for _, f := range z.R.File {
		if strings.HasPrefix(f.Name, prefix) {
			files = append(files, f.Name)
		}
	}
	return files
}

// PathError 路径错误
type PathError struct {
	Op   string
	Path string
	Err  error
}

func (e *PathError) Error() string {
	return e.Op + " " + e.Path + ": " + e.Err.Error()
}

func (e *PathError) Unwrap() error {
	return e.Err
}
