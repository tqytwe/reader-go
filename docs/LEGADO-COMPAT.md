# Legado / Reader 能力兼容矩阵

> 返回 [文档索引](./README.md) · 路线图 [ROADMAP.md](./ROADMAP.md)

本文档跟踪 Reader Go 对 Legado 书源及 [hectorqin/reader](https://github.com/hectorqin/reader) 功能的兼容情况。

**图例**：✅ 支持 · ⚠️ 部分支持 · ❌ 不支持 · 🔜 计划阶段

## 书源规则

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| CSS / `@default:` / `@css:` | ✅ | Phase 2 | RuleExecutor 完整支持，含索引语法、child 组合子、负索引 |
| XPath `@xpath:` | ✅ | Phase 2 | ParseXPath 完整支持，含括号、链式操作、&&/||/%% |
| JSONPath `@json:` | ✅ | Phase 2 | 完整支持 $、[*]、数组索引、切片、嵌套路径 |
| Regex `@regex:` | ✅ | Phase 2 | regexp2 完整支持，含 lookbehind、分组引用、替换 |
| 组合 `&&` / `||` / `%%` | ✅ | Phase 2 | RuleAnalyzer + Executor 完整支持，括号边界正确 |
| `@put` / `@get` 变量 | ✅ | Phase 2 | parser 分词器完整支持变量绑定 |
| `{{}}` 内嵌表达式 | ✅ | Phase 2 | JsEngine.EvaluateTemplate 完整支持 |
| `$1` 正则分组 | ✅ | Phase 2 | ParseRegex 完整支持分组引用和模板 |
| `##find##replace` 净化 | ✅ | Phase 2 | ParseRegex 完整支持替换模式 |
| 字段规则 `name::规则` + `@` | ✅ | Phase 1 | 搜索/详情/目录/正文全链路支持 |
| `ruleSearch` 对象格式 | ✅ | — | F-09 已修复搜索 |
| `BookSource.Mode` 四模式 | ✅ | Phase 2 | DB 字段 + 解析时正确使用 |

## URL 模板

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| `{{keyword}}` / `{{page}}` | ✅ | — | F-10 已支持 |
| 相对路径拼接 baseUrl | ✅ | — | — |
| POST searchUrl + body | ✅ | Phase 1 | 完整支持 POST 请求和 body |
| `<page1,2,3>` 页码多选 | ✅ | Phase 1 | url_template 完整实现 |
| `, {"method":"POST",...}` 选项 | ✅ | Phase 1 | JSON 选项解析完整 |
| `@js:` searchUrl / bookUrl | ✅ | Phase 1 | url_js.go 完整实现 |
| `<js>...</js>` URL 注入 | ✅ | Phase 1 | RunEmbeddedJS 完整实现 |

## JavaScript 书源

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| goja 基础执行 | ✅ | Phase 1 | JsEngine 完整实现并接入生产 |
| `@js:` 规则字段 | ✅ | Phase 1 | 完整支持 |
| Legado `java.*` 兼容层 | ✅ | Phase 1 | ajax、getString、put 等完整实现 |
| JS 超时 `JS_TIMEOUT_MS` | ✅ | Phase 1 | 30秒默认超时 + 上下文取消 |
| ajax 域名限制 | ✅ | Phase 1 | SSRF 防护完整（私有IP/localhost/file://拒绝） |
| WebView / 浏览器书源 | ❌ | Phase 3 | 需 Playwright 独立服务（已决定不做） |

## 书源管理

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| 合集 JSON 导入 | ✅ | — | F-01/F-07/F-11 |
| 链接导入 | ✅ | — | F-06 |
| 全字段 CRUD API | ✅ | Phase 1 | 后端全字段，前端高级 Tab 完整 |
| 导入错误反馈 | ✅ | Phase 0 | ImportResult 包含 Errors + SuccessCount |
| 书源分组 / 排序 | ✅ | Phase 1 | DB group/order + UI 展示 |
| 书源调试 SSE | ✅ | Phase 1 | BookSourceDebug 四步完整 |
| Cookie / Headers 注入请求 | ✅ | Phase 1 | parseHeaders 完整实现 |
| 书源登录 loginUrl | ⚠️ | Phase 3 | DB 字段已有，前端 UI 待完善 |

## 阅读链路

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| 并发搜索 | ✅ | — | ~30s 全源 |
| 书籍详情 | ✅ | Phase 0/2 | Executor 完整支持 |
| 目录列表 | ✅ | Phase 0 | 完整支持 |
| 章节正文 | ✅ | Phase 0/2 | content 字段完整 |
| 替换规则运行时 | ✅ | Phase 2 | scope 过滤完整实现 |
| 换源 | ✅ | Phase 3 | canonicalId + 章节对齐已实现 |
| 本地书 TXT/EPUB/CBZ | ✅ | Phase 3 | parser 完整，图片导出已实现 |

## 书海 Explore

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| exploreUrl | ✅ | Phase 2 | DB 字段已有，完整实现 |
| ruleExplore | ✅ | Phase 2 | 导入映射完整 |
| 发现页 UI | ✅ | Phase 2 | Explore.tsx 完整实现 |

## RSS

| 能力项 | 当前状态 | 目标阶段 | 备注 |
|--------|----------|----------|------|
| 标准 XML RSS | ✅ | — | — |
| 合集导入 | ✅ | Phase 0 | API + UI 完整 |
| ruleArticles 等自定义规则 | ✅ | Phase 2 | ConvertToFeed 完整保存规则 |
| GetRuleArticles() | ✅ | Phase 2 | 完整实现 |
| 链接导入订阅 UI | ✅ | Phase 0 | Rss.tsx 完整接入 |

## Reader 原版功能对标

| 功能 | 当前状态 | 目标阶段 | reader 对标 |
|------|----------|----------|-------------|
| 书架 SQLite 持久化 | ✅ | Phase 0 | 前后端完整对接 |
| 阅读进度 | ✅ | Phase 0 | PUT progress 完整 |
| WebDAV 同步 | ❌ | Phase 3 | ✅ reader 有（待实现） |
| 并发搜书 SSE | ✅ | Phase 3 | ✅ reader 有 |
| 替换规则 UI | ✅ | Phase 3 | ✅ reader 有 |
| Kindle / 听书 | ❌ | — | 低优先级（待实现） |
| 漫画阅读器 | ✅ | Phase 3 | CBZ parser + 图片模式 |
| 多用户 | ❌ | — | 个人版跳过（已决定不做） |

## bookKey 格式

| 场景 | Legado / Reader | Reader Go 当前 | 目标 |
|------|-----------------|----------------|------|
| 搜索返回 | 因源而异 | `{sourceId}::{url}` | ✅ 统一 |
| JS bookUrl | JS 返回值 | `{sourceId}::{jsResult}` | ✅ 已实现 |
| 书架存储 | 同上 | API 为准 | ✅ 已统一 |

## 安全与沙箱

| 策略 | 当前 | Phase 1 目标 |
|------|------|--------------|
| JS 执行超时 | ✅ 已实现 | 30秒默认超时 + 上下文取消 |
| 文件 IO | ✅ goja 默认禁止 | 安全 |
| 网络 ajax | ✅ SSRF 防护 | 私有IP/localhost/file:// 全部拒绝 |
| 上传校验 | 部分 | MAX_FILE_SIZE + 类型白名单 |
