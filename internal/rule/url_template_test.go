package rule

import (
	"net/http"
	"reflect"
	"testing"
)

// ==================== ParseURLOptions 测试 ====================

func TestParseURLOptions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantURL     string
		wantOptions string
	}{
		{
			name:        "无选项",
			input:       "https://example.com/api",
			wantURL:     "https://example.com/api",
			wantOptions: "",
		},
		{
			name:        "简单选项",
			input:       "https://example.com/api, {\"method\":\"POST\"}",
			wantURL:     "https://example.com/api",
			wantOptions: `{"method":"POST"}`,
		},
		{
			name:        "复杂选项",
			input:       "https://example.com/api, {\"method\":\"POST\", \"headers\":{\"User-Agent\":\"Reader\"}, \"retry\":3}",
			wantURL:     "https://example.com/api",
			wantOptions: `{"method":"POST", "headers":{"User-Agent":"Reader"}, "retry":3}`,
		},
		{
			name:        "选项前有空格",
			input:       "https://example.com/api , {\"method\":\"GET\"}",
			wantURL:     "https://example.com/api",
			wantOptions: `{"method":"GET"}`,
		},
		{
			name:        "URL 含占位符",
			input:       "https://example.com/api?book={{bookId}}, {\"method\":\"POST\"}",
			wantURL:     "https://example.com/api?book={{bookId}}",
			wantOptions: `{"method":"POST"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotOpts := parseURLOptions(tt.input)
			if gotURL != tt.wantURL {
				t.Errorf("parseURLOptions() url = %q, want %q", gotURL, tt.wantURL)
			}
			if gotOpts != tt.wantOptions {
				t.Errorf("parseURLOptions() options = %q, want %q", gotOpts, tt.wantOptions)
			}
		})
	}
}

// ==================== parsePageMultiSelect 测试 ====================

func TestParsePageMultiSelect(t *testing.T) {
	tests := []struct {
		name string
		input string
		want []int
	}{
		{
			name:  "简单序列",
			input: "1,2,3",
			want:  []int{1, 2, 3},
		},
		{
			name:  "带空格",
			input: "1, 2, 3",
			want:  []int{1, 2, 3},
		},
		{
			name:  "单值",
			input: "1",
			want:  []int{1},
		},
		{
			name:  "大数值",
			input: "1,2,10",
			want:  []int{1, 2, 10},
		},
		{
			name:  "空字符串",
			input: "",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePageMultiSelect(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePageMultiSelect(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ==================== parseURLOptionsJSON 测试 ====================

func TestParseURLOptionsJSON(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    *URLOptions
		wantErr bool
	}{
		{
			name:    "标准 JSON",
			jsonStr: `{"method":"POST","headers":{"User-Agent":"Reader"},"retry":3}`,
			want: &URLOptions{
				Method:  "POST",
				Headers: map[string]string{"User-Agent": "Reader"},
				Retry:   3,
			},
		},
		{
			name:    "带 body",
			jsonStr: `{"method":"POST","body":"{\"key\":\"value\"}"}`,
			want: &URLOptions{
				Method: "POST",
				Body:   `{"key":"value"}`,
			},
		},
		{
			name:    "带 charset",
			jsonStr: `{"method":"GET","charset":"utf-8"}`,
			want: &URLOptions{
				Method:  "GET",
				Charset: "utf-8",
			},
		},
		{
			name:    "无 method 默认 GET",
			jsonStr: `{"headers":{"Accept":"text/html"}}`,
			want: &URLOptions{
				Method:  http.MethodGet,
				Headers: map[string]string{"Accept": "text/html"},
			},
		},
		{
			name:    "空对象",
			jsonStr: `{}`,
			want: &URLOptions{
				Method: http.MethodGet,
			},
		},
		{
			name:    "宽松格式 - 单引号",
			jsonStr: `{'method':'POST','retry':5}`,
			want: &URLOptions{
				Method: "POST",
				Retry:  5,
			},
		},
		{
			name:    "宽松格式 - headers 单引号",
			jsonStr: `{'headers':{'User-Agent':'Reader','Accept':'text/html'}}`,
			want: &URLOptions{
				Method:  http.MethodGet,
				Headers: map[string]string{"User-Agent": "Reader", "Accept": "text/html"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseURLOptionsJSON(tt.jsonStr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseURLOptionsJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseURLOptionsJSON() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// ==================== calculatePageValue 测试 ====================

func TestCalculatePageValue(t *testing.T) {
	tests := []struct {
		name      string
		pageMulti []int
		page      int
		want      int
	}{
		{
			name:      "无页数多选",
			pageMulti: nil,
			page:      1,
			want:      1,
		},
		{
			name:      "空数组",
			pageMulti: []int{},
			page:      1,
			want:      1,
		},
		{
			name:      "第1页",
			pageMulti: []int{1, 2, 3},
			page:      1,
			want:      1,
		},
		{
			name:      "第2页",
			pageMulti: []int{1, 2, 3},
			page:      2,
			want:      2,
		},
		{
			name:      "第3页",
			pageMulti: []int{1, 2, 3},
			page:      3,
			want:      3,
		},
		{
			name:      "第4页（超出范围，用最后一个）",
			pageMulti: []int{1, 2, 3},
			page:      4,
			want:      3,
		},
		{
			name:      "第10页",
			pageMulti: []int{1, 2, 3},
			page:      10,
			want:      3,
		},
		{
			name:      "单值数组",
			pageMulti: []int{1},
			page:      5,
			want:      1,
		},
		{
			name:      "零页",
			pageMulti: []int{1, 2, 3},
			page:      0,
			want:      1,
		},
		{
			name:      "负页",
			pageMulti: []int{1, 2, 3},
			page:      -1,
			want:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePageValue(tt.pageMulti, tt.page)
			if got != tt.want {
				t.Errorf("calculatePageValue(%v, %d) = %d, want %d", tt.pageMulti, tt.page, got, tt.want)
			}
		})
	}
}

// ==================== ParseURLTemplate 测试 ====================

func TestParseURLTemplate(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *URLTemplate
		wantErr bool
	}{
		{
			name:    "空字符串",
			url:     "",
			wantErr: true,
		},
		{
			name:    "纯 URL 无模板",
			url:     "https://example.com/api",
			want: &URLTemplate{
				Raw:      "https://example.com/api",
				BaseURL:  "https://example.com/api",
				Options:  &URLOptions{Method: http.MethodGet},
			},
		},
		{
			name:    "含占位符",
			url:     "https://example.com/api?book={{bookId}}&page={{page}}",
			want: &URLTemplate{
				Raw:         "https://example.com/api?book={{bookId}}&page={{page}}",
				BaseURL:     "https://example.com/api?book=&page=",
				Placeholders: []string{"bookId", "page"},
				Options:     &URLOptions{Method: http.MethodGet},
			},
		},
		{
			name:    "含 JS 注入",
			url:     "https://example.com/api<js>generateUrl()</js>",
			want: &URLTemplate{
				Raw:           "https://example.com/api<js>generateUrl()</js>",
				BaseURL:       "https://example.com/api",
				JSInjections:  []string{"generateUrl()"},
				Options:       &URLOptions{Method: http.MethodGet},
			},
		},
		{
			name:    "含页数多选",
			url:     "https://example.com/api?page=<page1,2,3>",
			want: &URLTemplate{
				Raw:             "https://example.com/api?page=<page1,2,3>",
				BaseURL:         "https://example.com/api?page=",
				PageMultiSelect: []int{1, 2, 3},
				Options:         &URLOptions{Method: http.MethodGet},
			},
		},
		{
			name:    "含 URL 选项",
			url:     "https://example.com/api, {\"method\":\"POST\", \"headers\":{\"User-Agent\":\"Reader\"}, \"retry\":3}",
			want: &URLTemplate{
				Raw:     "https://example.com/api, {\"method\":\"POST\", \"headers\":{\"User-Agent\":\"Reader\"}, \"retry\":3}",
				BaseURL: "https://example.com/api",
				Options: &URLOptions{
					Method:  "POST",
					Headers: map[string]string{"User-Agent": "Reader"},
					Retry:   3,
				},
			},
		},
		{
			name:    "全部语法混合",
			url:     "https://example.com/api?book={{bookId}}&p={{page}}<page1,2,3><js>customEncode()</js>, {\"method\":\"POST\", \"headers\":{\"X-Custom\":\"Value\"}, \"body\":\"data=1\", \"charset\":\"utf-8\", \"retry\":5}",
			want: &URLTemplate{
				Raw:             "https://example.com/api?book={{bookId}}&p={{page}}<page1,2,3><js>customEncode()</js>, {\"method\":\"POST\", \"headers\":{\"X-Custom\":\"Value\"}, \"body\":\"data=1\", \"charset\":\"utf-8\", \"retry\":5}",
				BaseURL:         "https://example.com/api?book=&p=",
				Placeholders:    []string{"bookId", "page"},
				PageMultiSelect: []int{1, 2, 3},
				JSInjections:    []string{"customEncode()"},
				Options: &URLOptions{
					Method:  "POST",
					Headers: map[string]string{"X-Custom": "Value"},
					Body:    "data=1",
					Charset: "utf-8",
					Retry:   5,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseURLTemplate(tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseURLTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			if got.Raw != tt.want.Raw {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.want.Raw)
			}
			if got.BaseURL != tt.want.BaseURL {
				t.Errorf("BaseURL = %q, want %q", got.BaseURL, tt.want.BaseURL)
			}
			if !reflect.DeepEqual(got.Placeholders, tt.want.Placeholders) {
				t.Errorf("Placeholders = %v, want %v", got.Placeholders, tt.want.Placeholders)
			}
			if !reflect.DeepEqual(got.JSInjections, tt.want.JSInjections) {
				t.Errorf("JSInjections = %v, want %v", got.JSInjections, tt.want.JSInjections)
			}
			if !reflect.DeepEqual(got.PageMultiSelect, tt.want.PageMultiSelect) {
				t.Errorf("PageMultiSelect = %v, want %v", got.PageMultiSelect, tt.want.PageMultiSelect)
			}
			if !reflect.DeepEqual(got.Options, tt.want.Options) {
				t.Errorf("Options = %+v, want %+v", got.Options, tt.want.Options)
			}
		})
	}
}

// ==================== URLTemplate.Evaluate 测试 ====================

func TestURLTemplate_Evaluate(t *testing.T) {
	t.Run("简单 URL 无模板", func(t *testing.T) {
		tmpl, err := ParseURLTemplate("https://example.com/api")
		if err != nil {
			t.Fatalf("ParseURLTemplate() error = %v", err)
		}

		result, err := tmpl.Evaluate(nil, 1)
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}

		if result.URL != "https://example.com/api" {
			t.Errorf("URL = %q, want %q", result.URL, "https://example.com/api")
		}
		if result.Method != http.MethodGet {
			t.Errorf("Method = %q, want %q", result.Method, http.MethodGet)
		}
	})

	t.Run("占位符替换", func(t *testing.T) {
		tmpl, err := ParseURLTemplate("https://example.com/api?book={{bookId}}&page={{page}}")
		if err != nil {
			t.Fatalf("ParseURLTemplate() error = %v", err)
		}

		result, err := tmpl.Evaluate(map[string]string{"bookId": "12345"}, 2)
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}

		if result.URL != "https://example.com/api?book=12345&page=2" {
			t.Errorf("URL = %q, want %q", result.URL, "https://example.com/api?book=12345&page=2")
		}
		if result.Params["bookId"] != "12345" {
			t.Errorf("Params[bookId] = %q, want %q", result.Params["bookId"], "12345")
		}
	})

	t.Run("页数多选", func(t *testing.T) {
		tmpl, err := ParseURLTemplate("https://example.com/api?page=<page1,2,3>")
		if err != nil {
			t.Fatalf("ParseURLTemplate() error = %v", err)
		}

		// 第1页
		result1, _ := tmpl.Evaluate(nil, 1)
		if result1.PageValue != 1 {
			t.Errorf("page=1: PageValue = %d, want 1", result1.PageValue)
		}

		// 第2页
		result2, _ := tmpl.Evaluate(nil, 2)
		if result2.PageValue != 2 {
			t.Errorf("page=2: PageValue = %d, want 2", result2.PageValue)
		}

		// 第3页
		result3, _ := tmpl.Evaluate(nil, 3)
		if result3.PageValue != 3 {
			t.Errorf("page=3: PageValue = %d, want 3", result3.PageValue)
		}

		// 第5页（超出范围，用最后一个）
		result5, _ := tmpl.Evaluate(nil, 5)
		if result5.PageValue != 3 {
			t.Errorf("page=5: PageValue = %d, want 3", result5.PageValue)
		}
	})

	t.Run("URL 选项", func(t *testing.T) {
		tmpl, err := ParseURLTemplate("https://example.com/api, {\"method\":\"POST\", \"headers\":{\"User-Agent\":\"Reader\"}, \"body\":\"test\", \"charset\":\"utf-8\", \"retry\":3}")
		if err != nil {
			t.Fatalf("ParseURLTemplate() error = %v", err)
		}

		result, err := tmpl.Evaluate(nil, 1)
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}

		if result.Method != "POST" {
			t.Errorf("Method = %q, want POST", result.Method)
		}
		if result.Headers["User-Agent"] != "Reader" {
			t.Errorf("Headers[User-Agent] = %q, want Reader", result.Headers["User-Agent"])
		}
		if result.Body != "test" {
			t.Errorf("Body = %q, want test", result.Body)
		}
		if result.Charset != "utf-8" {
			t.Errorf("Charset = %q, want utf-8", result.Charset)
		}
		if result.Retry != 3 {
			t.Errorf("Retry = %d, want 3", result.Retry)
		}
	})

	t.Run("组合测试", func(t *testing.T) {
		tmpl, err := ParseURLTemplate("https://example.com/api?book={{bookId}}&p={{page}}<page1,2,3>, {\"method\":\"POST\", \"headers\":{\"X-Page\":\"{{page}}\"}, \"retry\":2}")
		if err != nil {
			t.Fatalf("ParseURLTemplate() error = %v", err)
		}

		result, err := tmpl.Evaluate(map[string]string{"bookId": "abc"}, 2)
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}

		if result.URL != "https://example.com/api?book=abc&p=2" {
			t.Errorf("URL = %q, want %q", result.URL, "https://example.com/api?book=abc&p=2")
		}
		if result.PageValue != 2 {
			t.Errorf("PageValue = %d, want 2", result.PageValue)
		}
	})

	t.Run("nil template", func(t *testing.T) {
		var tmpl *URLTemplate
		result, err := tmpl.Evaluate(nil, 1)
		if err == nil {
			t.Fatalf("Evaluate() on nil template should return error, got nil")
		}
		if result != nil {
			t.Errorf("Evaluate() on nil template returned non-nil result")
		}
	})
}

// ==================== URLTemplate 辅助方法测试 ====================

func TestURLTemplate_HasJSInjection(t *testing.T) {
	tmpl, _ := ParseURLTemplate("https://example.com/api<js>test()</js>")
	if !tmpl.HasJSInjection() {
		t.Error("HasJSInjection() should return true")
	}

	tmpl2, _ := ParseURLTemplate("https://example.com/api")
	if tmpl2.HasJSInjection() {
		t.Error("HasJSInjection() should return false")
	}
}

func TestURLTemplate_HasPageMultiSelect(t *testing.T) {
	tmpl, _ := ParseURLTemplate("https://example.com/api?page=<page1,2,3>")
	if !tmpl.HasPageMultiSelect() {
		t.Error("HasPageMultiSelect() should return true")
	}

	tmpl2, _ := ParseURLTemplate("https://example.com/api")
	if tmpl2.HasPageMultiSelect() {
		t.Error("HasPageMultiSelect() should return false")
	}
}

func TestURLTemplate_HasOptions(t *testing.T) {
	tmpl, _ := ParseURLTemplate("https://example.com/api, {\"method\":\"POST\"}")
	if !tmpl.HasOptions() {
		t.Error("HasOptions() should return true")
	}

	tmpl2, _ := ParseURLTemplate("https://example.com/api")
	if tmpl2.HasOptions() {
		t.Error("HasOptions() should return false")
	}
}

func TestURLTemplate_GetRequiredParams(t *testing.T) {
	tmpl, _ := ParseURLTemplate("https://example.com/api?book={{bookId}}&page={{page}}&cat={{category}}")
	params := tmpl.GetRequiredParams()
	want := []string{"bookId", "page", "category"}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("GetRequiredParams() = %v, want %v", params, want)
	}
}

// ==================== 真实场景测试 ====================

func TestParseURLTemplate_RealWorld(t *testing.T) {
	// 场景1：笔趣阁风格 - 章节列表页
	tmpl1, err := ParseURLTemplate("https://www.bqg.com/list/{{bookId}}/<page1,2,3>/, {\"method\":\"GET\", \"headers\":{\"User-Agent\":\"Mozilla/5.0\"}}")
	if err != nil {
		t.Fatalf("ParseURLTemplate() error = %v", err)
	}

	result1, err := tmpl1.Evaluate(map[string]string{"bookId": "12345"}, 1)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result1.URL != "https://www.bqg.com/list/12345/" {
		t.Errorf("URL = %q, want %q", result1.URL, "https://www.bqg.com/list/12345/")
	}
	if result1.PageValue != 1 {
		t.Errorf("PageValue = %d, want 1", result1.PageValue)
	}

	// 场景2：API 风格 - 带 JS 注入的 URL
	tmpl2, err := ParseURLTemplate("https://api.example.com/search<js>encodeURIComponent(query)</js>?q={{query}}&page={{page}}, {\"method\":\"POST\", \"headers\":{\"Content-Type\":\"application/json\"}, \"body\":\"{\\\"q\\\":\\\"{{query}}\\\"}\"}")
	if err != nil {
		t.Fatalf("ParseURLTemplate() error = %v", err)
	}

	if !tmpl2.HasJSInjection() {
		t.Error("Should have JS injection")
	}
	if len(tmpl2.JSInjections) != 1 {
		t.Errorf("JSInjections count = %d, want 1", len(tmpl2.JSInjections))
	}
	if tmpl2.JSInjections[0] != "encodeURIComponent(query)" {
		t.Errorf("JSInjections[0] = %q, want %q", tmpl2.JSInjections[0], "encodeURIComponent(query)")
	}

	// 场景3：分页 API - 页数映射
	tmpl3, err := ParseURLTemplate("https://api.example.com/books?page=<page1,2,10>, {\"method\":\"GET\", \"retry\":3}")
	if err != nil {
		t.Fatalf("ParseURLTemplate() error = %v", err)
	}

	// 第1页用1，第2页用2，第3页及以后用10
	r1, _ := tmpl3.Evaluate(nil, 1)
	if r1.PageValue != 1 {
		t.Errorf("page=1: PageValue = %d, want 1", r1.PageValue)
	}
	r2, _ := tmpl3.Evaluate(nil, 2)
	if r2.PageValue != 2 {
		t.Errorf("page=2: PageValue = %d, want 2", r2.PageValue)
	}
	r3, _ := tmpl3.Evaluate(nil, 3)
	if r3.PageValue != 10 {
		t.Errorf("page=3: PageValue = %d, want 10", r3.PageValue)
	}
	r100, _ := tmpl3.Evaluate(nil, 100)
	if r100.PageValue != 10 {
		t.Errorf("page=100: PageValue = %d, want 10", r100.PageValue)
	}
}

// ==================== 边界测试 ====================

func TestParseURLTemplate_BoundaryCases(t *testing.T) {
	// 只有空格
	_, err := ParseURLTemplate("   ")
	if err == nil {
		t.Error("ParseURLTemplate(whitespace only) should return error")
	}

	// 只有占位符
	tmpl, err := ParseURLTemplate("{{key}}")
	if err != nil {
		t.Fatalf("ParseURLTemplate('{{key}}') error = %v", err)
	}
	if len(tmpl.Placeholders) != 1 {
		t.Errorf("Placeholders = %v, want [key]", tmpl.Placeholders)
	}

	// 多个相同占位符
	tmpl2, _ := ParseURLTemplate("https://api.com?x={{id}}&y={{id}}")
	if len(tmpl2.Placeholders) != 2 {
		t.Errorf("Placeholders count = %d, want 2", len(tmpl2.Placeholders))
	}

	// 嵌套占位符（不支持，但不应崩溃）
	_, _ = ParseURLTemplate("https://api.com?{{{{nested}}}}")
	// 这个会被解析为 {{nested}} 和 {{ 的混合

	// 选项 JSON 格式错误
	_, err = ParseURLTemplate("https://api.com, {invalid json}")
	if err == nil {
		t.Error("ParseURLTemplate with invalid JSON should return error")
	}

	// JS 注入为空
	tmpl4, _ := ParseURLTemplate("https://api.com<js></js>")
	if len(tmpl4.JSInjections) != 1 {
		t.Errorf("JSInjections = %v, want [\"\"]", tmpl4.JSInjections)
	}

	// 页数多选为空
	tmpl5, _ := ParseURLTemplate("https://api.com?page=<page>")
	if len(tmpl5.PageMultiSelect) != 0 {
		t.Errorf("PageMultiSelect = %v, want []", tmpl5.PageMultiSelect)
	}
}
