package cbz

import (
	"encoding/xml"
	"fmt"
	"io"
)

// =============================================================================
// ComicInfo.xml 数据模型
// 参考 https://github.com/comictagger/comictagger/wiki/ComicInfo-v2.1
// =============================================================================

// ComicInfo 漫画元数据根结构
type ComicInfo struct {
	XMLName     xml.Name `xml:"ComicInfo"`
	PageCount   int      `xml:"Count,attr"`         // 可选：总页数
	CountTotal  int      `xml:"CountTotal,attr"`    // 可选：系列总页数
	Format      string   `xml:"Format"`             // 格式 (Comic, Manga, etc.)
	Manga       string   `xml:"Manga"`              // Yes/No/Unknown
	BlackAndWhite bool   `xml:"BlackAndWhite"`      // 是否黑白

	// 基本信息
	Title   string `xml:"Title"`       // 书名
	Series  string `xml:"Series"`      // 系列名
	Volume  string `xml:"Volume"`      // 卷号
	Issue   string `xml:"Issue"`       // 期号
	Count   string `xml:"Count"`       // 页数（文本形式）

	// 创作信息
	Writer    string `xml:"Writer"`        // 作者/编剧
	Penciller string `xml:"Penciller"`     // 画师
	Inker     string `xml:"Inker"`         // 勾线师
	Colorist  string `xml:"Colorist"`      // 上色师
	Letterer  string `xml:"Letterer"`      // 文字师
	Editor    string `xml:"Editor"`        // 编辑
	Translator string `xml:"Translator"`   // 翻译

	// 描述信息
	Summary   string `xml:"Summary"`       // 简介
	Notes     string `xml:"Notes"`         // 备注

	// 分类信息
	Genre     string `xml:"Genre"`         // 类型
	Web       string `xml:"Web"`           // 网站链接
	Notes2    string `xml:"Notes2"`        // 备用备注字段

	// 时间信息
	Year      string `xml:"Year"`          // 出版年份
	Month     string `xml:"Month"`         // 出版月份
	Day       string `xml:"Day"`           // 出版日期

	// 其他
	Language  string `xml:"LanguageISO"`   // 语言代码
	AgeRating string `xml:"AgeRating"`     // 年龄分级
	ScanInformation string `xml:"ScanInformation"` // 扫描信息
	StoryArc  string `xml:"StoryArc"`      // 故事弧
	StoryArcID string `xml:"StoryArcID"`   // 故事弧ID
	SequenceNumber string `xml:"SequenceNumber"` // 序列号
	Tags      string `xml:"Tags"`          // 标签
	AlternateSeries string `xml:"AlternateSeries"`
	AlternateNumber string `xml:"AlternateNumber"`
	AlternateCount string `xml:"AlternateCount"`
}

// ParseComicInfo 从 XML 数据解析 ComicInfo
func ParseComicInfo(data []byte) (*ComicInfo, error) {
	ci := &ComicInfo{}
	if err := xml.Unmarshal(data, ci); err != nil {
		return nil, err
	}
	return ci, nil
}

// ParseComicInfoFromReader 从 io.Reader 解析 ComicInfo
func ParseComicInfoFromReader(r io.Reader) (*ComicInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseComicInfo(data)
}

// ToBookInfo 将 ComicInfo 转换为 Book 的元数据
func (ci *ComicInfo) ToBookInfo() *Book {
	author := ci.Writer
	if author == "" {
		author = ci.Penciller
	}

	// 构造书名：Series + Volume + Issue
	name := ci.Title
	if name == "" {
		if ci.Series != "" {
			name = ci.Series
		}
	}
	if ci.Volume != "" || ci.Issue != "" {
		if name != "" {
			name += " "
		}
		if ci.Volume != "" {
			name += "Vol." + ci.Volume
		}
		if ci.Issue != "" {
			name += "#" + ci.Issue
		}
	}

	// 构造简介
	intro := ci.Summary
	if intro == "" && ci.ScanInformation != "" {
		intro = ci.ScanInformation
	}

	return &Book{
		Name:    name,
		Author:  author,
		Summary: intro,
		Tags:    ci.Genre,
		Metadata: map[string]string{
			"format":             ci.Format,
			"manga":              ci.Manga,
			"year":               ci.Year,
			"month":              ci.Month,
			"volume":             ci.Volume,
			"issue":              ci.Issue,
			"black_and_white":    fmt.Sprintf("%v", ci.BlackAndWhite),
			"age_rating":         ci.AgeRating,
			"language":           ci.Language,
			"story_arc":          ci.StoryArc,
			"alternate_series":   ci.AlternateSeries,
			"alternate_number":   ci.AlternateNumber,
			"scan_information":   ci.ScanInformation,
			"tags":               ci.Tags,
			"writer":             ci.Writer,
			"penciller":          ci.Penciller,
			"inker":              ci.Inker,
			"colorist":           ci.Colorist,
			"letterer":           ci.Letterer,
		},
	}
}
