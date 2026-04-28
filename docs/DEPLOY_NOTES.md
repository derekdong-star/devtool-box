# DevToolbox 自动部署说明

> 目标：当代码推送到 `main` 分支后，由 GitHub Actions 自动构建并推送 Docker 镜像，再通过 SSH 登录腾讯云服务器完成更新；公网流量统一由 Caddy 反向代理，并自动申请 HTTPS 证书。

## 1. 部署拓扑

部署完成后的结构如下：

- GitHub Actions 负责构建镜像并推送到 GHCR
- 腾讯云服务器负责拉取最新镜像并启动 `devtoolbox`
- Caddy 独立运行，统一处理域名、HTTPS 和反向代理
- `devtoolbox` 与 Caddy 通过外部 Docker 网络 `caddy_network` 通信

示意：

```text
git push main
  -> GitHub Actions
  -> GHCR
  -> SSH 到腾讯云服务器
  -> docker compose pull && docker compose up -d
  -> Caddy 反向代理到 devtoolbox:8080
```

## 2. 前提条件

开始前请先确认以下条件都已满足：

- 已有一台可 SSH 登录的腾讯云 Linux 服务器
- 域名已解析到该服务器公网 IP
- GitHub 仓库使用 `main` 作为生产部署分支
- 服务器已放行 `80` 和 `443` 端口
- 计划将应用部署到 `/opt/devtoolbox`
- 计划将 Caddy 部署到 `/opt/caddy`

## 3. 服务器初始化

### 3.1 安装 Docker

如果服务器位于国内网络环境，可使用阿里云源安装 Docker：

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg lsb-release

sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

CODENAME=$(lsb_release -cs)
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://mirrors.aliyun.com/docker-ce/linux/ubuntu ${CODENAME} stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

sudo systemctl start docker
sudo systemctl enable docker
```

可选：配置国内镜像加速器。

```bash
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json << 'EOF'
{
  "registry-mirrors": [
    "https://mirror.ccs.tencentyun.com",
    "https://docker.mirrors.ustc.edu.cn"
  ]
}
EOF

sudo systemctl restart docker
docker --version
docker compose version
```

### 3.2 给部署用户授予 Docker 权限

GitHub Actions 会通过 SSH 连接服务器执行部署命令，建议给目标用户加入 `docker` 用户组。以下示例使用 `ubuntu` 用户：

```bash
sudo usermod -aG docker ubuntu
newgrp docker
docker ps
```

如果你不想授予组权限，也可以在部署脚本中继续使用 `sudo docker`。

### 3.3 创建目录和共享网络

```bash
sudo mkdir -p /opt/devtoolbox/data /opt/caddy
sudo chown -R ubuntu:ubuntu /opt/devtoolbox /opt/caddy
sudo docker network create caddy_network || true
```

## 4. 服务配置

### 4.1 DevToolbox 的 Compose 文件

文件：`/opt/devtoolbox/docker-compose.yml`

```yaml
services:
  devtoolbox:
    image: ghcr.io/derekdong-star/devtool-box:latest
    container_name: devtoolbox
    expose:
      - "8080"
    volumes:
      - ./data:/app/data
    environment:
      - DATA_DIR=/app/data
    restart: unless-stopped
    networks:
      - caddy_network

networks:
  caddy_network:
    external: true
```

说明：

- 使用 `expose: 8080` 即可，不需要把应用端口直接暴露到公网
- `./data` 会映射到容器内 `/app/data`，用于持久化连接配置和命令模板
- 镜像地址需要与你的 GitHub 仓库发布路径保持一致

### 4.2 Caddy 的 Compose 文件

文件：`/opt/caddy/docker-compose.yml`

```yaml
services:
  caddy:
    image: caddy:2
    container_name: caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
      - caddy_config:/config
    networks:
      - caddy_network

volumes:
  caddy_data:
  caddy_config:

networks:
  caddy_network:
    external: true
```

### 4.3 Caddyfile

文件：`/opt/caddy/Caddyfile`

```text
tool.derekdong.com {
    reverse_proxy devtoolbox:8080
}
```

说明：

- `devtoolbox` 是 Docker Compose 中的服务名，也是 Caddy 访问后端时使用的主机名
- Caddy 和 DevToolbox 必须加入同一个 `caddy_network`
- 域名 `tool.derekdong.com` 需要提前解析到服务器公网 IP

## 5. GitHub Secrets

在 GitHub 仓库中进入 `Settings -> Secrets and variables -> Actions`，添加以下 Secret：

| Secret | 用途 |
| ------ | ---- |
| `TENCENT_CLOUD_HOST` | 腾讯云服务器公网 IP 或域名 |
| `TENCENT_CLOUD_USER` | SSH 登录用户名，例如 `ubuntu` |
| `TENCENT_CLOUD_SSH_KEY` | 部署用 SSH 私钥 |

### 5.1 生成部署专用 SSH 密钥

建议在服务器上生成一对专用密钥：

```bash
ssh-keygen -t ed25519 -C "github-actions" -f ~/.ssh/github_actions -N ""
cat ~/.ssh/github_actions.pub >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

查看私钥内容：

```bash
cat ~/.ssh/github_actions
```

把完整私钥内容复制到 GitHub Secret `TENCENT_CLOUD_SSH_KEY`。

## 6. GitHub Actions 工作流

