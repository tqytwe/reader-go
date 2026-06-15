# Reader Go API 参考

> 基础 URL：`http://localhost:6464/api`（Docker 内 nginx 反代：`http://localhost:6465/api`）

统一响应：`{ "code": 0, "message": "ok", "data": ... }`，`code !== 0` 为错误。

## 健康检查

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 服务状态 |

## 书源

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/bookSources` | 列表（完整 Legado 字段） |
| POST | `/bookSources` | 创建 |
| PUT | `/bookSources/:id` | 更新 |
| DELETE | `/bookSources/:id` | 删除 |
| POST | `/bookSources/import` | 批量导入，返回 `{imported,failed,errors[]}` |
| POST | `/bookSources/import/collection` | 链接导入合集 |
| GET | `/bookSources/stats` | 书源健康度 |
| GET | `/bookSources/debug` | SSE 调试（search/info/toc/content） |

## 搜索

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/search` | `q` | 并发搜书 |
| GET | `/search/stream` | `q` | SSE 流式结果 |

## 书籍

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/book/info` | `bookKey` | 详情 |
| GET | `/book/toc` | `bookKey` | 目录 `{chapters:[]}` |
| GET | `/book/content` | `bookKey`, `chapter` | 正文 `{content}` |
| GET | `/book/alternates` | `bookKey` | 换源候选 |

**bookKey 格式**：`{sourceId}::{url}`

## 书海 Explore

| 方法 | 路径 | 参数 |
|------|------|------|
| GET | `/explore` | `sourceId`, `tab?` |

## 书架

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/shelf` | 列表 |
| POST | `/shelf` | 加入 |
| PUT | `/shelf/:id` | 更新 |
| PUT | `/shelf/:id/progress` | 阅读进度 |
| DELETE | `/shelf/:id` | 删除（或 `?bookKey=`） |

## 替换规则

| 方法 | 路径 |
|------|------|
| GET/POST | `/replaceRules` |
| PUT/DELETE | `/replaceRules/:id` |

运行时 `scope`：`all` | `title` | `content` | `toc` | `search`

## WebDAV 同步（JSON）

| 方法 | 路径 |
|------|------|
| GET | `/sync/export` |
| POST | `/sync/import` |

## 本地书

| 方法 | 路径 |
|------|------|
| POST | `/localBooks`（multipart file） |
| GET | `/localBooks` |
| GET | `/localBooks/:id/content` |

## RSS

见 `rssApi`（`/rss/feeds` 等），订阅源含 `parseRules` JSON 字段。

常用接口：

| 方法 | 路径 |
|------|------|
| GET | `/api/rss/feeds` |
| POST | `/api/rss/feeds` |
| GET | `/api/rss/feeds/:id/items` |
| POST | `/api/rss/feeds/:id/preview` |
| POST | `/api/rss/feeds/:id/fetch` |
| POST | `/api/rss/import/collection` |
