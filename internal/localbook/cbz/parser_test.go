package cbz

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// 测试辅助函数
// =============================================================================

// createTestCBZ 创建一个测试用的 CBZ 文件
// 返回文件路径和清理函数
func createTestCBZ(t *testing.T, comicInfo *ComicInfo, images map[string][]byte) (string, func()) {
	t.Helper()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "cbz-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cbzPath := filepath.Join(tmpDir, "test.cbz")

	// 创建 zip 文件
	zipFile, err := os.Create(cbzPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)

	// 写入 ComicInfo.xml
	if comicInfo != nil {
		data, err := xml.Marshal(comicInfo)
		if err != nil {
			w.Close()
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to marshal comicinfo: %v", err)
		}
		f, err := w.Create("ComicInfo.xml")
		if err != nil {
			w.Close()
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create ComicInfo.xml: %v", err)
		}
		f.Write(data)
	}

	// 写入图片
	for name, data := range images {
		f, err := w.Create(name)
		if err != nil {
			w.Close()
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create %s: %v", name, err)
		}
		f.Write(data)
	}

	if err := w.Close(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to close zip writer: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return cbzPath, cleanup
}

// createTestCBZWithSubdir 创建带子目录结构的 CBZ 文件
func createTestCBZWithSubdir(t *testing.T, images map[string][]byte) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "cbz-test-subdir-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cbzPath := filepath.Join(tmpDir, "test_subdir.cbz")

	zipFile, err := os.Create(cbzPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)

	for name, data := range images {
		f, err := w.Create(name)
		if err != nil {
			w.Close()
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create %s: %v", name, err)
		}
		f.Write(data)
	}

	if err := w.Close(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to close zip writer: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return cbzPath, cleanup
}

// fakeImage 创建一个假的图片数据（PNG 文件头）
func fakeImage() []byte {
	// PNG 文件头（8 字节）
	return []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00}
}

// fakeJPEG 创建一个假的 JPEG 数据
func fakeJPEG() []byte {
	// JPEG 文件头
	return []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
}

// =============================================================================
// ComicInfo 解析测试
// =============================================================================

func TestParseComicInfo(t *testing.T) {
	xmlData := `<?xml version="1.0"?>
<ComicInfo xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <Title>测试漫画</Title>
  <Series>测试系列</Series>
  <Volume>1</Volume>
  <Issue>5</Issue>
  <Writer>作者A</Writer>
  <Penciller>画师B</Penciller>
  <Summary>这是一个测试简介。</Summary>
  <Year>2024</Year>
  <Month>6</Month>
  <Genre>冒险, 科幻</Genre>
  <Manga>Yes</Manga>
  <Count>20</Count>
</ComicInfo>`

	ci, err := ParseComicInfo([]byte(xmlData))
	if err != nil {
		t.Fatalf("ParseComicInfo failed: %v", err)
	}

	if ci.Title != "测试漫画" {
		t.Errorf("Title = %q, want %q", ci.Title, "测试漫画")
	}
	if ci.Series != "测试系列" {
		t.Errorf("Series = %q, want %q", ci.Series, "测试系列")
	}
	if ci.Volume != "1" {
		t.Errorf("Volume = %q, want %q", ci.Volume, "1")
	}
	if ci.Issue != "5" {
		t.Errorf("Issue = %q, want %q", ci.Issue, "5")
	}
	if ci.Writer != "作者A" {
		t.Errorf("Writer = %q, want %q", ci.Writer, "作者A")
	}
	if ci.Penciller != "画师B" {
		t.Errorf("Penciller = %q, want %q", ci.Penciller, "画师B")
	}
	if ci.Summary != "这是一个测试简介。" {
		t.Errorf("Summary = %q, want %q", ci.Summary, "这是一个测试简介。")
	}
	if ci.Year != "2024" {
		t.Errorf("Year = %q, want %q", ci.Year, "2024")
	}
	if ci.Month != "6" {
		t.Errorf("Month = %q, want %q", ci.Month, "6")
	}
	if ci.Genre != "冒险, 科幻" {
		t.Errorf("Genre = %q, want %q", ci.Genre, "冒险, 科幻")
	}
	if ci.Manga != "Yes" {
		t.Errorf("Manga = %q, want %q", ci.Manga, "Yes")
	}
}

