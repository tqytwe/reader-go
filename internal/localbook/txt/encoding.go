// Package txt TXT 书籍解析器
// 参考 legado 的 TextFile.kt 实现
package txt

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// =============================================================================
// BOM 定义
// =============================================================================

// BOM signatures for common encodings
var (
	// UTF-8 BOM: EF BB BF
	BOM_UTF8 = []byte{0xEF, 0xBB, 0xBF}
	// UTF-16 LE BOM: FF FE
	BOM_UTF16_LE = []byte{0xFF, 0xFE}
	// UTF-16 BE BOM: FE FF
	BOM_UTF16_BE = []byte{0xFE, 0xFF}
	// UTF-32 LE BOM: FF FE 00 00
	BOM_UTF32_LE = []byte{0xFF, 0xFE, 0x00, 0x00}
	// UTF-32 BE BOM: 00 00 FE FF
	BOM_UTF32_BE = []byte{0x00, 0x00, 0xFE, 0xFF}
)

// EncodingType 编码类型枚举
type EncodingType int

const (
	EncodingUnknown EncodingType = iota
	EncodingUTF8
	EncodingUTF8BOM
	EncodingGBK
	EncodingGB2312
	EncodingUTF16LE
	EncodingUTF16BE
	EncodingUTF32LE
	EncodingUTF32BE
)

func (e EncodingType) String() string {
	switch e {
	case EncodingUnknown:
		return "unknown"
	case EncodingUTF8:
		return "UTF-8"
	case EncodingUTF8BOM:
		return "UTF-8-BOM"
	case EncodingGBK:
		return "GBK"
	case EncodingGB2312:
		return "GB2312"
	case EncodingUTF16LE:
		return "UTF-16-LE"
	case EncodingUTF16BE:
		return "UTF-16-BE"
	case EncodingUTF32LE:
		return "UTF-32-LE"
	case EncodingUTF32BE:
		return "UTF-32-BE"
	default:
		return "unknown"
	}
}

// =============================================================================
// BOM 检测
// =============================================================================

// DetectBOM 检测文件开头的 BOM 标记
// 返回 BOM 类型和 BOM 字节长度
func DetectBOM(data []byte) (EncodingType, int) {
	if len(data) >= 4 {
		if bytes.Equal(data[:4], BOM_UTF32_LE) {
			return EncodingUTF32LE, 4
		}
		if bytes.Equal(data[:4], BOM_UTF32_BE) {
			return EncodingUTF32BE, 4
		}
	}

	if len(data) >= 3 {
		if bytes.Equal(data[:3], BOM_UTF8) {
			return EncodingUTF8BOM, 3
		}
	}

	if len(data) >= 2 {
		if bytes.Equal(data[:2], BOM_UTF16_LE) {
			return EncodingUTF16LE, 2
		}
		if bytes.Equal(data[:2], BOM_UTF16_BE) {
			return EncodingUTF16BE, 2
		}
	}

	return EncodingUnknown, 0
}

// =============================================================================
// 编码检测器
// =============================================================================

// EncodingDetector 编码检测器
type EncodingDetector struct {
	// 候选编码列表（按优先级排序）
	candidates []EncodingType
	// 采样字节数（用于检测）
	sampleSize int
}

// NewEncodingDetector 创建编码检测器
func NewEncodingDetector() *EncodingDetector {
	return &EncodingDetector{
		candidates: []EncodingType{
			EncodingUTF8,
			EncodingGBK,
			EncodingGB2312,
		},
		sampleSize: 4096, // 默认采样 4KB
	}
}

// SetCandidates 设置候选编码列表
func (d *EncodingDetector) SetCandidates(encodings []EncodingType) {
	d.candidates = encodings
}

// SetSampleSize 设置采样字节数
func (d *EncodingDetector) SetSampleSize(size int) {
	d.sampleSize = size
}

// Detect 从文件路径检测编码
func (d *EncodingDetector) Detect(filePath string) (EncodingType, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return EncodingUnknown, err
	}
	defer f.Close()

	// 读取采样数据
	sample := make([]byte, d.sampleSize)
	n, err := io.ReadFull(f, sample)
	if err != nil && err != io.ErrUnexpectedEOF {
		// 文件太小，尝试读取全部
		sample = make([]byte, 1024)
		n, _ = f.Read(sample)
	}
	sample = sample[:n]

	return d.DetectFromBytes(sample)
}

