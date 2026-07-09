# DevToolbox 项目上下文

## 产品与命名

- 仓库 / 服务名：`DevToolbox`
- 产品界面名：`Toolbox`
- UI 品牌调整时，不要修改 Go module、Docker service、container name 或部署目录。

## 架构边界

- 程序入口：`cmd/server`
- 应用组装与模块注册：`internal/app/app.go`
- HTTP handler：`internal/handler`
- 业务逻辑与持久化：`internal/service`
- DTO：`internal/model/dto.go`
- 前端资源：`internal/web`
- 统一响应辅助：`pkg/response`

新增功能模块时，沿用现有模式：

- 请求 / 响应结构放到 `internal/model`
- 业务逻辑放到 `internal/service`
- 路由注册放到 `internal/handler`
- 在 `internal/app/app.go` 的 `modules` 列表追加 handler
- 前端脚本放到 `internal/web/static/js`，并在 `internal/web/index.html` 引入

## 当前模块

- 数据库查询：MySQL / PostgreSQL / SQLite，表列表、表结构、SQL 执行，已保存连接支持自定义名称
- Redis 查询：连接、Key 扫描、Value 查看、原始命令执行
- 命令模板：SQL / Redis 常用命令模板
- Cookie / Session 解析：Cookie 字段拆解，gorilla/securecookie session 解码
- JWT 解析
- JSON 格式化 / 压缩 / 转义
- Base64 和 URL 编解码
- 时间戳与 UUID 工具
- 图片生成：文生图 / 图生图，支持 OpenAI SDK 和 OpenAI-compatible API
- 文件上传：腾讯云 COS 上传并返回资源链接

## 数据持久化

运行时数据优先使用 `DATA_DIR`，未设置时使用 `./data`。

- `db_conns.json`：数据库和 Redis 已保存连接
- `cmd_templates.json`：SQL / Redis 命令模板
- `image_config.json`：图片生成 API 配置
- `upload_config.json`：腾讯云 COS 上传配置

这些文件可能包含密钥或连接密码，不能提交到仓库。

## 部署事实

当前自动部署流程由 GitHub Release 触发。

- 工作流文件：`.github/workflows/deploy.yml`
- 触发方式：`release.published`
- 构建位置：GitHub Actions
- 镜像仓库：GHCR
- 服务器动作：SSH 到 `/opt/devtoolbox`，执行 `docker compose pull`、`docker compose up -d`、`docker image prune -f`
- 普通 `git push origin main` 不会直接触发部署。

涉及部署、发布、CI/CD、远端服务器时，先读取 `docs/DEPLOY_NOTES.md` 和 `.github/workflows/deploy.yml`。

## 本地运行与验证

- 本地 Docker：`docker compose up -d --build`，访问 `http://localhost:8090`
- 本地源码：`go run ./cmd/server`，访问 `http://localhost:8080`
- Go 测试：`go test ./...`
- Go 构建：`go build ./...`
- 前端语法检查：`node --check <changed-js-file>`

## Agent 工作约定

- 仓库级 agent 规则：`AGENTS.md`
- UI 品牌规则：`DESIGN.md`
- 部署说明：`docs/DEPLOY_NOTES.md`
- 用户侧功能总览：`README.md`
