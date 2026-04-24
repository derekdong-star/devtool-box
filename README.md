# DevToolbox

后端开发日常工具箱，单命令启动，浏览器操作。

## 功能

| 模块 | 功能 |
|------|------|
| **数据库查询** | MySQL / PostgreSQL / SQLite，查看表列表、表结构、执行 SQL；历史连接自动保存 |
| **Redis 查询** | 连接 Redis，前缀搜索 Key，查看各类型 Value，执行任意命令；历史连接自动保存 |
| **Cookie 解析** | 拆解 Cookie 字段；解析 gorilla/securecookie Session，支持带 Secret 验签 |
| **JWT 解析** | 解码 Header / Payload，时间戳字段自动转可读时间 |
| **JSON 工具** | 格式化 / 压缩 / 转义 |
| **编码转换** | Base64 / URL 编解码 |
| **时间 / UUID** | 时间戳互转，批量生成 UUID v4 |

## 快速开始

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

连接配置保存在可执行文件同级的 `db_conns.json`，包含 DSN 和密码，已加入 `.gitignore`，不会提交到仓库。
