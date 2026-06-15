package shelf

import "time"

// ShelfBook 书架中的书籍
type ShelfBook struct {
	ID           int64     `json:"id" db:"id"`
	BookKey      string    `json:"bookKey" db:"book_key"`            // 书籍唯一标识（来源ID+书名等）
	Name         string    `json:"name" db:"name"`                   // 书名
	Author       string    `json:"author" db:"author"`               // 作者
	CoverURL     string    `json:"coverUrl" db:"cover_url"`          // 封面URL
	Summary      string    `json:"summary" db:"summary"`             // 简介
	SourceID     int64     `json:"sourceId" db:"source_id"`          // 来源ID
	SourceName   string    `json:"sourceName" db:"source_name"`      // 来源名称
	// 阅读进度
	CurrentChapter string  `json:"currentChapter" db:"current_chapter"` // 当前章节
	LastReadAt     time.Time `json:"lastReadAt" db:"last_read_at"`      // 最后阅读时间
	ReadCount      int     `json:"readCount" db:"read_count"`           // 阅读次数
	// 用户自定义
	Note           string  `json:"note" db:"note"`                      // 笔记
	Order          int     `json:"order" db:"order"`                    // 排序
	// 元数据
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

// ShelfStats 书架统计
type ShelfStats struct {
	Total       int `json:"total"`
	Reading     int `json:"reading"`     // 阅读中（有进度）
	Unread      int `json:"unread"`      // 未读
	Completed   int `json:"completed"`   // 已读完
}
