package backend

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	gouruntime "runtime"
	"strings"
	"time"

	"geo_client2/backend/account"
	"geo_client2/backend/auth"
	"geo_client2/backend/database"
	"geo_client2/backend/database/repositories"
	"geo_client2/backend/logger"
	"geo_client2/backend/provider"
	"geo_client2/backend/search"
	"geo_client2/backend/settings"
	"geo_client2/backend/task"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx          context.Context
	accountSvc   *account.Service
	authSvc      *auth.Service
	settingsSvc  *settings.Service
	taskManager  *task.Manager
	searchSvc    *search.Service
	providerFact *provider.Factory
	logRepo      *repositories.LogRepository
	loginCancel  func()
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize database
	if err := database.Init(); err != nil {
		log.Fatalf("Failed to initialize database, application cannot start: %v", err)
	}

	db := database.GetDB()

	logger.SetDB(db)
	l := logger.GetLogger()
	stdout, _ := l.RedirectStdout()
	log.SetOutput(stdout)

	// Initialize repositories
	accountRepo := repositories.NewAccountRepository(db)
	authRepo := repositories.NewAuthRepository(db)
	settingsRepo := repositories.NewSettingsRepository(db)
	taskRepo := repositories.NewTaskRepository(db)
	loginRepo := repositories.NewLoginStatusRepository(db)
	logRepo := repositories.NewLogRepository(db)

	// Initialize services
	a.accountSvc = account.NewService(accountRepo)
	a.authSvc = auth.NewService(authRepo)
	a.settingsSvc = settings.NewService(settingsRepo)
	a.logRepo = logRepo

	// Initialize provider factory
	headlessStr, _ := a.settingsSvc.Get("browser_headless")
	headless := headlessStr != "false" // Default to true if not set or "true"
	a.providerFact = provider.NewFactory(headless, 60000)

	// Initialize task executor and manager
	executor := task.NewExecutor(taskRepo, a.providerFact, accountRepo, ctx)
	a.taskManager = task.NewManager(taskRepo, loginRepo, executor, ctx)

	// Initialize search service
	a.searchSvc = search.NewService(a.taskManager, a.providerFact, loginRepo, accountRepo, settingsRepo)
}

func (a *App) Shutdown(ctx context.Context) {
	database.Close()
}

// Greet returns a greeting (placeholder for Phase 1 binding test).
func (a *App) Greet(name string) string {
	return "Hello, " + name + "!"
}

// EmitTestEvent emits a test event for Phase 1 verification.
func (a *App) EmitTestEvent() {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "test", map[string]string{"message": "hello from Go"})
	}
}

// Settings methods
func (a *App) GetSetting(key string) (string, error) {
	if a.settingsSvc == nil {
		return "", fmt.Errorf("settings service not initialized")
	}
	return a.settingsSvc.Get(key)
}

func (a *App) SetSetting(key, value string) error {
	if a.settingsSvc == nil {
		return fmt.Errorf("settings service not initialized")
	}
	if a.providerFact != nil && key == "browser_headless" {
		a.providerFact.SetHeadless(value != "false")
	}
	return a.settingsSvc.Set(key, value)
}

// Task methods
func (a *App) CreateLocalSearchTask(keywordsJSON, platformsJSON string, queryCount int) (map[string]interface{}, error) {
	if a.taskManager == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}
	var keywords, platforms []string
	json.Unmarshal([]byte(keywordsJSON), &keywords)
	json.Unmarshal([]byte(platformsJSON), &platforms)

	taskID, err := a.taskManager.CreateLocalSearchTask(keywords, platforms, queryCount, "local_search", "local", nil, nil)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "taskId": taskID}, nil
}

func (a *App) GetAllTasks(limit, offset int, filtersJSON string) (map[string]interface{}, error) {
	if a.taskManager == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}
	var filters *repositories.TaskFilters
	if filtersJSON != "" {
		json.Unmarshal([]byte(filtersJSON), &filters)
	}
	tasks, err := a.taskManager.GetAllTasks(limit, offset, filters)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "tasks": tasks}, nil
}

