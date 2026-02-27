package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

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
}

type LongTaskConfig struct {
	Platforms             []string
	AccountIDs            map[string]string
	Article               Article
	AIConfig              AIPublishConfig
	MaxRetriesPerPlatform int
	DelayBetweenPlatforms time.Duration
}

func NewLongTaskRunner(manager *Manager, emit EventEmitter, config LongTaskConfig) *LongTaskRunner {
	taskID := fmt.Sprintf("longtask-%d", time.Now().UnixNano())

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

	return &LongTaskRunner{
		state:      state,
		manager:    manager,
		emit:       emit,
		aiConfig:   config.AIConfig,
		accountIDs: config.AccountIDs,
		logger:     logger.GetLogger(),
		pauseChan:  make(chan struct{}),
		resumeChan: make(chan struct{}),
	}
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

	r.emitStateUpdate()

	go r.runLoop()

	return nil
}

func (r *LongTaskRunner) runLoop() {
	defer func() {
		r.mu.Lock()
		now := time.Now()
		r.state.CompletedAt = &now
		if r.state.Status == LongTaskStatusRunning {
			r.state.Status = LongTaskStatusCompleted
		}
		r.mu.Unlock()
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
	r.emitStateUpdate()
}

func (r *LongTaskRunner) Resume() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state.Status != LongTaskStatusPaused {
		return
	}

	r.state.Status = LongTaskStatusRunning
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

type LongTaskManager struct {
	mu      sync.Mutex
	tasks   map[string]*LongTaskRunner
	manager *Manager
	logger  *logger.Logger
}

func NewLongTaskManager(manager *Manager) *LongTaskManager {
	return &LongTaskManager{
		tasks:   make(map[string]*LongTaskRunner),
		manager: manager,
		logger:  logger.GetLogger(),
	}
}

func (m *LongTaskManager) CreateTask(emit EventEmitter, config LongTaskConfig) *LongTaskRunner {
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

	return removed
}
