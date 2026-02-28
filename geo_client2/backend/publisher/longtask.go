package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"geo_client2/backend/database/repositories"
	"geo_client2/backend/logger"
)

type LongTaskStatus string

const (
	LongTaskStatusPending   LongTaskStatus = "pending"
	LongTaskStatusRunning   LongTaskStatus = "running"
	LongTaskStatusPaused    LongTaskStatus = "paused"
	LongTaskStatusCompleted LongTaskStatus = "completed"
	LongTaskStatusFailed    LongTaskStatus = "failed"
	LongTaskStatusCancelled LongTaskStatus = "cancelled"
)

type PlatformTaskState struct {
	Platform    string         `json:"platform"`
	Status      LongTaskStatus `json:"status"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	ArticleURL  string         `json:"article_url,omitempty"`
	Error       string         `json:"error,omitempty"`
	Retries     int            `json:"retries"`
	MaxRetries  int            `json:"max_retries"`
}

type LongTaskState struct {
	TaskID         string                        `json:"task_id"`
	Status         LongTaskStatus                `json:"status"`
	Article        Article                       `json:"article"`
	Platforms      []string                      `json:"platforms"`
	PlatformStates map[string]*PlatformTaskState `json:"platform_states"`
	CreatedAt      time.Time                     `json:"created_at"`
	StartedAt      *time.Time                    `json:"started_at,omitempty"`
	CompletedAt    *time.Time                    `json:"completed_at,omitempty"`
	CurrentIndex   int                           `json:"current_index"`
	TotalPlatforms int                           `json:"total_platforms"`
}

type LongTaskRunner struct {
	mu         sync.Mutex
	state      *LongTaskState
	manager    *Manager
	emit       EventEmitter
	aiConfig   AIPublishConfig
	accountIDs map[string]string
	ctx        context.Context
	cancel     context.CancelFunc
	logger     *logger.Logger
	pauseChan  chan struct{}
	resumeChan chan struct{}
	isPaused   bool
	repo       *repositories.LongTaskRepository
	skipCreate bool
}

type LongTaskConfig struct {
	TaskID                string
	ExistingRecord        bool
	Platforms             []string
	AccountIDs            map[string]string
	Article               Article
	AIConfig              AIPublishConfig
	MaxRetriesPerPlatform int
	DelayBetweenPlatforms time.Duration
	Repo                  *repositories.LongTaskRepository
}

func NewLongTaskRunner(manager *Manager, emit EventEmitter, config LongTaskConfig) *LongTaskRunner {
	taskID := config.TaskID
	if taskID == "" {
		taskID = fmt.Sprintf("longtask-%d", time.Now().UnixNano())
	}

	platformStates := make(map[string]*PlatformTaskState)
	for _, p := range config.Platforms {
		maxRetries := config.MaxRetriesPerPlatform
		if maxRetries == 0 {
			maxRetries = 2
		}
		platformStates[p] = &PlatformTaskState{
			Platform:   p,
			Status:     LongTaskStatusPending,
			MaxRetries: maxRetries,
		}
	}

	state := &LongTaskState{
		TaskID:         taskID,
		Status:         LongTaskStatusPending,
		Article:        config.Article,
		Platforms:      config.Platforms,
		PlatformStates: platformStates,
		CreatedAt:      time.Now(),
		CurrentIndex:   0,
		TotalPlatforms: len(config.Platforms),
	}

	runner := &LongTaskRunner{
		state:      state,
		manager:    manager,
		emit:       emit,
		aiConfig:   config.AIConfig,
		accountIDs: config.AccountIDs,
		logger:     logger.GetLogger(),
		pauseChan:  make(chan struct{}),
		resumeChan: make(chan struct{}),
		repo:       config.Repo,
		skipCreate: config.ExistingRecord,
	}

	if !runner.skipCreate {
		runner.persistCreate()
	}

	return runner
}

func (r *LongTaskRunner) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.state.Status == LongTaskStatusRunning {
		r.mu.Unlock()
		return fmt.Errorf("task already running")
	}

	r.ctx, r.cancel = context.WithCancel(ctx)
	now := time.Now()
	r.state.Status = LongTaskStatusRunning
	r.state.StartedAt = &now
	r.mu.Unlock()

	r.persistStarted()
	r.emitStateUpdate()

	go r.runLoop()

	return nil
}

func (r *LongTaskRunner) runLoop() {
	defer func() {
		r.mu.Lock()
		now := time.Now()
		r.state.CompletedAt = &now
		finalStatus := r.state.Status
		if finalStatus == LongTaskStatusRunning {
			finalStatus = LongTaskStatusCompleted
			r.state.Status = finalStatus
		}
		r.mu.Unlock()
		r.persistCompleted(finalStatus)
		r.emitStateUpdate()
		r.emit("longtask:all_done", map[string]interface{}{
			"task_id": r.state.TaskID,
			"status":  string(r.state.Status),
		})
	}()

	for r.state.CurrentIndex < len(r.state.Platforms) {
		select {
		case <-r.ctx.Done():
			r.mu.Lock()
			r.state.Status = LongTaskStatusCancelled
			r.mu.Unlock()
			r.persistStatus(LongTaskStatusCancelled)
			return
		default:
		}

		if r.isPaused {
			select {
			case <-r.resumeChan:
				r.isPaused = false
			case <-r.ctx.Done():
				return
			}
		}

		platform := r.state.Platforms[r.state.CurrentIndex]
		r.processPlatform(platform)

		r.mu.Lock()
		r.state.CurrentIndex++
		r.mu.Unlock()

		r.persistProgress()

		if r.state.CurrentIndex < len(r.state.Platforms) {
			select {
			case <-r.ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}
}

func (r *LongTaskRunner) processPlatform(platform string) {
	r.mu.Lock()
	ps := r.state.PlatformStates[platform]
	now := time.Now()
	ps.Status = LongTaskStatusRunning
	ps.StartedAt = &now
	r.mu.Unlock()

	r.emit("longtask:platform_started", map[string]interface{}{
		"task_id":  r.state.TaskID,
		"platform": platform,
		"index":    r.state.CurrentIndex,
		"total":    r.state.TotalPlatforms,
	})

	accountID := r.accountIDs[platform]

	var lastErr error
	for attempt := 0; attempt <= ps.MaxRetries; attempt++ {
		if attempt > 0 {
			r.logger.Info(fmt.Sprintf("[LongTask] Retrying %s, attempt %d/%d", platform, attempt, ps.MaxRetries))
			r.emit("longtask:retry", map[string]interface{}{
				"task_id":  r.state.TaskID,
				"platform": platform,
				"attempt":  attempt,
				"max":      ps.MaxRetries,
			})
			time.Sleep(5 * time.Second)
		}

		pub, err := NewPublisher(platform, r.manager.factory, accountID)
		if err != nil {
			lastErr = err
			continue
		}

		r.emit("publish:progress", map[string]string{"platform": platform, "message": "检查登录状态..."})
		loggedIn, loginErr := pub.CheckLoginStatus()
		if loginErr != nil {
			lastErr = fmt.Errorf("检查登录状态失败: %w", loginErr)
			pub.Close()
			break
		}
		if !loggedIn {
			lastErr = fmt.Errorf("平台 %s 登录已过期，请先重新登录后再发布", platform)
			r.emit("publish:progress", map[string]string{"platform": platform, "message": "登录已过期，请重新登录"})
			pub.Close()
			break
		}

		resumeCh := make(chan struct{}, 1)

		platformEmit := func(event string, data interface{}) {
			r.emit(event, data)
		}

		err = pub.Publish(r.ctx, r.state.Article, resumeCh, platformEmit, r.aiConfig)
		pub.Close()

		if err == nil || err == errAlreadyEmitted {
			r.mu.Lock()
			completedNow := time.Now()
			ps.Status = LongTaskStatusCompleted
			ps.CompletedAt = &completedNow
			ps.Retries = attempt
			r.mu.Unlock()

			r.emit("longtask:platform_completed", map[string]interface{}{
				"task_id":  r.state.TaskID,
				"platform": platform,
				"success":  true,
			})
			return
		}

		lastErr = err
		r.mu.Lock()
		ps.Retries = attempt + 1
		r.mu.Unlock()
	}

	r.mu.Lock()
	failedNow := time.Now()
	ps.Status = LongTaskStatusFailed
	ps.CompletedAt = &failedNow
	if lastErr != nil {
		ps.Error = lastErr.Error()
	}
	r.mu.Unlock()

	r.emit("longtask:platform_completed", map[string]interface{}{
		"task_id":  r.state.TaskID,
		"platform": platform,
		"success":  false,
		"error":    ps.Error,
	})
}

func (r *LongTaskRunner) Pause() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state.Status != LongTaskStatusRunning {
		return
	}

	r.isPaused = true
	r.state.Status = LongTaskStatusPaused
	r.persistStatus(LongTaskStatusPaused)
	r.emitStateUpdate()
}

func (r *LongTaskRunner) Resume() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state.Status != LongTaskStatusPaused {
		return
	}

	r.state.Status = LongTaskStatusRunning
	r.persistStatus(LongTaskStatusRunning)
	select {
	case r.resumeChan <- struct{}{}:
	default:
	}
	r.emitStateUpdate()
}

func (r *LongTaskRunner) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
	}
	r.state.Status = LongTaskStatusCancelled
	r.persistStatus(LongTaskStatusCancelled)
	r.emitStateUpdate()
}

func (r *LongTaskRunner) GetState() *LongTaskState {
	r.mu.Lock()
	defer r.mu.Unlock()

	stateCopy := *r.state
	return &stateCopy
}

func (r *LongTaskRunner) GetStateJSON() (string, error) {
	state := r.GetState()
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *LongTaskRunner) emitStateUpdate() {
	r.emit("longtask:state_update", map[string]interface{}{
		"task_id":         r.state.TaskID,
		"status":          string(r.state.Status),
		"current_index":   r.state.CurrentIndex,
		"total_platforms": r.state.TotalPlatforms,
		"platform_states": r.state.PlatformStates,
	})
}

func (r *LongTaskRunner) persistCreate() {
	if r.repo == nil {
		return
	}

	articleJSON, _ := json.Marshal(r.state.Article)
	platformsJSON, _ := json.Marshal(r.state.Platforms)
	accountIDsJSON, _ := json.Marshal(r.accountIDs)

	err := r.repo.Create(
		r.state.TaskID,
		string(r.state.Status),
		string(articleJSON),
		string(platformsJSON),
		string(accountIDsJSON),
		r.state.TotalPlatforms,
	)
	if err != nil {
		r.logger.Warn(fmt.Sprintf("[LongTask] Failed to persist task creation: %v", err))
	}
}

func (r *LongTaskRunner) persistProgress() {
	if r.repo == nil {
		return
	}

	platformStatesJSON, _ := json.Marshal(r.state.PlatformStates)
	err := r.repo.UpdateProgress(r.state.TaskID, r.state.CurrentIndex, string(platformStatesJSON))
	if err != nil {
		r.logger.Warn(fmt.Sprintf("[LongTask] Failed to persist progress: %v", err))
	}
}

func (r *LongTaskRunner) persistStatus(status LongTaskStatus) {
	if r.repo == nil {
		return
	}

	err := r.repo.UpdateStatus(r.state.TaskID, string(status))
	if err != nil {
		r.logger.Warn(fmt.Sprintf("[LongTask] Failed to persist status: %v", err))
	}
}

func (r *LongTaskRunner) persistStarted() {
	if r.repo == nil {
		return
	}

	err := r.repo.SetStarted(r.state.TaskID)
	if err != nil {
		r.logger.Warn(fmt.Sprintf("[LongTask] Failed to persist started: %v", err))
	}
}

func (r *LongTaskRunner) persistCompleted(status LongTaskStatus) {
	if r.repo == nil {
		return
	}

	platformStatesJSON, _ := json.Marshal(r.state.PlatformStates)
	_ = r.repo.UpdateProgress(r.state.TaskID, r.state.CurrentIndex, string(platformStatesJSON))
	err := r.repo.SetCompleted(r.state.TaskID, string(status))
	if err != nil {
		r.logger.Warn(fmt.Sprintf("[LongTask] Failed to persist completed: %v", err))
	}
}

type LongTaskManager struct {
	mu      sync.Mutex
	tasks   map[string]*LongTaskRunner
	manager *Manager
	logger  *logger.Logger
	repo    *repositories.LongTaskRepository
}

func NewLongTaskManager(manager *Manager, repo *repositories.LongTaskRepository) *LongTaskManager {
	return &LongTaskManager{
		tasks:   make(map[string]*LongTaskRunner),
		manager: manager,
		logger:  logger.GetLogger(),
		repo:    repo,
	}
}

func (m *LongTaskManager) CreateTask(emit EventEmitter, config LongTaskConfig) *LongTaskRunner {
	config.Repo = m.repo
	runner := NewLongTaskRunner(m.manager, emit, config)

	m.mu.Lock()
	m.tasks[runner.state.TaskID] = runner
	m.mu.Unlock()

	return runner
}

func (m *LongTaskManager) GetTask(taskID string) *LongTaskRunner {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tasks[taskID]
}

func (m *LongTaskManager) ListTasks() []*LongTaskState {
	m.mu.Lock()
	defer m.mu.Unlock()

	states := make([]*LongTaskState, 0, len(m.tasks))
	for _, task := range m.tasks {
		states = append(states, task.GetState())
	}
	return states
}

func (m *LongTaskManager) RemoveTask(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if task, ok := m.tasks[taskID]; ok {
		task.Cancel()
		delete(m.tasks, taskID)
	}

	if m.repo != nil {
		_ = m.repo.Delete(taskID)
	}
}

// DropTask stops a task runner and removes it from memory without deleting the DB record.
func (m *LongTaskManager) DropTask(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if task, ok := m.tasks[taskID]; ok {
		task.Cancel()
		delete(m.tasks, taskID)
	}
}

func (m *LongTaskManager) CleanupCompletedTasks(olderThan time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := 0
	now := time.Now()

	for taskID, task := range m.tasks {
		state := task.GetState()
		if state.CompletedAt != nil && now.Sub(*state.CompletedAt) > olderThan {
			delete(m.tasks, taskID)
			removed++
		}
	}

	if m.repo != nil {
		dbRemoved, _ := m.repo.DeleteOlderThan(olderThan)
		removed += int(dbRemoved)
	}

	return removed
}

func (m *LongTaskManager) RestoreUnfinishedTasks(emit EventEmitter, aiConfig AIPublishConfig) ([]*LongTaskState, error) {
	if m.repo == nil {
		return nil, nil
	}

	records, err := m.repo.GetUnfinished()
	if err != nil {
		return nil, fmt.Errorf("failed to get unfinished tasks: %w", err)
	}

	var restored []*LongTaskState

	for _, rec := range records {
		var article Article
		if err := json.Unmarshal([]byte(rec.ArticleJSON), &article); err != nil {
			m.logger.Warn(fmt.Sprintf("[LongTask] Failed to unmarshal article for task %s: %v", rec.TaskID, err))
			continue
		}

		var platforms []string
		if err := json.Unmarshal([]byte(rec.PlatformsJSON), &platforms); err != nil {
			m.logger.Warn(fmt.Sprintf("[LongTask] Failed to unmarshal platforms for task %s: %v", rec.TaskID, err))
			continue
		}

		var accountIDs map[string]string
		if err := json.Unmarshal([]byte(rec.AccountIDsJSON), &accountIDs); err != nil {
			m.logger.Warn(fmt.Sprintf("[LongTask] Failed to unmarshal account IDs for task %s: %v", rec.TaskID, err))
			continue
		}

		platformStates := make(map[string]*PlatformTaskState)
		if rec.PlatformStatesJSON.Valid && rec.PlatformStatesJSON.String != "" {
			if err := json.Unmarshal([]byte(rec.PlatformStatesJSON.String), &platformStates); err != nil {
				m.logger.Warn(fmt.Sprintf("[LongTask] Failed to unmarshal platform states for task %s: %v", rec.TaskID, err))
			}
		}

		if len(platformStates) == 0 {
			for _, p := range platforms {
				platformStates[p] = &PlatformTaskState{
					Platform:   p,
					Status:     LongTaskStatusPending,
					MaxRetries: 2,
				}
			}
		}

		state := &LongTaskState{
			TaskID:         rec.TaskID,
			Status:         LongTaskStatus(rec.Status),
			Article:        article,
			Platforms:      platforms,
			PlatformStates: platformStates,
			CreatedAt:      rec.CreatedAt,
			CurrentIndex:   rec.CurrentIndex,
			TotalPlatforms: rec.TotalPlatforms,
		}

		if rec.StartedAt.Valid {
			state.StartedAt = &rec.StartedAt.Time
		}
		if rec.CompletedAt.Valid {
			state.CompletedAt = &rec.CompletedAt.Time
		}

		runner := &LongTaskRunner{
			state:      state,
			manager:    m.manager,
			emit:       emit,
			aiConfig:   aiConfig,
			accountIDs: accountIDs,
			logger:     logger.GetLogger(),
			pauseChan:  make(chan struct{}),
			resumeChan: make(chan struct{}),
			repo:       m.repo,
			isPaused:   state.Status == LongTaskStatusPaused,
		}

		m.mu.Lock()
		m.tasks[rec.TaskID] = runner
		m.mu.Unlock()

		restored = append(restored, state)
		m.logger.Info(fmt.Sprintf("[LongTask] Restored task %s with status %s (index %d/%d)",
			rec.TaskID, rec.Status, rec.CurrentIndex, rec.TotalPlatforms))
	}

	return restored, nil
}

func (m *LongTaskManager) GetRepo() *repositories.LongTaskRepository {
	return m.repo
}
