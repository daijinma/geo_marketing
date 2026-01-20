# 大模型搜索功能 - 快速开始

## 安装依赖

```bash
cd geo_client
npm install
```

新增依赖：
- axios: 用于HTTP请求

## 启动应用

```bash
# 开发模式
make dev

# 或使用npm
npm run dev
```

## 使用流程

### 1. 登录系统
1. 启动应用后，输入用户名、密码和服务器地址
2. 点击登录

### 2. 平台授权
1. 进入"授权列表"页面
2. 在"网页大模型"部分找到DeepSeek或豆包
3. 点击"登录"按钮
4. 在打开的浏览器窗口中完成登录
5. 关闭浏览器窗口
6. 点击刷新按钮检查登录状态
7. 绿色对钩表示已登录成功

### 3. 执行搜索
1. 进入"大模型搜索"页面
2. 在关键词输入框中输入关键词，每行一个，例如：
   ```
   人工智能发展趋势
   机器学习应用
   深度学习框架
   ```
3. 选择平台（可多选）
4. 设置查询次数（建议1-3次）
5. 点击"开始搜索"
6. 等待搜索完成，查看实时结果

### 4. 查看结果
搜索结果提供三种视图：
- **汇总表格**：显示每次搜索的关键词、子查询、引用数和响应时间
- **Domain统计**：按域名统计引用次数
- **详细日志**：展开查看每个引用的详细信息（标题、摘要、链接等）

### 5. 提交到服务器
1. 搜索完成后，点击"提交到服务器"按钮
2. 系统会将所有搜索结果提交到服务器保存
3. 提交成功后会显示确认消息

## 目录结构

```
geo_client/
├── electron/                      # Electron主进程
│   ├── providers/                 # 搜索provider
│   │   ├── base.ts               # 基类
│   │   ├── deepseek.ts           # DeepSeek实现
│   │   ├── doubao.ts             # 豆包实现
│   │   └── index.ts              # 导出
│   ├── services/                 # 服务层
│   │   ├── auth.ts               # 认证
│   │   ├── search.ts             # 搜索
│   │   ├── task-manager.ts       # 任务管理
│   │   ├── task-queue.ts         # 任务队列
│   │   └── browser-pool.ts       # 浏览器池
│   ├── database/                 # 数据库
│   │   └── index.ts              # SQLite操作
│   ├── main.ts                   # 主进程入口
│   └── preload.ts                # Preload脚本
├── src/                          # React前端
│   ├── pages/
│   │   ├── Search.tsx            # 搜索页面 ✨新增
│   │   └── Auth.tsx              # 授权页面（更新）
│   ├── components/
│   │   ├── Sidebar.tsx           # 侧边栏（新增搜索菜单）
│   │   ├── LoginList.tsx         # 登录列表（更新）
│   │   └── SearchResults.tsx     # 搜索结果（更新）
│   └── stores/
│       └── loginStatusStore.ts   # 登录状态（更新）
└── MIGRATION_SUMMARY.md          # 详细技术文档

```

## 常见问题

### Q: 浏览器启动失败
**A**: 确保系统已安装Chrome浏览器。如果问题仍然存在，尝试删除 `~/.geo_client/browser/` 目录。

### Q: 登录状态检查失败
**A**: 
1. 确保平台已正确登录
2. 尝试关闭应用重新启动
3. 删除 `~/.geo_client/browser/{platform}/` 目录后重新登录

### Q: 搜索任务卡住
**A**: 
1. 检查网络连接
2. 查看是否有其他任务正在运行
3. 尝试取消任务后重新开始

### Q: 提交到服务器失败
**A**: 
1. 确保已登录系统
2. 检查服务器地址是否正确
3. 确认服务器运行正常

## 数据存储位置

- 本地数据库：`~/.geo_client/cache.db`
- 浏览器数据：`~/.geo_client/browser/{platform}/`

## 开发调试

### 查看日志
开发模式下会自动打开DevTools，可以在Console中查看日志。

### 数据库查看
```bash
sqlite3 ~/.geo_client/cache.db
.tables
SELECT * FROM login_status;
SELECT * FROM tasks;
```

### 重置数据
```bash
# 删除所有本地数据
rm -rf ~/.geo_client/
```

## 技术栈

- **Electron**: 跨平台桌面应用框架
- **React**: 前端UI框架
- **TypeScript**: 类型安全
- **Playwright**: 浏览器自动化
- **SQLite**: 本地数据库
- **Zustand**: 状态管理
- **Tailwind CSS**: 样式框架

## 更多信息

详细技术文档请参考 [MIGRATION_SUMMARY.md](./MIGRATION_SUMMARY.md)
