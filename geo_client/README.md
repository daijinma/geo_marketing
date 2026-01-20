# Geo Client

客户端抓取程序 - 基于 React + Electron 的桌面应用

## 技术栈

- **前端**: React + TypeScript + Vite
- **后端**: Electron (Node.js + TypeScript)
- **UI组件库**: shadcn/ui (Tailwind CSS)
- **状态管理**: Zustand
- **路由**: React Router
- **浏览器自动化**: Playwright
- **数据库**: better-sqlite3
- **浏览器显示**: Electron BrowserView（浏览器页面在应用内部显示）

## 前置要求

- **Node.js** 18+ （推荐 20 LTS）
- **npm** 或 **pnpm**（推荐）

## 安装流程

### 方式 1：一键安装（推荐）

```bash
# 进入项目目录
cd /path/to/geo_marketing/geo_client

# 运行安装脚本
./scripts/setup.sh
```

**国内用户建议使用镜像源**：
```bash
USE_CHINA_MIRROR=1 ./scripts/setup.sh
```

安装脚本会自动：
- ✅ 检查 Node.js 版本
- ✅ 安装所有依赖（自动选择 pnpm 或 npm）
- ✅ 下载 Electron 二进制
- ✅ 安装 Playwright 浏览器
- ✅ 编译 Electron 主进程代码
- ✅ 创建环境变量文件

### 方式 2：手动安装

#### 1. 克隆仓库并进入目录

```bash
cd /path/to/geo_marketing/geo_client
```

#### 2. 安装依赖

```bash
# 使用 pnpm（推荐）
pnpm install

# 或使用 npm
npm install
```

这一步会：
- 安装所有 Node.js 依赖
- 自动下载 Electron 二进制文件
- 自动安装 native 依赖（better-sqlite3）

**注意**：如果在国内网络环境下 Electron 下载缓慢，可以配置镜像：

```bash
# 设置 Electron 镜像
export ELECTRON_MIRROR=https://npmmirror.com/mirrors/electron/

# 然后重新安装
pnpm install
```

#### 3. 安装 Playwright 浏览器

```bash
# 只安装 Chromium（推荐，体积小）
npx playwright install chromium

# 或安装所有浏览器
npx playwright install
```

#### 4. 编译 Electron 主进程代码

在首次运行或修改 Electron 代码后，需要编译 TypeScript：

```bash
pnpm exec tsc -p electron
```

这会将 `electron/` 目录下的 TypeScript 代码编译到 `dist-electron/` 目录。

#### 5. 启动开发服务器

```bash
pnpm run electron:dev
```

这会同时启动：
- Vite 开发服务器（前端热更新，端口 1420）
- Electron 应用（自动加载前端页面）

## 命令速查表

| 操作 | 命令 |
|------|------|
| **安装依赖** | `pnpm install` |
| **编译主进程** | `pnpm exec tsc -p electron` |
| **开发模式** | `pnpm run electron:dev` |
| **快速开发** | `pnpm run electron:dev:fast` |
| **生产构建** | `pnpm run electron:build` |
| **类型检查** | `pnpm run type-check` |
| **仅前端开发** | `pnpm run dev` |

## 开发工作流

### 日常开发

1. **首次启动**：
   ```bash
   pnpm install                    # 安装依赖
   npx playwright install chromium # 安装浏览器
   pnpm exec tsc -p electron      # 编译主进程
   pnpm run electron:dev          # 启动开发服务器
   ```

2. **后续开发**：
   - 修改前端代码（`src/`）：无需重启，Vite 会自动热更新
   - 修改 Electron 代码（`electron/`）：需要重新编译并重启
     ```bash
     # 终端 1：重新编译
     pnpm exec tsc -p electron
     
     # 终端 2：重启 Electron（Ctrl+C 停止，然后重新运行）
     pnpm run electron:dev
     ```

### 自动编译开发模式

如果想让 Electron 代码也自动编译，可以使用 watch 模式：

```bash
# 终端 1：编译 Electron 代码（watch 模式）
pnpm exec tsc -p electron --watch

# 终端 2：启动 Electron
pnpm run electron:dev
```

## 项目结构

