package booksource

// BookSourceDTO API 层书源对象（与 Legado JSON 字段对齐）
type BookSourceDTO = BookSource

// ToDTO 转换为 API DTO（当前与模型一致）
func ToDTO(bs *BookSource) *BookSourceDTO {
	if bs == nil {
		return nil
	}
	cp := *bs
	return &cp
}

// FromDTO 从 API 请求填充模型
func FromDTO(dto *BookSourceDTO) *BookSource {
	if dto == nil {
		return nil
	}
	cp := *dto
	return &cp
}
