package booksource

import (
	"encoding/json"
	"strings"
)

// EnablePolicyResult 合集导入启用策略统计
type EnablePolicyResult struct {
	Enabled    int `json:"enabled"`
	Disabled   int `json:"disabled"`
	JSRequired int `json:"jsRequired"`
	NonJS      int `json:"nonJs"`
}

// ContainsJSRule 判断规则字符串是否依赖 Legado JS
func ContainsJSRule(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.Contains(strings.ToLower(s), "<js>") {
		return true
	}
	return strings.Contains(s, "@js:")
}

// RequiresWebView 判断 URL 配置是否需要 WebView（当前不支持）
func RequiresWebView(s string) bool {
	return strings.Contains(s, "webView")
}

// RequiresJSSearchLegacy 判断 Legado 书源搜索链路是否依赖 JS / WebView
func RequiresJSSearchLegacy(legacy *LegacyBookSource) bool {
	if legacy == nil {
		return false
	}
	if RequiresWebView(legacy.SearchURL) || ContainsJSRule(legacy.SearchURL) {
		return true
	}
	return ruleInterfaceContainsJS(legacy.RuleSearch)
}

// RequiresJSSearch 判断已转换书源的搜索链路是否依赖 JS / WebView
func RequiresJSSearch(bs *BookSource) bool {
	if bs == nil {
		return false
	}
	if RequiresWebView(bs.SearchURL) || ContainsJSRule(bs.SearchURL) {
		return true
	}
	return ContainsJSRule(bs.SearchRule)
}

// ApplyEnablePolicy 按策略设置书源启用状态并返回统计
// enableOnlyNonJS 为 true 时，仅启用搜索链路不依赖 JS 的源
func ApplyEnablePolicy(sources []*BookSource, enableOnlyNonJS bool) EnablePolicyResult {
	result := EnablePolicyResult{}
	for _, bs := range sources {
		if bs == nil {
			continue
		}
		js := RequiresJSSearch(bs)
		if js {
			result.JSRequired++
		} else {
			result.NonJS++
		}
		if enableOnlyNonJS {
			bs.Enabled = !js
		}
		if bs.Enabled {
			result.Enabled++
		} else {
			result.Disabled++
		}
	}
	return result
}

func ruleInterfaceContainsJS(rule interface{}) bool {
	if rule == nil {
		return false
	}
	switch v := rule.(type) {
	case string:
		return ContainsJSRule(v)
	case map[string]interface{}:
		for _, val := range v {
			if str, ok := val.(string); ok && ContainsJSRule(str) {
				return true
			}
		}
	default:
		b, err := json.Marshal(v)
		if err == nil {
			return ContainsJSRule(string(b))
		}
	}
	return false
}
