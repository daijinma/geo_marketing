package logger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Logger handles structured logging to both database and console.
type Logger struct {
	logDir string
	db     *sql.DB
}

var loggerInstance *Logger

// LogEntry represents a structured log entry.
type LogEntry struct {
	Level         string
	Source        string
	Message       string
	Details       map[string]interface{}
	TaskID        *int
	SessionID     string
	CorrelationID string
	Component     string
	UserAction    string
	PerformanceMS *int
}

// SetDB sets the database connection for the logger.
func SetDB(database *sql.DB) {
	if loggerInstance != nil {
		loggerInstance.db = database
	}
}

// GetLogger returns the singleton logger instance.
func GetLogger() *Logger {
	if loggerInstance == nil {
		homeDir, _ := os.UserHomeDir()
		logDir := filepath.Join(homeDir, ".geo_client2", "logs")
		os.MkdirAll(logDir, 0755)
		loggerInstance = &Logger{logDir: logDir}
	}
	return loggerInstance
}

// Info logs an info message.
func (l *Logger) Info(message string) {
	l.log(LogEntry{
		Level:   "INFO",
		Source:  l.getCallerSource(),
		Message: message,
	})
}

// InfoWithContext logs an info message with full context.
func (l *Logger) InfoWithContext(ctx context.Context, message string, details map[string]interface{}, taskID *int) {
	l.log(LogEntry{
		Level:         "INFO",
		Source:        l.getCallerSource(),
		Message:       message,
		Details:       details,
		TaskID:        taskID,
		SessionID:     extractSessionID(ctx),
		CorrelationID: extractCorrelationID(ctx),
	})
}

// Error logs an error message.
func (l *Logger) Error(message string, err error) {
	details := make(map[string]interface{})
	if err != nil {
		details["error"] = err.Error()
	}
	l.log(LogEntry{
		Level:   "ERROR",
		Source:  l.getCallerSource(),
		Message: message,
		Details: details,
	})
}

// ErrorWithContext logs an error message with full context.
func (l *Logger) ErrorWithContext(ctx context.Context, message string, details map[string]interface{}, err error, taskID *int) {
	if details == nil {
		details = make(map[string]interface{})
	}
	if err != nil {
		details["error"] = err.Error()
		details["error_type"] = fmt.Sprintf("%T", err)
	}
	l.log(LogEntry{
		Level:         "ERROR",
		Source:        l.getCallerSource(),
		Message:       message,
		Details:       details,
		TaskID:        taskID,
		SessionID:     extractSessionID(ctx),
		CorrelationID: extractCorrelationID(ctx),
	})
}

// Warn logs a warning message.
func (l *Logger) Warn(message string) {
	l.log(LogEntry{
		Level:   "WARN",
		Source:  l.getCallerSource(),
		Message: message,
	})
}

// WarnWithContext logs a warning message with context.
func (l *Logger) WarnWithContext(ctx context.Context, message string, details map[string]interface{}, taskID *int) {
	l.log(LogEntry{
		Level:         "WARN",
		Source:        l.getCallerSource(),
		Message:       message,
		Details:       details,
		TaskID:        taskID,
		SessionID:     extractSessionID(ctx),
		CorrelationID: extractCorrelationID(ctx),
	})
}

// Debug logs a debug message.
func (l *Logger) Debug(message string) {
	l.log(LogEntry{
		Level:   "DEBUG",
		Source:  l.getCallerSource(),
		Message: message,
	})
}

// DebugWithContext logs a debug message with context.
func (l *Logger) DebugWithContext(ctx context.Context, message string, details map[string]interface{}, taskID *int) {
	l.log(LogEntry{
		Level:         "DEBUG",
		Source:        l.getCallerSource(),
		Message:       message,
		Details:       details,
		TaskID:        taskID,
		SessionID:     extractSessionID(ctx),
		CorrelationID: extractCorrelationID(ctx),
	})
}

