# GEO Client2 Frontend - Complete Inventory

**Project**: geo_client2/frontend  
**Framework**: React 18 + Vite + TypeScript  
**State Management**: Zustand (2 stores)  
**Routing**: React Router v6  
**Styling**: Tailwind CSS + shadcn/ui  
**Communication**: Wails Bridge (Go ↔ React) + Runtime Events  

---

## 📍 Routes & Pages

### Application Structure
- **Entry Point**: `App.tsx` (main routing hub)
- **Layout**: `Layout.tsx` (wraps all pages with Sidebar + Header)
- **Base Path**: `/` (all routes are nested under Layout)

| Route | Component | Purpose |
|-------|-----------|---------|
| `/` | Dashboard | Home page with task stats (queue, running, completed, failed) and platform account counts |
| `/search` | Search | Create & execute search tasks across AI platforms (DeepSeek, 豆包, etc.) |
| `/auth` | Auth | Manage platform accounts & login status (11+ platforms: AI + social media) |
| `/publish` | Publish | Create articles and publish to social platforms with optional AI assist |
| `/publish-tasks` | PublishTasks | View list of all publish tasks (long-running multi-platform tasks) |
| `/publish-tasks/:taskId` | PublishTaskDetail | View detailed status of a specific publish task with per-platform progress |
| `/tasks` | Tasks | Unified task list with tabs for both search & publish tasks |
| `/logs` | Logs | Stream and view application logs with export/refresh options |
| `/settings` | Settings | Configure global settings (headless mode, AI assist settings) |

---

## 📄 Page Components (Detailed)

### 1. **Dashboard** (`pages/Dashboard.tsx`)
**Purpose**: Real-time overview of system status and resource utilization

**Features**:
- Task statistics: queue, running, completed, failed counts
- Account statistics: total accounts and breakdown by platform
- Auto-refresh every 5 seconds via `useEffect` interval
- Uses `wailsAPI.task.getStats()` and `wailsAPI.account.getStats()`

**State**:
- `stats`: Task statistics object
- `accountStats`: Account count data with platform breakdown

**Imports**:
- Icons: `LayoutDashboard`, `ListTodo`, `CheckCircle2`, `XCircle`, `Users` (lucide-react)
- API bridge via `wailsAPI`

---

### 2. **Search** (`pages/Search.tsx`)
**Purpose**: Create and execute AI search monitoring tasks

**Features**:
- Keyword input (multi-line support)
- Platform selection (deepseek, doubao, yiyan, yuanbao, etc.)
- Query count input (number of times to execute each keyword)
- Task creation via `wailsAPI.task.createLocalSearchTask()`
- Merge multiple task results for comparison
- MergedTaskViewer component for displaying combined results

**State**:
- `keywords`: Text input for search keywords
- `selectedPlatforms`: Array of selected platform IDs
- `queryCount`: Number of query iterations
- `isSearching`: Loading state during task creation
- `mergeTaskIds`: Comma-separated task IDs to merge
- `currentMergedIds`: Array of parsed task IDs

**Imports**:
- Icons: `Play`, `Loader2`, `Combine`
- Components: `MergedTaskViewer`
- Toast notifications via `sonner`

---

### 3. **Auth** (`pages/Auth.tsx`)
**Purpose**: Manage platform accounts and authentication

**Features**:
- 11 platform support: DeepSeek, 豆包, 文心一言, 腾讯元宝, 小红书, 知乎, 搜狐号, CSDN, 企鹅号, 百家号, 头条号
- Account CRUD operations (create, update, delete)
- Platform login status checking and management
- Active account selection per platform
- Account renaming
- Login UI launcher (opens headless browser for user login)
- Categorized platforms: AI + Social Media

**State**:
- `newAccountName`: Input for adding new accounts
- `addingPlatform`: Currently selected platform for new account
- All state managed via `useAccountStore`

**Account Store Integration**:
- `accountsByPlatform`: All accounts grouped by platform
- `activeAccounts`: Currently active account per platform
- Actions: loadAccounts, createAccount, setActiveAccount, deleteAccount, etc.

---

### 4. **Publish** (`pages/Publish.tsx`)
**Purpose**: Create and publish articles to social platforms with AI assistance

**Features**:
- Rich text editor for article content (via `RichContentEditor` component)
- Title & cover image input
- Platform selection (6 social platforms: 知乎, 搜狐号, CSDN, 企鹅号, 百家号, 头条号)
- Per-platform account selection
- AI-assisted title/content generation (optional)
- Real-time publish status tracking per platform
- Manual intervention prompts for platforms requiring interaction
- Event listener for publish progress updates (`publish:platformProgress`)

