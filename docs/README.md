# Reader Go 开发规划文档

本目录是 Reader Go 的**规划与状态文档**。编码时按 Phase 顺序推进；**当前做到哪一步，先看 [STATUS.md](./STATUS.md)**。

## 快速导航

| 我想… | 看这份 |
|--------|--------|
| **知道哪些功能做完了** | **[STATUS.md](./STATUS.md)** ← 首选 |
| 查任务编号 T-xxx | [BACKLOG.md](./BACKLOG.md) |
| 理解阶段顺序 | [ROADMAP.md](./ROADMAP.md) |
| 某 Phase 详细方案 | [phases/phase-0-foundation.md](./phases/phase-0-foundation.md) 等 |
| 接口联调 | [API.md](./API.md) |
| 架构与模块 | [ARCHITECTURE.md](./ARCHITECTURE.md) |
| Legado 兼容对照 | [LEGADO-COMPAT.md](./LEGADO-COMPAT.md) |
| Docker 部署 | [DOCKER.md](./DOCKER.md) |
| 历史 Bug 修复记录 | [ISSUES.md](../ISSUES.md) |

## 阅读顺序（新人）

1. [STATUS.md](./STATUS.md) — 现状
2. [ROADMAP.md](./ROADMAP.md) — 路线图
3. [ARCHITECTURE.md](./ARCHITECTURE.md) — 架构
4. 按需查阅 phases / API / DOCKER

## 愿景

**Docker 一键部署的个人阅读服务**，分阶段提升 Legado 书源兼容率，对齐 [hectorqin/reader](https://github.com/hectorqin/reader) 核心体验。

## 当前架构（2026-05-30）

- **`internal/rule`**：`RuleExecutor` 已接入 webbook 四流程（search/info/toc/content）。
- **`internal/webbook`**：Legado 规则解析 + Executor 回退；`parseRuleToSelectors` 已删除。
- **Phase 0～2**：Backlog 任务均已 **done**；Phase 3 后端 API 多项已就绪，前端 UI 见 STATUS「部分完成」。

## 不在范围

- 宿主机 systemd / nginx 运维（后期以 Docker 为主）
- WebView 书源、多用户（低优先级，见 BACKLOG superseded）