// LogPerformance logs performance metrics.
func (l *Logger) LogPerformance(ctx context.Context, operation string, durationMS int, details map[string]interface{}, taskID *int) {
	if details == nil {
		details = make(map[string]interface{})
	}
	l.log(LogEntry{
		Level:         "INFO",
		Source:        l.getCallerSource(),
		Message:       fmt.Sprintf("Performance: %s completed in %dms", operation, durationMS),
		Details:       details,
		TaskID:        taskID,
		SessionID:     extractSessionID(ctx),
		CorrelationID: extractCorrelationID(ctx),
		PerformanceMS: &durationMS,
	})
}

// LogUserAction logs a user action from frontend.
func (l *Logger) LogUserAction(sessionID, correlationID, component, action, message string, details map[string]interface{}) {
	l.log(LogEntry{
		Level:         "INFO",
		Source:        "frontend",
		Message:       message,
		Details:       details,
		SessionID:     sessionID,
		CorrelationID: correlationID,
		Component:     component,
		UserAction:    action,
	})
}

// log writes the log entry to both database and console.
func (l *Logger) log(entry LogEntry) {
	timestamp := time.Now()

	// Console output
	level := entry.Level
	msg := entry.Message
	if entry.Details != nil && len(entry.Details) > 0 {
		detailsJSON, _ := json.Marshal(entry.Details)
		msg = fmt.Sprintf("%s | details: %s", msg, string(detailsJSON))
	}
	fmt.Printf("[%s] [%s] [%s] %s\n", timestamp.Format(time.RFC3339), level, entry.Source, msg)

	// Database output (if available)
	if l.db != nil {
		var detailsJSON *string
		if entry.Details != nil && len(entry.Details) > 0 {
			jsonBytes, err := json.Marshal(entry.Details)
			if err == nil {
				str := string(jsonBytes)
				detailsJSON = &str
			}
		}

		_, err := l.db.Exec(`
			INSERT INTO logs (level, source, message, details, task_id, session_id, correlation_id, component, user_action, performance_ms, timestamp)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, entry.Level, entry.Source, entry.Message, detailsJSON, entry.TaskID,
			nullString(entry.SessionID), nullString(entry.CorrelationID),
			nullString(entry.Component), nullString(entry.UserAction),
			entry.PerformanceMS, timestamp.Format("2006-01-02 15:04:05"))

		if err != nil {
			// If DB write fails, at least log to console
			fmt.Printf("[ERROR] Failed to write log to database: %v\n", err)
		}
	}
}

// getCallerSource returns the file:line of the caller (2 levels up).
func (l *Logger) getCallerSource() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	// Get just the filename, not full path
	file = filepath.Base(file)
	return fmt.Sprintf("%s:%d", file, line)
}

// Helper functions for context extraction
type contextKey string

const (
	sessionIDKey     contextKey = "session_id"
	correlationIDKey contextKey = "correlation_id"
)

// WithSessionID adds session ID to context.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// WithCorrelationID adds correlation ID to context.
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

func extractSessionID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val := ctx.Value(sessionIDKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func extractCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val := ctx.Value(correlationIDKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// GetLogDir returns the log directory path.
func (l *Logger) GetLogDir() string {
	return l.logDir
}

// Timer helps track operation duration.
type Timer struct {
	logger    *Logger
	ctx       context.Context
	operation string
	taskID    *int
	startTime time.Time
}

// StartTimer creates a new performance timer.
func (l *Logger) StartTimer(ctx context.Context, operation string, taskID *int) *Timer {
	return &Timer{
		logger:    l,
		ctx:       ctx,
		operation: operation,
		taskID:    taskID,
		startTime: time.Now(),
	}
}

// End stops the timer and logs the performance.
func (t *Timer) End(details map[string]interface{}) {
	duration := int(time.Since(t.startTime).Milliseconds())
	t.logger.LogPerformance(t.ctx, t.operation, duration, details, t.taskID)
}
