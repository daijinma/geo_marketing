# 大模型搜索功能迁移总结

## 概述

本次更新将服务端的网页抓取功能迁移到客户端，实现了完整的"大模型搜索"功能。

## 主要功能

### 1. 大模型搜索
- 在侧边栏添加了"大模型搜索"菜单项
- 支持批量关键词搜索（每行一个关键词）
- 支持多平台并行搜索（DeepSeek、豆包）
- 支持设置每个关键词的查询次数

### 2. 客户端Provider服务
创建了客户端的provider服务，用于执行网页抓取：
- `BaseProvider`: Provider基类，提供通用功能
- `DeepSeekProvider`: DeepSeek平台搜索实现
- `DoubaoProvider`: 豆包平台搜索实现
- `ProviderFactory`: Provider工厂类

### 3. 本地任务管理
- 任务在本地创建并保存到SQLite数据库
- 使用任务队列管理搜索任务
- 支持任务状态跟踪（pending, running, completed, failed）
- 任务完成后可以提交到服务器保存

### 4. 搜索结果显示
- 实时显示搜索结果
- 三种视图：汇总表格、Domain统计、详细日志
- 支持展开查看详细信息
- 提供外部链接跳转

### 5. 平台授权管理
- 在授权列表页面显示已授权的平台
- 支持检查平台登录状态
- 支持打开平台登录（非headless模式）

## 技术架构

### 后端（Electron主进程）

#### 目录结构
```
electron/
├── providers/              # Provider服务
│   ├── base.ts            # 基类
│   ├── deepseek.ts        # DeepSeek实现
│   ├── doubao.ts          # 豆包实现
│   └── index.ts           # 导出
├── services/              # 服务层
│   ├── auth.ts            # 认证服务
│   ├── search.ts          # 搜索服务
│   ├── task-manager.ts    # 任务管理
│   ├── task-queue.ts      # 任务队列
│   ├── browser-pool.ts    # 浏览器池
│   └── browser-view-manager.ts
├── database/              # 数据库
│   ├── index.ts           # 数据库操作
│   └── types.d.ts         # 类型定义
├── main.ts                # 主进程入口
└── preload.ts             # Preload脚本

```

#### 核心服务

1. **搜索服务** (`services/search.ts`)
   - `search:createTask`: 创建搜索任务
   - `search:checkLoginStatus`: 检查平台登录状态
   - `search:getTasks`: 获取任务列表
   - `search:cancelTask`: 取消任务
   - `search:taskUpdated`: 任务更新事件

2. **任务管理服务** (`services/task-manager.ts`)
   - `task:getAuthorizedPlatforms`: 获取已授权平台
   - `task:saveToLocal`: 保存任务到本地
   - `task:submitToServer`: 提交任务到服务器

3. **浏览器池** (`services/browser-pool.ts`)
   - 管理浏览器实例
   - 支持浏览器复用
   - 任务队列和锁机制
   - 自动清理浏览器锁文件

### 前端（React）

#### 目录结构
```
src/
├── pages/
│   └── Search.tsx         # 搜索页面
├── components/
│   ├── Sidebar.tsx        # 侧边栏（添加搜索菜单）
│   ├── LoginList.tsx      # 登录列表（更新为显示授权状态）
│   └── SearchResults.tsx  # 搜索结果组件
├── stores/
│   └── loginStatusStore.ts # 登录状态Store
└── types/
    └── electron.d.ts      # Electron API类型定义
```

#### 核心组件

1. **搜索页面** (`pages/Search.tsx`)
   - 关键词输入（支持多行）
   - 平台选择（复选框）
   - 查询次数设置
   - 开始搜索按钮
   - 提交到服务器按钮
   - 搜索结果展示

2. **授权列表页面** (`pages/Auth.tsx`)
   - 平台登录管理
   - 任务队列展示
   - 已授权平台列表（新增）

3. **搜索结果组件** (`components/SearchResults.tsx`)
   - 汇总表格：显示查询词、Sub Query、引用数、响应时间
   - Domain统计：按域名统计引用次数
   - 详细日志：展示所有搜索结果详情

## 数据流

### 1. 搜索流程
```
用户输入 → 创建任务 → 任务队列 → Provider执行 → 返回结果 → 显示结果
```

详细步骤：
1. 用户在Search页面输入关键词和选择平台
2. 点击"开始搜索"创建任务（`search:createTask`）
3. 任务添加到任务队列（TaskQueue）
4. 任务队列按顺序执行任务
5. Provider打开浏览器，执行搜索，提取结果
6. 通过IPC发送结果到前端（`search:taskUpdated`）
7. 前端更新搜索结果显示

### 2. 保存流程
```
任务完成 → 保存到本地 → 用户确认 → 提交到服务器
```

详细步骤：
1. 搜索任务完成后，保存到本地SQLite数据库
2. 用户可以继续执行更多搜索
3. 所有任务完成后，点击"提交到服务器"按钮
4. 客户端调用服务器API提交任务数据
5. 服务器保存任务和搜索结果

