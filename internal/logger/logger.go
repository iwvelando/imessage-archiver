package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type Logger struct {
	level LogLevel
}

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func New(levelStr string) *Logger {
	level := parseLogLevel(levelStr)
	return &Logger{level: level}
}

func parseLogLevel(levelStr string) LogLevel {
	switch strings.ToLower(levelStr) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

func (l *Logger) Debug(msg string) {
	if l.level <= DEBUG {
		log.Printf("[DEBUG] %s", msg)
	}
}

func (l *Logger) Info(msg string) {
	if l.level <= INFO {
		log.Printf("[INFO] %s", msg)
	}
}

func (l *Logger) Warn(msg string) {
	if l.level <= WARN {
		log.Printf("[WARN] %s", msg)
	}
}

func (l *Logger) Error(msg string) {
	if l.level <= ERROR {
		log.Printf("[ERROR] %s", msg)
		fmt.Fprintf(os.Stderr, "[ERROR] %s\n", msg)
	}
}
