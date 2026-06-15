# Reader Go

> 用 Go + React 复刻「阅读」App，一个开源的开源小说阅读器后端服务。

## 项目简介

**Reader Go** 是一个基于 Go 和 React 的开源小说阅读器项目，复刻了 Android 端「阅读」App 的核心功能。它提供完整的 Web 后端 API 和现代化的 Web 前端界面，支持网络书源解析、本地书籍解析、书源管理、书架管理等功能。

📋 **[项目状态](docs/STATUS.md)** — 哪些做完、哪些进行中 · **[开发规划](docs/ROADMAP.md)** — 分阶段路线图与 Backlog（[`docs/`](docs/README.md)）

- 📖 **网络书源**：支持 XPath / CSS / JSONPath / Regex 规则解析，完成搜索、发现、详情、目录、正文全流程
- 📚 **本地书籍**：支持 TXT / EPUB / CBZ 格式解析
- 🛠 **书源管理**：增删改查、导入导出
- 📋 **书架管理**：分组、排序、阅读进度追踪
- 🔄 **替换规则**：正则替换，净化阅读内容

## 技术栈

| 层级 | 技术 |
|------|------|
| **后端** | Go 1.22 + Gin + SQLite |
| **前端** | React 18 + TypeScript + Vite + Ant Design |
| **规则引擎** | goquery / antchfx/xmlquery / gjson / regexp2 |
| **本地解析** | EPUB / CBZ / PDF 原生解析 |

### 后端依赖

```
github.com/gin-gonic/gin          # Web 框架
github.com/mattn/go-sqlite3       # 数据库
github.com/PuerkitoBio/goquery    # HTML/CSS 解析
github.com/antchfx/xmlquery       # XPath 解析
github.com/tidwall/gjson          # JSONPath 解析
github.com/dlclark/regexp2        # 正则（支持 lookbehind）
github.com/pdfcpu/pdfcpu          # PDF 处理
```

### 前端依赖

```
antd                              # UI 组件库
axios                             # HTTP 客户端
react-router-dom                  # 路由
zustand                           # 状态管理
@monaco-editor/react              # 代码编辑器
vitest                            # 测试框架
tailwindcss                       # 样式
```

## 快速开始

### 环境要求

- **Go** 1.22+
- **Node.js** 18+
- **npm** 8+

### 一键启动（推荐）

```bash
./dev-start.sh
```

这会同时启动后端和前端，按 `Ctrl+C` 停止所有服务。

### 分别启动

**终端 1 - 启动后端：**
```bash
./start-backend.sh
```

**终端 2 - 启动前端：**
```bash
./start-frontend.sh
```

### 手动启动

#### 1. 克隆项目

```bash
git clone https://github.com/your-org/reader-go.git
cd reader-go
```

#### 2. 后端启动

```bash
# 安装依赖
go mod download

# 启动服务（默认端口 6464）
CGO_ENABLED=1 go run cmd/server/main.go

# 或指定端口
PORT=8080 CGO_ENABLED=1 go run cmd/server/main.go

# 或指定数据库路径
DATABASE_URL=/path/to/reader.db CGO_ENABLED=1 go run cmd/server/main.go
```

#### 3. 前端启动

```bash
cd web
npm install
npm run dev
```

### 访问地址

启动后访问：

- **前端界面**: http://localhost:3000
- **后端 API**: http://localhost:6464
- **健康检查**: http://localhost:6464/health

### 生产环境 Nginx 配置

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:80;
    }

    location /api {
        proxy_pass http://localhost:8080;
    }

    # SSE（书源调试）
    location /api/bookSources/debug {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Connection '';
        chunked_transfer_encoding on;
    }
}
```

## API 文档

### 健康检查

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/health` | 服务健康检查 |

### 书源 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/bookSources` | 获取书源列表 |
| POST | `/api/bookSources` | 创建书源 |
| PUT | `/api/bookSources/:id` | 更新书源 |
| DELETE | `/api/bookSources/:id` | 删除书源 |
| POST | `/api/bookSources/import` | 批量导入书源 |

**书源模型：**

