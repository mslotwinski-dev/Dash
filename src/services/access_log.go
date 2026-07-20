package services

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

type AccessLogEntry struct {
	Timestamp string  `json:"timestamp"`
	ClientIP  string  `json:"client_ip"`
	Method    string  `json:"method"`
	Path      string  `json:"path"`
	Status    int     `json:"status"`
	Latency   float64 `json:"latency_ms"`
	UserAgent string  `json:"user_agent"`
}

var accessLogFile *os.File

func InitAccessLog() {
	err := os.MkdirAll("logs", 0755)
	if err != nil {
		log.Println("Błąd tworzenia folderu logs:", err)
		return
	}

	logPath := filepath.Join("logs", "access.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Println("Błąd otwarcia pliku access.log:", err)
		return
	}
	accessLogFile = file
}

func LogAccess(clientIP, method, path string, status int, latency time.Duration, userAgent string) {
	if accessLogFile == nil {
		return
	}

	entry := AccessLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		ClientIP:  clientIP,
		Method:    method,
		Path:      path,
		Status:    status,
		Latency:   float64(latency.Microseconds()) / 1000.0,
		UserAgent: userAgent,
	}

	data, err := json.Marshal(entry)
	if err == nil {
		data = append(data, '\n')
		accessLogFile.Write(data)
	}
}

func CloseAccessLog() {
	if accessLogFile != nil {
		accessLogFile.Close()
	}
}
