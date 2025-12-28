# LLM Sentry Monitor Service

本仓库是 `LLM Sentry` 项目的监控抓取模块，负责模拟用户行为并解析 AI 回答。

## 1. 目录结构
```text
llm_sentry_monitor/
├── main.py              # 业务入口
├── requirements.txt     # 依赖清单
├── core/                # 核心解析逻辑
├── providers/           # 模型适配器 (DeepSeek, 豆包)
└── README.md            # 本说明文件
```

## 2. 快速启动

### 安装依赖
```bash
cd llm_sentry_monitor
pip install -r requirements.txt
playwright install chromium
```

### 运行监测
```bash
python main.py
```

## 3. 核心逻辑
- **Web 自动化**: 使用 Playwright 模拟真实浏览器操作，绕过 API 限制。
- **持久化登录**: 首次运行请手动登录，Cookie 将保存在 `./browser_data`。
- **引用解析**: 自动提取回答中的外部链接并统计域名占比。

## 4. 对接新模型
只需在 `providers/` 目录下继承 `BaseProvider` 并实现 `search` 方法即可。
