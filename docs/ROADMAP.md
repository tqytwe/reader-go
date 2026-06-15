# Reader Go 总路线图

> 返回 [文档索引](./README.md)

## 愿景

Docker 一键部署的个人阅读服务，Legado 书源兼容率分阶段提升，核心阅读链路（搜索 → 详情 → 目录 → 正文 → 书架进度）稳定可用。

## 四阶段里程碑

时间盒为**实现复杂度估算**，非日历承诺。

| 阶段 | 周期估算 | 目标 | 关键交付 |
|------|----------|------|----------|
| [Phase 0](./phases/phase-0-foundation.md) | 3–5 天 | 可稳定 Docker + 核心链路可用 | healthcheck、阅读器/书架 API 接线 |
| [Phase 1](./phases/phase-1-short-term.md) | 2–3 周 | 书源可维护 + JS URL/基础 JS 规则 | DTO、高级表单、`@js:` URL |
| [Phase 2](./phases/phase-2-medium-term.md) | 3–4 周 | 发现页 + 统一规则引擎 + RSS 抓取 | `RuleExecutor`、Explore API、RSS rules 列 |
| [Phase 3](./phases/phase-3-long-term.md) | 持续 | 原版高级能力 | 换源、WebDAV、并发搜书 SSE 等 |

## 默认落地路径

```mermaid
flowchart LR
  P0[Phase0 Docker+契约]
  P1A[Phase1 字段对齐]
  P1B[Phase1 JS书源]
  P2A[Phase2 RuleExecutor]
  P2B[Phase2 Explore+RSS]
  P3[Phase3 长期池]
  P0 --> P1A --> P1B --> P2A
  P2A --> P2B --> P3
```

## 不在范围

- 宿主机 systemd、独立 nginx 运维文档（仅 Docker 内 nginx 或 compose 双服务）
- WebView 书源 Playwright 服务（Phase 3 低优先级）
- 多用户 / 租户（个人版可跳过）

## 横切能力（贯穿各 Phase）

以下优化写入各阶段文档，可在 Phase 0/1 并行启动：

### 可观测性

- 结构化日志：`sourceId`、`ruleMode`、`latency`
- 可选 `/metrics`（Prometheus）

### 书源健康度

- `source_stats` 表：`last_success`、`error_rate`
- 书源管理页展示成功率

### 搜索体验

- 单源超时跳过、全局 30s 上限可配置
- 结果去重（书名 + 作者）

### 缓存层

- Explore / 书详情短期内存缓存（按 `sourceId + url` hash）

### 安全

- CORS 白名单（compose 环境变量）
- 上传大小与类型校验
- JS 沙箱策略（见 [LEGADO-COMPAT.md](./LEGADO-COMPAT.md)）

### 质量门禁

- CI：`go test ./...` + `web npm run build`
- Docker build 作为 release gate

### 文档

- 单一 [API.md](./API.md)（待建）：从 handlers 生成或手维，修正 `keyword`→`q`、`bookKey` 格式说明

## 相关文档

| 文档 | 用途 |
|------|------|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 目标架构与模块边界 |
| [LEGADO-COMPAT.md](./LEGADO-COMPAT.md) | 能力兼容矩阵 |
| [DOCKER.md](./DOCKER.md) | 部署拓扑与修复清单 |
| [BACKLOG.md](./BACKLOG.md) | 可追踪任务列表 |

## 人力估算（单人全职参考）

| 阶段 | 后端 | 前端 | 联调/测试 |
|------|------|------|-----------|
| Phase 0 | 1–2 天 | 1–2 天 | 1 天 |
| Phase 1 | 1–1.5 周 | 0.5–1 周 | 2–3 天 |
| Phase 2 | 2–2.5 周 | 1–1.5 周 | 3–4 天 |
| Phase 3 | 按功能池逐项 | 按功能池逐项 | — |
