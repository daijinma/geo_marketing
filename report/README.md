# GEO 报告网站

独立的通用报告网站（React + Vite + Tailwind）。报告内容通过 `src/config/report.json` 配置文件注入，替换配置即可生成新的报告页面。

## 使用方式

```bash
cd report
pnpm install
pnpm dev
```

## 配置文件

- `src/config/report.json`
  - `meta`: 标题、日期、客户、摘要与首页指标
  - `sections`: 章节列表，支持 `image` / `table` / `rich` / `mixed` / `recommendations` 类型

## 图片资源

将图片放入 `src/assets/`，并在配置中使用相对路径：

```json
{ "src": "assets/图片文件名.png", "alt": "说明" }
```

本次报告的图片源文件来自 `百保力GEO报告-0227.files/`，建议拷贝到 `report/src/assets/`。

## 适配新报告

1. 替换 `src/config/report.json`
2. 拷贝对应图片到 `src/assets/`
3. 运行 `pnpm dev` 预览
