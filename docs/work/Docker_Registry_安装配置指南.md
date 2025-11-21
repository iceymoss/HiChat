# Docker Registry 安装配置指南

**服务器 IP**: 192.168.4.104  
**Registry 端口**: 5000  
**Registry UI 端口**: 8080  
**数据存储路径**: `/opt/docker-registry`

---

## 快速安装（Docker Compose）

> **⚠️ 重要提示**：本配置使用 **HTTP**（非 HTTPS）Registry。安装完成后，**所有客户端机器**（包括 CI/CD Runner、K8s 节点等）都必须配置 `insecure-registries`，否则无法推送/拉取镜像。详见下方"配置客户端"章节。

### 1. 安装 Docker Compose（如未安装）

```bash
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

### 2. 创建目录和认证

```bash
# 创建目录
sudo mkdir -p /opt/docker-registry/{data,auth}

# 安装 htpasswd
sudo apt-get install -y apache2-utils

# 创建用户（admin/admin123）
sudo htpasswd -Bbn admin admin123 > /opt/docker-registry/auth/htpasswd
```

### 3. 创建 docker-compose.yml

```bash
cd /opt/docker-registry
sudo nano docker-compose.yml
```

内容：

```yaml
version: '3.8'

services:
  registry:
    image: registry:2
    container_name: docker-registry
    restart: always
    ports:
      - "5000:5000"
    environment:
      REGISTRY_AUTH: htpasswd
      REGISTRY_AUTH_HTPASSWD_PATH: /auth/htpasswd
      REGISTRY_AUTH_HTPASSWD_REALM: Registry Realm
    volumes:
      - ./data:/var/lib/registry
      - ./auth:/auth
    networks:
      - registry-net

  registry-ui:
    image: joxit/docker-registry-ui:latest
    container_name: registry-ui
    restart: always
    ports:
      - "8080:80"
    environment:
      REGISTRY_URL: http://registry:5000
      REGISTRY_SECURE: false
      DELETE_IMAGES: true
      REGISTRY_USERNAME: admin
      REGISTRY_PASSWORD: admin123
    depends_on:
      - registry
    networks:
      - registry-net

networks:
  registry-net:
    driver: bridge
```

### 4. 启动服务

```bash
sudo docker-compose up -d

# 查看日志
sudo docker-compose logs -f
```

### 5. 验证

```bash
# 检查容器
docker ps | grep -E "docker-registry|registry-ui"

# 测试 API
curl -I http://192.168.4.104:5000/v2/
# 应返回 401 Unauthorized（正常）

# 访问 Web UI
# http://192.168.4.104:8080
```

---

## 常用操作

### 登录 Registry

```bash
docker login 192.168.4.104:5000
# 用户名：admin
# 密码：admin123
```

### 推送镜像

```bash
# 1. 标记镜像
docker tag <image-name>:<tag> 192.168.4.104:5000/<image-name>:<tag>

# 2. 推送镜像
docker push 192.168.4.104:5000/<image-name>:<tag>
```

### 拉取镜像

```bash
docker pull 192.168.4.104:5000/<image-name>:<tag>
```

### 配置客户端（HTTP Registry）

**重要**：由于本指南使用 HTTP（非 HTTPS）Registry，**所有客户端机器都必须配置 `insecure-registries`**，否则无法推送/拉取镜像。

#### 在客户端机器上配置

编辑 `/etc/docker/daemon.json`：

```json
{
  "insecure-registries": ["192.168.4.104:5000"]
}
```

**注意**：
- 如果文件不存在，需要创建
- 如果文件已有其他配置，需要合并（不能覆盖）
- 可以配置多个 Registry，用逗号分隔

然后重启 Docker：

```bash
sudo systemctl restart docker
```

#### 验证配置

```bash
# 检查配置是否生效
docker info | grep -A 5 "Insecure Registries"