```json
{
  "id": 1,
  "name": "示例书源",
  "baseUrl": "https://example.com",
  "searchUrl": "/search?keyword={{keyword}}&page={{page}}",
  "bookInfoUrl": "/book/{{bookKey}}",
  "tocUrl": "/toc/{{bookKey}}",
  "contentUrl": "/content/{{bookKey}}/{{chapterIndex}}",
  "searchRule": "@XPath://div[@class='book-item']",
  "bookInfoRule": "@XPath://div[@class='book-detail']",
  "tocRule": "@XPath://ul[@class='chapter-list']/li",
  "contentRule": "@XPath://div[@class='content']",
  "searchMode": "xpath",
  "bookInfoMode": "xpath",
  "tocMode": "xpath",
  "contentMode": "xpath",
  "userAgent": "Mozilla/5.0",
  "headers": "{\"Accept\":\"text/html\"}",
  "cookie": "",
  "timeout": 10,
  "enabled": true,
  "group": "默认",
  "order": 0
}
```

### 搜索 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/search` | 全网搜索书籍 |

**查询参数：**

| 参数 | 类型 | 描述 |
|------|------|------|
| `q` | string | 搜索关键词（必填） |
| `page` | number | 页码，默认 1 |
| `sourceId` | number | 指定书源 ID，不传则搜索全部 |

**响应示例：**

```json
{
  "results": [
    {
      "name": "书名",
      "author": "作者",
      "bookKey": "sourceId:bookName",
      "sourceId": 1,
      "sourceName": "书源名称",
      "coverUrl": "https://...",
      "summary": "简介..."
    }
  ]
}
```

### 书籍 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/book/info` | 获取书籍详情 |
| GET | `/api/book/toc` | 获取目录列表 |
| GET | `/api/book/content` | 获取章节正文 |

**查询参数：**

| 参数 | 类型 | 描述 |
|------|------|------|
| `bookKey` | string | 书籍唯一标识（必填） |
| `sourceId` | number | 书源 ID（必填） |
| `chapterIndex` | number | 章节索引（content 接口必填） |

### 书架 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/shelf` | 获取书架列表 |
| POST | `/api/shelf` | 加入书架 |
| PUT | `/api/shelf/:id` | 更新书架书籍（进度/笔记） |
| DELETE | `/api/shelf/:id` | 从书架移除 |

**书架模型：**

```json
{
  "id": 1,
  "bookKey": "1:书名",
  "name": "书名",
  "author": "作者",
  "coverUrl": "https://...",
  "summary": "简介",
  "sourceId": 1,
  "sourceName": "书源名称",
  "currentChapter": "第一章",
  "lastReadAt": "2024-01-01T12:00:00Z",
  "readCount": 5,
  "note": "笔记内容",
  "order": 0
}
```

### 替换规则 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/replaceRules` | 获取替换规则列表 |
| POST | `/api/replaceRules` | 创建替换规则 |
| PUT | `/api/replaceRules/:id` | 更新替换规则 |
| DELETE | `/api/replaceRules/:id` | 删除替换规则 |

**替换规则模型：**

```json
{
  "id": 1,
  "name": "净化广告",
  "pattern": "\\[.*?\\]",
  "replacement": "",
  "scope": "content",
  "caseInsensitive": false,
  "enabled": true,
  "order": 0
}
```

**作用范围：** `all` | `title` | `content` | `toc` | `search`

## 规则语法说明

Reader Go 的规则引擎支持多种解析模式和丰富的内嵌语法，灵活应对各种网页结构。

### 解析模式

规则字符串以 `@模式:` 开头指定解析方式：

| 模式 | 前缀 | 说明 |
|------|------|------|
| **default / css** | `@default:` 或 `@css:` | JSoup CSS 选择器 |
| **xpath** | `@xpath:` | XPath 1.0 表达式 |
| **jsonpath** | `@jsonpath:` | JSONPath 表达式 |
| **regex** | `@regex:` | 正则表达式（AllInOne 模式） |
| **js** | `@js:` | JavaScript 代码 |

**示例：**

```
@xpath://div[@class='book-list']/div
@css:.book-item
@jsonpath:$..books[*]
@regex:书名：(.+?)\n作者：(.+?)
```

### 组合操作符

多个规则片段可以通过操作符组合，实现复杂的解析逻辑：

