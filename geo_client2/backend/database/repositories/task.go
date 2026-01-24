package repositories

import (
	"database/sql"
	"fmt"
)

// TaskFilters for filtering tasks.
type TaskFilters struct {
	Status    *string
	Platform  *string
	Source    *string
	TaskType  *string
	CreatedBy *string
}

// TaskRepository handles task data.
type TaskRepository struct {
	*BaseRepository
}

// NewTaskRepository creates a new task repository.
func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{BaseRepository: NewBaseRepository(db)}
}

// Save saves a task.
func (r *TaskRepository) Save(taskID *int, keywords, platforms string, queryCount int, status string, resultData *string, taskType, source string, createdBy *string, priority int, scheduledAt *string, settings *string) (int64, error) {
	res, err := r.db.Exec(
		`INSERT OR REPLACE INTO tasks 
		(task_id, keywords, platforms, query_count, status, result_data, 
		 task_type, source, created_by, priority, scheduled_at, settings, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		taskID, keywords, platforms, queryCount, status, resultData,
		taskType, source, createdBy, priority, scheduledAt, settings,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetByID retrieves a task by ID.
func (r *TaskRepository) GetByID(id int) (map[string]interface{}, error) {
	rows, err := r.db.Query("SELECT * FROM tasks WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRowsToMap(rows)
}

// GetByTaskID retrieves a task by task_id.
func (r *TaskRepository) GetByTaskID(taskID int) (map[string]interface{}, error) {
	rows, err := r.db.Query("SELECT * FROM tasks WHERE task_id = ?", taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRowsToMap(rows)
}

// UpdateStatus updates task status.
func (r *TaskRepository) UpdateStatus(id int, status string, resultData *string) error {
	_, err := r.db.Exec(
		"UPDATE tasks SET status = ?, result_data = ?, updated_at = datetime('now') WHERE id = ?",
		status, resultData, id,
	)
	return err
}

// UpdateName updates task name.
func (r *TaskRepository) UpdateName(id int, name string) error {
	_, err := r.db.Exec(
		"UPDATE tasks SET name = ?, updated_at = datetime('now') WHERE id = ?",
		name, id,
	)
	return err
}

// UpdateStats updates task statistics.
func (r *TaskRepository) UpdateStats(id int, totalQueries, completedQueries, totalRecords, completedRecords, failedRecords, totalCitations int) error {
	_, err := r.db.Exec(
		`UPDATE tasks SET 
			total_queries = ?, completed_queries = ?, 
			total_records = ?, completed_records = ?, 
			failed_records = ?, total_citations = ?,
			updated_at = datetime('now')
		WHERE id = ?`,
		totalQueries, completedQueries, totalRecords, completedRecords, failedRecords, totalCitations, id,
	)
	return err
}

// GetAll retrieves tasks with filters.
func (r *TaskRepository) GetAll(limit, offset int, filters *TaskFilters) ([]map[string]interface{}, error) {
	query := "SELECT * FROM tasks WHERE 1=1"
	args := []interface{}{}

	if filters != nil {
		if filters.Status != nil {
			query += " AND status = ?"
			args = append(args, *filters.Status)
		}
		if filters.Platform != nil {
			query += " AND platforms LIKE ?"
			args = append(args, fmt.Sprintf("%%%s%%", *filters.Platform))
		}
		if filters.Source != nil {
			query += " AND source = ?"
			args = append(args, *filters.Source)
		}
		if filters.TaskType != nil {
			query += " AND task_type = ?"
			args = append(args, *filters.TaskType)
		}
		if filters.CreatedBy != nil {
			query += " AND created_by = ?"
			args = append(args, *filters.CreatedBy)
		}
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
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

// GetStats retrieves task statistics.
func (r *TaskRepository) GetStats() (map[string]interface{}, error) {
	var total, running, completed, failed, localCount, serverCount, localSearchCount, serverSyncCount int
	err := r.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status = 'running' OR status = 'pending' THEN 1 ELSE 0 END) as running,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
			SUM(CASE WHEN source = 'local' THEN 1 ELSE 0 END) as local_count,
			SUM(CASE WHEN source = 'server' THEN 1 ELSE 0 END) as server_count,
			SUM(CASE WHEN task_type = 'local_search' THEN 1 ELSE 0 END) as local_search_count,
			SUM(CASE WHEN task_type = 'server_sync' THEN 1 ELSE 0 END) as server_sync_count
		FROM tasks
	`).Scan(&total, &running, &completed, &failed, &localCount, &serverCount, &localSearchCount, &serverSyncCount)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":     total,
		"running":   running,
		"completed": completed,
		"failed":    failed,
		"bySource": map[string]int{
			"local":  localCount,
			"server": serverCount,
		},
		"byType": map[string]int{
			"local_search": localSearchCount,
			"server_sync":  serverSyncCount,
		},
	}, nil
}

func scanRowsToMap(rows *sql.Rows) (map[string]interface{}, error) {
	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
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
	return entry, nil
}

