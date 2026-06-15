# Docker 部署与优化

> 返回 [文档索引](./README.md) · Phase 0 [phase-0-foundation.md](./phases/phase-0-foundation.md)

后期部署以 **Docker Compose 为主**，不纳入宿主机 systemd/nginx 运维项。

## 当前拓扑

```mermaid
flowchart LR
  User[用户浏览器]
  Web[reader-go-web :6465]
  Nginx[nginx 容器内]
  API[reader-go-api :6464]
  Vol["volume ./data"]
  User -->|http://host:6465| Web
  Web --> Nginx
  Nginx -->|/api/| API
  API --> Vol
```

| 服务 | 镜像 | 端口 | 说明 |
|------|------|------|------|
| `reader-go-api` | [`Dockerfile`](../Dockerfile) | `6464:6464` | Go API + SQLite |
| `reader-go-web` | [`Dockerfile.web`](../Dockerfile.web) | `6465:80` | 静态前端 + nginx 反代 |

**数据卷**：`./data` → `/app/data`（含 `reader.db`）

**网络**：`reader-go-net` bridge；web 容器通过服务名 `reader-go-api:6464` 访问 API。

## Phase 0 已知缺陷（必修复）

| ID | 问题 | 根因 | 修复方案 | 文件 |
|----|------|------|----------|------|
| P0-D01 | healthcheck 失败 | 运行时镜像未安装 `wget`，但 healthcheck 使用 wget | 安装 `wget` **或** 改用 `curl` / 内置 HTTP | [`Dockerfile`](../Dockerfile), [`docker-compose.yml`](../docker-compose.yml) |
| P0-D02 | nginx `/health` 502 | 反代到 `:8080`，API 实际 `:6464` | 改为 `reader-go-api:6464` | [`nginx.conf`](../nginx.conf) L29 |
| P0-D03 | 本地默认端口不一致 | `main.go` 默认 `8080`，compose 用 `6464` | 默认 `PORT=6464`；README 对齐 | [`cmd/server/main.go`](../cmd/server/main.go), [`README.md`](../README.md) |
| P0-D04 | README Docker 端口错误 | 写 `8080` / `localhost` 混用 | 前端 `:6465`，API `:6464` | [`README.md`](../README.md) |

### 验收（Phase 0 Docker）

```bash
docker-compose up -d --build
docker-compose ps   # reader-go-api 应为 healthy
curl -sf http://localhost:6464/health
curl -sf http://localhost:6465/health   # 经 nginx 反代
curl -sf http://localhost:6465/api/bookSources
```

## 环境变量

| 变量 | 默认值 | 服务 | 说明 |
|------|--------|------|------|
| `PORT` | 6464 | api | API 监听端口 |
| `DATABASE_URL` | /app/data/reader.db | api | SQLite 路径 |
| `GIN_MODE` | debug | api | **建议** release |
| `JS_TIMEOUT_MS` | — | api | Phase 1：JS 超时 |
| `CACHE_DIR` | — | api | 可选缓存目录 |

### Phase 0 建议 compose 增强

```yaml
environment:
  - PORT=6464
  - DATABASE_URL=/app/data/reader.db
  - GIN_MODE=release
  - JS_TIMEOUT_MS=5000
  - CACHE_DIR=/app/data/.cache
```

## 构建说明

### API 多阶段构建

[`Dockerfile`](../Dockerfile)：

1. **builder**：`golang:1.22-alpine` + CGO（go-sqlite3）
2. **runtime**：`alpine:3.19` + ca-certificates + tzdata

当前 runtime 镜像已包含 `wget`，可用于 healthcheck。

## 运维脚本

推荐直接使用仓库内脚本：

```bash
./deploy/rebuild-compose.sh
./deploy/smoke-test.sh
./deploy/cleanup-docker.sh
```

- `rebuild-compose.sh`：使用 `docker build --network=host` 重建 API 镜像并拉起 compose
- `smoke-test.sh`：检查 health、书海、RSS 抓取、Chromium 可执行路径
- `cleanup-docker.sh`：清理悬空镜像和 builder 缓存

### Web 镜像

[`Dockerfile.web`](../Dockerfile.web) + [`nginx.conf`](../nginx.conf)：

- SPA `try_files` fallback
- `/api/` → `reader-go-api:6464`
- 静态资源长期缓存

## 优化路线图

### 构建缓存

- go mod download 单独 layer（已实现）
- 可选 BuildKit cache mount：`--mount=type=cache,target=/go/pkg/mod`

### 多架构

```yaml
# 示例：buildx 矩阵
platforms:
  - linux/amd64
  - linux/arm64
```

go-sqlite3 需在 arm64 验证 CGO 交叉编译。

### Compose Profiles

| Profile | 用途 |
|---------|------|
| `default` | 生产双服务 |
| `dev` | 挂载源码 + air/ vite 热重载 |
| `all-in-one` | 单容器 nginx+api（可选） |

### 备份卷

```yaml
volumes:
  - ./data:/app/data
  - ./backups:/app/backups   # 定时 sqlite .backup
```

### 生产建议

- `GIN_MODE=release`
- `restart: unless-stopped`（已配置）
- 资源限制：

```yaml
deploy:
  resources:
    limits:
      cpus: '1'
      memory: 512M
```

- 日志驱动：`json-file`，`max-size: 10m`

### depends_on 健康检查

[`docker-compose.yml`](../docker-compose.yml) 中 web 依赖 `service_healthy`。**P0-D01 修复前 web 可能永远无法启动**。

## 与宿主机部署的关系

根目录 [`nginx-server.conf`](../nginx-server.conf) 与 [`deploy/`](../deploy/) 保留供非 Docker 场景，**不在主路线图维护范围**。新功能验收以 compose 为准。

## 故障排查

| 现象 | 检查 |
|------|------|
| web 容器一直 Starting | `docker logs reader-go-api`；healthcheck 是否 wget 失败 |
| API 404 on /api | nginx `proxy_pass` 是否带 trailing slash |
| 书架为空换容器后 | Phase 0：前端是否接 POST /api/shelf |
| 书源导入后搜不到 | WebBook 热加载（F-08 已修）；确认 API healthy |