## 数据库结构

### tasks表
```sql
CREATE TABLE tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id INTEGER,                    -- 服务器任务ID
  keywords TEXT NOT NULL,             -- 关键词列表（逗号分隔）
  platforms TEXT NOT NULL,            -- 平台列表（逗号分隔）
  query_count INTEGER NOT NULL,       -- 查询次数
  status TEXT NOT NULL,               -- 状态
  result_data TEXT,                   -- 结果数据（JSON）
  created_at TEXT DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(task_id)
);
```

### login_status表
```sql
CREATE TABLE login_status (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  platform_type TEXT NOT NULL,        -- 平台类型（llm/platform）
  platform_name TEXT NOT NULL,        -- 平台名称
  is_logged_in INTEGER NOT NULL DEFAULT 0, -- 是否已登录
  last_check_at TEXT,                 -- 最后检查时间
  updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(platform_name)
);
```

## API接口

### Electron IPC接口

#### 搜索相关
- `search:createTask(params)`: 创建搜索任务
- `search:checkLoginStatus(platform)`: 检查平台登录状态
- `search:getTasks()`: 获取任务列表
- `search:cancelTask(taskId)`: 取消任务
- `search:taskUpdated` (event): 任务更新事件

#### 任务管理相关
- `task:getAuthorizedPlatforms()`: 获取已授权平台
- `task:saveToLocal(keywords, platforms, queryCount, status, resultData)`: 保存任务到本地
- `task:submitToServer(taskId, apiBaseUrl, token)`: 提交任务到服务器

### 服务器API接口
- `POST /tasks/create`: 创建任务
  ```json
  {
    "keywords": ["关键词1", "关键词2"],
    "platforms": ["deepseek", "doubao"],
    "query_count": 1
  }
  ```

## 使用说明

### 1. 平台授权
1. 进入"授权列表"页面
2. 找到需要授权的平台（如DeepSeek、豆包）
3. 点击"登录"按钮
4. 在弹出的浏览器窗口中完成登录
5. 关闭浏览器窗口，平台授权完成
6. 在页面底部可以看到"已授权的平台"列表

### 2. 执行搜索
1. 进入"大模型搜索"页面
2. 在关键词输入框中输入关键词（每行一个）
3. 选择要使用的平台（可多选）
4. 设置每个关键词的查询次数（建议1-3次）
5. 点击"开始搜索"按钮
6. 等待搜索完成，实时查看搜索结果

### 3. 提交结果
1. 搜索完成后，页面会显示"提交到服务器"按钮
2. 点击按钮将所有搜索结果提交到服务器
3. 提交成功后，数据会保存到服务器数据库

## 注意事项

1. **浏览器管理**
   - 客户端使用系统安装的Chrome浏览器
   - 浏览器数据保存在 `~/.geo_client/browser/{platform}/`
   - 支持headless和非headless模式

2. **任务队列**
   - 默认同时只运行1个任务（避免资源冲突）
   - 任务按创建时间顺序执行
   - 可以取消未执行的任务

3. **数据存储**
   - 本地数据库位置：`~/.geo_client/cache.db`
   - 搜索结果临时保存在内存中
   - 提交到服务器后可清除本地数据

4. **平台支持**
   - 目前支持DeepSeek和豆包
   - 其他平台可以通过实现Provider接口扩展

## 后续优化方向

1. **性能优化**
   - 支持多任务并行（需要管理多个浏览器实例）
   - 优化页面加载等待时间
   - 缓存已登录状态

2. **功能增强**
   - 支持更多平台（ChatGPT、Claude等）
   - 支持搜索结果导出（CSV、Excel）
   - 支持搜索历史管理

3. **用户体验**
   - 添加搜索进度条
   - 支持搜索任务暂停/恢复
   - 优化错误提示和异常处理

## 相关文件

### 新增文件
- `electron/providers/base.ts`
- `electron/providers/deepseek.ts`
- `electron/providers/doubao.ts`
- `electron/providers/index.ts`
- `electron/services/search.ts`
- `electron/services/task-manager.ts`
- `src/pages/Search.tsx`

### 修改文件
- `electron/main.ts`: 注册新的IPC处理程序
- `electron/preload.ts`: 添加新的API暴露
- `src/App.tsx`: 添加Search路由
- `src/components/Sidebar.tsx`: 添加搜索菜单项
- `src/components/LoginList.tsx`: 添加授权状态更新
- `src/components/SearchResults.tsx`: 更新结果展示结构
- `src/pages/Auth.tsx`: 添加已授权平台显示
- `src/stores/loginStatusStore.ts`: 添加updateLoginStatus方法
- `src/types/electron.d.ts`: 添加新的API类型定义

## 测试建议

1. **单元测试**
   - Provider功能测试
   - 任务队列管理测试
   - 数据库操作测试

2. **集成测试**
   - 完整搜索流程测试
   - 多任务并发测试
   - 服务器同步测试

3. **端到端测试**
   - 用户授权流程
   - 搜索和结果展示
   - 数据提交流程
