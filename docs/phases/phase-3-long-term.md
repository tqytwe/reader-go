# Phase 3：长期功能池

> 返回 [ROADMAP](../ROADMAP.md) · 任务 [BACKLOG](../BACKLOG.md) · **持续迭代**

**目标**：对齐 [hectorqin/reader](https://github.com/hectorqin/reader) 高级能力，按价值/复杂度排序逐项交付。

**前置**：Phase 2 完成（RuleExecutor 稳定 + Explore + RSS）。

## 功能池（优先级排序）

| 功能 | 方案要点 | 依赖 | reader 对标 |
|------|----------|------|-------------|
| **换源** | `canonicalId` + 候选源 API + 章节名模糊对齐 | 稳定 bookKey、完整 toc | ✅ |
| **WebDAV** | 配置加密存储；同步书架/书源/进度/规则；compose sidecar 可选 | 书架进度 API、版本号 | ✅ |
| **并发搜书 SSE** | `GET /api/search/stream` | Executor 稳定 | ✅ |
| **替换规则 UI** | 新页面 + scope 可视化 | Phase 2 scope 修复 | ✅ |
| **本地书/漫画** | 接 [`localbook`](../internal/localbook/) parser + 阅读器 | 本地书 API 已有 | ✅ |
| **书源登录** | loginUrl + Cookie 回写 UI | JS 引擎 1.2c | 部分 |
| **WebView 书源** | 独立 Playwright 服务 | 低优先级 | 部分 |
| **多用户** | tenant_id 或分库 | 个人版可跳过 | 可选 |

## 换源（详细方案）

### 数据模型

```go
type CanonicalBook struct {
    CanonicalID string   // hash(title+author) 或手动
    Title       string
    Author      string
    Candidates  []BookCandidate // 多书源同一书
}

type BookCandidate struct {
    SourceID int
    BookKey  string
    MatchScore float64 // 章节对齐得分
}
```

### API 草案

```
GET  /api/book/alternates?bookKey=
POST /api/book/switch-source  { fromBookKey, toSourceId, toBookKey }
```

### 章节对齐

- Levenshtein / 归一化章节名
- 缓存 `canonicalId → [{sourceId, chapterMap}]`

### 验收

- 书架书 A 源失效，可一键换到 B 源并恢复阅读进度（最近章节）

## WebDAV 同步

### 同步包格式（JSON）

```json
{
  "version": 1,
  "exportedAt": "2026-05-30T00:00:00Z",
  "shelf": [...],
  "bookSources": [...],
  "replaceRules": [...],
  "progress": [...]
}
```

### 安全

- WebDAV 密码 AES 加密存 DB
- 冲突策略：LWW 或用户选择

### Compose 可选 sidecar

```yaml
  webdav:
    image: ...
    profiles: [sync]
```

## 并发搜书 SSE

```
GET /api/search/stream?q=斗破&sourceIds=1,2,3
Event: result  data: {...}
Event: done     data: {"total":42,"elapsedMs":8500}
```

- 单源超时跳过
- 前端 Search 页实时追加结果

## 替换规则 UI

- 列表 + 启用开关 + scope 多选（all/title/content/toc/search）
- 与 Phase 2 运行时 scope 一致

## 本地书 / 漫画

已有模块：

- [`localbook/txt`](../internal/localbook/txt/)
- [`localbook/epub`](../internal/localbook/epub/)
- [`localbook/cbz`](../internal/localbook/cbz/)

待做：上传 UI、阅读器图片模式、进度同步。

## 书源登录

1. 书源配置 `loginUrl` + `loginCheckJs`
2. 管理页「登录」按钮 → 执行 JS → Cookie 写回 DB
3. 后续请求自动带 Cookie

## WebView 书源

- 独立 microservice：Playwright + HTTP API
- reader-go 调用 `POST /webview/fetch { url, waitSelector }`
- **低优先级**：部署复杂、资源占用高

## 多用户（可选）

个人 Docker 版可跳过。若做：

- `users` 表 + JWT
- 或每用户独立 SQLite 文件

## 不在 Phase 3 核心

- Kindle 推送
- 听书 TTS
- 定时更新书架（可单独 backlog 项）

## 选型原则

每项开工前在 [BACKLOG](../BACKLOG.md) 登记：

1. 用户价值（高/中/低）
2. 实现复杂度（人天）
3. 依赖任务 ID
4. 可验收 curl/UI 步骤

## 参考

- [hectorqin/reader README](https://github.com/hectorqin/reader) — 功能清单
- [LEGADO-COMPAT](../LEGADO-COMPAT.md) — 兼容矩阵更新
