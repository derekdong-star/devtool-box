# DevToolbox

后端开发日常工具箱，单命令启动，浏览器操作。

## 功能

| 模块 | 功能 |
|------|------|
| **数据库查询** | MySQL / PostgreSQL / SQLite，查看表列表、表结构、执行 SQL；历史连接自动保存；SQL 命令历史与模板 |
| **Redis 查询** | 连接 Redis，前缀搜索 Key，查看各类型 Value，执行任意命令；历史连接自动保存；Redis 命令历史与模板 |
| **命令模板** | SQL / Redis 常用命令保存为模板，一键回填并自动复制到剪贴板；模板数据后端持久化 |
| **Cookie 解析** | 拆解 Cookie 字段；解析 gorilla/securecookie Session，支持带 Secret 验签 |
| **JWT 解析** | 解码 Header / Payload，时间戳字段自动转可读时间 |
| **JSON 工具** | 格式化 / 压缩 / 转义 |
| **编码转换** | Base64 / URL 编解码 |
| **时间 / UUID** | 时间戳互转，批量生成 UUID v4 |

## 快速开始

### 方式一：Docker Compose（推荐）

无需安装 Go，有 Docker 即可：

```bash
git clone <repo-url>
cd devtoolbox
docker compose up -d
```

浏览器打开 [http://localhost:8090](http://localhost:8090)

> 数据自动持久化到 `./data/` 目录，容器删除重建后配置不丢失。

### 方式二：本地源码运行

**依赖：** Go 1.21+

```bash
git clone <repo-url>
cd devtoolbox
go run ./cmd/server
```

浏览器打开 [http://localhost:8080](http://localhost:8080)

或编译为单一可执行文件：

```bash
go build -o devtoolbox ./cmd/server
./devtoolbox
```

## 项目结构

```
cmd/server/         # 程序入口
internal/
  app/              # 生命周期，模块注册
  handler/          # HTTP 层，路由注册
  service/          # 业务逻辑层
  model/            # DTO
  web/              # 前端资源（go:embed 打包进二进制）
pkg/response/       # 统一响应格式
```

## 本地数据

所有数据统一保存在 `./data/` 目录：

| 文件 | 内容 |
|---|---|
| `db_conns.json` | 数据库 / Redis 连接配置（含密码） |
| `cmd_templates.json` | SQL / Redis 命令模板 |

已加入 `.gitignore`，不会提交到仓库。本地启动和 Docker 共用同一套数据文件。