func (a *App) GetTaskDetail(taskID int) (map[string]interface{}, error) {
	if a.taskManager == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}
	detail, err := a.taskManager.GetTaskDetail(taskID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "data": detail}, nil
}

func (a *App) CancelTask(taskID int) error {
	if a.taskManager == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.taskManager.CancelTask(taskID)
}

func (a *App) RetryTask(taskID int) error {
	if a.taskManager == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.taskManager.RetryTask(taskID)
}

func (a *App) ContinueTask(taskID int) error {
	if a.taskManager == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.taskManager.ContinueTask(taskID)
}

func (a *App) DeleteTask(taskID int) error {
	if a.taskManager == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.taskManager.DeleteTask(taskID)
}

func (a *App) UpdateTaskName(taskID int, name string) error {
	if a.taskManager == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.taskManager.UpdateTaskName(taskID, name)
}

func (a *App) GetStats() (map[string]interface{}, error) {
	if a.taskManager == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}
	return a.taskManager.GetStats()
}

// Search methods
func (a *App) CheckLoginStatus(platform string) (map[string]interface{}, error) {
	if a.searchSvc == nil {
		return nil, fmt.Errorf("search service not initialized")
	}
	loggedIn, err := a.searchSvc.CheckLoginStatus(platform)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "isLoggedIn": loggedIn}, nil
}

func (a *App) BatchCheckLoginStatus() error {
	if a.accountSvc == nil {
		return fmt.Errorf("account service not initialized")
	}
	if a.providerFact == nil {
		return fmt.Errorf("provider factory not initialized")
	}

	go func() {
		l := logger.GetLogger()
		l.Info("[BATCH-CHECK] Starting batch login status check")

		runtime.EventsEmit(a.ctx, "batch-check:started", map[string]interface{}{
			"message": "开始批量检测登录状态",
		})

		platforms := []string{"deepseek", "doubao", "yiyan", "yuanbao", "xiaohongshu"}
		totalPlatforms := len(platforms)
		checked := 0

		for _, platform := range platforms {
			activeAccount, err := a.accountSvc.GetActiveAccount(platform)
			if err != nil || activeAccount == nil {
				l.Info(fmt.Sprintf("[BATCH-CHECK] Skipping %s: no active account", platform))
				checked++
				runtime.EventsEmit(a.ctx, "batch-check:progress", map[string]interface{}{
					"platform": platform,
					"checked":  checked,
					"total":    totalPlatforms,
					"skipped":  true,
					"message":  fmt.Sprintf("跳过 %s (无活跃账号)", platform),
				})
				continue
			}

			l.Info(fmt.Sprintf("[BATCH-CHECK] Checking %s (account: %s)", platform, activeAccount.AccountID))

			runtime.EventsEmit(a.ctx, "batch-check:progress", map[string]interface{}{
				"platform":   platform,
				"account_id": activeAccount.AccountID,
				"checked":    checked,
				"total":      totalPlatforms,
				"checking":   true,
				"message":    fmt.Sprintf("正在检测 %s...", platform),
			})

			prov, err := a.providerFact.GetProvider(platform, true, 60000, activeAccount.AccountID)
			if err != nil {
				l.Error(fmt.Sprintf("[BATCH-CHECK] Failed to get provider for %s", platform), err)
				checked++
				runtime.EventsEmit(a.ctx, "batch-check:progress", map[string]interface{}{
					"platform":   platform,
					"account_id": activeAccount.AccountID,
					"checked":    checked,
					"total":      totalPlatforms,
					"error":      true,
					"message":    fmt.Sprintf("%s 检测失败: %s", platform, err.Error()),
				})
				continue
			}

			isLoggedIn, err := prov.CheckLoginStatus()
			prov.Close()

			if err != nil {
				l.Error(fmt.Sprintf("[BATCH-CHECK] Error checking %s", platform), err)
				checked++
				runtime.EventsEmit(a.ctx, "batch-check:progress", map[string]interface{}{
					"platform":   platform,
					"account_id": activeAccount.AccountID,
					"checked":    checked,
					"total":      totalPlatforms,
					"error":      true,
					"message":    fmt.Sprintf("%s 检测出错: %s", platform, err.Error()),
				})
				continue
			}

			checked++
			l.Info(fmt.Sprintf("[BATCH-CHECK] %s result: %v", platform, isLoggedIn))

			runtime.EventsEmit(a.ctx, "batch-check:progress", map[string]interface{}{
				"platform":     platform,
				"account_id":   activeAccount.AccountID,
				"checked":      checked,
				"total":        totalPlatforms,
				"is_logged_in": isLoggedIn,
				"message":      fmt.Sprintf("%s: %s", platform, map[bool]string{true: "登录正常", false: "登录已过期"}[isLoggedIn]),
			})
		}

		runtime.EventsEmit(a.ctx, "batch-check:completed", map[string]interface{}{
			"checked": checked,
			"total":   totalPlatforms,
			"message": "批量检测完成",
		})

		l.Info("[BATCH-CHECK] Batch check completed")
	}()

	return nil
}

