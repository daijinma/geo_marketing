package repositories

import (
	"database/sql"
)

// LogRepository handles log data.
type LogRepository struct {
	*BaseRepository
}

// NewLogRepository creates a new log repository.
func NewLogRepository(db *sql.DB) *LogRepository {
	return &LogRepository{BaseRepository: NewBaseRepository(db)}
}

// Add adds a log entry.
func (r *LogRepository) Add(level, source, message string, details *string, taskID *int) error {
	_, err := r.db.Exec(
		"INSERT INTO logs (level, source, message, details, task_id, timestamp) VALUES (?, ?, ?, ?, ?, datetime('now'))",
		level, source, message, details, taskID,
	)
	return err
}

// AddWithContext adds a log entry with additional context.
func (r *LogRepository) AddWithContext(level, source, message string, details *string, taskID *int, sessionID, correlationID, component, userAction *string, performanceMS *int) error {
	_, err := r.db.Exec(`
		INSERT INTO logs (
			level, source, message, details, task_id, 
			session_id, correlation_id, component, user_action, performance_ms,
			timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		level, source, message, details, taskID,
		sessionID, correlationID, component, userAction, performanceMS,
	)
	return err
}

// ClearOldLogs deletes logs older than specified days.
func (r *LogRepository) ClearOldLogs(daysToKeep int) error {
	_, err := r.db.Exec("DELETE FROM logs WHERE timestamp < datetime('now', '-' || ? || ' days')", daysToKeep)
	return err
}

// DeleteAllLogs deletes all logs from the database.
func (r *LogRepository) DeleteAllLogs() (int64, error) {
	result, err := r.db.Exec("DELETE FROM logs")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetLogsCount returns total number of logs.
func (r *LogRepository) GetLogsCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&count)
	return count, err
}

// GetLogsCountOlderThan returns count of logs older than specified days.
func (r *LogRepository) GetLogsCountOlderThan(days int) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM logs WHERE timestamp < datetime('now', '-' || ? || ' days')"
	err := r.db.QueryRow(query, days).Scan(&count)
	return count, err
}

// GetAll retrieves logs with optional filters.
func (r *LogRepository) GetAll(limit, offset int, level, source *string, taskID *int) ([]map[string]interface{}, error) {
	query := "SELECT * FROM logs WHERE 1=1"
	args := []interface{}{}

	if level != nil {
		query += " AND level = ?"
		args = append(args, *level)
	}
	if source != nil {
		query += " AND source = ?"
		args = append(args, *source)
	}
	if taskID != nil {
		query += " AND task_id = ?"
		args = append(args, *taskID)
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	cols, _ := rows.Columns()
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		entry := make(map[string]interface{})
		for i, col := range cols {
			entry[col] = values[i]
		}
		results = append(results, entry)
	}
	return results, nil
}
