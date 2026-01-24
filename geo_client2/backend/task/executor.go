package task

import (
	"context"
	"fmt"
	stdRuntime "runtime"
	"sync"
	"time"

	"geo_client2/backend/database/repositories"
	"geo_client2/backend/logger"
	"geo_client2/backend/provider"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Executor executes tasks.
type Executor struct {
	taskRepo     *repositories.TaskRepository
	providerFact *provider.Factory
	accountRepo  *repositories.AccountRepository
	logger       *logger.Logger
	eventCtx     context.Context
	runningTasks map[int]context.CancelFunc
	accountLocks sync.Map
	mu           sync.RWMutex
}

// NewExecutor creates a new executor.
func NewExecutor(taskRepo *repositories.TaskRepository, providerFact *provider.Factory, accountRepo *repositories.AccountRepository, eventCtx context.Context) *Executor {
	return &Executor{
		taskRepo:     taskRepo,
		providerFact: providerFact,
		accountRepo:  accountRepo,
		logger:       logger.GetLogger(),
		eventCtx:     eventCtx,
		runningTasks: make(map[int]context.CancelFunc),
		accountLocks: sync.Map{},
	}
}

// ExecuteLocalTask executes a local search task.
func (e *Executor) ExecuteLocalTask(taskID int, keywords, platforms []string, queryCount int, settings map[string]interface{}) error {
	ctx, cancel := context.WithCancel(e.eventCtx)

	e.mu.Lock()
	e.runningTasks[taskID] = cancel
	e.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 2048)
			n := stdRuntime.Stack(buf, false)
			stack := string(buf[:n])
			e.logger.ErrorWithContext(ctx, "Panic in ExecuteLocalTask", map[string]interface{}{
				"panic": r,
				"stack": stack,
			}, fmt.Errorf("%v", r), &taskID)
			e.taskRepo.UpdateStatus(taskID, "failed", nil)
		}
		e.mu.Lock()
		delete(e.runningTasks, taskID)
		e.mu.Unlock()
		cancel()
	}()

	timer := e.logger.StartTimer(ctx, "ExecuteLocalTask", &taskID)

	e.logger.InfoWithContext(ctx, "Task execution started", map[string]interface{}{
		"keywords":    keywords,
		"platforms":   platforms,
		"query_count": queryCount,
	}, &taskID)
	e.taskRepo.UpdateStatus(taskID, "running", nil)

	delay := 5 * time.Second
	headless := true
	if e.providerFact != nil {
		headless = e.providerFact.IsHeadless()
	}

	if settings != nil {
		if d, ok := settings["delay_between_tasks"].(float64); ok {
			delay = time.Duration(d) * time.Second
		}
		if h, ok := settings["headless"].(bool); ok {
			headless = h
		}
	}

	totalQueries := len(keywords) * len(platforms) * queryCount
	completedQueries := 0
	failedQueries := 0

	providers := make(map[string]provider.Provider)
	defer func() {
		for _, p := range providers {
			p.Close()
		}
	}()

	for round := 1; round <= queryCount; round++ {
		if ctx.Err() != nil {
			e.taskRepo.UpdateStatus(taskID, "cancelled", nil)
			return ctx.Err()
		}

		for _, keyword := range keywords {
			if ctx.Err() != nil {
				e.taskRepo.UpdateStatus(taskID, "cancelled", nil)
				return ctx.Err()
			}

			for _, platformName := range platforms {
				if ctx.Err() != nil {
					e.taskRepo.UpdateStatus(taskID, "cancelled", nil)
					return ctx.Err()
				}

				activeAccount, err := e.accountRepo.GetActiveAccount(platformName)
				if err != nil {
					e.logger.ErrorWithContext(ctx, "Failed to get active account", map[string]interface{}{
						"platform": platformName,
					}, err, &taskID)
					e.taskRepo.SaveSearchRecord(taskID, keyword, platformName, keyword, "", 0, "failed", fmt.Sprintf("failed to get active account: %v", err), round)
					failedQueries++
					continue
				}
				if activeAccount == nil {
					e.logger.WarnWithContext(ctx, "No active account for platform", map[string]interface{}{
						"platform": platformName,
					}, &taskID)
					e.taskRepo.SaveSearchRecord(taskID, keyword, platformName, keyword, "", 0, "failed", "no active account found for this platform", round)
					failedQueries++
					continue
				}

				cacheKey := fmt.Sprintf("%s:%s", platformName, activeAccount.AccountID)

				lock, _ := e.accountLocks.LoadOrStore(activeAccount.AccountID, &sync.Mutex{})
				mutex := lock.(*sync.Mutex)
				mutex.Lock()

				prov, ok := providers[cacheKey]
				if !ok {
					var errProv error
					prov, errProv = e.providerFact.GetProvider(platformName, headless, 60000, activeAccount.AccountID)
					if errProv != nil {
						mutex.Unlock()
						e.logger.ErrorWithContext(ctx, "Failed to get provider", map[string]interface{}{
							"platform":   platformName,
							"account_id": activeAccount.AccountID,
						}, errProv, &taskID)
						e.taskRepo.SaveSearchRecord(taskID, keyword, platformName, keyword, "", 0, "failed", fmt.Sprintf("failed to initialize provider: %v", errProv), round)
						failedQueries++
						continue
					}
					providers[cacheKey] = prov
				}

				searchStartTime := time.Now()
				searchTimer := e.logger.StartTimer(ctx, fmt.Sprintf("Search:%s:%s", platformName, keyword), &taskID)
				result, err := prov.Search(ctx, keyword, keyword)
				searchDuration := time.Since(searchStartTime)

				searchDetails := map[string]interface{}{
					"platform": platformName,
					"keyword":  keyword,
					"success":  err == nil,
				}
				if result != nil {
					searchDetails["result_length"] = len(result.FullText)
					searchDetails["citations_count"] = len(result.Citations)
				}
				searchTimer.End(searchDetails)

				mutex.Unlock()

				if err != nil {
					e.logger.ErrorWithContext(ctx, fmt.Sprintf("[%s] Search failed for: %s", platformName, keyword), map[string]interface{}{
						"platform": platformName,
						"keyword":  keyword,
					}, err, &taskID)
					failedQueries++
				} else {
					e.logger.InfoWithContext(ctx, fmt.Sprintf("[%s] Search success: %s", platformName, keyword), map[string]interface{}{
						"platform": platformName,
						"keyword":  keyword,
						"result":   result,
					}, &taskID)
					completedQueries++
				}

				status := "completed"
				errorMsg := ""
				if err != nil {
					status = "failed"
					errorMsg = err.Error()
				}

				fullAnswer := ""
				if result != nil {
					fullAnswer = result.FullText
				}

				recordID, saveErr := e.taskRepo.SaveSearchRecord(taskID, keyword, platformName, keyword, fullAnswer, int(searchDuration.Milliseconds()), status, errorMsg, round)
				if saveErr == nil && result != nil {
					for i, citation := range result.Citations {
						e.taskRepo.SaveCitation(recordID, i, citation.URL, citation.Domain, citation.Title, citation.Snippet, "")
					}
					for i, query := range result.Queries {
						e.taskRepo.SaveSearchQuery(recordID, query, i)
					}
				}

				e.taskRepo.UpdateStats(taskID, totalQueries, completedQueries+failedQueries, 0, completedQueries, failedQueries, 0)

				runtime.EventsEmit(e.eventCtx, "search:taskUpdated", map[string]interface{}{
					"taskId": taskID,
					"progress": map[string]interface{}{
						"completed": completedQueries + failedQueries,
						"total":     totalQueries,
						"success":   completedQueries,
						"failed":    failedQueries,
					},
				})

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
				}
			}
		}
	}

	finalStatus := "completed"
	if completedQueries == 0 && totalQueries > 0 {
		finalStatus = "failed"
	} else if failedQueries > 0 {
		finalStatus = "partial_completed"
	}

	e.taskRepo.UpdateStatus(taskID, finalStatus, nil)
	timer.End(map[string]interface{}{
		"completed_queries": completedQueries,
		"failed_queries":    failedQueries,
		"total_queries":     totalQueries,
	})
	e.logger.InfoWithContext(ctx, "Task execution completed", map[string]interface{}{
		"status":            finalStatus,
		"completed_queries": completedQueries,
		"failed_queries":    failedQueries,
		"total_queries":     totalQueries,
	}, &taskID)
	return nil
}