```
geo_client/
├── src/                    # React 前端代码
│   ├── pages/             # 页面组件
│   │   ├── Dashboard.tsx  # 仪表盘
│   │   ├── Login.tsx      # 登录页面
│   │   ├── Search.tsx     # 搜索页面
│   │   ├── Tasks.tsx      # 任务管理
│   │   └── Logs.tsx       # 日志查看
│   ├── components/        # 通用组件
│   │   ├── Layout.tsx     # 布局组件
│   │   ├── Sidebar.tsx    # 侧边栏
│   │   ├── LoginList.tsx  # 登录列表
│   │   └── TaskQueue.tsx  # 任务队列
│   ├── stores/            # Zustand 状态管理
│   │   ├── authStore.ts   # 认证状态
│   │   ├── loginStatusStore.ts  # 登录状态
│   │   └── taskStore.ts   # 任务状态
│   ├── utils/             # 工具函数
│   │   └── api.ts         # API 封装
│   └── types/             # TypeScript 类型定义
│       ├── auth.ts        # 认证类型
│       ├── provider.ts    # Provider 类型
│       ├── task.ts        # 任务类型
│       └── electron.d.ts  # Electron API 类型
├── electron/               # Electron 后端代码（TypeScript）
│   ├── main.ts            # 主进程入口
│   ├── preload.ts         # 预加载脚本
│   ├── tsconfig.json      # TypeScript 配置
│   ├── database/          # 数据库模块
│   │   ├── index.ts       # SQLite 操作
│   │   └── types.d.ts     # better-sqlite3 类型定义
│   ├── services/          # 服务模块
│   │   ├── auth.ts        # 认证服务（IPC 处理）
│   │   ├── browser-pool.ts # 浏览器池管理
│   │   ├── browser-view-manager.ts # BrowserView 管理
│   │   └── task-queue.ts  # 任务队列
│   └── providers/         # Provider 实现
│       ├── base.ts        # 基础 Provider 类
│       └── doubao.ts      # 豆包 Provider
├── dist-electron/          # 编译后的 Electron 代码（自动生成）
│   ├── main.js            # 编译后的主进程
│   ├── preload.js         # 编译后的预加载脚本
│   └── package.json       # 声明为 CommonJS
├── dist/                   # 前端构建输出（生产构建时生成）
├── scripts/               # 便捷脚本
│   └── electron-dev.sh    # 开发启动脚本
├── package.json           # Node.js 依赖和脚本
├── electron-builder.yml   # Electron 打包配置
└── vite.config.ts         # Vite 配置
```

## 环境变量

创建 `.env.development` 文件配置开发环境：

```env
VITE_API_BASE_URL=http://127.0.0.1:8000
VITE_APP_ENV=development
```

创建 `.env.production` 文件配置生产环境：

```env
VITE_API_BASE_URL=https://api.example.com
VITE_APP_ENV=production
```

## 数据存储

应用数据存储在用户目录下：

- **macOS**: `~/.geo_client/cache.db`
- **Windows**: `%USERPROFILE%\.geo_client\cache.db`
- **Linux**: `~/.geo_client/cache.db`

数据库包含以下表：
- `auth` - 认证 token
- `tasks` - 本地任务数据
- `login_status` - 平台登录状态
- `settings` - 用户设置
- `logs` - 日志记录

## 浏览器自动化说明

应用使用 Playwright 进行浏览器自动化：

- **浏览器引擎**: Chromium（使用系统安装的 Chrome）
- **显示方式**: 浏览器页面在 Electron 窗口内部显示（BrowserView）
- **并发控制**: 一次只运行一个浏览器实例和一个任务
- **用户数据目录**: `~/.geo_client/browser_data/`

### BrowserView 特性

- 浏览器页面占据主窗口右侧 50%
- 支持显示/隐藏切换
- 自动适应窗口大小

## 故障排除

### 问题：Electron 安装失败

**症状**：
```
Error: Electron failed to install correctly
```

**解决方案**：

1. 配置镜像后重新安装：
   ```bash
   export ELECTRON_MIRROR=https://npmmirror.com/mirrors/electron/
   pnpm install --force
   ```

2. 或手动触发下载：
   ```bash
   node node_modules/electron/install.js
   ```

### 问题：找不到 dist-electron/main.js

**症状**：
```
Cannot find module 'dist-electron/main.js'
```

**解决方案**：

编译 Electron 主进程代码：
```bash
pnpm exec tsc -p electron
```

### 问题：Playwright 浏览器未安装

**症状**：
```
Error: browserType.launch: Executable doesn't exist
```

**解决方案**：

```bash
npx playwright install chromium
```

### 问题：端口 1420 被占用

**症状**：
```
Error: Port 1420 is already in use
```

**解决方案**：

```bash
# macOS/Linux
lsof -ti:1420 | xargs kill -9

# 或找到并手动终止进程
lsof -i:1420
```

### 问题：TypeScript 编译错误

**解决方案**：

1. 运行类型检查查看具体错误：
   ```bash
   pnpm run type-check
   ```

2. 确保安装了所有类型定义：
   ```bash
   pnpm install
   ```

### 问题：better-sqlite3 编译失败

**症状**：
```
Error: Cannot find module 'better-sqlite3'
```

**解决方案**：

重新构建 native 模块：
```bash
pnpm rebuild better-sqlite3
```

### 问题：数据库权限问题

**解决方案**：

确保有写入权限：
```bash
chmod -R 755 ~/.geo_client
```

## 构建和打包

### 开发构建

```bash
pnpm run build
```

这会：
1. 编译前端代码（React）
2. 编译 Electron 主进程代码
3. 输出到 `dist/` 和 `dist-electron/`

### 生产打包

```bash
pnpm run electron:build
```

这会生成可分发的安装包：
- **macOS**: `.dmg` 文件（支持 x64 和 arm64）
- **Windows**: `.exe` 安装程序
- **Linux**: `.AppImage` 文件

