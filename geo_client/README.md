# Geo Client

客户端抓取程序 - 基于 React + Tauri 的桌面应用

## 技术栈

- **前端**: React + TypeScript + Vite
- **后端**: Tauri (Rust)
- **UI组件库**: shadcn/ui (Tailwind CSS)
- **状态管理**: Zustand
- **路由**: React Router
- **浏览器自动化**: chromiumoxide (Rust)

## 命令速查表

| 操作 | 命令 |
|------|------|
| **初始化项目** | `./scripts/setup.sh` |
| **开发模式（推荐）** | `npm run tauri:dev:fast` |
| **标准开发模式** | `npm run tauri:dev` |
| **生产构建** | `npm run tauri:build` |
| **离线构建** | `./scripts/build-offline.sh` |
| **类型检查** | `npm run type-check` |
| **预构建依赖** | `npm run tauri:prebuild` |
| **检查依赖** | `./scripts/check-deps.sh` |
| **修复索引问题** | `./scripts/fix-cargo-index.sh` |

## 快速开始

### 一键初始化

```bash
# 运行初始化脚本（自动检测并安装所需依赖）
./scripts/setup.sh
```

初始化脚本会自动：
- ✅ 检测 Node.js 和 Rust 是否已安装
- ✅ 自动安装缺失的 Rust（支持国内镜像源）
- ✅ 安装 Node.js 依赖（优先使用 pnpm）
- ✅ 安装 Tauri CLI
- ✅ 创建环境变量文件

### 手动安装

#### 前置要求

- **Node.js** 18+ 
- **Rust** 1.70+
- **Tauri CLI**

#### 安装步骤

1. **安装 Node.js 依赖**（优先使用 pnpm）：
   ```bash
   # 如果已安装 pnpm
   pnpm install
   
   # 或使用 npm
   npm install
   ```

2. **安装 Rust**（如果未安装）：
   ```bash
   # 官方源
   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
   
   # 或使用国内镜像源（如果官方源无法访问）
   export RUSTUP_DIST_SERVER=https://mirrors.tuna.tsinghua.edu.cn/rustup
   export RUSTUP_UPDATE_ROOT=https://mirrors.tuna.tsinghua.edu.cn/rustup/rustup
   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
   ```

3. **安装 Tauri CLI**：
   ```bash
   cargo install tauri-cli
   ```

## 开发命令

### 开发模式

#### 快速开发模式（推荐）

智能检测依赖状态，自动跳过已安装的依赖：

```bash
npm run tauri:dev:fast
# 或
pnpm run tauri:dev:fast
```

#### 标准开发模式

```bash
npm run tauri:dev
# 或
pnpm run tauri:dev
```

#### 仅前端开发

```bash
npm run dev
# 或
pnpm run dev
```

### 构建命令

#### 生产构建

```bash
npm run tauri:build
# 或
pnpm run tauri:build
```

#### 离线构建（跳过索引更新）

如果 Cargo 索引更新卡住，可以使用离线模式：

```bash
./scripts/build-offline.sh
```

或手动执行：

```bash
cd src-tauri
cargo build --offline --release
```

### 其他命令

#### 类型检查

```bash
npm run type-check
# 或
pnpm run type-check
```

#### 预构建 Rust 依赖

首次运行或更新依赖后，可以预先下载依赖：

```bash
npm run tauri:prebuild
# 或
pnpm run tauri:prebuild
```

#### 检查依赖状态

检查 Rust 和 Node.js 依赖是否已安装：

```bash
./scripts/check-deps.sh
```

## 脚本说明

项目提供了多个便捷脚本，位于 `scripts/` 目录：

### `setup.sh` - 项目初始化

自动检测并安装所有必需的依赖：

```bash
./scripts/setup.sh
```

**功能**：
- 检测 Node.js 和 Rust 是否已安装
- 自动安装缺失的 Rust（支持镜像源）
- 智能选择包管理器（优先 pnpm）
- 安装 Tauri CLI
- 创建环境变量文件

### `dev.sh` - 智能开发启动

优化的开发启动脚本，自动检测依赖状态：

```bash
./scripts/dev.sh
# 或通过 npm 命令
npm run tauri:dev:fast
```

**功能**：
- 检测 Rust 依赖是否已构建
- 检测 Node.js 依赖是否已安装
- 自动跳过已安装的依赖
- 启动开发服务器

### `check-deps.sh` - 依赖检查

检查依赖是否已安装和构建：

```bash
./scripts/check-deps.sh
```

### `build-offline.sh` - 离线构建

使用离线模式构建，跳过索引更新：

```bash
./scripts/build-offline.sh
```

### `fix-cargo-index.sh` - 修复索引问题

修复 Cargo 索引更新卡住的问题：

```bash
./scripts/fix-cargo-index.sh
```

