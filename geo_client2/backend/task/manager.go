package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"geo_client2/backend/database/repositories"
	"geo_client2/backend/logger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Manager handles task management.
type Manager struct {
	taskRepo     *repositories.TaskRepository
	loginRepo    *repositories.LoginStatusRepository
	executor     *Executor
	logger       *logger.Logger
	eventContext context.Context
}

// NewManager creates a new task manager.
func NewManager(taskRepo *repositories.TaskRepository, loginRepo *repositories.LoginStatusRepository, executor *Executor, eventCtx context.Context) *Manager {
	return &Manager{
		taskRepo:     taskRepo,
		loginRepo:    loginRepo,
		executor:     executor,
		logger:       logger.GetLogger(),
		eventContext: eventCtx,
	}
}

// CreateLocalSearchTask creates a local search task.
func (m *Manager) CreateLocalSearchTask(keywords []string, platforms []string, queryCount int, taskType, source string, createdBy *string, settings map[string]interface{}) (int64, error) {
	keywordsJSON, _ := json.Marshal(keywords)
	platformsJSON, _ := json.Marshal(platforms)
	var settingsJSON *string
	if settings != nil {
		s, _ := json.Marshal(settings)
		str := string(s)
		settingsJSON = &str
	}

	taskID, err := m.taskRepo.Save(
		nil, // task_id
		string(keywordsJSON),
		string(platformsJSON),
		queryCount,
		"pending",
		nil, // result_data
		taskType,
		source,
		createdBy,
		0,   // priority
		nil, // scheduled_at
		settingsJSON,
	)
	if err != nil {
		return 0, err
	}

	// Start execution asynchronously
	go func() {
		if err := m.executor.ExecuteLocalTask(int(taskID), keywords, platforms, queryCount, settings); err != nil {
			m.logger.Error("Task execution failed", err)
		}
	}()

	return taskID, nil
}

// GetAllTasks retrieves tasks with filters.
func (m *Manager) GetAllTasks(limit, offset int, filters *repositories.TaskFilters) ([]map[string]interface{}, error) {
	return m.taskRepo.GetAll(limit, offset, filters)
}

// GetTaskDetail retrieves task detail with related records.
func (m *Manager) GetTaskDetail(taskID int) (map[string]interface{}, error) {
	return m.taskRepo.GetByID(taskID)
}

// CancelTask cancels a running task.
func (m *Manager) CancelTask(taskID int) error {
	return m.executor.CancelTask(taskID)
}

func (m *Manager) RetryTask(taskID int) error {
	_ = m.executor.CancelTask(taskID)

	taskData, err := m.taskRepo.GetByID(taskID)
	if err != nil {
		return err
	}

	var keywords []string
	var platforms []string
	var settings map[string]interface{}

	getValue := func(key string) string {
		for k, v := range taskData {
			if strings.EqualFold(k, key) {
				switch val := v.(type) {
				case string:
					return val
				case []byte:
					return string(val)
				}
			}
		}
		return ""
	}

	kStr := getValue("keywords")
	if kStr != "" {
		json.Unmarshal([]byte(kStr), &keywords)
	}

	pStr := getValue("platforms")
	if pStr != "" {
		json.Unmarshal([]byte(pStr), &platforms)
	}

	sStr := getValue("settings")
	if sStr != "" {
		json.Unmarshal([]byte(sStr), &settings)
	}

	queryCount := 1
	for k, v := range taskData {
		if strings.EqualFold(k, "query_count") {
			switch val := v.(type) {
			case int64:
				queryCount = int(val)
			case int:
				queryCount = val
			case float64:
				queryCount = int(val)
			}
			break
		}
	}

	m.logger.InfoWithContext(m.eventContext, "Retrying task - Resetting state", map[string]interface{}{
		"taskID":     taskID,
		"keywords":   keywords,
		"platforms":  platforms,
		"queryCount": queryCount,
	}, &taskID)

	if err := m.taskRepo.DeleteSearchRecordsByTaskID(taskID); err != nil {
		m.logger.WarnWithContext(m.eventContext, "Failed to delete old search records during retry", map[string]interface{}{"error": err.Error()}, &taskID)
	}

	m.taskRepo.UpdateStatus(taskID, "pending", nil)
	m.taskRepo.UpdateStats(taskID, len(keywords)*len(platforms)*queryCount, 0, 0, 0, 0, 0)

	runtime.EventsEmit(m.eventContext, "search:taskUpdated", map[string]interface{}{
		"taskId": taskID,
		"progress": map[string]interface{}{
			"completed": 0,
			"total":     len(keywords) * len(platforms) * queryCount,
			"success":   0,
			"failed":    0,
		},
	})

	go func() {
		if err := m.executor.ExecuteLocalTask(taskID, keywords, platforms, queryCount, settings); err != nil {
			m.logger.Error("Task retry failed", err)
		}
	}()

	return nil
}

// GetStats retrieves task statistics.
func (m *Manager) GetStats() (map[string]interface{}, error) {
	return m.taskRepo.GetStats()
}

// SubmitToServer submits a task to the server.
func (m *Manager) SubmitToServer(taskID int, apiBaseURL, token string) error {
	task, err := m.taskRepo.GetByID(taskID)
	if err != nil {
		return err
	}

	keywordsStr := ""
	if k, ok := task["keywords"].(string); ok {
		keywordsStr = k
	} else if k, ok := task["keywords"].([]byte); ok {
		keywordsStr = string(k)
	}

	platformsStr := ""
	if p, ok := task["platforms"].(string); ok {
		platformsStr = p
	} else if p, ok := task["platforms"].([]byte); ok {
		platformsStr = string(p)
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"keywords":    []string{keywordsStr},
		"platforms":   []string{platformsStr},
		"query_count": 1,
	})

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/tasks/create", apiBaseURL), bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return nil
}

// GetSearchRecords retrieves search records for a task.
func (m *Manager) GetSearchRecords(taskID int) ([]map[string]interface{}, error) {
	return m.taskRepo.GetSearchRecordsByTaskID(taskID)
}