| 操作符 | 含义 | 示例 |
|--------|------|------|
| `&&` | **交集** — 取两个规则结果的交集 | `@xpath://div && @xpath://div[@class='active']` |
| `||` | **并集** — 取两个规则结果的并集 | `@xpath://h1 || @xpath://h2` |
| `%%` | **交叉合并** — 交叉合并两个结果集 | `@xpath://td[1] %% @xpath://td[2]` |

**示例：**

```
# 搜索书名和作者交叉合并
@xpath://div[@class='name'] %% @xpath://div[@class='author']

# 并集匹配 h1 或 h2 标题
@xpath://h1 || @xpath://h2

# 交集筛选特定元素
@xpath://div[@class='book'] && @xpath://div[@data-status='online']
```

### 内嵌语法

#### @put / @get — 变量绑定

```
@put:title{@xpath://h1}          # 绑定变量 title
@put:author{@xpath://span[@class='author']}
书名：{{title}}  作者：{{author}}
```

- `@put:key{...}` — 将规则结果绑定到变量 `key`
- `@get:key` — 引用之前绑定的变量

#### {{}} — 内嵌 JS 表达式

```
@put:length{{len .}}              # 获取结果数量
{{strings.ToUpper .}}             # 字符串转大写
{{strings.Replace . "旧" "新" -1}}
{{printf "%d章" .}}               # 格式化输出
```

#### $1 / $2 — 正则分组引用

```
@regex:第(.+?)章(.+?)             # 匹配 "第一章 开篇"
结果：$1 - $2                      # 输出 "一 - 开篇"
```

#### ##find##replace — 正则替换

```
@xpath://div[@class='content']    ##广告##  ##\d{4}-\d{2}-\d{2}##
```

### 完整示例

```
# 搜索规则：提取书名、作者、封面，交叉合并
@put:name{@xpath://div[@class='name']/text()} %%
@put:author{@xpath://div[@class='author']/text()} %%
@put:cover{@xpath://img[@class='cover']/@src}

# 书籍信息规则
@put:title{@xpath://h1[@class='title']}
@put:author{@xpath://span[@class='author']}
@put:summary{@xpath://div[@class='summary']/text()}
书名：{{title}}  作者：{{author}}

# 目录规则
@xpath://ul[@class='chapters']/li/a

# 正文规则（含净化）
@xpath://div[@class='content']/p/text()
  ##广告文字##  ##\s{2,}##\n
```

### URL 模板语法

书源的 URL 字段支持模板语法：

| 语法 | 说明 | 示例 |
|------|------|------|
| `{{key}}` | 参数占位符 | `/search?kw={{keyword}}` |
| `{{page}}` | 页码占位符 | `/list?page={{page}}` |
| `<js>...</js>` | JS 注入 | `<js>localStorage.getItem('token')</js>` |
| `<page1,2,3>` | 页数多选 | 第1页用1，第2页用2，之后用3 |
| `, {...}` | URL 选项 | `, {"method":"POST","headers":{...},"retry":3}` |

**示例：**

```
/search?keyword={{keyword}}&page={{page}}<page1,2,3>
, {"method":"POST","headers":{"User-Agent":"Reader"},"retry":3}
```

## 项目结构

