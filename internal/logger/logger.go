package logger

import (
	"io/fs"
	"log"
	"os"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
)

type Logger struct {
	level LogLevel
}

func NewLogger(level LogLevel) *Logger {
	return &Logger{level}
}

func SetOutputToFile() (*os.File, error) {
	fileName := "app.log"
	flag := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	fileMode := fs.FileMode(0644)
	logFile, err := os.OpenFile(fileName, flag, fileMode)
	if err != nil {
		return nil, err
	}
	log.SetOutput(logFile)
	return logFile, nil
}

func (l *Logger) Debug(message string) {
	if l.level <= LogLevelDebug {
		log.Println("[DEBUG] " + message)
	}
}

func (l *Logger) Info(message string) {
	if l.level <= LogLevelInfo {
		log.Println("[INFO] " + message)
	}
}

func (l *Logger) Warning(message string) {
	if l.level <= LogLevelWarning {
		log.Println("[WARNING] " + message)
	}
}

func (l *Logger) Error(message string) {
	if l.level <= LogLevelError {
		log.Println("[ERROR] " + message)
	}
}