// Account methods
func (a *App) CreateAccount(platform, accountName string) (map[string]interface{}, error) {
	if a.accountSvc == nil {
		return nil, fmt.Errorf("account service not initialized")
	}
	l := logger.GetLogger()
	l.Debug(fmt.Sprintf("App.CreateAccount called with: platform=%s, name=%s", platform, accountName))
	acc, err := a.accountSvc.CreateAccount(platform, accountName)
	if err != nil {
		l.Error("App.CreateAccount error", err)
		return nil, err
	}
	l.Debug(fmt.Sprintf("App.CreateAccount success: %v", acc.AccountID))
	return map[string]interface{}{
		"success": true,
		"account": map[string]interface{}{
			"id":            acc.ID,
			"platform":      acc.Platform,
			"account_id":    acc.AccountID,
			"account_name":  acc.AccountName,
			"user_data_dir": acc.UserDataDir,
			"is_active":     acc.IsActive,
			"category":      acc.Category,
			"created_at":    acc.CreatedAt,
			"updated_at":    acc.UpdatedAt,
		},
	}, nil
}

func (a *App) ListAccounts(platform string) (map[string]interface{}, error) {
	if a.accountSvc == nil {
		return nil, fmt.Errorf("account service not initialized")
	}
	accounts, err := a.accountSvc.ListAccounts(platform)
	if err != nil {
		return nil, err
	}
	accountList := make([]map[string]interface{}, len(accounts))
	for i, acc := range accounts {
		accountList[i] = map[string]interface{}{
			"id":            acc.ID,
			"platform":      acc.Platform,
			"account_id":    acc.AccountID,
			"account_name":  acc.AccountName,
			"user_data_dir": acc.UserDataDir,
			"is_active":     acc.IsActive,
			"category":      acc.Category,
			"created_at":    acc.CreatedAt,
			"updated_at":    acc.UpdatedAt,
		}
	}
	return map[string]interface{}{"success": true, "accounts": accountList}, nil
}

func (a *App) GetActiveAccount(platform string) (map[string]interface{}, error) {
	if a.accountSvc == nil {
		return nil, fmt.Errorf("account service not initialized")
	}
	acc, err := a.accountSvc.GetActiveAccount(platform)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return map[string]interface{}{"success": true, "account": nil}, nil
	}
	return map[string]interface{}{
		"success": true,
		"account": map[string]interface{}{
			"id":            acc.ID,
			"platform":      acc.Platform,
			"account_id":    acc.AccountID,
			"account_name":  acc.AccountName,
			"user_data_dir": acc.UserDataDir,
			"is_active":     acc.IsActive,
			"category":      acc.Category,
			"created_at":    acc.CreatedAt,
			"updated_at":    acc.UpdatedAt,
		},
	}, nil
}

func (a *App) SetActiveAccount(platform, accountID string) error {
	if a.accountSvc == nil {
		return fmt.Errorf("account service not initialized")
	}
	return a.accountSvc.SetActiveAccount(platform, accountID)
}

func (a *App) DeleteAccount(accountID string) error {
	if a.accountSvc == nil {
		return fmt.Errorf("account service not initialized")
	}
	return a.accountSvc.DeleteAccount(accountID)
}

