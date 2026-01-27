# Logging System Documentation

This document describes the comprehensive logging system implemented in GEO Client2.

## Overview

The logging system provides structured, queryable logging across both frontend and backend with support for:
- Multiple log levels (DEBUG, INFO, WARN, ERROR)
- Session tracking for user journey analysis
- Correlation IDs for tracking related operations
- Performance metrics
- User action tracking
- Database storage for easy querying
- LLM-friendly structured format

## Architecture

```
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│  Frontend   │────────▶│  Wails API   │────────▶│  Backend     │
│  Logger     │         │  AddLog()    │         │  Logger      │
└─────────────┘         └──────────────┘         └──────────────┘
                                                         │
                                                         ▼
                                                  ┌──────────────┐
                                                  │  SQLite DB   │
                                                  │  logs table  │
                                                  └──────────────┘
```

## Backend Usage

### Basic Logging

```go
import "geo_client2/backend/logger"

// Simple logging
logger.GetLogger().Info("Operation completed")
logger.GetLogger().Error("Operation failed", err)
logger.GetLogger().Warn("Warning message")
logger.GetLogger().Debug("Debug information")
```

### Contextual Logging

```go
import (
    "context"
    "geo_client2/backend/logger"
)

ctx := context.Background()
taskID := 42

// Log with full context
logger.GetLogger().InfoWithContext(ctx, "Task started", map[string]interface{}{
    "keywords": []string{"test"},
    "platforms": []string{"doubao"},
}, &taskID)

// Log errors with context
logger.GetLogger().ErrorWithContext(ctx, "Search failed", map[string]interface{}{
    "platform": "doubao",
    "keyword": "test",
}, err, &taskID)
```

### Performance Tracking

```go
// Start a timer
timer := logger.GetLogger().StartTimer(ctx, "SearchOperation", &taskID)

// ... do work ...

// End timer (automatically logs duration)
timer.End(map[string]interface{}{
    "results_count": 10,
    "success": true,
})
```

### Session and Correlation IDs

```go
import "geo_client2/backend/logger"

// Add session ID to context
ctx = logger.WithSessionID(ctx, "session-123")

// Add correlation ID to context
ctx = logger.WithCorrelationID(ctx, "corr-abc")

// These IDs will automatically be included in all logs
logger.GetLogger().InfoWithContext(ctx, "Message", nil, nil)
```

## Frontend Usage

### Basic Logging

```typescript
import { logger } from '@/utils/logger';

// Simple logging
logger.info('User clicked button');
logger.error('Failed to load data', {
  component: 'Dashboard',
  details: { error: 'Network timeout' }
});
logger.warn('Deprecated feature used');
logger.debug('State updated');
```

### User Action Tracking

```typescript
logger.logUserAction('create_task', 'TaskForm', {
  keywords: ['test'],
  platforms: ['doubao']
});
```

### API Call Tracking

```typescript
import { generateCorrelationId } from '@/utils/logger';

const correlationId = generateCorrelationId();

// Before API call
logger.logApiCall('CreateTask', { keywords, platforms }, correlationId);

try {
  const result = await wailsAPI.task.createLocalSearchTask(config);
  logger.logApiResponse('CreateTask', true, duration, correlationId);
} catch (error) {
  logger.logApiError('CreateTask', error, correlationId);
}
```

### Performance Tracking

```typescript
const timer = logger.startTimer('LoadDashboard', 'Dashboard');

// ... do work ...

timer.end({ items_loaded: 42 });
```

### Navigation Tracking

```typescript
// Automatically tracked by NavigationLogger in App.tsx
logger.logNavigation('/dashboard', '/tasks');
```

## Log Viewer UI

Access the log viewer at `/logs` in the application.

### Features

- **Filtering**: Filter by level, source, task ID
- **Search**: Full-text search in messages and details
- **Pagination**: Browse logs in pages of 50
- **Expandable Details**: Click on log entries to see full JSON details
- **Session Tracking**: View session ID and correlation ID for debugging
- **Real-time**: Refresh button to load latest logs

### Filters

- **Level**: ERROR, WARN, INFO, DEBUG
- **Source**: frontend, backend, or specific files
- **Task ID**: Filter logs for a specific task
- **Search**: Free-text search across all fields

## Database Schema

```sql
CREATE TABLE logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    level TEXT NOT NULL,                -- ERROR, WARN, INFO, DEBUG
    source TEXT NOT NULL,               -- File:line or "frontend"
    message TEXT NOT NULL,              -- Human-readable message
    details TEXT,                       -- JSON object with metadata
    task_id INTEGER,                    -- Associated task ID
    session_id TEXT,                    -- User session identifier
    correlation_id TEXT,                -- Links related operations
    component TEXT,                     -- Component/module name
    user_action TEXT,                   -- User action that triggered this
    performance_ms INTEGER,             -- Operation duration in ms
    timestamp TEXT DEFAULT (datetime('now', 'localtime'))
);
```