```
reader-go/
├── cmd/
│   └── server/
│       └── main.go              # 后端入口
├── internal/
│   ├── booksource/              # 书源模块
│   │   ├── model.go             # 书源数据模型
│   │   └── service.go           # 书源业务逻辑
│   ├── localbook/               # 本地书籍解析
│   │   ├── cbz/                 # CBZ 漫画解析
│   │   │   ├── parser.go
│   │   │   └── comicinfo.go
│   │   ├── epub/                # EPUB 电子书解析
│   │   │   ├── parser.go
│   │   │   ├── types.go
│   │   │   ├── content.go
│   │   │   └── toc.go
│   │   └── txt/                 # TXT 文本解析
│   │       ├── parser.go
│   │       ├── encoding.go      # 编码检测
│   │       └── toc_rules.go     # 目录规则
│   ├── replace/                 # 替换规则模块
│   │   ├── model.go
│   │   └── service.go
│   ├── rule/                    # 规则解析引擎
│   │   ├── types.go             # 类型定义（RuleMode, RuleSegment 等）
│   │   ├── analyzer.go          # 规则切分器（&& || %%）
│   │   ├── parser.go            # 内嵌语法解析器
│   │   ├── jsoup.go             # JSoup CSS 解析
│   │   ├── xpath.go             # XPath 解析
│   │   ├── jsonpath.go          # JSONPath 解析
│   │   ├── regex.go             # 正则解析
│   │   ├── js_engine.go         # JS 引擎
│   │   └── url_template.go      # URL 模板解析
│   ├── shelf/                   # 书架模块
│   │   ├── model.go
│   │   └── service.go
│   ├── web/                     # Web API
│   │   ├── server.go            # Gin 服务器
│   │   ├── handlers.go          # 路由处理器
│   │   └── middleware/
│   │       └── cors.go          # CORS 中间件
│   └── webbook/                 # 网络书籍获取
│       ├── webbook.go           # 入口
│       ├── book_list.go         # 搜索结果
│       ├── book_info.go         # 书籍详情
│       ├── chapter_list.go      # 目录
│       ├── book_content.go      # 正文
│       └── concurrent.go        # 并发控制
├── data/
│   └── reader.db                # SQLite 数据库（运行时生成）
├── web/                         # 前端
│   ├── src/
│   │   ├── api/
│   │   │   └── client.ts        # Axios 实例
│   │   ├── components/
│   │   │   └── Layout.tsx       # 布局组件
│   │   ├── pages/
│   │   │   ├── Home.tsx         # 首页
│   │   │   ├── Bookshelf.tsx    # 书架
│   │   │   ├── Search.tsx       # 搜索
│   │   │   ├── Reader.tsx       # 阅读器
│   │   │   ├── BookSourceManage.tsx  # 书源管理
│   │   │   └── types.ts         # 类型定义
│   │   ├── store/
│   │   │   └── useStore.ts      # Zustand 状态管理
│   │   ├── router.tsx           # React Router 配置
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── tailwind.config.js
│   └── postcss.config.js
├── go.mod
├── go.sum
└── README.md
```

## 开发计划

详细路线图、阶段任务与验收标准见 **[docs/ROADMAP.md](docs/ROADMAP.md)** 与 **[docs/BACKLOG.md](docs/BACKLOG.md)**。

### 已完成

- [x] 基础 Web 框架（Gin + SQLite）
- [x] 书源 CRUD + 导入导出
- [x] 搜索 API（全网并发搜索）
- [x] 书籍详情 / 目录 / 正文 API
- [x] 书架管理（增删改 + 进度追踪）
- [x] 替换规则管理
- [x] 规则引擎（XPath / CSS / JSONPath / Regex）
- [x] 规则组合操作符（&& || %%）
- [x] 内嵌语法（@put/@get / {{}} / $1 / ##）
- [x] URL 模板解析（占位符 / JS 注入 / 页数多选 / 请求选项）
- [x] 本地书籍解析（TXT / EPUB / CBZ）
- [x] 前端基础框架（React + TypeScript + Vite + Ant Design）
- [x] 前端页面（首页 / 书架 / 搜索 / 阅读器 / 书源管理）
- [x] 单元测试（规则引擎 / 本地解析）

### 计划中（摘要）

- Phase 0：Docker 稳定 + 阅读器/书架 API 契约（见 [phase-0](docs/phases/phase-0-foundation.md)）
- Phase 1：书源全字段对齐 + JS 书源
- Phase 2：RuleExecutor + 书海 Explore + RSS 自定义规则
- Phase 3：换源、WebDAV、SSE 搜书等（见 [phase-3](docs/phases/phase-3-long-term.md)）

<details>
<summary>原 ISSUES 时代计划项（部分已纳入 docs）</summary>

- [ ] 本地书籍导入（拖拽上传）
- [ ] 阅读器功能（字体/主题/进度保存）
- [ ] 书源发现/推荐
- [ ] 阅读历史记录
- [ ] 笔记/高亮功能
- [ ] 书源版本管理
- [ ] 定时更新书架
- [ ] 多用户/权限系统
- [ ] WebSocket 实时推送
- [x] Docker Compose 一键部署

</details>

---

## License

MIT