func TestParseComicInfo_Empty(t *testing.T) {
	ci, err := ParseComicInfo([]byte(`<ComicInfo></ComicInfo>`))
	if err != nil {
		t.Fatalf("ParseComicInfo failed: %v", err)
	}

	if ci.Title != "" {
		t.Errorf("Title should be empty, got %q", ci.Title)
	}
}

func TestParseComicInfo_InvalidXML(t *testing.T) {
	_, err := ParseComicInfo([]byte(`not valid xml`))
	if err == nil {
		t.Fatal("expected error for invalid XML, got nil")
	}
}

func TestComicInfo_ToBookInfo(t *testing.T) {
	ci := &ComicInfo{
		Title:  "漫画标题",
		Series: "系列名",
		Volume: "3",
		Issue:  "12",
		Writer: "作者",
		Summary: "简介内容",
		Genre:  "动作, 冒险",
		Year:   "2023",
		Manga:  "Yes",
	}

	book := ci.ToBookInfo()

	if book.Name != "漫画标题 Vol.3#12" {
		t.Errorf("Name = %q, want %q", book.Name, "漫画标题 Vol.3#12")
	}
	if book.Author != "作者" {
		t.Errorf("Author = %q, want %q", book.Author, "作者")
	}
	if book.Summary != "简介内容" {
		t.Errorf("Summary = %q, want %q", book.Summary, "简介内容")
	}
	if book.Tags != "动作, 冒险" {
		t.Errorf("Tags = %q, want %q", book.Tags, "动作, 冒险")
	}
	if book.Metadata["year"] != "2023" {
		t.Errorf("Metadata[year] = %q, want %q", book.Metadata["year"], "2023")
	}
	if book.Metadata["manga"] != "Yes" {
		t.Errorf("Metadata[manga] = %q, want %q", book.Metadata["manga"], "Yes")
	}
}

func TestComicInfo_ToBookInfo_NoTitle(t *testing.T) {
	ci := &ComicInfo{
		Series: "仅系列名",
		Writer: "作者",
	}

	book := ci.ToBookInfo()

	if book.Name != "仅系列名" {
		t.Errorf("Name = %q, want %q", book.Name, "仅系列名")
	}
}

func TestComicInfo_ToBookInfo_FallbackAuthor(t *testing.T) {
	// 没有 Writer，应该 fallback 到 Penciller
	ci := &ComicInfo{
		Title:   "标题",
		Penciller: "画师",
	}

	book := ci.ToBookInfo()

	if book.Author != "画师" {
		t.Errorf("Author = %q, want %q", book.Author, "画师")
	}
}

// =============================================================================
// CbzFile 解析测试
// =============================================================================

func TestParse(t *testing.T) {
	images := map[string][]byte{
		"page_001.png": fakeImage(),
		"page_002.png": fakeImage(),
		"page_003.png": fakeImage(),
	}

	cbzPath, cleanup := createTestCBZ(t, &ComicInfo{
		Title:  "测试漫画",
		Writer: "作者",
		Summary: "测试简介",
	}, images)
	defer cleanup()

	book, err := Parse(cbzPath)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if book.Name != "测试漫画" {
		t.Errorf("Name = %q, want %q", book.Name, "测试漫画")
	}
	if book.Author != "作者" {
		t.Errorf("Author = %q, want %q", book.Author, "作者")
	}
	if book.Summary != "测试简介" {
		t.Errorf("Summary = %q, want %q", book.Summary, "测试简介")
	}
}

