package replace

import "time"

// ReplaceRule 替换规则实体
type ReplaceRule struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`           // 规则名称
	Pattern   string    `json:"pattern" db:"pattern"`     // 正则匹配模式
	Replacement string  `json:"replacement" db:"replacement"` // 替换内容
	// 作用范围
	Scope       string    `json:"scope" db:"scope"`       // 作用范围: all|title|content|toc|search
	// 选项
	CaseInsensitive bool  `json:"caseInsensitive" db:"case_insensitive"` // 忽略大小写
	Enabled       bool    `json:"enabled" db:"enabled"`                 // 是否启用
	Order         int     `json:"order" db:"order"`                     // 排序权重
	// 元数据
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time `json:"updatedAt" db:"updated_at"`
}

// RuleScope 作用范围常量
const (
	ScopeAll     = "all"
	ScopeTitle   = "title"
	ScopeContent = "content"
	ScopeToc     = "toc"
	ScopeSearch  = "search"
)
