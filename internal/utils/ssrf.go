package utils

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateURL 校验目标 URL，防止 SSRF 攻击
// 拒绝以下类型的 URL:
//   - 私有 IP 地址（10.x, 172.16-31.x, 192.168.x）
//   - 回环地址（127.x, ::1, localhost）
//   - 链路本地地址（169.254.x）
//   - 文件协议（file://）
//   - 元数据端点（169.254.169.254）
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL is empty")
	}

	// 拒绝 file:// 协议
	if strings.HasPrefix(strings.ToLower(rawURL), "file://") {
		return fmt.Errorf("file:// protocol is not allowed")
	}

	// 拒绝 data:// 协议
	if strings.HasPrefix(strings.ToLower(rawURL), "data://") {
		return fmt.Errorf("data:// protocol is not allowed")
	}

	// 解析 URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// 只允许 HTTP/HTTPS 协议
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http/https protocol allowed, got: %s", u.Scheme)
	}

	// 拒绝空主机
	if u.Host == "" {
		return fmt.Errorf("URL has no host")
	}

	host := u.Hostname()

	// 拒绝 localhost
	if host == "localhost" || host == "localhost." {
		return fmt.Errorf("localhost is not allowed")
	}

	// 解析 IP 地址
	ips, err := net.LookupIP(host)
	if err != nil {
		// DNS 解析失败时，检查是否是明显的内网 IP 格式
		if isPrivateIP(host) {
			return fmt.Errorf("private IP address is not allowed: %s", host)
		}
		// DNS 解析失败但看起来是公网域名则允许
		return nil
	}

	// 检查所有解析结果，确保没有指向私有地址
	for _, ip := range ips {
		if isPrivateIP(ip.String()) {
			return fmt.Errorf("private IP address is not allowed: %s (%s)", host, ip)
		}
	}

	return nil
}

// isPrivateIP 判断 IP 是否为私有地址
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		// 不是 IP 地址格式，可能是域名
		return false
	}

	// 回环地址
	if ip.IsLoopback() {
		return true
	}

	// 未指定地址
	if ip.IsUnspecified() {
		return true
	}

	// 链路本地地址
	if ip.IsLinkLocalUnicast() {
		return true
	}

	// 链路本地多播
	if ip.IsLinkLocalMulticast() {
		return true
	}

	// 私有地址段
	privateNets := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10", // CGNAT
		"fc00::/7",      // 唯一本地地址
	}

	for _, cidr := range privateNets {
		_, net, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if net.Contains(ip) {
			return true
		}
	}

	return false
}
