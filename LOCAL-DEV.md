# Reader Go 本地开发指南

> 完全本地开发，无需 Docker，无需服务器

## 🚀 快速开始

### 方式一：一键启动（推荐）

```bash
./dev-start.sh
```

这会同时启动后端和前端，按 `Ctrl+C` 停止所有服务。

### 方式二：分别启动

**终端 1 - 启动后端：**
```bash
./start-backend.sh
```

**终端 2 - 启动前端：**
```bash
./start-frontend.sh
```

## 📍 访问地址

启动后访问：

- **前端界面**: http://localhost:3000
- **后端 API**: http://localhost:6464
- **健康检查**: http://localhost:6464/health

## 🔧 环境要求

| 工具 | 版本要求 | 检查命令 |
|------|----------|----------|
| **Go** | 1.22+ | `go version` |
| **Node.js** | 18+ | `node --version` |
| **npm** | 8+ | `npm --version` |

## 📁 项目结构

```
reader-go/
├── cmd/server/           # 后端入口
├── internal/             # 后端核心代码
│   ├── booksource/       # 书源管理
│   ├── localbook/        # 本地书籍解析
│   ├── replace/          # 替换规则
│   ├── rule/             # 规则引擎
│   ├── shelf/            # 书架管理
│   ├── web/              # Web API
│   └── webbook/          # 网络书籍获取
├── web/                  # 前端 React 项目
│   ├── src/
│   │   ├── api/         # API 客户端
│   │   ├── components/  # 组件
│   │   ├── pages/       # 页面
│   │   └── store/       # 状态管理
│   └── package.json
├── data/                 # 数据目录（自动创建）
├── start-backend.sh      # 启动后端
├── start-frontend.sh     # 启动前端
└── dev-start.sh          # 一键启动
```

## 🔨 开发命令

### 后端

```bash
# 运行后端
go run cmd/server/main.go

# 编译后端
go build -o reader-go-api ./cmd/server

# 运行测试
go test ./...

# 格式化代码
gofmt -w .
```

### 前端

```bash
cd web

# 安装依赖
npm install

# 开发模式
npm run dev

# 构建生产版本
npm run build

# 运行测试
npm run test

# 代码检查
npm run lint
```

## 🗄️ 数据库

数据库文件存储在 `data/reader.db`，使用 SQLite。

**重置数据库：**
```bash
rm data/reader.db*
# 重启后端会自动创建新数据库
```

## 🌐 API 接口

### 书源管理
- `GET /api/bookSources` - 获取书源列表
- `POST /api/bookSources` - 创建书源
- `PUT /api/bookSources/:id` - 更新书源
- `DELETE /api/bookSources/:id` - 删除书源

### 搜索
- `GET /api/search?q=关键词` - 搜索书籍

### 书架
- `GET /api/shelf` - 获取书架
- `POST /api/shelf` - 加入书架
- `PUT /api/shelf/:id` - 更新书架

### 本地书籍
- `POST /api/localBooks` - 上传本地书籍
- `GET /api/localBooks` - 获取本地书籍列表

完整 API 文档见 [docs/API.md](docs/API.md)

## 🔍 调试技巧

### 后端调试

设置环境变量开启调试日志：
```bash
export READER_GO_DEBUG_LOG=./debug.log
go run cmd/server/main.go
```

### 前端调试

浏览器开发者工具 (F12) 可以查看：
- Network 标签：API 请求
- Console 标签：错误日志
- Application 标签：本地存储

## 📝 开发流程

1. **修改后端代码** → 自动重启（使用 `go run`）
2. **修改前端代码** → Vite 热更新，无需刷新
3. **查看效果** → 浏览器访问 http://localhost:3000

## 🐛 常见问题

### 后端启动失败

**错误**: `CGO_ENABLED` 相关错误
```bash
# 确保启用 CGO
CGO_ENABLED=1 go run cmd/server/main.go
```

**错误**: 数据库锁定
```bash
# 删除数据库文件
rm data/reader.db*
```

### 前端启动失败

**错误**: `node_modules` 缺失
```bash
cd web
npm install
```

**错误**: 端口被占用
```bash
# 修改 vite.config.ts 中的端口
# 或杀掉占用端口的进程
lsof -i :3000
kill -9 <PID>
```

## 🎯 下一步

- 阅读 [docs/ROADMAP.md](docs/ROADMAP.md) 了解开发计划
- 查看 [docs/API.md](docs/API.md) 了解 API 详情
- 参考 [CLAUDE.md](CLAUDE.md) 了解项目规范

---

**Happy Coding!** 🎉
