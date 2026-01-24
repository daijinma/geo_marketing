package logger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type Logger struct {
	logDir  string
	db      *sql.DB
	logFile *os.File
	mu      sync.Mutex
}

var loggerInstance *Logger

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

func SetDB(database *sql.DB) {
	if loggerInstance != nil {
		loggerInstance.db = database
	}
}

func GetLogger() *Logger {
	if loggerInstance == nil {
		homeDir, _ := os.UserHomeDir()
		logDir := filepath.Join(homeDir, ".geo_client2", "logs")
		os.MkdirAll(logDir, 0755)

		loggerInstance = &Logger{logDir: logDir}
		loggerInstance.openLogFile()
	}
	return loggerInstance
}

func (l *Logger) openLogFile() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		l.logFile.Close()
	}

	logPath := filepath.Join(l.logDir, "app.log")

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to open log file: %v\n", err)
		return
	}

	l.logFile = f
}

func (l *Logger) rotateLogIfNeeded() {
	logPath := filepath.Join(l.logDir, "app.log")

	info, err := os.Stat(logPath)
	if err != nil || info.Size() < 10*1024*1024 {
		return
	}

	l.mu.Lock()
	if l.logFile != nil {
		l.logFile.Close()
		l.logFile = nil
	}

	timestamp := time.Now().Format("20060102_150405")
	oldPath := filepath.Join(l.logDir, fmt.Sprintf("app_%s.log", timestamp))
	os.Rename(logPath, oldPath)
	l.mu.Unlock()

	l.openLogFile()
}

func (l *Logger) GetLogFilePath() string {
	return filepath.Join(l.logDir, "app.log")
}

func (l *Logger) ReadLogFile(lines int) (string, error) {
	logPath := l.GetLogFilePath()

	content, err := os.ReadFile(logPath)
	if err != nil {
		return "", err
	}

	if lines <= 0 {
		return string(content), nil
	}

	allLines := splitLines(string(content))
	if len(allLines) <= lines {
		return string(content), nil
	}

	lastLines := allLines[len(allLines)-lines:]
	result := ""
	for _, line := range lastLines {
		result += line + "\n"
	}
	return result, nil
}

func splitLines(content string) []string {
	var lines []string
	current := ""
	for _, ch := range content {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		l.logFile.Close()
		l.logFile = nil
	}
}

func (l *Logger) Info(message string) {
	l.log(LogEntry{
		Level:   "INFO",
		Source:  l.getCallerSource(),
		Message: message,
	})
}

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

func (l *Logger) Warn(message string) {
	l.log(LogEntry{
		Level:   "WARN",
		Source:  l.getCallerSource(),
		Message: message,
	})
}

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

func (l *Logger) Debug(message string) {
	l.log(LogEntry{
		Level:   "DEBUG",
		Source:  l.getCallerSource(),
		Message: message,
	})
}

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

func (l *Logger) log(entry LogEntry) {
	timestamp := time.Now()

	level := entry.Level
	msg := entry.Message
	if entry.Details != nil && len(entry.Details) > 0 {
		detailsJSON, _ := json.Marshal(entry.Details)
		msg = fmt.Sprintf("%s | details: %s", msg, string(detailsJSON))
	}
	logLine := fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp.Format(time.RFC3339), level, entry.Source, msg)

	fmt.Print(logLine)

	l.mu.Lock()
	if l.logFile != nil {
		l.logFile.WriteString(logLine)
	}
	l.mu.Unlock()

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
			errLine := fmt.Sprintf("[ERROR] Failed to write log to database: %v\n", err)
			fmt.Print(errLine)
			l.mu.Lock()
			if l.logFile != nil {
				l.logFile.WriteString(errLine)
			}
			l.mu.Unlock()
		}
	}

	if timestamp.Unix()%100 == 0 {
		go l.rotateLogIfNeeded()
	}
}

func (l *Logger) getCallerSource() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	file = filepath.Base(file)
	return fmt.Sprintf("%s:%d", file, line)
}

type contextKey string

const (
	sessionIDKey     contextKey = "session_id"
	correlationIDKey contextKey = "correlation_id"
)

func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

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

func (l *Logger) GetLogDir() string {
	return l.logDir
}

type Timer struct {
	logger    *Logger
	ctx       context.Context
	operation string
	taskID    *int
	startTime time.Time
}

func (l *Logger) StartTimer(ctx context.Context, operation string, taskID *int) *Timer {
	return &Timer{
		logger:    l,
		ctx:       ctx,
		operation: operation,
		taskID:    taskID,
		startTime: time.Now(),
	}
}

func (t *Timer) End(details map[string]interface{}) {
	duration := int(time.Since(t.startTime).Milliseconds())
	t.logger.LogPerformance(t.ctx, t.operation, duration, details, t.taskID)
}

func (l *Logger) RedirectStdout() (io.Writer, io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile == nil {
		return os.Stdout, os.Stderr
	}

	stdoutWriter := io.MultiWriter(os.Stdout, l.logFile)
	stderrWriter := io.MultiWriter(os.Stderr, l.logFile)

	return stdoutWriter, stderrWriter
}
