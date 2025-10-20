// pkg/logger/logger.go
package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger for structured logging
type Logger struct {
	*zap.SugaredLogger
	level zap.AtomicLevel
}

// NewLogger creates a new structured logger
func NewLogger() *Logger {
	// Set up log level
	level := zap.NewAtomicLevel()
	
	// Default to info level, can be changed via environment variable
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		switch logLevel {
		case "debug":
			level.SetLevel(zap.DebugLevel)
		case "warn":
			level.SetLevel(zap.WarnLevel)
		case "error":
			level.SetLevel(zap.ErrorLevel)
		default:
			level.SetLevel(zap.InfoLevel)
		}
	} else {
		level.SetLevel(zap.InfoLevel)
	}

	// Configure encoder
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Set up outputs
	var output io.Writer = os.Stdout
	
	// In production, write to file and stdout
	if os.Getenv("ENVIRONMENT") == "production" {
		logDir := "logs"
		if err := os.MkdirAll(logDir, 0755); err == nil {
			logFile := filepath.Join(logDir, "url-shortener.log")
			file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				output = io.MultiWriter(os.Stdout, file)
			}
		}
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(output),
		level,
	)

	// Add caller information in development
	development := os.Getenv("ENVIRONMENT") == "development"
	var zapLogger *zap.Logger
	if development {
		zapLogger = zap.New(core, zap.AddCaller(), zap.Development())
	} else {
		zapLogger = zap.New(core)
	}

	return &Logger{
		SugaredLogger: zapLogger.Sugar(),
		level:         level,
	}
}

// GetStandardLogger returns a standard library logger (simplified for GORM)
func (l *Logger) GetStandardLogger() *log.Logger {
	// Return a simple stdlib logger that GORM can use
	return log.New(os.Stdout, "[GORM] ", log.LstdFlags)
}

// WithFields adds structured fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		zapFields = append(zapFields, k, v)
	}
	
	return &Logger{
		SugaredLogger: l.SugaredLogger.With(zapFields...),
		level:         l.level,
	}
}

// SetLevel dynamically changes the log level
func (l *Logger) SetLevel(level string) {
	switch level {
	case "debug":
		l.level.SetLevel(zap.DebugLevel)
	case "info":
		l.level.SetLevel(zap.InfoLevel)
	case "warn":
		l.level.SetLevel(zap.WarnLevel)
	case "error":
		l.level.SetLevel(zap.ErrorLevel)
	}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() {
	_ = l.SugaredLogger.Sync()
}