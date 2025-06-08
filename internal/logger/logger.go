package logger

import (
	"log"
	"os"
	"strings"
)

type Logger struct {
	level       LogLevel
	infoLogger  *log.Logger // For DEBUG and INFO - goes to stdout
	errorLogger *log.Logger // For WARN and ERROR - goes to stderr
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
	return &Logger{
		level:       level,
		infoLogger:  log.New(os.Stdout, "", log.LstdFlags),
		errorLogger: log.New(os.Stderr, "", log.LstdFlags),
	}
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
		l.infoLogger.Printf("[DEBUG] %s", msg)
	}
}

func (l *Logger) Info(msg string) {
	if l.level <= INFO {
		l.infoLogger.Printf("[INFO] %s", msg)
	}
}

func (l *Logger) Warn(msg string) {
	if l.level <= WARN {
		l.errorLogger.Printf("[WARN] %s", msg)
	}
}

func (l *Logger) Error(msg string) {
	if l.level <= ERROR {
		l.errorLogger.Printf("[ERROR] %s", msg)
	}
}