# 应该看到：192.168.4.104:5000
```

#### 常见错误

**错误 1**: `http: server gave HTTP response to HTTPS client`

**原因**: 客户端未配置 `insecure-registries`，Docker 默认使用 HTTPS 连接

**解决**: 按照上述步骤配置 `insecure-registries` 并重启 Docker

**错误 2**: `Get "https://192.168.4.104:5000/v2/": dial tcp: lookup 192.168.4.104`

**原因**: 网络连接问题或 DNS 解析失败

**解决**: 
- 检查网络连接：`ping 192.168.4.104`
- 检查防火墙是否开放 5000 端口
- 如果使用域名，检查 DNS 配置

#### 服务端配置说明

**当前配置使用 HTTP**：
- `docker-compose.yml` 中 Registry 使用默认 HTTP 协议
- `REGISTRY_SECURE: false` 表示 Registry UI 使用 HTTP 连接 Registry

**如果需要使用 HTTPS**（生产环境推荐）：

1. 准备 SSL 证书（`domain.crt` 和 `domain.key`）
2. 创建证书目录：`mkdir -p /opt/docker-registry/certs`
3. 将证书文件放入该目录
4. 修改 `docker-compose.yml`，添加证书挂载：

```yaml
registry:
  # ... 其他配置 ...
  volumes:
    - ./data:/var/lib/registry
    - ./auth:/auth
    - ./certs:/certs
  environment:
    # ... 其他环境变量 ...
    REGISTRY_HTTP_TLS_CERTIFICATE: /certs/domain.crt
    REGISTRY_HTTP_TLS_KEY: /certs/domain.key
```

5. 客户端配置改为使用 HTTPS（无需 `insecure-registries`）：
   ```bash
   docker login https://192.168.4.104:5000
   ```

### 容器管理

```bash
cd /opt/docker-registry

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 重启
docker-compose restart

# 停止
docker-compose stop

# 启动
docker-compose start

# 停止并删除容器
docker-compose down
```

---

## 问题排查

### 推送/拉取镜像失败

#### 错误 1: `http: server gave HTTP response to HTTPS client`

**原因**: 
- 服务端使用 HTTP，但客户端未配置 `insecure-registries`
- Docker 客户端默认使用 HTTPS 连接 Registry

**解决步骤**:
1. 在客户端编辑 `/etc/docker/daemon.json`，添加：
   ```json
   {
     "insecure-registries": ["192.168.4.104:5000"]
   }
   ```
2. 重启 Docker：`sudo systemctl restart docker`
3. 验证配置：`docker info | grep "Insecure Registries"`
4. 重新尝试推送/拉取

**注意**: 
- **所有需要访问 Registry 的客户端机器都必须配置**
- 包括：开发机器、CI/CD Runner、K8s 节点等
- 配置后必须重启 Docker 服务才能生效

#### 错误 2: `unauthorized: authentication required`

**原因**: Registry 启用了认证，但未登录

**解决**: 
```bash
# 先登录
docker login 192.168.4.104:5000
# 用户名：admin
# 密码：admin123

# 然后再推送/拉取
```

#### 错误 3: `denied: requested access to the resource is denied`

**原因**: 
- 认证失败（用户名或密码错误）
- 认证文件配置错误

**解决**:
1. 检查登录凭据是否正确
2. 重新登录：`docker login 192.168.4.104:5000`
3. 检查服务端认证文件：`cat /opt/docker-registry/auth/htpasswd`

### Registry UI 无法连接

**检查**:
```bash
# 查看日志
docker-compose logs registry-ui

# 检查 REGISTRY_URL（应使用容器名）
docker inspect registry-ui | grep REGISTRY_URL
```

**解决**: 确保 `REGISTRY_URL=http://registry:5000`（使用容器名，不是 IP）

### 数据丢失

**检查数据卷挂载**:
```bash
docker inspect docker-registry | grep -A 10 Mounts
ls -la /opt/docker-registry/data
```

---

## 备份与恢复

### 备份

```bash
cd /opt/docker-registry
docker-compose stop
tar -czf registry-backup-$(date +%Y%m%d).tar.gz data auth
docker-compose start
```

### 恢复

```bash
cd /opt/docker-registry
docker-compose stop
tar -xzf registry-backup-YYYYMMDD.tar.gz
docker-compose start
```

---

## 访问地址

- **Registry API**: `http://192.168.4.104:5000`
- **Registry UI**: `http://192.168.4.104:8080`

**相关文档**:
- [GitLab CI/CD 配置指南](../GITLAB_CI的必要配置.md)
- [GitLab Runner 远程安装配置指南](./GitLab_Runner_远程安装配置指南.md)
