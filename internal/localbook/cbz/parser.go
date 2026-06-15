package cbz

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// =============================================================================
// 公共类型定义（复用 reader-go 项目中的概念）
// =============================================================================

// Book 书籍信息
type Book struct {
	Name     string            `json:"name"`
	Author   string            `json:"author,omitempty"`
	CoverURL string            `json:"coverUrl,omitempty"`
	Summary  string            `json:"summary,omitempty"`
	Tags     string            `json:"tags,omitempty"`
	// 元数据（扩展字段，存放 ComicInfo 中的额外信息）
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Chapter 章节
type Chapter struct {
	Name    string `json:"name"`
	Index   int    `json:"-"` // 内部索引
	IsCover bool   `json:"-"` // 是否为封面
}

// =============================================================================
// CbzFile CBZ 漫画文件解析器
// =============================================================================

// CbzFile CBZ 文件解析器
type CbzFile struct {
	zipReader *zip.ReadCloser // 原始 zip 文件引用
	archive   *zip.Reader     // zip 归档
	ci        *ComicInfo     // 解析后的 ComicInfo
	images    []imageEntry   // 图片文件列表（按名称排序）
	coverIdx  int            // 封面索引（第一张图片）
	filePath  string         // 原始文件路径
}

// imageEntry 内部图片条目
type imageEntry struct {
	Path    string // zip 内的文件路径
	Name    string // 文件名（不含路径）
	Ext     string // 扩展名（小写，不含点）
	Size    int64  // 文件大小
	IsCover bool   // 是否为封面
}

// 支持的图片扩展名
var supportedExtensions = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".gif":  {},
	".png":  {},
	".bmp":  {},
	".webp": {},
	".svg":  {},
}

// isImageFile 判断是否为支持的图片文件
func isImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := supportedExtensions[ext]
	return ok
}

// stripPathPrefix 去除路径前缀（如 "comic/" 或 "./"）
func stripPathPrefix(p string) string {
	// 去除开头的 ./
	p = strings.TrimPrefix(p, "./")
	// 去除末尾的 /
	p = strings.TrimSuffix(p, "/")
	return p
}

// Parse 解析 CBZ 文件，返回 Book 信息
// 这是主要的入口函数，打开 CBZ 文件并解析所有元数据和图片列表
func Parse(path string) (*Book, error) {
	cbz, err := Open(path)
	if err != nil {
		return nil, err
	}
	return cbz.GetBookInfo(), nil
}

// Open 打开 CBZ 文件并初始化解析器
// 返回 CbzFile 实例，供后续获取章节和图片使用
func Open(path string) (*CbzFile, error) {
	// 打开 zip 文件
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open cbz file: %w", err)
	}

	cbz := &CbzFile{
		archive:  &r.Reader,
		filePath: path,
	}

	// 扫描所有条目，分离图片和 ComicInfo.xml
	var comicInfoPath string
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		name := stripPathPrefix(f.Name)
		lowerName := strings.ToLower(name)

		if lowerName == "comicinfo.xml" {
			comicInfoPath = f.Name
			continue
		}

		if isImageFile(name) {
			cbz.images = append(cbz.images, imageEntry{
				Path: f.Name,
				Name: filepath.Base(name),
				Ext:  strings.ToLower(filepath.Ext(name)),
				Size: int64(f.UncompressedSize64),
			})
		}
	}

	// 解析 ComicInfo.xml
	if comicInfoPath != "" {
		idx := findFileIndex(r.File, comicInfoPath)
		if idx >= 0 {
			f := r.File[idx]
			rc, err := f.Open()
			if err == nil {
				ci, parseErr := ParseComicInfoFromReader(rc)
				if parseErr == nil {
					cbz.ci = ci
				}
				rc.Close()
			}
		}
	}

	// 按文件名排序图片
	sort.Slice(cbz.images, func(i, j int) bool {
		return naturalLess(cbz.images[i].Name, cbz.images[j].Name)
	})

	if len(cbz.images) == 0 {
		r.Close()
		return nil, fmt.Errorf("no supported image files found in cbz archive")
	}

	// 第一张图片作为封面
	cbz.images[0].IsCover = true
	cbz.coverIdx = 0

	// 保存 zip 文件引用（延迟关闭）
	cbz.zipReader = r

	return cbz, nil
}

// findFileIndex 在 zip.File 切片中查找指定路径的索引
func findFileIndex(files []*zip.File, path string) int {
	for i, f := range files {
		if f.Name == path {
			return i
		}
	}
	return -1
}