**State**:
- `selectedPlatforms`: Array of target platforms
- `selectedAccountIds`: Map of platform → account ID
- `title`, `content`, `coverImage`: Article data
- `publishStates`: Per-platform publishing status
- `isPublishing`: Overall publishing state

**Publish Status States**:
- `idle`: Not started
- `running`: Publishing in progress
- `waiting_manual`: Awaiting user intervention
- `completed`: Successfully published
- `failed`: Publishing failed

---

### 5. **PublishTasks** (`pages/PublishTasks.tsx`)
**Purpose**: List all publish tasks with status overview

**Features**:
- Displays all long-running publish tasks
- Status badges for each task (running, pending, paused, completed, failed, cancelled)
- Task info: title, article preview, platform list, progress
- Navigation to detail page
- Delete/retry operations
- Auto-refresh every 3 seconds

**Status Display**:
- `running`: Blue badge
- `pending`: Slate badge
- `paused`: Yellow badge
- `completed`: Green badge
- `failed`: Red badge
- `cancelled`: Zinc badge

---

### 6. **PublishTaskDetail** (`pages/PublishTaskDetail.tsx`)
**Purpose**: Detailed view of a single publish task execution

**Features**:
- Full article display (title, cover, content via RichContentEditor)
- Per-platform execution status with detailed information
- Platform states: status, started_at, completed_at, article_url, error, retry count
- Progress tracking (current_index/total_platforms)
- Retry and delete operations
- Account information per platform
- Formatted timestamps

**Platform State Tracking**:
```typescript
interface PlatformState {
  platform: string;
  status: LongTaskStatus;
  started_at?: string;
  completed_at?: string;
  article_url?: string;
  error?: string;
  retries?: number;
  max_retries?: number;
}
```

---

### 7. **Tasks** (`pages/Tasks.tsx`)
**Purpose**: Unified task management for both search and publish tasks

**Features**:
- Tabbed interface: Search Tasks | Publish Tasks
- Search task list with expandable details
- Publish task list with inline viewer
- Local task creation UI (embedded `LocalTaskCreator`)
- Local task details viewer (embedded `LocalTaskDetail`)
- Status filtering
- Task deletion with confirmation
- Task execution/retry
- Auto-refresh every 2 seconds

**Search Task Properties**:
```typescript
interface Task {
  id: number;
  task_id?: number;
  name?: string;
  keywords: string;
  platforms: string;
  query_count: number;
  status: string;
  task_type: string;
  source: string;
  created_by: string | null;
  created_at: string;
}
```

---

### 8. **Logs** (`pages/Logs.tsx`)
**Purpose**: Real-time application log viewing and management

**Features**:
- Real-time log streaming
- Auto-scroll to bottom when new logs arrive
- Auto-refresh toggle (2-second interval when enabled)
- Manual refresh button
- Log export to file (multiple formats: JSON, CSV, TXT)
- Log clearing/deletion with confirmation
- Folder open (native file browser)
- Keyboard shortcuts support

**Controls**:
- Play/Pause auto-refresh
- Manual refresh button
- Export menu (format selection)
- Delete menu (confirmation)

---

### 9. **Settings** (`pages/Settings.tsx`)
**Purpose**: Global application configuration

**Features**:
- **Browser Mode**: Toggle headless mode (default: true)
- **AI Publish Assist**: Enable/disable AI-assisted article generation
  - AI Base URL configuration
  - AI API Key configuration
  - Save/test AI config

**Settings Persistence**:
- Uses `wailsAPI.settings.get/set()` for backend storage
- Configuration keys:
  - `browser_headless`: Browser automation mode
  - `ai_publish_assist`: AI assistance toggle
  - `ai_publish_base_url`: AI service endpoint
  - `ai_publish_api_key`: AI service authentication

---

## 🏪 Global State Stores (Zustand)

### 1. **authStore** (`stores/authStore.ts`)
**Purpose**: User authentication state and token management

**State**:
```typescript
interface AuthState {
  token: string | null;
  expiresAt: number | null;
  username: string | null;
  userId: string | null;
  isAdmin: boolean;
}
```

**Actions**:
- `setToken()`: Set user auth token with expiration
- `clearToken()`: Clear all auth data
- `isTokenValid()`: Check if token is not expired
- `checkAndRefreshToken()`: Validate token (currently non-functional)

**Persistence**: Uses `zustand/middleware` persist with key `geo_client2-auth`

---

### 2. **accountStore** (`stores/accountStore.ts`)
**Purpose**: Multi-platform account management

**State**:
```typescript
interface AccountState {
  activeAccounts: Record<PlatformName, Account | null>;
  accountsByPlatform: Record<PlatformName, Account[]>;
  loading: Record<string, boolean>;
}
```