## Log Entry Structure

### Backend Log Entry

```json
{
  "id": 123,
  "level": "ERROR",
  "source": "executor.go:75",
  "message": "Failed to get active account",
  "details": {
    "platform": "doubao",
    "error": "no active account found",
    "error_type": "*errors.errorString"
  },
  "task_id": 42,
  "session_id": "sess-abc-123",
  "correlation_id": "corr-xyz-789",
  "component": null,
  "user_action": null,
  "performance_ms": null,
  "timestamp": "2026-01-23 10:30:45"
}
```

### Frontend Log Entry

```json
{
  "id": 124,
  "level": "INFO",
  "source": "frontend",
  "message": "User action: create_task",
  "details": {
    "keywords": ["test"],
    "platforms": ["doubao"]
  },
  "task_id": null,
  "session_id": "sess-abc-123",
  "correlation_id": "corr-xyz-789",
  "component": "TaskForm",
  "user_action": "create_task",
  "performance_ms": null,
  "timestamp": "2026-01-23 10:30:46"
}
```

## Best Practices

### For Developers

1. **Use appropriate log levels**
   - `ERROR`: Failures that prevent operations
   - `WARN`: Issues that don't block operations
   - `INFO`: Important state changes, user actions
   - `DEBUG`: Detailed debugging information

2. **Include context in details**
   ```go
   logger.GetLogger().ErrorWithContext(ctx, "Database query failed", map[string]interface{}{
       "query": "SELECT * FROM users",
       "duration_ms": 5000,
       "row_count": 0,
   }, err, nil)
   ```

3. **Use correlation IDs for request tracking**
   - Generate at the start of an operation
   - Pass through all related function calls
   - Links frontend request → backend processing → result

4. **Track performance for slow operations**
   ```go
   timer := logger.GetLogger().StartTimer(ctx, "DatabaseQuery", nil)
   defer timer.End(map[string]interface{}{"rows": count})
   ```

5. **Don't log sensitive data**
   - Avoid passwords, tokens, personal information
   - Sanitize user input before logging

### For LLMs

The logging system is optimized for LLM analysis:

1. **Structured JSON format**: Easy to parse and query
2. **Session tracking**: Understand user journey leading to issues
3. **Correlation IDs**: Trace operations across frontend/backend
4. **Rich context**: Error types, stack traces, operation details
5. **Performance metrics**: Identify slow operations automatically

### Query Examples

```go
// Get all errors for a specific task
logs, _ := logRepo.GetAll(100, 0, &"ERROR", nil, &taskID)

// Get all logs for a user session
logs, _ := logRepo.GetBySession("sess-abc-123", 100, 0)

// Get all logs related to an operation
logs, _ := logRepo.GetByCorrelation("corr-xyz-789")

// Get error statistics
stats, _ := logRepo.GetErrorStats(nil, nil)
```

## Maintenance

### Log Rotation

```go
// Clear logs older than 30 days
err := logRepo.ClearOldLogs(30)
```

Or via the UI (future feature):
```
Settings → Logs → Auto-delete logs older than: [30] days
```

### Performance Considerations

- Indexes on `level`, `source`, `session_id`, `correlation_id`, `task_id`
- Database writes are async (don't block operations)
- Consider log rotation for long-running installations
- Monitor database size in `~/.geo_client2/cache.db`

## Troubleshooting

### Logs not appearing in UI

1. Check that database migration ran successfully
2. Verify logger is initialized with DB: `logger.SetDB(db)`
3. Check browser console for frontend errors
4. Verify Wails backend is running

### Performance issues

1. Check database size: `du -h ~/.geo_client2/cache.db`
2. Run log cleanup: Delete old logs
3. Check for excessive DEBUG logging in production

### Missing context in logs

1. Ensure context is passed through function calls
2. Use `WithSessionID` and `WithCorrelationID` helpers
3. Check that `LogEntry` includes all required fields

## Future Enhancements

- [ ] Export logs to JSON/CSV
- [ ] Real-time log streaming (WebSocket)
- [ ] Advanced analytics dashboard
- [ ] Log aggregation and pattern detection
- [ ] Alert system for error thresholds
- [ ] Integration with external monitoring tools

## Related Files

- Backend:
  - `backend/logger/logger.go` - Core logger implementation
  - `backend/database/repositories/log.go` - Database operations
  - `backend/database/schema.go` - Table schema
  - `backend/app.go` - Wails bindings

- Frontend:
  - `frontend/src/utils/logger.ts` - Frontend logger
  - `frontend/src/pages/Logs.tsx` - Log viewer UI
  - `frontend/src/components/ErrorBoundary.tsx` - Error handling
  - `frontend/src/App.tsx` - Integration points