// naturalLess 自然排序比较（支持数字前缀如 "001.jpg", "002.jpg", "10.jpg"）
func naturalLess(a, b string) bool {
	// 简单的自然排序：先比较数字部分，再比较字符串部分
	for len(a) > 0 && len(b) > 0 {
		if a[0] >= '0' && a[0] <= '9' && b[0] >= '0' && b[0] <= '9' {
			// 都是数字，提取完整数字
			i := 0
			for i < len(a) && a[i] >= '0' && a[i] <= '9' {
				i++
			}
			j := 0
			for j < len(b) && b[j] >= '0' && b[j] <= '9' {
				j++
			}
			// 比较数字长度（前导零少的数字更大）
			if i != j {
				return i < j
			}
			// 长度相同，逐位比较
			for k := 0; k < i; k++ {
				if a[k] != b[k] {
					return a[k] < b[k]
				}
			}
			// 数字部分相同，跳过
			a = a[i:]
			b = b[j:]
		} else {
			if a[0] != b[0] {
				return a[0] < b[0]
			}
			a = a[1:]
			b = b[1:]
		}
	}
	return len(a) < len(b)
}

// =============================================================================
// 公共接口方法
// =============================================================================

// GetBookInfo 获取书籍信息（包括从 ComicInfo.xml 解析的元数据）
func (cbz *CbzFile) GetBookInfo() *Book {
	if cbz.ci != nil {
		return cbz.ci.ToBookInfo()
	}

	// 没有 ComicInfo.xml，使用文件名作为书名
	name := strings.TrimSuffix(filepath.Base(cbz.filePath), filepath.Ext(cbz.filePath))
	return &Book{
		Name: name,
	}
}

// GetChapters 获取所有章节列表
// 每张图片作为一个章节，封面单独标记
func (cbz *CbzFile) GetChapters() []Chapter {
	chapters := make([]Chapter, len(cbz.images))
	for i, img := range cbz.images {
		chapters[i] = Chapter{
			Name:    img.Name,
			Index:   i,
			IsCover: img.IsCover,
		}
	}
	return chapters
}

// GetImage 获取指定索引的图片数据
// chapterIndex 从 0 开始，0 表示封面（第一张图片）
func (cbz *CbzFile) GetImage(chapterIndex int) ([]byte, error) {
	if cbz.zipReader == nil {
		return nil, fmt.Errorf("cbz archive is closed")
	}
	if chapterIndex < 0 || chapterIndex >= len(cbz.images) {
		return nil, fmt.Errorf("invalid chapter index %d, must be in [0, %d)", chapterIndex, len(cbz.images))
	}

	entry := cbz.images[chapterIndex]
	var target *zip.File
	for _, f := range cbz.zipReader.File {
		if f.Name == entry.Path {
			target = f
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("image file not found in archive: %s", entry.Path)
	}

	rc, err := target.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open image %s: %w", entry.Path, err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// GetCover 获取封面图片数据（第一张图片）
func (cbz *CbzFile) GetCover() ([]byte, error) {
	return cbz.GetImage(cbz.coverIdx)
}

// GetImageCount 获取图片总数
func (cbz *CbzFile) GetImageCount() int {
	return len(cbz.images)
}

// GetComicInfo 获取解析的 ComicInfo 元数据
func (cbz *CbzFile) GetComicInfo() *ComicInfo {
	return cbz.ci
}

// Close 关闭 CBZ 文件
func (cbz *CbzFile) Close() error {
	if cbz.zipReader != nil {
		return cbz.zipReader.Close()
	}
	return nil
}

// =============================================================================
// 便捷函数：一次性解析并获取所有数据
// =============================================================================

// ParseAll 解析 CBZ 文件并返回完整数据
// 包含书籍信息、所有章节和图片数据
type CBZResult struct {
	Book     *Book     `json:"book"`
	Chapters []Chapter `json:"chapters"`
	Cover    []byte    `json:"cover,omitempty"`
	Images   [][]byte  `json:"-"` // 原始图片数据，不序列化
}

// ParseAll 解析整个 CBZ 文件
func ParseAll(path string) (*CBZResult, error) {
	cbz, err := Open(path)
	if err != nil {
		return nil, err
	}
	defer cbz.Close()

	result := &CBZResult{
		Book:     cbz.GetBookInfo(),
		Chapters: cbz.GetChapters(),
	}

	// 获取封面
	cover, err := cbz.GetCover()
	if err == nil {
		result.Cover = cover
	}

	// 获取所有图片
	result.Images = make([][]byte, len(cbz.images))
	for i := range cbz.images {
		img, err := cbz.GetImage(i)
		if err != nil {
			// 单张图片失败不中断，记录空数据
			img = nil
		}
		result.Images[i] = img
	}

	return result, nil
}

// =============================================================================
// 文件操作：将解析结果保存到目录
// =============================================================================

// Export 将 CBZ 内容导出到指定目录
func (cbz *CbzFile) Export(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	// 导出 ComicInfo.xml（如果存在）
	if cbz.ci != nil {
		data, err := xml.MarshalIndent(cbz.ci, "", "  ")
		if err == nil {
			os.WriteFile(filepath.Join(dir, "ComicInfo.xml"), data, 0644)
		}
	}

	// 导出所有图片
	for i := range cbz.images {
		img, err := cbz.GetImage(i)
		if err != nil {
			continue
		}
		outPath := filepath.Join(dir, fmt.Sprintf("%04d_%s", i+1, cbz.images[i].Name))
		os.WriteFile(outPath, img, 0644)
	}

	return nil
}
