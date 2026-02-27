package repositories

import (
	"database/sql"
	"time"
)

type LongTaskRecord struct {
	ID                 int64
	TaskID             string
	Status             string
	ArticleJSON        string
	PlatformsJSON      string
	AccountIDsJSON     string
	PlatformStatesJSON sql.NullString
	CurrentIndex       int
	TotalPlatforms     int
	CreatedAt          time.Time
	StartedAt          sql.NullTime
	CompletedAt        sql.NullTime
	UpdatedAt          time.Time
}

type LongTaskRepository struct {
	*BaseRepository
}

func NewLongTaskRepository(db *sql.DB) *LongTaskRepository {
	return &LongTaskRepository{BaseRepository: NewBaseRepository(db)}
}

func (r *LongTaskRepository) Create(taskID, status, articleJSON, platformsJSON, accountIDsJSON string, totalPlatforms int) error {
	_, err := r.db.Exec(`
		INSERT INTO long_tasks (task_id, status, article_json, platforms_json, account_ids_json, total_platforms, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now', 'localtime'), datetime('now', 'localtime'))
	`, taskID, status, articleJSON, platformsJSON, accountIDsJSON, totalPlatforms)
	return err
}

func (r *LongTaskRepository) UpdateStatus(taskID, status string) error {
	_, err := r.db.Exec(`
		UPDATE long_tasks SET status = ?, updated_at = datetime('now', 'localtime') WHERE task_id = ?
	`, status, taskID)
	return err
}

func (r *LongTaskRepository) UpdateProgress(taskID string, currentIndex int, platformStatesJSON string) error {
	_, err := r.db.Exec(`
		UPDATE long_tasks 
		SET current_index = ?, platform_states_json = ?, updated_at = datetime('now', 'localtime') 
		WHERE task_id = ?
	`, currentIndex, platformStatesJSON, taskID)
	return err
}

func (r *LongTaskRepository) SetStarted(taskID string) error {
	_, err := r.db.Exec(`
		UPDATE long_tasks 
		SET status = 'running', started_at = datetime('now', 'localtime'), updated_at = datetime('now', 'localtime') 
		WHERE task_id = ?
	`, taskID)
	return err
}

func (r *LongTaskRepository) SetCompleted(taskID, status string) error {
	_, err := r.db.Exec(`
		UPDATE long_tasks 
		SET status = ?, completed_at = datetime('now', 'localtime'), updated_at = datetime('now', 'localtime') 
		WHERE task_id = ?
	`, status, taskID)
	return err
}

func (r *LongTaskRepository) GetByTaskID(taskID string) (*LongTaskRecord, error) {
	row := r.db.QueryRow(`
		SELECT id, task_id, status, article_json, platforms_json, account_ids_json, 
		       platform_states_json, current_index, total_platforms, 
		       created_at, started_at, completed_at, updated_at
		FROM long_tasks WHERE task_id = ?
	`, taskID)

	var rec LongTaskRecord
	var createdAtStr, updatedAtStr string
	var startedAtStr, completedAtStr sql.NullString

	err := row.Scan(
		&rec.ID, &rec.TaskID, &rec.Status, &rec.ArticleJSON, &rec.PlatformsJSON, &rec.AccountIDsJSON,
		&rec.PlatformStatesJSON, &rec.CurrentIndex, &rec.TotalPlatforms,
		&createdAtStr, &startedAtStr, &completedAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	rec.CreatedAt, _ = time.ParseInLocation("2006-01-02 15:04:05", createdAtStr, time.Local)
	rec.UpdatedAt, _ = time.ParseInLocation("2006-01-02 15:04:05", updatedAtStr, time.Local)
	if startedAtStr.Valid {
		t, _ := time.ParseInLocation("2006-01-02 15:04:05", startedAtStr.String, time.Local)
		rec.StartedAt = sql.NullTime{Time: t, Valid: true}
	}
	if completedAtStr.Valid {
		t, _ := time.ParseInLocation("2006-01-02 15:04:05", completedAtStr.String, time.Local)
		rec.CompletedAt = sql.NullTime{Time: t, Valid: true}
	}

	return &rec, nil
}

func (r *LongTaskRepository) GetUnfinished() ([]*LongTaskRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, task_id, status, article_json, platforms_json, account_ids_json, 
		       platform_states_json, current_index, total_platforms, 
		       created_at, started_at, completed_at, updated_at
		FROM long_tasks 
		WHERE status IN ('pending', 'running', 'paused')
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRecords(rows)
}

func (r *LongTaskRepository) GetAll() ([]*LongTaskRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, task_id, status, article_json, platforms_json, account_ids_json, 
		       platform_states_json, current_index, total_platforms, 
		       created_at, started_at, completed_at, updated_at
		FROM long_tasks 
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRecords(rows)
}

func (r *LongTaskRepository) scanRecords(rows *sql.Rows) ([]*LongTaskRecord, error) {
	var records []*LongTaskRecord

	for rows.Next() {
		var rec LongTaskRecord
		var createdAtStr, updatedAtStr string
		var startedAtStr, completedAtStr sql.NullString

		err := rows.Scan(
			&rec.ID, &rec.TaskID, &rec.Status, &rec.ArticleJSON, &rec.PlatformsJSON, &rec.AccountIDsJSON,
			&rec.PlatformStatesJSON, &rec.CurrentIndex, &rec.TotalPlatforms,
			&createdAtStr, &startedAtStr, &completedAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}

		rec.CreatedAt, _ = time.ParseInLocation("2006-01-02 15:04:05", createdAtStr, time.Local)
		rec.UpdatedAt, _ = time.ParseInLocation("2006-01-02 15:04:05", updatedAtStr, time.Local)
		if startedAtStr.Valid {
			t, _ := time.ParseInLocation("2006-01-02 15:04:05", startedAtStr.String, time.Local)
			rec.StartedAt = sql.NullTime{Time: t, Valid: true}
		}
		if completedAtStr.Valid {
			t, _ := time.ParseInLocation("2006-01-02 15:04:05", completedAtStr.String, time.Local)
			rec.CompletedAt = sql.NullTime{Time: t, Valid: true}
		}

		records = append(records, &rec)
	}

	return records, nil
}

func (r *LongTaskRepository) Delete(taskID string) error {
	_, err := r.db.Exec("DELETE FROM long_tasks WHERE task_id = ?", taskID)
	return err
}

func (r *LongTaskRepository) DeleteOlderThan(duration time.Duration) (int64, error) {
	cutoff := time.Now().Add(-duration).Format("2006-01-02 15:04:05")
	result, err := r.db.Exec(`
		DELETE FROM long_tasks 
		WHERE status IN ('completed', 'failed', 'cancelled') 
		AND completed_at < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
