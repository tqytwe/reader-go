# Reader Go - 项目规范

> **开发规划见 [`docs/`](docs/README.md)**（[ROADMAP.md](docs/ROADMAP.md) · [BACKLOG.md](docs/BACKLOG.md) · [DOCKER.md](docs/DOCKER.md)）

## 项目概述

Reader Go 是一个基于 Go + React 的开源小说阅读器，复刻 hectorqin/reader 功能。

## 技术栈

| 组件 | 技术 | 版本要求 |
|------|------|----------|
| **后端** | Go | 1.22+ |
| **Web 框架** | Gin | v1.9+ |
| **数据库** | SQLite | - |
| **HTML 解析** | goquery | v1.8+ |
| **XPath** | antchfx/xmlquery | v1.3+ |
| **JSON 解析** | gjson | v1.17+ |
| **正则** | regexp2 | v1.11+ (支持 lookbehind) |
| **JS 引擎** | goja | - |
| **前端** | React | 18.x |
| **构建工具** | Vite | 5.x |
| **UI 组件** | Ant Design | 5.x |
| **状态管理** | Zustand | 4.x |
| **路由** | React Router | 6.x |

## 项目结构

```
reader-go/
├── cmd/server/           # 后端入口
├── internal/             # 核心业务模块
│   ├── booksource/       # 书源管理
│   ├── localbook/        # 本地书籍解析
│   ├── replace/          # 替换规则
│   ├── rule/             # 规则解析引擎
│   ├── shelf/            # 书架管理
│   ├── web/              # Web API
│   └── webbook/          # 网络书籍获取
├── web/                  # 前端 React 项目
│   ├── src/
│   │   ├── api/         # API 客户端
│   │   ├── components/  # 公共组件
│   │   ├── pages/       # 页面组件
│   │   ├── store/        # 状态管理
│   │   └── types/        # 类型定义
│   ├── package.json
│   └── vite.config.ts
├── data/                 # 数据目录（运行时）
├── go.mod
├── go.sum
└── README.md
```

## 环境变量

| 变量 | 默认值 | 描述 |
|------|--------|------|
| `PORT` | 6464 | API 服务端口 |
| `DATABASE_URL` | ./data/reader.db | 数据库路径 |
| `CACHE_DIR` | ./data/.cache | 缓存目录 |
| `MAX_FILE_SIZE` | 52428800 | 最大上传文件（50MB） |

## 开发规范

### 代码风格

- **Go**: 使用 `gofmt` 格式化，遵循标准 Go 项目布局
- **React/TypeScript**: 遵循项目 ESLint 规则，启用 `strict` 模式
- **命名**: 语义化命名，中文注释

### Git 提交规范

```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式（不影响功能）
refactor: 重构
test: 测试相关
chore: 构建/工具相关
```

### API 设计

- RESTful 风格
- 统一响应格式: `{ "code": 0, "message": "ok", "data": ... }`
- 错误码: 负数表示错误，正数表示成功

### 测试要求

- 规则引擎必须编写单元测试
- 本地书籍解析必须编写单元测试
- 提交前确保 `go test ./...` 通过

## 部署说明

### Docker 部署

```bash
docker-compose up -d
# 前端: http://localhost:6465
# 后端 API: http://localhost:6464
```

### 直接部署

```bash
# 后端
go mod download
go run cmd/server/main.go

# 前端
cd web
npm install
npm run dev
```

## 注意事项

1. **CGO**: go-sqlite3 需要 CGO 支持，编译时设置 `CGO_ENABLED=1`
2. **正则**: Go 标准库 `regexp` 不支持 lookbehind，使用 `regexp2`
3. **编码**: 默认使用 UTF-8，本地书籍支持 GBK/GB2312 自动检测