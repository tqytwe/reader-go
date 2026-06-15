package pdf

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Parser PDF 解析器
// 使用纯 Go 实现的基础解析，支持文本提取
type Parser struct{}

// NewParser 创建新的 PDF 解析器
func NewParser() *Parser {
	return &Parser{}
}

// SupportedExtensions 返回支持的文件扩展名
func (p *Parser) SupportedExtensions() []string {
	return []string{".pdf"}
}

// Parse 解析 PDF 文件
func (p *Parser) Parse(reader io.Reader, filename string) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read data: %w", err)
	}

	return p.parseBytes(data, filename)
}

// ParseFile 解析 PDF 文件（从文件路径）
func (p *Parser) ParseFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return p.parseBytes(data, filepath.Base(filePath))
}

// parseBytes 从字节数据解析 PDF
func (p *Parser) parseBytes(data []byte, filename string) (string, error) {
	// 简单的文本提取：查找 PDF 中的文本流
	var sb strings.Builder

	// 提取标题（从文件名）
	title := strings.TrimSuffix(filename, filepath.Ext(filename))
	sb.WriteString(fmt.Sprintf("文档: %s\n\n", title))

	// 尝试提取文本内容
	texts := extractTextFromPDF(data)
	if len(texts) > 0 {
		sb.WriteString("内容:\n")
		for _, t := range texts {
			if t != "" {
				sb.WriteString(t)
				sb.WriteString("\n")
			}
		}
	} else {
		sb.WriteString("(PDF 内容无法直接提取，请使用阅读器查看)")
	}

	return sb.String(), nil
}

// extractTextFromPDF 从 PDF 数据中提取文本
// 这是一个基础实现，完整实现需要更复杂的 PDF 解析
func extractTextFromPDF(data []byte) []string {
	var texts []string

	// 查找文本流 (BT...ET 块)
	start := 0
	for {
		btIdx := bytes.Index(data[start:], []byte("BT"))
		if btIdx == -1 {
			break
		}
		btIdx += start

		etIdx := bytes.Index(data[btIdx:], []byte("ET"))
		if etIdx == -1 {
			break
		}
		etIdx += btIdx

		// 提取这个文本块中的文本
		textBlock := data[btIdx:etIdx]
		text := extractTextFromBlock(textBlock)
		if text != "" {
			texts = append(texts, text)
		}

		start = etIdx
	}

	return texts
}

// extractTextFromBlock 从 PDF 文本块中提取文本
func extractTextFromBlock(block []byte) string {
	var sb strings.Builder
	i := 0

	for i < len(block) {
		// 查找 Tj 或 TJ 操作符
		if i+2 < len(block) && block[i] == '<' && block[i+2] == '>' {
			// 十六进制编码的字符串 <...>
			end := bytes.Index(block[i:], []byte(">"))
			if end == -1 || end < 3 {
				i++
				continue
			}
			hexStr := string(block[i+1 : i+end])
			text := decodeHexString(hexStr)
			if text != "" {
				sb.WriteString(text)
			}
			i += end + 1
		} else if i+1 < len(block) && block[i] == '(' && block[i+1] != ')' {
			// 括号编码的字符串
			end := findMatchingParen(block[i+1:])
			if end > 0 {
				text := string(block[i+1 : i+1+end])
				text = decodeEscapeString(text)
				if text != "" {
					sb.WriteString(text)
				}
				i += end + 2
				continue
			}
		} else if i+1 < len(block) && block[i] == 'T' {
			// 检查 Tj 或 TJ
			if block[i+1] == 'j' || (block[i+1] == 'J' && i+2 < len(block) && block[i+2] == '<') {
				// 跳过 Tj/J
				i += 2
				continue
			}
		}

		i++
	}

	return sb.String()
}

// findMatchingParen 查找匹配的右括号
func findMatchingParen(data []byte) int {
	depth := 1
	for i, b := range data {
		if b == '\\' && i+1 < len(data) {
			i++ // 跳过转义字符
			continue
		}
		if b == '(' {
			depth++
		} else if b == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// decodeHexString 解码十六进制字符串
func decodeHexString(hex string) string {
	// 移除空格和换行
	hex = strings.ReplaceAll(hex, " ", "")
	hex = strings.ReplaceAll(hex, "\n", "")
	hex = strings.ReplaceAll(hex, "\r", "")

	if len(hex)%2 != 0 {
		hex = hex + "0"
	}

	var sb strings.Builder
	for i := 0; i < len(hex); i += 2 {
		var b byte
		fmt.Sscanf(hex[i:i+2], "%02x", &b)
		if b >= 32 && b < 127 {
			sb.WriteByte(b)
		} else if b == 0 {
			sb.WriteByte(' ')
		}
	}

	return sb.String()
}

// decodeEscapeString 解码转义字符串
func decodeEscapeString(s string) string {
	var sb strings.Builder
	escapeMap := map[byte]byte{
		'n':  '\n',
		'r':  '\r',
		't':  '\t',
		'\\': '\\',
		'(':  '(',
		')':  ')',
	}

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			if repl, ok := escapeMap[s[i+1]]; ok {
				sb.WriteByte(repl)
				i++
				continue
			}
		}
		if s[i] >= 32 && s[i] < 127 {
			sb.WriteByte(s[i])
		}
	}

	return sb.String()
}

// GetMetadata 获取 PDF 元数据
func (p *Parser) GetMetadata(filePath string) (map[string]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	metadata["title"] = filepath.Base(filePath)
	metadata["pageCount"] = estimatePageCount(data)

	return metadata, nil
}

// estimatePageCount 估算 PDF 页数
func estimatePageCount(data []byte) string {
	// 通过计算 /Type /Page 出现的次数来估算
	count := bytes.Count(data, []byte("/Type /Page"))
	count += bytes.Count(data, []byte("/Type/Page"))
	if count == 0 {
		count = bytes.Count(data, []byte("%%Page"))
	}
	if count == 0 {
		return "1"
	}
	return fmt.Sprintf("%d", count)
}

// GetPageCount 获取页数
func (p *Parser) GetPageCount(filePath string) (int, error) {
	metadata, err := p.GetMetadata(filePath)
	if err != nil {
		return 0, err
	}

	var count int
	fmt.Sscanf(metadata["pageCount"], "%d", &count)
	return count, nil
}

// ParseBuffer 从字节 buffer 解析
func (p *Parser) ParseBuffer(data []byte) (string, error) {
	return p.parseBytes(data, "document.pdf")
}