// CancelTask cancels a running task.
func (e *Executor) CancelTask(taskID int) error {
	e.mu.Lock()
	cancel, ok := e.runningTasks[taskID]
	if ok {
		delete(e.runningTasks, taskID)
	}
	e.mu.Unlock()

	if ok {
		cancel()
		e.taskRepo.UpdateStatus(taskID, "cancelled", nil)
		runtime.EventsEmit(e.eventCtx, "search:taskUpdated", map[string]interface{}{
			"taskId": taskID,
			"progress": map[string]interface{}{
				"completed": 0,
				"total":     0,
				"success":   0,
				"failed":    0,
			},
		})
		e.logger.InfoWithContext(context.Background(), "Task cancelled successfully", map[string]interface{}{
			"task_id": taskID,
		}, &taskID)
		return nil
	}

	// Force cancel logic: if task status is 'running' in DB, mark it as cancelled
	task, err := e.taskRepo.GetByID(taskID)
	if err == nil && task != nil {
		if status, ok := task["status"].(string); ok && status == "running" {
			e.taskRepo.UpdateStatus(taskID, "cancelled", nil)
			runtime.EventsEmit(e.eventCtx, "search:taskUpdated", map[string]interface{}{
				"taskId": taskID,
				"progress": map[string]interface{}{
					"completed": 0,
					"total":     0,
					"success":   0,
					"failed":    0,
				},
			})
			e.logger.InfoWithContext(context.Background(), "Task force cancelled (was stuck in running state)", map[string]interface{}{
				"task_id": taskID,
			}, &taskID)
			return nil
		}
	}

	// If not in runningTasks, it might have already finished or failed
	err = fmt.Errorf("task %d not found in running tasks (it may have already finished or failed)", taskID)
	e.logger.WarnWithContext(context.Background(), "Attempted to cancel task that is not running", map[string]interface{}{
		"task_id": taskID,
	}, &taskID)
	return err
}

func stringPtr(s string) *string {
	return &s
}
