package utils

import (
	"bytes"
	"io"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// DecodeBodyToUTF8 按响应头与页面内容探测编码，并统一转换为 UTF-8。
func DecodeBodyToUTF8(data []byte, contentType string) string {
	if len(data) == 0 {
		return ""
	}

	enc, _, _ := charset.DetermineEncoding(data, contentType)
	reader := transform.NewReader(bytes.NewReader(data), enc.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return string(data)
	}
	return string(decoded)
}