func (a *App) UpdateAccountName(accountID, name string) error {
	if a.accountSvc == nil {
		return fmt.Errorf("account service not initialized")
	}
	return a.accountSvc.UpdateAccountName(accountID, name)
}

func (a *App) GetAccountStats() (map[string]interface{}, error) {
	if a.accountSvc == nil {
		return nil, fmt.Errorf("account service not initialized")
	}
	return a.accountSvc.GetStats()
}

// Auth methods

// Login performs user authentication and saves the token.
func (a *App) Login(username, password, apiBaseURL string) (map[string]interface{}, error) {
	if a.authSvc == nil {
		return nil, fmt.Errorf("auth service not initialized")
	}
	resp, err := a.authSvc.Login(username, password, apiBaseURL)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}
	return map[string]interface{}{
		"success":    resp.Success,
		"token":      resp.Token,
		"expires_at": resp.ExpiresAt,
	}, nil
}

// Login methods

// StartLogin opens a browser for the specified account to login.
func (a *App) StartLogin(platform, accountID string) error {
	if a.providerFact == nil {
		return fmt.Errorf("provider factory not initialized")
	}
	if a.loginCancel != nil {
		a.loginCancel()
		a.loginCancel = nil
	}

	// Always headless=false for login
	prov, err := a.providerFact.GetProvider(platform, false, 0, accountID)
	if err != nil {
		logger.GetLogger().Error("Failed to get provider for login", err)
		return err
	}

	cancel, err := prov.StartLogin()
	if err != nil {
		logger.GetLogger().Error("Failed to start login browser", err)
		return err
	}

	a.loginCancel = cancel
	return nil
}

// StopLogin closes the current login browser.
func (a *App) StopLogin() error {
	if a.loginCancel != nil {
		a.loginCancel()
		a.loginCancel = nil
	}
	return nil
}

// Log methods
func (a *App) GetLogs(limit, offset int, filtersJSON string) (map[string]interface{}, error) {
	if a.logRepo == nil {
		return nil, fmt.Errorf("log repository not initialized")
	}
	var level, source *string
	var taskID *int

	if filtersJSON != "" {
		var filters map[string]interface{}
		if err := json.Unmarshal([]byte(filtersJSON), &filters); err == nil {
			if lvl, ok := filters["level"].(string); ok && lvl != "" {
				level = &lvl
			}
			if src, ok := filters["source"].(string); ok && src != "" {
				source = &src
			}
			if tid, ok := filters["task_id"].(float64); ok {
				tidInt := int(tid)
				taskID = &tidInt
			}
		}
	}

	logs, err := a.logRepo.GetAll(limit, offset, level, source, taskID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"logs":    logs,
		"total":   len(logs),
	}, nil
}

func (a *App) AddLog(level, source, message, detailsJSON, sessionID, correlationID, component, userAction string, performanceMS, taskID *int) error {
	l := logger.GetLogger()
	l.Debug(fmt.Sprintf("AddLog called: level=%s, message=%s, taskID=%v", level, message, taskID))
	if a.logRepo == nil {
		l.Error("App.AddLog called but logRepo is nil. Startup might have failed.", nil)
		return fmt.Errorf("log repository not initialized")
	}

	var details *string
	if detailsJSON != "" {
		details = &detailsJSON
	}

	var sessionIDPtr, correlationIDPtr, componentPtr, userActionPtr *string
	if sessionID != "" {
		sessionIDPtr = &sessionID
	}
	if correlationID != "" {
		correlationIDPtr = &correlationID
	}
	if component != "" {
		componentPtr = &component
	}
	if userAction != "" {
		userActionPtr = &userAction
	}

	return a.logRepo.AddWithContext(level, source, message, details, taskID, sessionIDPtr, correlationIDPtr, componentPtr, userActionPtr, performanceMS)
}

func (a *App) ClearOldLogs(daysToKeep int) (map[string]interface{}, error) {
	count, err := a.logRepo.GetLogsCountOlderThan(daysToKeep)
	if err != nil {
		return nil, err
	}

	err = a.logRepo.ClearOldLogs(daysToKeep)
	if err != nil {
		return nil, err
	}

	// Log the cleanup operation
	logger.GetLogger().Info(fmt.Sprintf("Old logs cleared: deleted %d logs older than %d days", count, daysToKeep))

	return map[string]interface{}{
		"success": true,
		"deleted": count,
	}, nil
}