// SaveSearchRecord saves a search record.
func (r *TaskRepository) SaveSearchRecord(taskID int, keyword, platform, prompt, fullAnswer string, responseTime int, status string, errorMsg string, round int) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO search_records 
		(task_id, keyword, platform, prompt, full_answer, response_time_ms, search_status, error_message, round_number, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		taskID, keyword, platform, prompt, fullAnswer, responseTime, status, errorMsg, round,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// SaveCitation saves a citation.
func (r *TaskRepository) SaveCitation(recordID int64, citeIndex int, url, domain, title, snippet, siteName string) error {
	_, err := r.db.Exec(
		`INSERT OR IGNORE INTO citations 
		(record_id, cite_index, url, domain, title, snippet, site_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		recordID, citeIndex, url, domain, title, snippet, siteName,
	)
	return err
}

// SaveSearchQuery saves a search query.
func (r *TaskRepository) SaveSearchQuery(recordID int64, query string, queryOrder int) error {
	_, err := r.db.Exec(
		`INSERT INTO search_queries 
		(record_id, query, query_order, created_at)
		VALUES (?, ?, ?, datetime('now'))`,
		recordID, query, queryOrder,
	)
	return err
}

func (r *TaskRepository) DeleteSearchRecordsByTaskID(taskID int) error {
	_, err := r.db.Exec(`
		DELETE FROM citations 
		WHERE record_id IN (SELECT id FROM search_records WHERE task_id = ?)
	`, taskID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`
		DELETE FROM search_queries 
		WHERE record_id IN (SELECT id FROM search_records WHERE task_id = ?)
	`, taskID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec("DELETE FROM search_records WHERE task_id = ?", taskID)
	return err
}

func (r *TaskRepository) GetSearchRecordsByTaskID(taskID int) ([]map[string]interface{}, error) {
	rows, err := r.db.Query("SELECT * FROM search_records WHERE task_id = ? ORDER BY created_at ASC", taskID)
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

		var recordID int64
		var ok bool

		switch v := entry["id"].(type) {
		case int64:
			recordID = v
			ok = true
		case int:
			recordID = int64(v)
			ok = true
		case float64:
			recordID = int64(v)
			ok = true
		}

		if ok {
			citations, _ := r.GetCitationsByRecordID(recordID)
			entry["citations"] = citations

			queries, _ := r.GetQueriesByRecordID(recordID)
			entry["queries"] = queries
		}

		results = append(results, entry)
	}
	return results, nil
}

func (r *TaskRepository) GetMergedSearchRecords(taskIDs []int) ([]map[string]interface{}, error) {
	if len(taskIDs) == 0 {
		return []map[string]interface{}{}, nil
	}

	query := "SELECT * FROM search_records WHERE task_id IN ("
	args := make([]interface{}, len(taskIDs))
	for i, id := range taskIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += ") ORDER BY task_id ASC, created_at ASC"

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

		var recordID int64
		var ok bool

		switch v := entry["id"].(type) {
		case int64:
			recordID = v
			ok = true
		case int:
			recordID = int64(v)
			ok = true
		case float64:
			recordID = int64(v)
			ok = true
		}

		if ok {
			citations, _ := r.GetCitationsByRecordID(recordID)
			entry["citations"] = citations

			queries, _ := r.GetQueriesByRecordID(recordID)
			entry["queries"] = queries
		}

		results = append(results, entry)
	}
	return results, nil
}

func (r *TaskRepository) GetCitationsByRecordID(recordID int64) ([]map[string]interface{}, error) {
	rows, err := r.db.Query("SELECT * FROM citations WHERE record_id = ? ORDER BY cite_index ASC", recordID)
	if err != nil {
		return []map[string]interface{}{}, err
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

// GetQueriesByRecordID retrieves queries for a record.
func (r *TaskRepository) GetQueriesByRecordID(recordID int64) ([]map[string]interface{}, error) {
	rows, err := r.db.Query("SELECT * FROM search_queries WHERE record_id = ? ORDER BY query_order ASC", recordID)
	if err != nil {
		return []map[string]interface{}{}, err
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

// Delete deletes a task and all related data (search_records, citations, queries).
func (r *TaskRepository) Delete(taskID int) error {
	// Delete in order: citations -> search_queries -> search_records -> task
	if err := r.DeleteSearchRecordsByTaskID(taskID); err != nil {
		return fmt.Errorf("failed to delete search records: %w", err)
	}

	_, err := r.db.Exec("DELETE FROM tasks WHERE id = ?", taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// IsSearchRecordCompleted checks if a search record exists and is completed.
func (r *TaskRepository) IsSearchRecordCompleted(taskID int, keyword, platform string, round int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM search_records 
			WHERE task_id = ? AND keyword = ? AND platform = ? AND round_number = ? AND search_status = 'completed'
		)
	`, taskID, keyword, platform, round).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