**Supported Platforms** (11 total):
- AI: `deepseek`, `doubao`, `yiyan`, `yuanbao`
- Social Media: `xiaohongshu`, `zhihu`, `sohu`, `csdn`, `qie`, `baijiahao`, `toutiao`

**Key Actions**:
- `loadAccounts(platform)`: Fetch all accounts for a platform
- `loadActiveAccount(platform)`: Fetch currently active account
- `createAccount(platform, name)`: Create new account
- `setActiveAccount(platform, accountID)`: Set active account
- `deleteAccount(accountID)`: Delete an account
- `updateAccountName(accountID, name)`: Rename account
- `startLogin(platform, accountID)`: Launch login browser
- `stopLogin()`: Stop active login process
- `refreshAccounts(platform)`: Reload accounts after changes

**Account Data Structure**:
```typescript
interface Account {
  id: string;
  platform: string;
  name: string;
  is_active: boolean;
  login_status?: string;
  created_at: string;
  updated_at: string;
}
```

---

## 🔌 Major Components (Non-Page)

| Component | File | Purpose |
|-----------|------|---------|
| Layout | `components/Layout.tsx` | Page wrapper with Sidebar + Header + Outlet |
| Header | `components/Header.tsx` | Top navigation bar |
| Sidebar | `components/Sidebar.tsx` | Left navigation with route links |
| ErrorBoundary | `components/ErrorBoundary.tsx` | Global error boundary |
| LocalTaskCreator | `components/LocalTaskCreator.tsx` | Inline form for creating tasks |
| LocalTaskDetail | `components/LocalTaskDetail.tsx` | Display task details (expandable) |
| MergedTaskViewer | `components/MergedTaskViewer.tsx` | Compare/merge multiple task results |
| PublishTaskDetailModal | `components/PublishTaskDetailModal.tsx` | Modal for viewing publish task details |
| RichContentEditor | `components/RichContentEditor.tsx` | Rich text editor for article content |
| Button (UI) | `components/ui/button.tsx` | Custom button component (shadcn/ui) |
| Switch (UI) | `components/ui/switch.tsx` | Toggle switch component (shadcn/ui) |

---

## 🌊 Event Stream (Wails Events)

The application listens for backend events via `EventsOn`:

| Event | Payload | Handler Location |
|-------|---------|------------------|
| `test` | `{ message: string }` | App.tsx (toast notification) |
| `search:taskUpdated` | Task update data | App.tsx (logging), Search page (UI update) |
| `login-status-changed` | `{ platform: string; isLoggedIn: boolean }` | App.tsx (toast warning) |
| `task-login-required` | `{ platformName: string; keyword: string }` | App.tsx (error toast) |
| `publish:platformProgress` | Platform publish status | Publish page (status update) |

---

## 🔗 Wails API Bridge

All backend communication flows through `utils/wails-api.ts`:

### Task API
```typescript
wailsAPI.task.createLocalSearchTask()
wailsAPI.task.getStats()
wailsAPI.task.listRecords()
wailsAPI.task.getRecord()
wailsAPI.task.execute()
wailsAPI.task.cancel()
wailsAPI.task.retry()
wailsAPI.task.delete()
```

### Account API
```typescript
wailsAPI.account.list(platform)
wailsAPI.account.getActive(platform)
wailsAPI.account.create(platform, name)
wailsAPI.account.setActive(platform, accountID)
wailsAPI.account.delete(accountID)
wailsAPI.account.update(accountID, name)
wailsAPI.account.startLogin(platform, accountID)
wailsAPI.account.stopLogin()
wailsAPI.account.getStats()
```

### LongTask API (Publishing)
```typescript
wailsAPI.longTask.createRecord()
wailsAPI.longTask.listRecords()
wailsAPI.longTask.getRecord(taskId)
wailsAPI.longTask.executeTask(taskId)
wailsAPI.longTask.pauseTask(taskId)
wailsAPI.longTask.resumeTask(taskId)
wailsAPI.longTask.cancelTask(taskId)
wailsAPI.longTask.deleteRecord(taskId)
```

### Settings API
```typescript
wailsAPI.settings.get(key)
wailsAPI.settings.set(key, value)
```

### Logs API
```typescript
wailsAPI.logs.getCurrentLog()
wailsAPI.logs.clearLogs()
wailsAPI.logs.exportLogs(format)
```

---

## 🎯 Key Application Flows