提供三种解决方案：
1. 切换到官方源 + sparse protocol
2. 使用离线模式
3. 清除索引缓存

## 项目结构

```
geo_client/
├── src/                    # React前端代码
│   ├── pages/             # 页面组件
│   ├── components/        # 通用组件
│   ├── hooks/             # React Hooks
│   ├── stores/            # Zustand状态管理
│   ├── utils/             # 工具函数
│   └── types/             # TypeScript类型定义
├── src-tauri/             # Tauri后端代码（Rust）
│   ├── src/
│   │   ├── commands/      # Tauri命令
│   │   ├── providers/     # Provider实现
│   │   ├── models/        # 数据模型
│   │   ├── storage/       # 存储模块
│   │   └── queue/         # 任务队列
│   ├── .cargo/            # Cargo配置
│   │   └── config.toml    # Cargo优化配置
│   ├── capabilities/      # Tauri权限配置
│   └── Cargo.toml         # Rust依赖配置
├── scripts/               # 便捷脚本
│   ├── setup.sh           # 初始化脚本
│   ├── dev.sh             # 开发启动脚本
│   ├── check-deps.sh      # 依赖检查脚本
│   ├── build-offline.sh   # 离线构建脚本
│   └── fix-cargo-index.sh # 索引修复脚本
└── package.json           # Node.js依赖
```

## 环境变量

创建 `.env.development` 和 `.env.production` 文件配置API地址：

**`.env.development`**:
```env
VITE_API_BASE_URL=http://127.0.0.1:8000
VITE_APP_ENV=development
```

**`.env.production`**:
```env
VITE_API_BASE_URL=https://api.example.com
VITE_APP_ENV=production
```

## Cargo 配置优化

项目已配置 Cargo 优化选项（`src-tauri/.cargo/config.toml`）：

- ✅ **增量编译**：只编译更改的部分
- ✅ **系统 Git CLI**：使用系统 Git 而不是 libgit2（更快更稳定）
- ✅ **Sparse Protocol**：只下载需要的包信息（更快）
- ✅ **镜像源支持**：支持中科大镜像源（可配置）

### 切换镜像源

如果官方源较慢，可以编辑 `src-tauri/.cargo/config.toml`：

```toml
# 使用中科大镜像
[source.crates-io]
replace-with = 'ustc'

[source.ustc]
registry = "https://mirrors.ustc.edu.cn/crates.io-index"
```

如果镜像源更新索引卡住，可以：
1. 临时注释掉镜像源配置，使用官方源
2. 或运行 `./scripts/fix-cargo-index.sh` 自动修复

## 故障排除

### 问题：Cargo 索引更新卡住

**解决方案**：

1. **使用离线模式**（如果依赖已下载）：
   ```bash
   ./scripts/build-offline.sh
   ```

2. **切换到官方源**：
   ```bash
   ./scripts/fix-cargo-index.sh
   # 选择选项 1
   ```

3. **清除缓存**：
   ```bash
   rm -rf ~/.cargo/registry/index
   rm -rf ~/.cargo/.package-cache
   ```

### 问题：Rust 安装失败（网络问题）

**解决方案**：

1. **使用国内镜像源**：
   ```bash
   export RUSTUP_DIST_SERVER=https://mirrors.tuna.tsinghua.edu.cn/rustup
   export RUSTUP_UPDATE_ROOT=https://mirrors.tuna.tsinghua.edu.cn/rustup/rustup
   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
   ```

2. **或运行初始化脚本**（自动处理）：
   ```bash
   ./scripts/setup.sh
   ```

### 问题：TypeScript 类型错误

**解决方案**：

1. **运行类型检查**：
   ```bash
   npm run type-check
   ```

2. **确保环境变量类型定义存在**：
   检查 `src/vite-env.d.ts` 文件是否存在

### 问题：依赖已安装但仍重新安装

**解决方案**：

1. **使用快速开发模式**（自动检测）：
   ```bash
   npm run tauri:dev:fast
   ```

2. **手动检查依赖**：
   ```bash
   ./scripts/check-deps.sh
   ```

### 问题：强制重新构建

**解决方案**：

```bash
# 清理 Rust 构建缓存
cd src-tauri
cargo clean

# 清理 Node.js 依赖（可选）
rm -rf node_modules
pnpm install  # 或 npm install
```

## 开发计划

本项目按阶段开发：

1. ✅ 阶段一：项目初始化和基础框架
2. ⏳ 阶段二：认证模块
3. ⏳ 阶段三：登录列表管理
4. ⏳ 阶段四：Rust Provider重写
5. ⏳ 阶段五：任务队列管理
6. ⏳ 阶段六：本地关键词搜索功能
7. ⏳ 阶段七：RPA集成和优化
8. ⏳ 阶段八：结果上传和优化
9. ⏳ 阶段九：日志查看和打包

## 许可证

MIT
