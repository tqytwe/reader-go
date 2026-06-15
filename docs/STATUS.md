# Reader Go 项目状态总览

> **一眼看懂**：本文是「现在怎样了」的单一真相来源。  
> 计划与任务 ID 见 [BACKLOG.md](./BACKLOG.md) · 分阶段设计见 [ROADMAP.md](./ROADMAP.md) · 历史 Bug 见 [ISSUES.md](../ISSUES.md)

**最后更新**：2026-05-30

---

## 文档怎么读？

| 文档 | 性质 | 何时看 |
|------|------|--------|
| **[STATUS.md](./STATUS.md)**（本文） | **现状快照** | 想知道哪些做完了、哪些只有一半 |
| [BACKLOG.md](./BACKLOG.md) | **任务台账** | 查 T-xxx 编号、验收、done/open |
| [ROADMAP.md](./ROADMAP.md) | **路线图** | 理解 Phase 0→3 顺序与依赖 |
| [phases/phase-*.md](./phases/) | **阶段设计稿** | 开工某 Phase 时的详细方案 |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | **架构说明** | 模块边界、RuleExecutor 数据流 |
| [API.md](./API.md) | **接口参考** | 联调、写 curl |
| [LEGADO-COMPAT.md](./LEGADO-COMPAT.md) | **兼容矩阵** | Legado 能力对照（部分条目待同步） |
| [ISSUES.md](../ISSUES.md) | **历史修复** | F-01～F-12 已修记录；下方「待解决」多为过期描述 |

```mermaid
flowchart LR
  STATUS[STATUS 现状]
  BACKLOG[BACKLOG 任务]
  PHASE[phases 设计]
  ROADMAP[ROADMAP 路线]
  ROADMAP --> PHASE
  PHASE --> BACKLOG
  BACKLOG --> STATUS
```

---

## Phase 进度

| 阶段 | 状态 | 说明 |
|------|------|------|
| Phase 0 Docker & 契约 | ✅ 完成 | T-001～T-010 |
| Phase 1 书源 & JS | ✅ 完成 | DTO、@js: URL、java.* 兼容层 |
| Phase 2 Executor & Explore & RSS | ✅ 完成 | RuleExecutor 四流程、Explore API、RSS 规则 |
| Phase 3 长期池 | ✅ 核心完成 | 换源/同步/SSE/Explore 规则解析已落地 |

---

## 明确放弃的功能（个人版）

| 功能 | 决定 | 说明 |
|------|------|------|
| **多用户 / 认证** | ❌ 不做 | 个人 Docker 单用户场景，无租户需求 |
| **loginUrl 登录 UI** | ❌ 不做 | DB 字段保留；需登录书源请在书源高级 Tab **手填 Cookie** |
| **WebView 书源** | ❌ 不做 | 见下方决策说明 |

### WebView 要不要做？

**建议：个人版不需要实现。**

| 维度 | 说明 |
|------|------|
| 成本 | 需独立 Playwright/Chromium 服务，内存 500MB+，compose 多一个 sidecar |
| 收益 | 仅少数书源依赖浏览器渲染；绝大多数 XPath/CSS/JS 书源已覆盖 |
| 原版 | hectorqin/reader 个人版同样不主推 WebView |
| 替代 | 优先换书源；或手填 Cookie 解决登录类问题 |

若将来有**大量** WebView 专属书源需求，再单独立项（T-036 复活），不与当前主线耦合。

---

## 功能清单（按实现程度）

### ✅ 已完成（前后端可用）

| 功能 | 入口 |
|------|------|
| Docker / healthcheck | `docker-compose.yml` |
| 搜索 / 详情 / 目录 / 正文 | 搜索页 → 阅读器 |
| 书源导入（文件 + 链接合集） | 书源管理 |
| RuleExecutor 四流程 | `internal/webbook/rule_exec.go` |
| 书海 Explore API | `/explore` + Explore 页 |
| RSS 订阅 + 合集导入 | RSS 页 |
| 替换规则 CRUD + scope 运行时 | ReplaceRules 页 |
| 本地书上传 | LocalBooks 页 |
| 书架 API 同步 | Bookshelf 页 |
| 书源调试四步 | BookSourceDebug 页 |
| CI | `.github/workflows/ci.yml` |
| 结构化日志 / source_stats | `logutil` + `/bookSources/stats` |
| 数据同步 export/import | 数据同步页 `/sync` |
| 搜索 SSE 流式 | 搜索页「流式」开关 |
| 书源高级编辑 + 健康度 | 书源管理 |
| Explore 规则解析书单 | 书海页 |
| 换源 + 章节对齐 | 书架换源按钮 |

### ⚠️ 部分完成

_当前无阻塞性 partial 项。_

### 📋 可选 / 未做
| 功能 | 说明 |
|------|------|
| WebDAV sidecar | 同步包已有，WebDAV 协议未接 |
| Kindle / TTS / 定时更新 | 不在核心范围 |

### ❌ 已放弃（个人版）

| 功能 | 说明 |
|------|------|
| WebView 书源 | Playwright 独立服务，不实现 |
| 多用户 / 认证 | 不实现 |
| loginUrl 登录 UI | 不实现；Cookie 手填 |

---

## 代码目录速查

```
reader-go/
├── cmd/server/          # 入口
├── internal/
│   ├── web/             # HTTP 层
│   ├── webbook/         # 书源执行（Legado 规则 + Executor）
│   ├── rule/            # 规则引擎
│   ├── booksource/      # 书源 CRUD / 导入
│   ├── shelf/           # 书架
│   ├── rss/             # RSS
│   ├── replace/         # 替换规则
│   ├── localbook/       # 本地书解析
│   └── migrate/         # DB 迁移
├── web/src/             # React 前端（正式）
├── docs/                # 规划文档（本目录）
├── deploy/              # systemd 安装脚本
├── docker-compose.yml
├── ISSUES.md            # 历史问题（勿与 BACKLOG 双轨维护）
│
└── web/dist/            # 构建产物（改 src 后需 npm run build）
```

---

## 已知限制

1. **JS 书源**：复杂 Legado JS 仍可能失败；`@js:` URL 已支持。
2. **换源**：跨源搜索 + 章节名模糊对齐；极端异名章节可能需手动调整进度。
3. **Explore**：依赖书源 `exploreRule`；无规则时回退 `searchRule`。

---

## 下一步

可选：WebDAV sidecar、Kindle/TTS 等 — 见 [BACKLOG.md](./BACKLOG.md)