### 1. Search Task Creation → Execution → Results
```
Search Page
  ↓ User inputs keywords, selects platforms
  ↓ wailsAPI.task.createLocalSearchTask()
  ↓ Backend creates task record
  ↓ Event: search:taskUpdated
  ↓ Tasks Page displays results
  ↓ User can merge/compare multiple task results
```

### 2. Account Authentication
```
Auth Page
  ↓ User clicks "Login" for platform
  ↓ wailsAPI.account.startLogin()
  ↓ Backend launches headless browser
  ↓ User completes login in browser
  ↓ Event: login-status-changed
  ↓ Auth Page updates account status
```

### 3. Article Publishing
```
Publish Page
  ↓ User creates article (title, content, cover)
  ↓ Selects platforms & accounts
  ↓ Clicks "Publish" (or AI-assisted generation first)
  ↓ wailsAPI.longTask.createRecord()
  ↓ Backend starts publishing to each platform
  ↓ Event: publish:platformProgress (per-platform updates)
  ↓ PublishTasks → PublishTaskDetail for status tracking
```

### 4. Real-Time Monitoring
```
Dashboard
  ↓ Initial load: wailsAPI.task.getStats(), wailsAPI.account.getStats()
  ↓ Auto-refresh every 5 seconds
  ↓ Displays current queue/running/completed/failed task counts
  ↓ Shows total accounts + breakdown by platform
```

---

## 📊 Data Structures

### Task
```typescript
{
  id: number;
  task_id?: number;
  name?: string;
  keywords: string;           // Comma/newline-separated
  platforms: string;          // Comma-separated platform IDs
  query_count: number;        // Iteration count
  status: string;             // 'pending', 'running', 'completed', 'failed'
  task_type: string;          // 'search', 'publish'
  source: string;             // Task origin
  created_by: string | null;
  created_at: string;         // ISO timestamp
}
```

### Account
```typescript
{
  id: string;                 // UUID or account ID
  platform: string;           // Platform name
  name: string;               // User-defined account name
  is_active: boolean;         // Is currently active
  login_status?: string;      // 'logged_in', 'logged_out', etc.
  created_at: string;         // ISO timestamp
  updated_at: string;         // ISO timestamp
}
```

### PublishLongTask
```typescript
{
  task_id: string;
  status: 'pending' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled';
  article?: {
    title: string;
    content: string;
    cover_image?: string;
  };
  platforms?: string[];
  account_ids?: Record<string, string>;
  platform_states?: Record<string, PlatformState>;
  current_index?: number;
  total_platforms?: number;
  created_at?: string;
  updated_at?: string;
}
```

---

## 🎨 UI Component Library

- **Framework**: Tailwind CSS
- **Components**: shadcn/ui (button, switch)
- **Icons**: lucide-react
- **Notifications**: sonner (toast)
- **Editor**: RichContentEditor (custom component for markdown/rich text)

---

## 🔐 Authentication Flow

- **Entry Point**: Login page (if no token)
- **Token Storage**: authStore (persisted to localStorage)
- **Token Validation**: `isTokenValid()` checks expiration
- **Automatic Logout**: Toast warning if token expired during session
- **Protected Routes**: Layout component checks auth state

---

## 📝 Logging & Debugging

**Frontend Logger** (`utils/logger.ts`):
- `logger.info()`: Information logs
- `logger.debug()`: Debug logs
- `logger.warn()`: Warning logs
- `logger.error()`: Error logs
- `logger.logNavigation()`: Route change tracking
- Structured logging with contextual data

**Backend Integration**:
- Logs streamed via `wailsAPI.logs.getCurrentLog()`
- Auto-refresh in Logs page every 2 seconds
- Export to multiple formats (JSON, CSV, TXT)

---

## 🎯 Summary for UI/UX Walkthrough

### Screens to Review:
1. **Dashboard** - KPI overview (task stats, account counts)
2. **Search** - Keyword input, platform selection, task creation
3. **Auth** - Multi-platform account management, login flow
4. **Publish** - Rich article editor, platform publishing, status tracking
5. **PublishTasks** - List of all publishing tasks
6. **PublishTaskDetail** - Per-platform execution details
7. **Tasks** - Unified task management (search + publish)
8. **Logs** - Real-time log viewer with export
9. **Settings** - Global configuration (headless mode, AI settings)

### Key UX Patterns:
- **Tab Navigation**: Search/Publish task tabs in Tasks page
- **Expandable Details**: Local task expansion in task lists
- **Status Badges**: Color-coded task status indicators
- **Real-Time Updates**: Auto-refresh intervals + event listeners
- **Modal Interactions**: Login browser, publish confirmations
- **Toast Notifications**: User feedback for actions
- **Account Switching**: Per-platform active account selection
- **Rich Editing**: Markdown support for article content