打包后的文件在 `release/` 目录。

## IPC 通信

前端通过 `window.electronAPI` 与主进程通信：

### 认证相关

```typescript
// 登录
const response = await window.electronAPI.auth.login(username, password, apiBaseUrl);

// 获取 token
const tokenInfo = await window.electronAPI.auth.getToken();

// 退出登录
await window.electronAPI.auth.logout();

// 检查 token 是否有效
const isValid = await window.electronAPI.auth.checkTokenValid();
```

### BrowserView 控制

```typescript
// 显示浏览器页面
await window.electronAPI.browserView.show('https://www.example.com');

// 隐藏浏览器页面
await window.electronAPI.browserView.hide();
```

## 开发计划

本项目按阶段开发：

1. ✅ 阶段一：项目初始化和基础框架
2. ✅ 阶段二：认证模块
3. ⏳ 阶段三：登录列表管理
4. ✅ 阶段四：Provider 实现（豆包）
5. ✅ 阶段五：任务队列管理
6. ⏳ 阶段六：本地关键词搜索功能
7. ⏳ 阶段七：浏览器自动化优化
8. ⏳ 阶段八：结果上传和优化
9. ⏳ 阶段九：日志查看和打包

## 开发注意事项

### 1. 模块系统

- **前端（src/）**: ES Module
- **Electron（electron/）**: 编译为 CommonJS
- **dist-electron/**: 包含 `package.json` 声明为 CommonJS

### 2. 类型检查

在开发过程中定期运行类型检查：

```bash
pnpm run type-check
```

### 3. 代码修改后的操作

- **前端代码（src/）**: 自动热更新，无需操作
- **Electron 代码（electron/）**: 
  1. 重新编译：`pnpm exec tsc -p electron`
  2. 重启应用：Ctrl+C 停止，然后 `pnpm run electron:dev`

### 4. BrowserView 使用

浏览器页面会在主窗口右侧显示（占 50% 宽度），可以：
- 通过 IPC 控制显示/隐藏
- 自动适应窗口大小变化
- 独立的浏览器上下文（不共享 cookies）

### 5. 浏览器池管理

- 同时只运行 1 个浏览器实例
- 同时只处理 1 个任务
- 任务完成后自动关闭浏览器（如果没有活跃标签页）

## 常见开发场景

### 场景 1：添加新的 Provider

1. 在 `electron/providers/` 创建新文件（如 `deepseek.ts`）
2. 继承 `BaseProvider` 类并实现 `search()` 方法
3. 重新编译：`pnpm exec tsc -p electron`
4. 重启应用

### 场景 2：添加新的 IPC 通道

1. 在 `electron/preload.ts` 添加 API 定义
2. 在 `src/types/electron.d.ts` 添加类型定义
3. 在相应的服务文件（如 `electron/services/xxx.ts`）添加 IPC 处理
4. 在 `electron/main.ts` 初始化新的处理程序
5. 重新编译并重启

### 场景 3：修改数据库结构

1. 修改 `electron/database/index.ts` 的建表语句
2. 更新 `schema_version` 并添加迁移逻辑
3. 重新编译并重启
4. 数据库会自动迁移

## 性能优化建议

### 1. 开发模式优化

使用 watch 模式自动编译：

```bash
# 终端 1
pnpm exec tsc -p electron --watch

# 终端 2
pnpm run electron:dev
```

### 2. 减少编译时间

- 使用 `skipLibCheck: true`（已配置）
- 使用增量编译（TypeScript 默认开启）

### 3. 减少包体积

生产构建时会自动：
- Tree-shaking（Vite）
- 压缩代码（esbuild）
- 只打包必要的依赖

## 调试技巧

### 调试主进程

在 `electron/main.ts` 添加：
```typescript
console.log('调试信息:', someVariable);
```

输出会显示在启动 Electron 的终端。

### 调试渲染进程

1. 开发模式会自动打开 DevTools
2. 或在应用中按 `Cmd+Option+I`（macOS）或 `Ctrl+Shift+I`（Windows/Linux）

### 调试 Playwright

在 Provider 代码中添加：
```typescript
await page.screenshot({ path: 'debug.png' });
console.log('Current URL:', page.url());
```

### 查看数据库

```bash
# 使用 sqlite3 命令行工具
sqlite3 ~/.geo_client/cache.db

# 查看所有表
.tables

# 查看认证信息
SELECT * FROM auth;

# 退出
.quit
```

## 许可证

MIT

## 更新日志

### v0.1.0 (2024-01-20)

- ✅ 从 Tauri + Rust 迁移到 Electron + Node.js/TypeScript
- ✅ 实现基础框架（主进程、预加载脚本、IPC）
- ✅ 实现认证模块（登录、token 管理）
- ✅ 实现数据库模块（better-sqlite3）
- ✅ 实现浏览器池管理（Playwright）
- ✅ 实现 Provider（豆包）
- ✅ 集成 BrowserView（浏览器页面内部显示）
- ✅ 实现任务队列
