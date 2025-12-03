package debuglog

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"glm-tool/config"

	"github.com/gophertool/tool/log"
)

type DebugEntry struct {
	Timestamp string         `json:"timestamp"`
	Request   map[string]any `json:"request"`
	Response  map[string]any `json:"response"`
	Error     string         `json:"error,omitempty"`
}

var (
	mutex      sync.Mutex
	logEntries []DebugEntry
)

func LogRequest(request map[string]any, response map[string]any, err error) {
	if !config.AppConfig.Debug {
		return
	}

	entry := DebugEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Request:   request,
		Response:  response,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	mutex.Lock()
	defer mutex.Unlock()

	logEntries = append(logEntries, entry)

	if err := writeToFile(); err != nil {
		log.Warnf("写入 debug 日志失败: %v", err)
	}
}

func writeToFile() error {
	data, err := json.MarshalIndent(logEntries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(config.AppConfig.DebugLogFile, data, 0644)
}

func GetEntries() []DebugEntry {
	mutex.Lock()
	defer mutex.Unlock()
	return logEntries
}

func ClearEntries() {
	mutex.Lock()
	defer mutex.Unlock()
	logEntries = []DebugEntry{}
	os.Remove(config.AppConfig.DebugLogFile)
}