// DetectFromBytes 从字节数据检测编码
func (d *EncodingDetector) DetectFromBytes(data []byte) (EncodingType, error) {
	if len(data) == 0 {
		return EncodingUnknown, fmt.Errorf("empty data")
	}

	// 1. 首先检测 BOM
	bomType, bomLen := DetectBOM(data)
	if bomType != EncodingUnknown {
		// 有 BOM 直接返回（BOM 优先级最高）
		_ = bomLen // 避免 unused variable
		return bomType, nil
	}

	// 2. 无 BOM 时，通过内容分析检测编码
	type score struct {
		enc   EncodingType
		score float64
	}
	scores := make([]score, 0, len(d.candidates))

	for _, enc := range d.candidates {
		s := d.scoreEncoding(data, enc)
		if s > 0 {
			scores = append(scores, score{enc: enc, score: s})
		}
	}

	if len(scores) == 0 {
		// 所有编码都失败，默认 UTF-8
		return EncodingUTF8, nil
	}

	// 选择得分最高的编码
	best := scores[0]
	for i := 1; i < len(scores); i++ {
		if scores[i].score > best.score {
			best = scores[i]
		}
	}

	// UTF-8 优先校验：如果数据包含合法 UTF-8 多字节序列，
	// 即使 GBK 分数略高也应选择 UTF-8。
	// 这是因为 UTF-8 中文的 3 字节序列在 GBK 解码时可能产生看似有效的 CJK 字符。
	if best.enc != EncodingUTF8 && d.hasUTF8MultiByte(data) {
		for _, s := range scores {
			if s.enc == EncodingUTF8 && d.isProbableUTF8(data) {
				return EncodingUTF8, nil
			}
		}
	}

	return best.enc, nil
}

