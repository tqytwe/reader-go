# Legado / Reader 能力兼容矩阵

> 返回 [文档索引](./README.md) · 路线图 [ROADMAP.md](./ROADMAP.md)

本文档跟踪 Reader Go 对 Legado 书源及 [hectorqin/reader](https://github.com/hectorqin/reader) 功能的兼容情况。

**图例**：✅ 支持 · ⚠️ 部分支持 · ❌ 不支持 · 🔜 计划阶段

## 书源规则

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| CSS / `@default:` / `@css:` | ⚠️ | Phase 2 | 搜索走 legado_rule；详情/目录/正文仅 goquery，未读 mode |
| XPath `@xpath:` | ⚠️ | Phase 2 | `internal/rule` 有实现，webbook 未统一调用 |
| JSONPath `@json:` | ⚠️ | Phase 2 | 搜索部分支持；详情等未接入 |
| Regex `@regex:` | ⚠️ | Phase 2 | 规则包有 regexp2；webbook 路径有限 |
| 组合 `&&` / `||` / `%%` | ❌ | Phase 2 | RuleAnalyzer 存在，Executor 未接入生产 |
| `@put` / `@get` 变量 | ❌ | Phase 2 | parser 有实现，webbook 未用 |
| `{{}}` 内嵌表达式 | ❌ | Phase 2 | — |
| `$1` 正则分组 | ❌ | Phase 2 | — |
| `##find##replace` 净化 | ❌ | Phase 2 | — |
| 字段规则 `name::规则` + `@` | ⚠️ | Phase 1 | 搜索已支持 legado_rule；info/toc/content 仍 CSS |
| `ruleSearch` 对象格式 | ✅ | — | F-09 已修复搜索 |
| `BookSource.Mode` 四模式 | ❌ | Phase 2 | DB 有字段，解析时未使用 |

## URL 模板

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| `{{keyword}}` / `{{page}}` | ✅ | — | F-10 已支持 |
| 相对路径拼接 baseUrl | ✅ | — | — |
| POST searchUrl + body | ⚠️ | Phase 1 | 部分书源可用 |
| `<page1,2,3>` 页码多选 | ❌ | Phase 1 | url_template 有代码，webbook 未全接 |
| `, {"method":"POST",...}` 选项 | ⚠️ | Phase 1 | — |
| `@js:` searchUrl / bookUrl | ❌ | Phase 1 | buildSearchURL 直接报错 |
| `<js>...</js>` URL 注入 | ❌ | Phase 1 | hasUnsupportedJS 阻断搜索 |

## JavaScript 书源

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| goja 基础执行 | ⚠️ | Phase 1 | JsEngine 存在，生产未接 |
| `@js:` 规则字段 | ❌ | Phase 1 | 1.2b |
| Legado `java.*` 兼容层 | ❌ | Phase 1 | 1.2c：ajax、getString 等优先级表 |
| JS 超时 `JS_TIMEOUT_MS` | ❌ | Phase 1 | compose 环境变量 |
| ajax 域名限制 | ❌ | Phase 1 | 建议限制在 baseUrl 子域 |
| WebView / 浏览器书源 | ❌ | Phase 3 | 需 Playwright 独立服务 |

## 书源管理

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| 合集 JSON 导入 | ✅ | — | F-01/F-07/F-11 |
| 链接导入 | ✅ | — | F-06 |
| 全字段 CRUD API | ⚠️ | Phase 1 | 后端有字段，前端仅 8 字段 |
| 导入错误反馈 | ❌ | Phase 0 | Import 静默 continue |
| 书源分组 / 排序 | ⚠️ | Phase 1 | DB 有 group/order，UI 简易 |
| 书源调试 SSE | ⚠️ | Phase 1 | BookSourceDebug 扩展四步测试 |
| Cookie / Headers 注入请求 | ⚠️ | Phase 1 | parseHeaders 曾空实现 |
| 书源登录 loginUrl | ❌ | Phase 3 | 依赖 JS 1.2c |

## 阅读链路

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| 并发搜索 | ✅ | — | ~30s 全源 |
| 书籍详情 | ⚠️ | Phase 0/2 | 前端契约 + Executor |
| 目录列表 | ⚠️ | Phase 0 | useReader 二次解包问题 |
| 章节正文 | ⚠️ | Phase 0/2 | content 字段名 |
| 替换规则运行时 | ❌ | Phase 2 | 忽略 scope |
| 换源 | ❌ | Phase 3 | canonicalId + 章节对齐 |
| 本地书 TXT/EPUB/CBZ | ⚠️ | Phase 3 | parser 有，UI 有限 |

## 书海 Explore

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| exploreUrl | ❌ | Phase 2 | 无 DB 字段 |
| ruleExplore | ❌ | Phase 2 | 导入未映射 |
| 发现页 UI | ❌ | Phase 2 | Explore.tsx 待建 |

## RSS

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| 标准 XML RSS | ✅ | — | — |
| 合集导入 | ⚠️ | Phase 0 | API 有，UI 部分 |
| ruleArticles 等自定义规则 | ❌ | Phase 2 | ConvertToFeed 丢弃 |
| GetRuleArticles() | ❌ | Phase 2 | 恒空 |
| 链接导入订阅 UI | ❌ | Phase 0 | Rss.tsx 未接 |

## Reader 原版功能对标

| 功能 | 当前状态 | 目标阶段 | reader 对标 |
|------|----------|----------|-------------|
| 书架 SQLite 持久化 | ⚠️ | Phase 0 | 前端未全接 API |
| 阅读进度 | ⚠️ | Phase 0 | PUT progress 待暴露 |
| WebDAV 同步 | ❌ | Phase 3 | ✅ reader 有 |
| 并发搜书 SSE | ❌ | Phase 3 | ✅ reader 有 |
| 替换规则 UI | ❌ | Phase 3 | ✅ reader 有 |
| Kindle / 听书 | ❌ | — | 低优先级 |
| 漫画阅读器 | ⚠️ | Phase 3 | CBZ parser 已有 |
| 多用户 | ❌ | — | 个人版跳过 |

## bookKey 格式

| 场景 | Legado / Reader | Reader Go 当前 | 目标 |
|------|-----------------|----------------|------|
| 搜索返回 | 因源而异 | `{sourceId}::{url}` | 统一 |
| JS bookUrl | JS 返回值 | ❌ 报错 | `{sourceId}::{jsResult}` |
| 书架存储 | 同上 | 混用 Zustand | API 为准 |

## 安全与沙箱

| 策略 | 当前 | Phase 1 目标 |
|------|------|--------------|
| JS 执行超时 | 无 | `JS_TIMEOUT_MS` 默认 5000 |
| 文件 IO | goja 默认禁止 | 文档化 + 测试 |
| 网络 ajax | 未限制 | 建议 baseUrl 同源/子域 |
| 上传校验 | 部分 | MAX_FILE_SIZE + 类型白名单 |