func (a *App) DeleteAllLogs() (map[string]interface{}, error) {
	count, err := a.logRepo.DeleteAllLogs()
	if err != nil {
		return nil, err
	}

	// Log the deletion operation (this will be the only log left after deletion)
	logger.GetLogger().Warn(fmt.Sprintf("All logs deleted by user: %d logs removed", count))

	return map[string]interface{}{
		"success": true,
		"deleted": count,
	}, nil
}

// GetVersionInfo returns version and build time information
func (a *App) GetVersionInfo() map[string]interface{} {
	buildTime := BuildTime
	formattedTime := buildTime

	if buildTime != "unknown" && buildTime != "" {
		if t, err := time.Parse(time.RFC3339, buildTime); err == nil {
			formattedTime = t.Format("2006-01-02 15:04:05")
		}
	}

	return map[string]interface{}{
		"version":   Version,
		"buildTime": formattedTime,
	}
}

func (a *App) GetSearchRecords(taskID int) (map[string]interface{}, error) {
	if a.taskManager == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}
	records, err := a.taskManager.GetSearchRecords(taskID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "records": records}, nil
}

func (a *App) GetMergedSearchRecords(taskIDsJSON string) (map[string]interface{}, error) {
	if a.taskManager == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}
	var taskIDs []int
	if err := json.Unmarshal([]byte(taskIDsJSON), &taskIDs); err != nil {
		return nil, fmt.Errorf("invalid task IDs format: %w", err)
	}

	records, err := a.taskManager.GetMergedSearchRecords(taskIDs)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "records": records}, nil
}

func (a *App) GetLogFileContent(lines int) (map[string]interface{}, error) {
	l := logger.GetLogger()
	content, err := l.ReadLogFile(lines)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to read log file: %v", err),
		}, nil
	}
	return map[string]interface{}{
		"success": true,
		"content": content,
	}, nil
}

func (a *App) OpenLogsFolder() error {
	l := logger.GetLogger()
	logDir := l.GetLogDir()

	var cmd *exec.Cmd
	switch gouruntime.GOOS {
	case "darwin":
		cmd = exec.Command("open", logDir)
	case "windows":
		cmd = exec.Command("explorer", logDir)
	case "linux":
		cmd = exec.Command("xdg-open", logDir)
	default:
		return fmt.Errorf("unsupported platform: %s", gouruntime.GOOS)
	}
	return cmd.Start()
}

func (a *App) SaveExcelFile(filename string, base64Content string) (map[string]interface{}, error) {
	data, err := b64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 content: %w", err)
	}

	filepath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename:      filename,
		Title:                "保存 Excel 文件",
		Filters:              []runtime.FileFilter{{DisplayName: "Excel Files (*.xlsx)", Pattern: "*.xlsx"}},
		CanCreateDirectories: true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open save dialog: %w", err)
	}

	if filepath == "" {
		return map[string]interface{}{"success": false, "message": "Save cancelled"}, nil
	}

	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"path":    filepath,
	}, nil
}

func (a *App) ExportLogs(timeRange string) (map[string]interface{}, error) {
	l := logger.GetLogger()
	now := time.Now()

	content, err := l.ReadLogFile(0)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	if len(content) == 0 {
		return map[string]interface{}{
			"success": false,
			"message": "Log file is empty",
		}, nil
	}

	filename := fmt.Sprintf("geo_client_logs_%s_%s.txt", timeRange, now.Format("20060102_150405"))
	filepath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultDirectory:     ".",
		DefaultFilename:      filename,
		Title:                "Export Logs",
		Filters:              []runtime.FileFilter{{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"}},
		CanCreateDirectories: true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open save dialog: %w", err)
	}

	if filepath == "" {
		return map[string]interface{}{"success": false, "message": "Export cancelled"}, nil
	}

	err = os.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	lines := len(strings.Split(content, "\n"))
	return map[string]interface{}{
		"success": true,
		"path":    filepath,
		"count":   lines,
	}, nil
}
