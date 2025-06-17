package utils

import (
	"example.com/config"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"time"
)

// LogLevel represents the severity level of the log entry
type LogLevel string

// Log levels
const (
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	DebugLevel LogLevel = "debug"
)

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	LogDir        string
	MaxFileSize   int64    // Maximum size of log file in bytes (0 = no limit)
	MaxFiles      int      // Maximum number of log files to keep (0 = no limit)
	EnableConsole bool     // Whether to also log to console
	JSONFormat    bool     // Whether to use JSON format
	Level         LogLevel // Minimum log level
}

// Logger is our custom logger that writes to date-based files
type Logger struct {
	config     LoggerConfig
	logger     *slog.Logger
	currentDay string
	logFile    *os.File
	mutex      sync.RWMutex
}

// NewLogger creates a new instance of Logger with default configuration
func NewLogger(loggerConf config.LoggerConfig) (*Logger, error) {
	config := LoggerConfig{
		LogDir:        loggerConf.FilePath,
		MaxFileSize:   int64(loggerConf.MaxSize), // 100MB default
		MaxFiles:      loggerConf.MaxFiles,       // Keep 7 days of logs
		EnableConsole: loggerConf.EnableConsole,
		JSONFormat:    loggerConf.Format == "json",
		Level:         LogLevel(loggerConf.Level),
	}
	return NewLoggerWithConfig(config)
}

// NewLoggerWithConfig creates a new instance of Logger with custom configuration
func NewLoggerWithConfig(config LoggerConfig) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	l := &Logger{
		config: config,
	}

	// Set up initial log file
	if err := l.rotateLogFile(); err != nil {
		return nil, err
	}

	// Clean up old log files
	if err := l.cleanupOldLogs(); err != nil {
		// Log error but don't fail initialization
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup old logs: %v\n", err)
	}

	return l, nil
}

// rotateLogFile creates a new log file for the current day
func (l *Logger) rotateLogFile() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	today := time.Now().Format("2006-01-02")

	// If we're already using today's log file, check if rotation is needed
	if l.currentDay == today && l.logFile != nil {
		if l.config.MaxFileSize > 0 {
			if stat, err := l.logFile.Stat(); err == nil && stat.Size() < l.config.MaxFileSize {
				return nil // No rotation needed
			}
			// File is too large, create a new one with timestamp
			today = time.Now().Format("2006-01-02_15-04-05")
		} else {
			return nil // No rotation needed
		}
	}

	// Close existing file if open
	if l.logFile != nil {
		l.logFile.Close()
	}

	// Create new log file
	fileName := filepath.Join(l.config.LogDir, fmt.Sprintf("app-%s.log", today))
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Determine output writer
	var writer io.Writer = file
	if l.config.EnableConsole {
		writer = io.MultiWriter(file, os.Stderr)
	}

	// Set up slog handler
	var handler slog.Handler
	handlerOptions := &slog.HandlerOptions{
		Level: l.getSlogLevel(),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize the timestamp format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(time.Now().Format(time.RFC3339)),
				}
			}
			return a
		},
	}

	if l.config.JSONFormat {
		handler = slog.NewJSONHandler(writer, handlerOptions)
	} else {
		handler = slog.NewTextHandler(writer, handlerOptions)
	}

	l.logger = slog.New(handler)
	l.logFile = file
	l.currentDay = today

	return nil
}

