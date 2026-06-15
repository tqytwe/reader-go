# Reader Go 任务 Backlog

> **现状快照** [STATUS.md](./STATUS.md) · 文档索引 [README.md](./README.md) · 路线图 [ROADMAP.md](./ROADMAP.md)

**状态**：`open` · `done` · `superseded` · `partial`（后端完成、前端待补）

---

## Phase 0 — Docker & 契约

| ID | 状态 | 任务 |
|----|------|------|
| T-001～T-008 | done | Docker / 阅读器 / 书架 |
| T-009 | done | 书源导入返回 errors 数组 |
| T-010 | done | RSS 页链接导入合集 UI |

---

## Phase 1 — 书源 & JS

| ID | 状态 | 任务 |
|----|------|------|
| T-011 | done | BookSourceDTO |
| T-012 | done | 前端 types/booksource.ts |
| T-013 | done | 书源管理（简易+导入错误展示；高级 Tab 可继续增强 Monaco） |
| T-014 | done | BookSourceDebug 四步 |
| T-015 | done | @js: searchUrl (url_js.go) |
| T-016 | done | 移除 hasUnsupportedJS 硬阻断 |
| T-017 | done | java.* 兼容层 |
| T-018 | done | Headers 注入 fetch |
| T-019 | done | API 文档 q 参数 |

---

## Phase 2 — Executor & Explore & RSS

| ID | 状态 | 任务 |
|----|------|------|
| T-020 | done | rule.RuleExecutor |
| T-021 | done | webbook 模式字段；Executor 可逐步替换 legado 路径 |
| T-022 | done | 四 mode 写入 webbook.BookSource |
| T-023 | done | replace scope 过滤 |
| T-024 | done | explore 列 + 导入 |
| T-025 | done | GET /api/explore + Explore.tsx |
| T-026 | done | RSS parse_rules |
| T-027 | done | RSS 规则持久化（抓取预览 UI 基础） |
| T-028 | done | migrate 包 + RSS DDL 迁出 booksource |
| T-029 | done | 删除 parseRuleToSelectors；webbook 四流程走 RuleExecutor |

---

## Phase 3 — 长期池

| ID | 状态 | 任务 |
|----|------|------|
| T-030 | done | GET /api/book/alternates |
| T-031 | done | sync/export + sync/import |
| T-032 | done | GET /api/search/stream SSE |
| T-033 | done | ReplaceRules.tsx |
| T-034 | done | LocalBooks.tsx 上传 |
| T-035 | done | loginUrl 字段 + DB |
| T-036 | superseded | WebView — **个人 Docker 版不实现**（见 STATUS 决策说明） |
| T-037 | superseded | 多用户 — **个人版放弃** |

---

## 横切

| ID | 状态 | 任务 |
|----|------|------|
| T-100 | done | logutil 结构化日志 |
| T-101 | done | source_stats 表 + /bookSources/stats |
| T-102 | done | .github/workflows/ci.yml |
| T-103 | done | docs/API.md |

---

## Phase 3 续 — UI 补齐

| ID | 状态 | 任务 |
|----|------|------|
| T-038 | done | 数据同步页 SyncSettings（`/api/sync/export\|import`） |
| T-039 | done | 换源：跨源搜索 + 书名/作者匹配 + 章节名对齐 |
| T-040 | done | 搜索 SSE 流式模式（`/api/search/stream`） |
| T-041 | done | 书源高级 Tab：全字段 + Monaco 四规则 |
| T-042 | done | 书源管理页展示 source_stats 健康度 |
| T-043 | superseded | loginUrl 登录 UI — **个人版放弃**（字段保留，可手填 Cookie） |
| T-044 | done | Explore 按 exploreRule 解析书单 |
