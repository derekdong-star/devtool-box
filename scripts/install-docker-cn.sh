#!/bin/bash
set -e

# 1. 更新 apt
sudo apt-get update

# 2. 安装必要依赖
sudo apt-get install -y ca-certificates curl gnupg lsb-release

# 3. 添加阿里云 Docker GPG key
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

# 4. 添加阿里云 Docker apt 源
CODENAME=$(lsb_release -cs)
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://mirrors.aliyun.com/docker-ce/linux/ubuntu ${CODENAME} stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 5. 安装 Docker
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 6. 启动并设置开机自启
sudo systemctl start docker
sudo systemctl enable docker

# 7. 配置国内镜像加速器 (腾讯云)
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json << 'EOF'
{
  "registry-mirrors": [
    "https://mirror.ccs.tencentyun.com",
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com"
  ]
}
EOF

# 8. 重启 Docker 生效
sudo systemctl restart docker

# 9. 验证
echo "Docker 版本:"
docker --version

echo "Docker Compose 版本:"
docker compose version

echo "运行测试容器:"
sudo docker run --rm hello-world

echo "✅ Docker 安装完成"