func TestOpen(t *testing.T) {
	images := map[string][]byte{
		"01_cover.jpg":  fakeJPEG(),
		"02_page.jpg":   fakeJPEG(),
		"03_page.jpg":   fakeJPEG(),
		"04_page.jpg":   fakeJPEG(),
	}

	cbzPath, cleanup := createTestCBZ(t, &ComicInfo{
		Title:  "测试漫画",
		Series: "系列",
	}, images)
	defer cleanup()

	cbz, err := Open(cbzPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer cbz.Close()

	// 验证图片数量
	if cbz.GetImageCount() != 4 {
		t.Errorf("GetImageCount = %d, want 4", cbz.GetImageCount())
	}

	// 验证章节
	chapters := cbz.GetChapters()
	if len(chapters) != 4 {
		t.Fatalf("GetChapters returned %d chapters, want 4", len(chapters))
	}

	if !chapters[0].IsCover {
		t.Error("first chapter should be cover")
	}
	if chapters[0].Name != "01_cover.jpg" {
		t.Errorf("first chapter name = %q, want %q", chapters[0].Name, "01_cover.jpg")
	}

	// 验证封面
	cover, err := cbz.GetCover()
	if err != nil {
		t.Fatalf("GetCover failed: %v", err)
	}
	if len(cover) == 0 {
		t.Error("cover data should not be empty")
	}
	if !bytes.HasPrefix(cover, []byte{0xFF, 0xD8}) {
		t.Error("cover should be JPEG")
	}

	// 验证第二张图片
	img2, err := cbz.GetImage(1)
	if err != nil {
		t.Fatalf("GetImage(1) failed: %v", err)
	}
	if len(img2) == 0 {
		t.Error("image 2 data should not be empty")
	}
}

func TestOpen_NoComicInfo(t *testing.T) {
	images := map[string][]byte{
		"page1.png": fakeImage(),
		"page2.png": fakeImage(),
	}

	cbzPath, cleanup := createTestCBZ(t, nil, images) // 没有 ComicInfo
	defer cleanup()

	cbz, err := Open(cbzPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer cbz.Close()

	book := cbz.GetBookInfo()
	// 没有 ComicInfo 时，使用文件名（去掉 .cbz 扩展名）
	if !strings.Contains(book.Name, "test") {
		t.Errorf("Name should contain 'test', got %q", book.Name)
	}
	if book.Author != "" {
		t.Errorf("Author should be empty, got %q", book.Author)
	}
}

func TestOpen_NoImages(t *testing.T) {
	// 只有 ComicInfo，没有图片
	cbzPath, cleanup := createTestCBZ(t, &ComicInfo{Title: "测试"}, nil)
	defer cleanup()

	_, err := Open(cbzPath)
	if err == nil {
		t.Fatal("expected error for no images, got nil")
	}
	if !strings.Contains(err.Error(), "no supported image files") {
		t.Errorf("error message should mention 'no supported image files', got %q", err.Error())
	}
}

func TestOpen_UnsupportedImagesOnly(t *testing.T) {
	// 只有不支持的格式
	images := map[string][]byte{
		"readme.txt": []byte("not an image"),
		"data.json":  []byte("{}"),
	}

	cbzPath, cleanup := createTestCBZ(t, nil, images)
	defer cleanup()

	_, err := Open(cbzPath)
	if err == nil {
		t.Fatal("expected error for no supported images, got nil")
	}
}

func TestOpen_SupportsAllFormats(t *testing.T) {
	images := map[string][]byte{
		"01.jpg":  fakeJPEG(),
		"02.jpeg": fakeJPEG(),
		"03.png":  fakeImage(),
		"04.gif":  []byte("GIF89a"),
		"05.bmp":  []byte("BM"),
		"06.webp": []byte("RIFF\x00\x00\x00\x00WEBP"),
		"07.svg":  []byte("<svg></svg>"),
	}

	cbzPath, cleanup := createTestCBZ(t, nil, images)
	defer cleanup()

	cbz, err := Open(cbzPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer cbz.Close()

	if cbz.GetImageCount() != 7 {
		t.Errorf("GetImageCount = %d, want 7", cbz.GetImageCount())
	}
}

func TestOpen_NaturalSort(t *testing.T) {
	// 测试自然排序：1, 2, ..., 9, 10, 11
	images := map[string][]byte{
		"page_10.png":  fakeImage(),
		"page_2.png":   fakeImage(),
		"page_1.png":   fakeImage(),
		"page_11.png":  fakeImage(),
		"page_20.png":  fakeImage(),
		"page_3.png":   fakeImage(),
	}

	cbzPath, cleanup := createTestCBZ(t, nil, images)
	defer cleanup()

	cbz, err := Open(cbzPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer cbz.Close()

	chapters := cbz.GetChapters()
	expectedOrder := []string{"page_1.png", "page_2.png", "page_3.png", "page_10.png", "page_11.png", "page_20.png"}

	for i, expected := range expectedOrder {
		if chapters[i].Name != expected {
			t.Errorf("chapter[%d] = %q, want %q", i, chapters[i].Name, expected)
		}
	}
}

func TestOpen_Subdirectory(t *testing.T) {
	// 图片在子目录中
	images := map[string][]byte{
		"images/01.png": fakeImage(),
		"images/02.png": fakeImage(),
		"images/03.png": fakeImage(),
	}

	cbzPath, cleanup := createTestCBZWithSubdir(t, images)
	defer cleanup()

	cbz, err := Open(cbzPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer cbz.Close()

	if cbz.GetImageCount() != 3 {
		t.Errorf("GetImageCount = %d, want 3", cbz.GetImageCount())
	}
	if chapters := cbz.GetChapters(); chapters[0].Name != "01.png" {
		t.Errorf("first chapter name = %q, want %q", chapters[0].Name, "01.png")
	}
}

func TestGetImage_InvalidIndex(t *testing.T) {
	images := map[string][]byte{
		"01.png": fakeImage(),
	}

	cbzPath, cleanup := createTestCBZ(t, nil, images)
	defer cleanup()

	cbz, err := Open(cbzPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer cbz.Close()

	_, err = cbz.GetImage(-1)
	if err == nil {
		t.Error("GetImage(-1) should return error")
	}

	_, err = cbz.GetImage(10)
	if err == nil {
		t.Error("GetImage(10) should return error")
	}
}

func TestGetImage_AfterClose(t *testing.T) {
	images := map[string][]byte{
		"01.png": fakeImage(),
	}

	cbzPath, cleanup := createTestCBZ(t, nil, images)
	defer cleanup()

	cbz, err := Open(cbzPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	cbz.Close()

	_, err = cbz.GetImage(0)
	if err == nil {
		t.Error("GetImage after Close should return error")
	}
}

func TestGetComicInfo(t *testing.T) {
	ci := &ComicInfo{
		Title:  "测试",
		Writer: "作者",
		Year:   "2024",
	}

	images := map[string][]byte{
		"01.png": fakeImage(),
	}

	cbzPath, cleanup := createTestCBZ(t, ci, images)
	defer cleanup()

	cbz, err := Open(cbzPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer cbz.Close()

	info := cbz.GetComicInfo()
	if info == nil {
		t.Fatal("GetComicInfo returned nil")
	}
	if info.Title != "测试" {
		t.Errorf("Title = %q, want %q", info.Title, "测试")
	}
}

// =============================================================================
// ParseAll 测试
// =============================================================================

func TestParseAll(t *testing.T) {
	images := map[string][]byte{
		"01.png": fakeImage(),
		"02.png": fakeImage(),
		"03.png": fakeImage(),
	}

	cbzPath, cleanup := createTestCBZ(t, &ComicInfo{
		Title:  "测试漫画",
		Writer: "作者",
		Summary: "简介",
	}, images)
	defer cleanup()

	result, err := ParseAll(cbzPath)
	if err != nil {
		t.Fatalf("ParseAll failed: %v", err)
	}

	if result.Book.Name != "测试漫画" {
		t.Errorf("Book.Name = %q, want %q", result.Book.Name, "测试漫画")
	}
	if len(result.Chapters) != 3 {
		t.Errorf("len(Chapters) = %d, want 3", len(result.Chapters))
	}
	if len(result.Cover) == 0 {
		t.Error("Cover should not be empty")
	}
	if len(result.Images) != 3 {
		t.Errorf("len(Images) = %d, want 3", len(result.Images))
	}
}

// =============================================================================
// Export 测试
// =============================================================================

func TestExport(t *testing.T) {
	images := map[string][]byte{
		"01_cover.png": fakeImage(),
		"02_page.png":  fakeImage(),
	}

	ci := &ComicInfo{
		Title:  "导出测试",
		Writer: "作者",
	}

	cbzPath, cleanup := createTestCBZ(t, ci, images)
	defer cleanup()

	cbz, err := Open(cbzPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer cbz.Close()

	// 创建导出目录
	exportDir, err := os.MkdirTemp("", "cbz-export-*")
	if err != nil {
		t.Fatalf("failed to create export dir: %v", err)
	}
	defer os.RemoveAll(exportDir)

	if err := cbz.Export(exportDir); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// 验证导出的文件
	files, err := os.ReadDir(exportDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	// 应该有 ComicInfo.xml + 2 张图片
	if len(files) != 3 {
		t.Errorf("exported %d files, want 3", len(files))
	}

	// 验证 ComicInfo.xml
	comicInfoPath := filepath.Join(exportDir, "ComicInfo.xml")
	if _, err := os.Stat(comicInfoPath); os.IsNotExist(err) {
		t.Error("ComicInfo.xml not exported")
	}

	// 验证图片
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".png") {
			info, _ := f.Info()
			if info.Size() == 0 {
				t.Errorf("exported image %s is empty", f.Name())
			}
		}
	}
}

// =============================================================================
// 边界情况测试
// =============================================================================

func TestOpen_FileNotFound(t *testing.T) {
	_, err := Open("/nonexistent/path/test.cbz")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestOpen_InvalidZip(t *testing.T) {
	// 创建一个不是 zip 格式的文件
	tmpFile, err := os.CreateTemp("", "fake-cbz-*.cbz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.WriteString("this is not a zip file")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = Open(tmpFile.Name())
	if err == nil {
		t.Fatal("expected error for invalid zip, got nil")
	}
}

func TestNaturalLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"01.png", "02.png", true},
		{"02.png", "01.png", false},
		{"1.png", "10.png", true},
		{"10.png", "2.png", false},
		{"10.png", "11.png", true},
		{"2.png", "10.png", true},
		{"a1.png", "a2.png", true},
		{"img_01.png", "img_02.png", true},
		{"chapter_1.png", "chapter_10.png", true},
		{"chapter_2.png", "chapter_10.png", true},
	}

	for _, tt := range tests {
		got := naturalLess(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("naturalLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"image.jpg", true},
		{"image.jpeg", true},
		{"image.png", true},
		{"image.gif", true},
		{"image.bmp", true},
		{"image.webp", true},
		{"image.svg", true},
		{"IMAGE.JPG", true},
		{"image.JpEg", true},
		{"readme.txt", false},
		{"data.json", false},
		{"archive.zip", false},
		{"ComicInfo.xml", false},
	}

	for _, tt := range tests {
		got := isImageFile(tt.name)
		if got != tt.want {
			t.Errorf("isImageFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestStripPathPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"./images/page1.png", "images/page1.png"},
		{"images/page1.png", "images/page1.png"},
		{"page1.png", "page1.png"},
		{"./page1.png", "page1.png"},
		{"comic/01.png", "comic/01.png"},
	}

	for _, tt := range tests {
		got := stripPathPrefix(tt.input)
		if got != tt.want {
			t.Errorf("stripPathPrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
