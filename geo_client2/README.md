# geo_client2 - Wails 版 GEO 监测客户端

基于 Wails v2 + Go + React + TypeScript 的 GEO 监测平台客户端。

## 架构

- **桌面壳**: Wails v2 (Go 主进程 + 系统 WebView)
- **前端**: React 18 + Vite + TypeScript + Tailwind + Zustand + shadcn/ui
- **后端**: Go (认证、任务、搜索、Provider、数据库、设置、日志)
- **RPA**: go-rod (替代 Playwright，UserDataDir 持久登录，HijackRequests 拦截 SSE)
- **数据库**: modernc.org/sqlite (纯 Go SQLite)
- **前后端通信**: Wails Bind + Runtime Events

## 前置要求

- Go 1.21+
- Node.js 18+ / pnpm
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

## 项目结构

```
geo_client2/
├── main.go                 # Wails 入口
├── backend/                # Go 业务逻辑
│   ├── app.go              # 绑定给前端的结构体
│   ├── auth/               # 认证服务
│   ├── task/               # 任务管理
│   ├── search/             # 搜索服务
│   ├── provider/           # Provider (RPA)
│   ├── database/           # 数据库层
│   ├── settings/           # 设置服务
│   └── logger/             # 日志服务
└── frontend/               # React 前端
    ├── src/
    │   ├── pages/          # 页面组件
    │   ├── components/      # 通用组件
    │   ├── stores/         # Zustand 状态
    │   ├── utils/           # 工具函数
    │   └── wailsjs/        # Wails 绑定桥接
    └── ...
```

## 开发

### 安装依赖

```bash
# 前端依赖
cd frontend && pnpm install

# Go 依赖（需要 Go 环境）
cd .. && go mod tidy
```

### 运行开发模式

```bash
# 启动 Wails 开发服务器（同时启动 Vite + Wails）
wails dev
```

开发服务器会：
- 启动 Vite 在 `http://localhost:5173`
- 启动 Wails 应用窗口
- 支持热重载

### 仅前端开发（无 Wails）

```bash
cd frontend && pnpm run dev
```

注意：此时 Wails 后端不可用，Go 绑定调用会失败（但 UI 可正常开发）。

## 构建

```bash
# 构建前端
cd frontend && pnpm run build

# 构建 Wails 应用
cd .. && wails build
```

构建产物：
- macOS: `build/bin/geo_client2.app`
- Windows: `build/bin/geo_client2.exe`

## 配置

应用配置目录：`~/.geo_client2/`
- 数据库: `~/.geo_client2/cache.db`
- 日志: `~/.geo_client2/logs/task_YYYY-MM-DD.log`
- 浏览器数据: `~/.geo_client2/browser_data/{doubao|deepseek}`

## 功能

- ✅ 用户认证（登录、Token 管理）
- ✅ 任务管理（创建、执行、取消、重试）
- ✅ 大模型搜索（DeepSeek、豆包）
- ✅ 登录授权管理（平台登录状态检查）
- ✅ 设置管理（无头模式等）
- ⏳ Provider RPA 实现（go-rod，待完善）

## 与 geo_client (Electron) 的差异

| 特性 | geo_client (Electron) | geo_client2 (Wails) |
|------|----------------------|---------------------|
| 桌面壳 | Electron + Chromium | Wails + 系统 WebView |
| 后端 | Node.js + TypeScript | Go |
| RPA | Playwright | go-rod |
| IPC | Electron IPC | Wails Bind + Events |
| 登录窗口 | BrowserView 内嵌 | rod headed 浏览器窗口 |
| 包体积 | ~150MB+ | ~10-20MB |

## 待完善

- [ ] Provider 完整实现（go-rod SSE 拦截、DOM 解析）
- [ ] 任务详情完整展示（汇总表格、Domain 统计、详细日志）
- [ ] 日志查看页面
- [ ] shadcn/ui 组件补充（Button, Input, Card, Table, Tabs, Dialog 等）
- [ ] 暗色模式切换
- [ ] 侧边栏折叠

## 开发状态

- ✅ Phase 1: Wails 脚手架 + 前端壳
- ✅ Phase 2: 数据层 + Go 服务
- ✅ Phase 3: 前端功能 + UI
- ⏳ Phase 4: 联调、打包、文档

## 许可证

与 geo_client 相同
