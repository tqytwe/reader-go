package web

import (
	"encoding/json"
	"os"
	"time"
)

const debugLogEnv = "READER_GO_DEBUG_LOG"

// #region agent log
func agentDebugLog(location, message, hypothesisID, runID string, data map[string]interface{}) {
	debugLogPath := os.Getenv(debugLogEnv)
	if debugLogPath == "" {
		return
	}
	payload := map[string]interface{}{
		"location":     location,
		"message":      message,
		"hypothesisId": hypothesisID,
		"runId":        runID,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	f, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, '\n'))
}

// #endregion
