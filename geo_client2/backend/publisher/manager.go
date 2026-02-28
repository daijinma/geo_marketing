package publisher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"geo_client2/backend/logger"
	"geo_client2/backend/provider"
)

// Article represents the content to be published.
type Article struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	CoverImage string `json:"cover_image,omitempty"` // URL or base64 of cover image
}

// PublishResult represents the result of a publish operation for one platform.
type PublishResult struct {
	Platform                string `json:"platform"`
	Success                 bool   `json:"success"`
	ArticleURL              string `json:"article_url,omitempty"`
	Error                   string `json:"error,omitempty"`
	NeedsManualIntervention bool   `json:"needs_manual_intervention"`
	InterventionPrompt      string `json:"intervention_prompt,omitempty"`
}

// Publisher interface for platform publishers.
type Publisher interface {
	Publish(ctx context.Context, article Article, resume <-chan struct{}, emit EventEmitter, aiConfig AIPublishConfig) error
	CheckLoginStatus() (bool, error)
	GetLoginUrl() string
	StartLogin() (func(), error)
	Close() error
}

// EventEmitter is a callback for emitting progress events to the frontend.
type EventEmitter func(event string, data interface{})

// publishJob tracks the state of a single platform publish task.
type publishJob struct {
	platform string
	taskID   string
	resume   chan struct{}
	cancel   context.CancelFunc
}

// Manager manages concurrent publish jobs.
type Manager struct {
	mu             sync.Mutex
	jobs           map[string]*publishJob
	resumeRegistry map[string]chan struct{}
	logger         *logger.Logger
	factory        *provider.Factory
}

// NewManager creates a new publish manager.
func NewManager(factory *provider.Factory) *Manager {
	return &Manager{
		jobs:           make(map[string]*publishJob),
		resumeRegistry: make(map[string]chan struct{}),
		logger:         logger.GetLogger(),
		factory:        factory,
	}
}

// RegisterResumeChannel registers an external resume channel (e.g. longtask per-platform) keyed by taskID.
func (m *Manager) RegisterResumeChannel(taskID string, ch chan struct{}) {
	m.mu.Lock()
	m.resumeRegistry[taskID] = ch
	m.mu.Unlock()
}

// UnregisterResumeChannel removes a previously registered resume channel.
func (m *Manager) UnregisterResumeChannel(taskID string) {
	m.mu.Lock()
	delete(m.resumeRegistry, taskID)
	m.mu.Unlock()
}

// StartPublish starts publishing an article to multiple platforms concurrently.
// accountIDs maps platform -> accountID.
func (m *Manager) StartPublish(
	ctx context.Context,
	platforms []string,
	accountIDs map[string]string,
	article Article,
	emit EventEmitter,
	aiConfig AIPublishConfig,
) error {
	for _, platform := range platforms {
		accountID := accountIDs[platform]
		taskID := fmt.Sprintf("%s-%d", platform, time.Now().UnixNano())

		jobCtx, cancel := context.WithCancel(ctx)
		resumeCh := make(chan struct{}, 1)

		job := &publishJob{
			platform: platform,
			taskID:   taskID,
			resume:   resumeCh,
			cancel:   cancel,
		}

		m.mu.Lock()
		m.jobs[taskID] = job
		m.mu.Unlock()

		go func(p, tid, aid string, j *publishJob) {
			defer func() {
				m.mu.Lock()
				delete(m.jobs, tid)
				m.mu.Unlock()
				cancel()
			}()

			emit("publish:started", map[string]string{"platform": p, "taskId": tid})

			pub, err := NewPublisher(p, m.factory, aid)
			if err != nil {
				emit("publish:completed", map[string]interface{}{
					"platform": p,
					"success":  false,
					"error":    fmt.Sprintf("不支持的平台: %v", err),
				})
				return
			}
			defer pub.Close()

			if fp, ok := pub.(*FlowPublisher); ok {
				fp.jobTaskID = tid
			}

			emit("publish:progress", map[string]string{"platform": p, "message": "检查登录状态..."})
			loggedIn, err := pub.CheckLoginStatus()
			if err != nil {
				emit("publish:completed", map[string]interface{}{
					"platform": p,
					"success":  false,
					"error":    fmt.Sprintf("检查登录状态失败: %v", err),
				})
				return
			}

			if !loggedIn {
				emit("publish:completed", map[string]interface{}{
					"platform": p,
					"success":  false,
					"error":    "登录已过期，请先重新登录后再发布",
				})
				return
			}

			if err := pub.Publish(jobCtx, article, j.resume, emit, aiConfig); err != nil {
				if err == errAlreadyEmitted {
					// Publisher handled its own publish:completed event; nothing to do.
					return
				}
				if jobCtx.Err() != nil {
					emit("publish:completed", map[string]interface{}{
						"platform": p,
						"success":  false,
						"error":    "已取消",
					})
					return
				}
				emit("publish:completed", map[string]interface{}{
					"platform": p,
					"success":  false,
					"error":    err.Error(),
				})
				return
			}

			emit("publish:completed", map[string]interface{}{
				"platform": p,
				"success":  true,
			})
		}(platform, taskID, accountID, job)
	}

	// Watch all jobs and emit all_done when complete
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.mu.Lock()
				remaining := len(m.jobs)
				m.mu.Unlock()
				if remaining == 0 {
					emit("publish:all_done", map[string]interface{}{})
					return
				}
			}
		}
	}()

	return nil
}

// ResumePublish signals a waiting job to continue after manual intervention.
func (m *Manager) ResumePublish(taskID string) error {
	m.mu.Lock()
	job, inJobs := m.jobs[taskID]
	ch, inRegistry := m.resumeRegistry[taskID]
	m.mu.Unlock()

	if inJobs {
		select {
		case job.resume <- struct{}{}:
		default:
		}
		return nil
	}
	if inRegistry {
		select {
		case ch <- struct{}{}:
		default:
		}
		return nil
	}
	return fmt.Errorf("publish job not found: %s", taskID)
}

// CancelPublish cancels a running publish job.
func (m *Manager) CancelPublish(taskID string) error {
	m.mu.Lock()
	job, ok := m.jobs[taskID]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("publish job not found: %s", taskID)
	}

	job.cancel()
	return nil
}
