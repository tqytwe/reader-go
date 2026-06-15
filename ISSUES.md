# Reader Go 问题清单

> **规划与待办已迁移至 [`docs/`](docs/README.md)**  
> - **现状**：[`docs/STATUS.md`](docs/STATUS.md)  
> - **任务**：[`docs/BACKLOG.md`](docs/BACKLOG.md)  
> - **路线**：[`docs/ROADMAP.md`](docs/ROADMAP.md)  
>
> **本文件仅保留历史修复（F-01～F-12）**。下方「待解决」章节写于同一轮排查，**多数已在后续 Phase 中修复**；请以 STATUS/BACKLOG 为准，勿按本节逐项开工。

> 检查时间：2026-05-30  
> 对照项目：[hectorqin/reader](https://github.com/hectorqin/reader) / [hectorqin/reader-legado](https://github.com/hectorqin/reader-legado)

---

## 已修复（本次）

| ID | 严重度 | 问题 | 根因 | 修复 |
|----|--------|------|------|------|
| F-01 | **P0** | 书源/合集导入失败，数据库始终为空 | `book_sources` INSERT 语句 23 列却有 24 个 `?` 占位符，报错 `24 values for 23 columns` | 修正 `internal/booksource/service.go` Create/Import SQL |
| F-02 | **P0** | 书源列表/RSS 列表前端不显示 | ① 后端空 slice 序列化为 `data:null`；② 前端 axios 未解包 `{code,message,data}` 信封 | `okList` 返回 `[]`；`web/src/api/client.ts` 增加 `unwrap()` |
| F-03 | **P0** | RSS 添加后列表接口 500 | `AddFeed` 写入 `time.Time` 字符串，但 `ListFeeds` 按 `int64` 读取 | `internal/rss/service.go` 统一使用 Unix 时间戳 |
| F-04 | **P0** | RSS 手动添加订阅失败 | 前端 POST `{url}`，后端只认 `{feedUrl}` | 后端兼容 `url`/`feedUrl`；前端改为 `feedUrl` |
| F-05 | **P1** | 服务端口与 nginx/docker 不一致 | 进程跑在 8080，nginx/docker 期望 6464 | 提供 `deploy/reader-go-api.service`（PORT=6464） |
| F-06 | **P1** | 源码缺少「链接导入书源」UI | `web/dist` 有该功能但 `web/src` 被回退 | 恢复 `BookSourceManage.tsx` 链接导入模态框 |

| F-07 | **P0** | Legado 书源导入缺 searchUrl/ruleSearch | LegacyBookSource 未映射 searchUrl；ruleSearch 对象被丢弃 | 扩展 `import.go`，JSON 对象完整存储 |
| F-08 | **P0** | 导入后 WebBook 不热加载 | init 时加载，导入后不刷新 | `reloadWebBookSources()` + CRUD/导入后调用 |
| F-09 | **P0** | 搜索忽略 SearchRule | searchWithSource 用空解析器 | 新增 `legado_rule.go`，Legado HTML/JSON 规则解析 |
| F-10 | **P1** | buildSearchURL 不支持 {{key}}/POST | 仅支持 {key} GET | 支持 Legado `{{key}}`、相对路径、POST body |
| F-11 | **P1** | 合集导入为增量，旧脏数据残留 | 旧导入缺字段的数据仍在库中 | 合集导入改为全量替换（DeleteAll） |
| F-12 | **P1** | 搜索页 bookKey 未映射 | 前端未读 API 的 bookKey 字段 | 修复 `Search.tsx` |

**验证结果（第二轮）：**
- 26/26 书源均含 searchUrl
- `GET /api/search?q=斗破苍穹` 返回多源搜索结果（约 30s 并发搜书）
- JS 书源（如 `@js:` bookUrl）仍无法获取完整 bookKey，属已知限制

---

## ~~待解决~~ 历史记录 — 部署/运维（P0-P1）

> ⚠️ 过期描述，见 [docs/STATUS.md](docs/STATUS.md)

| ID | 严重度 | 问题 | 说明 |
|----|--------|------|------|
| D-01 | **P0** | 无 systemd 服务或未安装 | 项目原先无 unit 文件；已添加 `deploy/reader-go-api.service` + `deploy/install-service.sh`，需 `sudo bash deploy/install-service.sh` |
| D-02 | **P0** | nginx 未配置/未监听前端端口 | `nginx-server.conf` 监听 6465，但当前 nginx 未加载该配置；6465 不可访问 |
| D-03 | **P1** | 服务器未安装 Go | 只能使用预编译 `reader-go-api`；源码修改后需在有 Go+CGO 环境重新编译 |
| D-04 | **P1** | 默认端口文档混乱 | README 写 8080/6464 混用；docker-compose 用 6464，main.go 默认 8080 |
| D-05 | **P2** | 前端 dist 与 src 不同步 | 修改 `web/src` 后需 `npm run build` 更新 `web/dist` |

---

## ~~待解决~~ 历史记录 — 书源/规则引擎（P1-P2）

> ⚠️ 过期描述，见 [docs/STATUS.md](docs/STATUS.md)

| ID | 严重度 | 问题 | 文件 |
|----|--------|------|------|
| R-01 | **P1** | `parseHeaders()` 空实现，书源自定义请求头无效 | `internal/web/server.go:118-124` |
| R-02 | **P1** | 书源导入后未热加载到 WebBook | 导入成功但 `app.WebBook` 不刷新，需重启服务才能搜索 | `internal/web/server.go` init |
| R-03 | **P1** | 旧版书源格式转换不完整 | 缺少 `searchUrl`、`bookInfoUrl`、`contentUrl` 等 URL 模板字段映射 | `internal/booksource/import.go` |
| R-04 | **P1** | JS 书源引擎不完整 | 原版 legado 支持完整 JS 规则；goja 引擎功能有限 | `internal/rule/js_engine.go` |
| R-05 | **P2** | 不支持书源登录 | 原版 README 明确不支持；复刻同样缺失 | — |
| R-06 | **P2** | 不支持 webview 书源 | 需额外部署接口 | — |
| R-07 | **P2** | `Import()` 静默跳过失败行 | SQL 错误被 `continue` 吞掉，无错误反馈 | `internal/booksource/service.go:284` |

---

## ~~待解决~~ 历史记录 — RSS/订阅（P1-P2）

> ⚠️ 过期描述，见 [docs/STATUS.md](docs/STATUS.md)

| ID | 严重度 | 问题 | 文件 |
|----|--------|------|------|
| S-01 | **P1** | RSS 页面无「链接导入订阅源合集」UI | API 已有 `/api/rss/import/collection`，前端 Rss.tsx 未接入 | `web/src/pages/Rss.tsx` |
| S-02 | **P1** | 自定义 RSS 规则未存储 | `ConvertToFeed` 丢弃 `ruleArticles` 等 legado 规则字段 | `internal/rss/import.go` |
| S-03 | **P2** | RSS 页面英文 UI | 与其他页面中文不一致 | `web/src/pages/Rss.tsx` |
| S-04 | **P2** | `GetRuleArticles()` 永远返回空 | 占位方法未实现 | `internal/rss/import.go:64` |

---

### ~~待解决~~ 历史记录 — 规则引擎（P0，核心能力）

> ⚠️ 过期描述，E-01～E-05 已在 Phase 2 完成

| ID | 严重度 | 问题 | 说明 |
|----|--------|------|------|
| E-01 | **P0** | `internal/rule` 与 `internal/webbook` 未打通 | 规则包有 XPath/JSONPath/JS，但 webbook 只用 goquery CSS |
| E-02 | **P0** | 搜索忽略 `SearchRule` | `searchWithSource` 使用空规则硬编码选择器 |
| E-03 | **P0** | Legado 规则格式不兼容 | 导入用 `author::规则`+`@`，解析器要 `author:规则`+`&&` |
| E-04 | **P1** | 书源 Cookie/Headers 未注入 HTTP 请求 | 导致大量书源无法访问 |
| E-05 | **P1** | 无书海/发现(Explore)功能 | 原版 legado 核心功能缺失 |

---

## ~~待解决~~ 历史记录 — 前端/API（P1-P2）

> ⚠️ 过期描述，见 [docs/STATUS.md](docs/STATUS.md)

| ID | 严重度 | 问题 | 文件 |
|----|--------|------|------|
| A-01 | **P1** | 书源管理字段与后端不一致 | 前端用 `mode`/`headers`(object)，后端用 `searchMode`/`headers`(JSON string) | `BookSourceManage.tsx` vs `booksource/model.go` |
| A-02 | **P1** | 搜索结果显示字段映射不完整 | 后端返回 `name`/`bookKey`，前端期望 `bookName`/`bookId`（已有 fallback 但不完整） | `Search.tsx` |
| A-03 | **P1** | README 搜索参数写 `keyword`，实际 API 用 `q` | 文档错误 | `README.md:180` |
| A-04 | **P2** | 缺少大量原版功能页面 | 无书海、换源、WebDAV 同步、定时更新、并发搜书、Kindle、漫画、听书等 | 对照 [reader](https://github.com/hectorqin/reader) |
| A-05 | **P2** | Gin 运行 debug 模式 | 生产应设 `GIN_MODE=release` | `internal/web/server.go` |

---

## ~~待解决~~ 历史记录 — 数据/架构（P2）

> ⚠️ 过期描述，X-01 migrate 包已拆分 RSS DDL

| ID | 严重度 | 问题 | 说明 |
|----|--------|------|------|
| X-01 | **P2** | `booksource.Service.Init()` 创建 RSS 表 | 书源服务不应管理 RSS schema，职责混乱 | `internal/booksource/service.go:56-91` |
| X-02 | **P2** | SQLite 单连接池 | `SetMaxOpenConns(1)` 限制并发 | `internal/web/server.go:43` |
| X-03 | **P2** | 无用户认证/多用户 | 原版为个人服务器版；复刻同样无认证 | — |

---

## 服务部署指南（非 Docker）

```bash
# 1. 编译（需 Go 1.22+ 且 CGO_ENABLED=1）
cd /data/1panel/rida/reader-go
CGO_ENABLED=1 go build -o reader-go-api ./cmd/server/

# 2. 安装 systemd 服务
sudo bash deploy/install-service.sh

# 3. 配置 nginx（前端静态 + API 反代）
sudo cp nginx-server.conf /etc/nginx/sites-available/reader-go.conf
sudo ln -sf /etc/nginx/sites-available/reader-go.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

# 4. 构建前端（修改 src 后）
cd web && npm install && npm run build
```

访问：`http://服务器IP:6465`（前端） → API 反代到 `127.0.0.1:6464`

---

## 调试日志

本次修复在以下位置保留了调试埋点（验证通过后可移除）：
- `internal/web/handlers.go` — `importBookSourceCollection`、`listBookSources`

日志文件：`.cursor/debug-9e907b.log`