// getSlogLevel converts our LogLevel to slog.Level
func (l *Logger) getSlogLevel() slog.Level {
	switch l.config.Level {
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// cleanupOldLogs removes old log files based on MaxFiles configuration
func (l *Logger) cleanupOldLogs() error {
	if l.config.MaxFiles <= 0 {
		return nil // No cleanup needed
	}

	entries, err := os.ReadDir(l.config.LogDir)
	if err != nil {
		return err
	}

	// Filter and sort log files by modification time
	var logFiles []os.FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if filepath.Ext(info.Name()) == ".log" {
			logFiles = append(logFiles, info)
		}
	}

	// Sort by modification time (oldest first)
	for i := 0; i < len(logFiles)-1; i++ {
		for j := i + 1; j < len(logFiles); j++ {
			if logFiles[i].ModTime().After(logFiles[j].ModTime()) {
				logFiles[i], logFiles[j] = logFiles[j], logFiles[i]
			}
		}
	}

	// Remove excess files
	if len(logFiles) > l.config.MaxFiles {
		for i := 0; i < len(logFiles)-l.config.MaxFiles; i++ {
			filePath := filepath.Join(l.config.LogDir, logFiles[i].Name())
			if err := os.Remove(filePath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to remove old log file %s: %v\n", filePath, err)
			}
		}
	}

	return nil
}

// checkRotation checks if we need to rotate the log file
func (l *Logger) checkRotation() error {
	l.mutex.RLock()
	needsRotation := false

	// Check if day has changed
	today := time.Now().Format("2006-01-02")
	if l.currentDay != today {
		needsRotation = true
	}

	// Check file size if limit is set
	if !needsRotation && l.config.MaxFileSize > 0 && l.logFile != nil {
		if stat, err := l.logFile.Stat(); err == nil && stat.Size() >= l.config.MaxFileSize {
			needsRotation = true
		}
	}
	l.mutex.RUnlock()

	if needsRotation {
		return l.rotateLogFile()
	}
	return nil
}

// logAttrs converts a map to slog.Attr array
func logAttrs(fields map[string]interface{}) []any {
	if fields == nil {
		return nil
	}

	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	return attrs
}

// Log writes a log entry with the specified level and fields
func (l *Logger) Log(level LogLevel, message string, fields map[string]interface{}) error {
	if err := l.checkRotation(); err != nil {
		return err
	}

	attrs := logAttrs(fields)

	l.mutex.RLock()
	logger := l.logger
	l.mutex.RUnlock()

	if logger == nil {
		return fmt.Errorf("logger not initialized")
	}

	switch level {
	case InfoLevel:
		logger.Info(message, attrs...)
	case WarnLevel:
		logger.Warn(message, attrs...)
	case ErrorLevel:
		logger.Error(message, attrs...)
	case DebugLevel:
		logger.Debug(message, attrs...)
	default:
		logger.Info(message, attrs...)
	}

	return nil
}

// Info logs an info level message
func (l *Logger) Info(message string, fields map[string]interface{}) error {
	return l.Log(InfoLevel, message, fields)
}

// Warn logs a warning level message
func (l *Logger) Warn(message string, fields map[string]interface{}) error {
	return l.Log(WarnLevel, message, fields)
}

// Error logs an error level message
func (l *Logger) Error(message string, fields map[string]interface{}) error {
	return l.Log(ErrorLevel, message, fields)
}

// Debug logs a debug level message
func (l *Logger) Debug(message string, fields map[string]interface{}) error {
	return l.Log(DebugLevel, message, fields)
}

// With creates a new logger with predefined fields
func (l *Logger) With(fields map[string]interface{}) *ContextLogger {
	return &ContextLogger{
		logger: l,
		fields: fields,
	}
}

// Close closes the logger and its associated file
func (l *Logger) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// ContextLogger wraps the main logger with predefined fields
type ContextLogger struct {
	logger *Logger
	fields map[string]interface{}
}

// mergeFields combines context fields with additional fields
func (cl *ContextLogger) mergeFields(additional map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Add context fields first
	for k, v := range cl.fields {
		merged[k] = v
	}

	// Add additional fields (they can override context fields)
	for k, v := range additional {
		merged[k] = v
	}

	return merged
}

// Info logs an info level message with context fields
func (cl *ContextLogger) Info(message string, fields map[string]interface{}) error {
	return cl.logger.Info(message, cl.mergeFields(fields))
}

// Warn logs a warning level message with context fields
func (cl *ContextLogger) Warn(message string, fields map[string]interface{}) error {
	return cl.logger.Warn(message, cl.mergeFields(fields))
}

// Error logs an error level message with context fields
func (cl *ContextLogger) Error(message string, fields map[string]interface{}) error {
	return cl.logger.Error(message, cl.mergeFields(fields))
}

// Debug logs a debug level message with context fields
func (cl *ContextLogger) Debug(message string, fields map[string]interface{}) error {
	return cl.logger.Debug(message, cl.mergeFields(fields))
}
