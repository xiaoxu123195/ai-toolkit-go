package util

import (
	"log"
	"net/http"
	"Zai/internal/config"
)

// debug日志函数
func DebugLog(format string, args ...interface{}) {
	if config.DEBUG_MODE {
		log.Printf("[DEBUG] "+format, args...)
	}
}

func SetCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}
