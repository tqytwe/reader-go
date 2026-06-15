package web

import "log"

// LogRule 结构化规则/书源日志
func LogRule(sourceID int64, step string, latencyMs int64, err error) {
	if err != nil {
		log.Printf(`{"sourceId":%d,"step":%q,"latencyMs":%d,"error":%q}`, sourceID, step, latencyMs, err.Error())
		return
	}
	log.Printf(`{"sourceId":%d,"step":%q,"latencyMs":%d}`, sourceID, step, latencyMs)
}

// logError 记录内部错误到服务端日志，不向客户端暴露详情
func logError(err error) {
	if err != nil {
		log.Printf("[ERROR] internal: %v", err)
	}
}