在仓库中创建文件 `.github/workflows/deploy.yml`：

```yaml
name: Build and Deploy

on:
  push:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=sha,prefix={{branch}}-
            type=raw,value=latest,enable={{is_default_branch}}

      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  deploy:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to Tencent Cloud
        uses: appleboy/ssh-action@v1.0.3
        with:
          host: ${{ secrets.TENCENT_CLOUD_HOST }}
          username: ${{ secrets.TENCENT_CLOUD_USER }}
          key: ${{ secrets.TENCENT_CLOUD_SSH_KEY }}
          script: |
            set -e
            cd /opt/devtoolbox
            sudo docker compose pull
            sudo docker compose up -d
            sudo docker image prune -f

            cd /opt/caddy
            sudo docker compose exec -T caddy caddy reload --config /etc/caddy/Caddyfile
```

说明：

- `build` 阶段负责构建并推送镜像到 GHCR
- `deploy` 阶段通过 SSH 登录服务器并拉起最新容器
- 如果 Caddy 配置没有变化，`reload` 仍然是安全的；它不会重启容器，只会热加载配置

## 7. 云防火墙和域名

腾讯云安全组至少需要放行以下入站规则：

| 协议端口 | 来源 |
| -------- | ---- |
| TCP:80 | 0.0.0.0/0 |
| TCP:443 | 0.0.0.0/0 |

补充说明：

- 如果应用只通过 Caddy 暴露，就不需要额外开放 `8080` 或 `8090`
- 若域名尚未解析到服务器，Caddy 无法自动申请 HTTPS 证书

## 8. 首次上线

第一次部署时建议按下面顺序执行：

### 8.1 启动 Caddy

```bash
cd /opt/caddy
sudo docker compose up -d
```

### 8.2 启动 DevToolbox

```bash
cd /opt/devtoolbox
sudo docker compose up -d
```

### 8.3 验证状态

```bash
sudo docker compose -f /opt/caddy/docker-compose.yml ps
sudo docker compose -f /opt/caddy/docker-compose.yml logs --tail 20

sudo docker compose -f /opt/devtoolbox/docker-compose.yml ps
sudo docker compose -f /opt/devtoolbox/docker-compose.yml logs --tail 20

sudo docker network inspect caddy_network
```

验证通过后，访问：

```text
https://tool.derekdong.com
```

## 9. 后续发布

后续发布流程只有一条：

```bash
git add .
git commit -m "feat: xxx"
git push origin main
```

推送到 `main` 后：

1. GitHub Actions 自动构建镜像
2. 新镜像推送到 GHCR
3. 工作流 SSH 到服务器执行 `docker compose pull`
4. DevToolbox 容器以最新镜像重建
5. Caddy 保持运行，仅执行配置热加载

通常 2 到 3 分钟内可完成更新。

## 10. 常用排查命令

```bash
sudo docker compose -f /opt/devtoolbox/docker-compose.yml ps
sudo docker compose -f /opt/devtoolbox/docker-compose.yml logs --tail 50

sudo docker compose -f /opt/caddy/docker-compose.yml ps
sudo docker compose -f /opt/caddy/docker-compose.yml logs --tail 50

sudo docker network inspect caddy_network
sudo docker images | head
```

## 11. 故障排查

| 现象 | 常见原因 | 处理方式 |
| ---- | -------- | -------- |
| 页面返回 `502` | Caddy 无法连接到后端 | 检查 `devtoolbox` 容器是否正常运行，以及服务名是否仍为 `devtoolbox` |
| 容器正常但仍然 `502` | Caddy 和 DevToolbox 不在同一网络 | 确认两边的 Compose 文件都声明了外部网络 `caddy_network` |
| HTTPS 证书申请失败 | 域名解析未生效，或 `80/443` 未放行 | 用 `nslookup` 验证域名解析，用安全组确认端口规则 |
| `docker compose pull` 失败 | GHCR 镜像访问权限或网络问题 | 手动执行 `sudo docker pull ghcr.io/derekdong-star/devtool-box:latest` 验证 |
| 修改 Caddyfile 后未生效 | Caddy 配置未重新加载 | 执行 `cd /opt/caddy && sudo docker compose exec -T caddy caddy reload --config /etc/caddy/Caddyfile` |

## 12. 扩展新服务

如果未来还要通过 Caddy 代理其他服务，保持同样模式即可：新服务加入 `caddy_network`，然后在 Caddyfile 增加一个站点块。

示例：

```yaml
services:
  myapi:
    image: my-api:latest
    container_name: myapi
    expose:
      - "3000"
    networks:
      - caddy_network

networks:
  caddy_network:
    external: true
```

对应的 Caddyfile：

```text
tool.derekdong.com {
    reverse_proxy devtoolbox:8080
}

api.derekdong.com {
    reverse_proxy myapi:3000
}
```

修改完成后执行：

```bash
cd /opt/caddy
sudo docker compose exec -T caddy caddy reload --config /etc/caddy/Caddyfile
```

## 13. 数据持久化

应用数据保存在服务器 `/opt/devtoolbox/data/` 目录，并映射到容器内 `/app/data`。只要宿主机目录不删除，容器重建不会丢失以下内容：

- 数据库和 Redis 连接配置
- SQL / Redis 命令模板

---

最后更新时间：`2026-04-28`
