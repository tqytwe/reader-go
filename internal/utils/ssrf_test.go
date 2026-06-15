package utils

import (
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		// 允许的 URL
		{"正常 HTTPS", "https://www.google.com/search", false, ""},
		{"正常 HTTP", "http://example.com/path", false, ""},
		{"带端口", "https://api.example.com:8080/v1", false, ""},

		// 拒绝 file://
		{"file 协议", "file:///etc/passwd", true, "file://"},
		{"file 协议大写", "FILE:///etc/passwd", true, "file://"},

		// 拒绝 data://
		{"data 协议", "data://text/plain,test", true, "data://"},

		// 拒绝非 http/https
		{"ftp 协议", "ftp://ftp.example.com/file", true, "ftp"},
		{"gopher 协议", "gopher://localhost:70", true, "gopher"},

		// 拒绝 localhost
		{"localhost", "http://localhost:8080/api", true, "localhost"},
		{"localhost.", "http://localhost./api", true, "localhost"},
		{"127.0.0.1", "http://127.0.0.1:8080/api", true, "127.0.0.1"},
		{"127.0.0.1 https", "https://127.0.0.1:8080", true, "127.0.0.1"},

		// 拒绝私有 IP
		{"10.x.x.x", "http://10.0.0.1:8080/api", true, "private"},
		{"192.168.x.x", "http://192.168.1.1:8080", true, "private"},
		{"172.16.x.x", "http://172.16.0.1:8080", true, "private"},
		{"172.31.x.x", "http://172.31.255.255:8080", true, "private"},

		// 拒绝链路本地
		{"169.254.x.x", "http://169.254.169.254/latest/meta-data/", true, "private"},

		// 空 URL
		{"空 URL", "", true, "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateURL(%q) expected error, got nil", tt.url)
					return
				}
				if tt.errMsg != "" && !containsStr(err.Error(), tt.errMsg) {
					t.Errorf("ValidateURL(%q) error = %v, want %q", tt.url, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateURL(%q) unexpected error: %v", tt.url, err)
				}
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	privateIPs := []string{
		"10.0.0.1",
		"10.255.255.255",
		"172.16.0.1",
		"172.31.255.255",
		"192.168.0.1",
		"192.168.255.255",
		"127.0.0.1",
		"127.0.0.255",
		"::1",
		"169.254.0.1",
		"169.254.169.254",
		"100.64.0.1",
		"fc00::1",
	}

	publicIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"203.0.113.1",
		"198.51.100.1",
		"172.15.255.255", // 不在 172.16/12 范围内
		"172.32.0.1",     // 不在 172.16/12 范围内
	}

	for _, ip := range privateIPs {
		if !isPrivateIP(ip) {
			t.Errorf("isPrivateIP(%q) = false, want true", ip)
		}
	}

	for _, ip := range publicIPs {
		if isPrivateIP(ip) {
			t.Errorf("isPrivateIP(%q) = true, want false", ip)
		}
	}

	// 非 IP 字符串应返回 false
	if isPrivateIP("google.com") {
		t.Error("isPrivateIP(\"google.com\") = true, want false")
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