// scoreEncoding 对指定编码打分（0-1 之间）
// 基于：
// 1. 可解码字符比例
// 2. 中文字符比例（对于中文编码）
// 3. 控制字符比例（越低越好）
func (d *EncodingDetector) scoreEncoding(data []byte, enc EncodingType) float64 {
	var dec encoding.Encoding
	var isChineseEncoding bool

	switch enc {
	case EncodingUTF8:
		dec = encoding.Nop
		isChineseEncoding = false
	case EncodingGBK:
		dec = simplifiedchinese.GBK
		isChineseEncoding = true
	case EncodingGB2312:
		dec = simplifiedchinese.HZGB2312
		isChineseEncoding = true
	default:
		return 0
	}

	// 尝试解码并统计
	decoder := transform.NewReader(bytes.NewReader(data), dec.NewDecoder())
	decoded, err := io.ReadAll(decoder)
	if err != nil {
		// 解码失败，得分很低
		return 0.01
	}

	if len(decoded) == 0 {
		return 0
	}

	score := 0.0

	// 1. 基础分：解码成功即给基础分
	score += 0.3

	// 2. 控制字符比例（排除换行、制表等合法控制字符）
	controlCount := 0
	for _, b := range data {
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			controlCount++
		}
	}
	controlRatio := float64(controlCount) / float64(len(data))
	if controlRatio < 0.01 {
		score += 0.3
	} else if controlRatio < 0.05 {
		score += 0.15
	}

	// 3. 中文编码特有：检测中文字符比例
	if isChineseEncoding {
		// 统计 CJK 字符（Unicode 范围 4E00-9FFF 等）
		cjkCount := 0
		for _, r := range string(decoded) {
			if (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
				(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
				(r >= 0x20000 && r <= 0x2A6DF) || // CJK Extension B
				(r >= 0xF900 && r <= 0xFAFF) { // CJK Compatibility Ideographs
				cjkCount++
			}
		}
		cjkRatio := float64(cjkCount) / float64(len(decoded))
		if cjkRatio > 0.1 {
			score += 0.4
		} else if cjkRatio > 0.05 {
			score += 0.2
		} else if cjkRatio > 0.01 {
			score += 0.1
		}
	}

	// 4. UTF-8 特有：检测是否为合法 UTF-8
	if enc == EncodingUTF8 {
		if d.isProbableUTF8(data) {
			score += 0.3
		}
	}

	return score
}

// isProbableUTF8 判断数据是否可能是 UTF-8 编码
func (d *EncodingDetector) isProbableUTF8(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	i := 0
	validSeqCount := 0
	invalidSeqCount := 0

	for i < len(data) {
		b := data[i]

		if b <= 0x7F {
			// ASCII
			i++
			validSeqCount++
			continue
		}

		// 判断 UTF-8 序列长度
		var seqLen int
		if b <= 0xBF {
			// 无效：10xxxxxx 不能作为起始字节
			invalidSeqCount++
			i++
			continue
		} else if b <= 0xDF {
			seqLen = 2
		} else if b <= 0xEF {
			seqLen = 3
		} else if b <= 0xF4 {
			seqLen = 4
		} else {
			// 无效起始字节
			invalidSeqCount++
			i++
			continue
		}

		// 检查后续字节
		if i+seqLen > len(data) {
			invalidSeqCount++
			break
		}

		valid := true
		for j := 1; j < seqLen; j++ {
			if data[i+j] < 0x80 || data[i+j] > 0xBF {
				valid = false
				break
			}
		}

		if valid {
			validSeqCount++
			i += seqLen
		} else {
			invalidSeqCount++
			i++
		}
	}

	if len(data) == 0 {
		return false
	}

	totalSeqs := validSeqCount + invalidSeqCount
	if totalSeqs == 0 {
		return false
	}

	ratio := float64(validSeqCount) / float64(totalSeqs)
	return ratio > 0.8 || (ratio > 0.5 && invalidSeqCount < 5)
}

// hasUTF8MultiByte 检测数据是否包含 UTF-8 多字节序列
// UTF-8 中文通常是 3 字节序列：1110xxxx 10xxxxxx 10xxxxxx
// 如果检测到 ≥3 个连续的合法 3 字节 UTF-8 序列，则很可能是 UTF-8
func (d *EncodingDetector) hasUTF8MultiByte(data []byte) bool {
	count := 0
	i := 0
	for i < len(data) {
		b := data[i]
		if b >= 0xE0 && b <= 0xEF {
			// 3-byte UTF-8 sequence
			if i+2 < len(data) && data[i+1] >= 0x80 && data[i+1] <= 0xBF &&
				data[i+2] >= 0x80 && data[i+2] <= 0xBF {
				count++
				if count >= 3 {
					return true
				}
				i += 3
				continue
			}
		}
		i++
	}
	return false
}

// GetDecoder 获取编码对应的解码器
func GetDecoder(enc EncodingType) (encoding.Encoding, error) {
	switch enc {
	case EncodingUTF8, EncodingUTF8BOM:
		return encoding.Nop, nil
	case EncodingGBK:
		return simplifiedchinese.GBK, nil
	case EncodingGB2312:
		return simplifiedchinese.HZGB2312, nil
	case EncodingUTF16LE:
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), nil
	case EncodingUTF16BE:
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), nil
	case EncodingUTF32LE:
		// UTF-32 not directly supported in golang.org/x/text, use UTF-16 as fallback
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), nil
	case EncodingUTF32BE:
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), nil
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", enc.String())
	}
}

// ConvertToUTF8 将指定编码的数据转换为 UTF-8
func ConvertToUTF8(data []byte, enc EncodingType) ([]byte, error) {
	decoder, err := GetDecoder(enc)
	if err != nil {
		return nil, err
	}

	// 跳过 BOM（如果有）
	start := 0
	if enc == EncodingUTF8BOM && len(data) >= 3 {
		start = 3
	} else if enc == EncodingUTF16LE && len(data) >= 2 {
		start = 2
	} else if enc == EncodingUTF16BE && len(data) >= 2 {
		start = 2
	}

	reader := transform.NewReader(bytes.NewReader(data[start:]), decoder.NewDecoder())
	return io.ReadAll(reader)
}
