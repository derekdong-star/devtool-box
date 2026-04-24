# ── Build Stage ─────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /build

# 先复制依赖文件，利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制源码（包含 go:embed 的前端静态文件）
COPY . .

# 静态编译：禁用 CGO，静态链接
ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o devtoolbox ./cmd/server

# ── Runtime Stage ───────────────────────────────────────────────
FROM alpine:3.21

# 安装 ca-certificates（HTTPS 请求需要）
RUN apk add --no-cache ca-certificates

# 从 builder 复制二进制到 PATH
COPY --from=builder /build/devtoolbox /usr/local/bin/devtoolbox

# 工作目录用于存放数据文件（db_conns.json / cmd_templates.json）
WORKDIR /app

# 暴露端口
EXPOSE 8080

# 运行
ENTRYPOINT ["devtoolbox"]
