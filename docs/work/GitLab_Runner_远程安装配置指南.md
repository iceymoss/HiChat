# GitLab Runner 远程安装配置指南

本文档说明如何在服务器B上安装 GitLab Runner，并连接到服务器A上的 GitLab。

## 环境说明

- **服务器A（GitLab）**: 192.168.4.102
- **服务器B（Runner）**: 需要安装 GitLab Runner
- **跳板机**: 172.34.2.11

---

## 步骤 1: 在服务器B上安装 Docker

### 1.1 安装 curl（如果还没有）

```bash
apt update
apt install -y curl
```

### 1.2 安装 Docker

```bash
curl -fsSL https://get.docker.com | sh
```

### 1.3 启动 Docker 服务

```bash
systemctl start docker
systemctl enable docker
```

### 1.4 验证 Docker 安装

```bash
docker --version
docker ps
```

---

## 步骤 2: 在服务器B上安装 GitLab Runner

### 重要说明：Runner 的运行方式

有两种方式运行 GitLab Runner：

#### 方式 A：系统服务方式（传统方式）
- Runner 作为系统服务运行在主机上
- 通过 `apt-get install` 安装
- 使用 `systemctl` 管理

#### 方式 B：Docker 容器方式（推荐）⭐
- Runner 本身也运行在 Docker 容器中
- 更易于管理和迁移
- 配置更灵活

**CI/CD 任务的执行方式**（两种方式相同）：
- ✅ Runner 使用 **Docker executor**，所以**CI/CD 任务会在 Docker 容器中执行**
- 当 Pipeline 触发时，Runner 会：
  1. 从 GitLab 获取任务
  2. 在服务器B上启动 Docker 容器（例如 `docker:24`）
  3. 在容器中执行构建任务
  4. 任务完成后清理容器

**架构示意（Docker 方式）**：
```
服务器B主机
└── Docker Engine
    ├── GitLab Runner 容器（持续运行）
    │   └── 监听 GitLab 的任务
    └── CI/CD 任务容器（每次任务时临时创建）
        └── docker:24 容器中执行构建脚本
```

---

## 方式 A：系统服务方式安装 Runner

### A.1 添加 GitLab Runner 仓库

```bash
curl -L "https://packages.gitlab.com/install/repositories/runner/gitlab-runner/script.deb.sh" | sudo bash
```

### A.2 安装 GitLab Runner

```bash
apt-get install gitlab-runner
```

安装后，Runner 会作为系统服务注册，可以通过 `systemctl` 管理。

### A.3 将当前用户添加到 docker 组（可选，如果使用非 root 用户）

```bash
usermod -aG docker $USER
newgrp docker
```

**注意**：如果使用系统服务模式，Runner 会以 `gitlab-runner` 用户运行，需要确保该用户有权限访问 Docker。

---

## 方式 B：Docker 容器方式安装 Runner（推荐）⭐

### B.1 创建配置目录

```bash
# 创建配置文件存储目录
sudo mkdir -p /srv/gitlab-runner/config
```

### B.2 运行 GitLab Runner 容器

```bash
docker run -d \
  --name gitlab-runner \
  --restart always \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /srv/gitlab-runner/config:/etc/gitlab-runner \
  gitlab/gitlab-runner:latest
```

**参数说明**：
- `--name gitlab-runner`: 容器名称
- `--restart always`: 自动重启策略
- `-v /var/run/docker.sock:/var/run/docker.sock`: 挂载 Docker socket，让 Runner 容器能够创建其他容器
- `-v /srv/gitlab-runner/config:/etc/gitlab-runner`: 挂载配置目录，持久化配置

### B.3 验证容器运行

```bash
# 查看容器状态
docker ps | grep gitlab-runner

# 查看容器日志
docker logs gitlab-runner
```

应该看到容器正在运行。

**注意**：使用 Docker 方式时，后续的注册和配置命令都需要在容器内执行（使用 `docker exec`）。

---

## 步骤 3: 从服务器A的 GitLab 获取注册 Token

### 3.1 访问 GitLab（通过 SSH 端口转发或直接访问）

如果使用 SSH 端口转发（从本地电脑）：

```bash
# 在本地电脑执行
ssh -L 8080:192.168.4.102:80 user@172.34.2.11
```

然后访问：`http://localhost:8080`

### 3.2 获取 Runner 注册 Token

1. 登录 GitLab（用户名：`root`，密码：从服务器A的 `/etc/gitlab/initial_root_password` 获取）
2. 进入项目页面（例如：`root/hichat`）
3. 进入 **Settings > CI/CD**
4. 展开 **Runners** 部分
5. 在 **Project runners** 区域，点击 **Create project runner** 或查看现有的注册 token
6. 复制显示的 **Registration token**（格式类似：`glrt-8-xxxxx...`）

**或者使用项目级别的注册命令**：
- 在 Runners 页面，会显示类似这样的命令：
  ```
  gitlab-runner register --url http://192.168.4.102 --token <YOUR_TOKEN>
  ```

---

## 步骤 4: 在服务器B上注册 Runner

### 4.1 注册 Runner

```bash
sudo gitlab-runner register
```

### 4.2 交互式配置

按提示输入以下信息：

1. **GitLab instance URL**: 
   ```
   http://192.168.4.102
   ```
   （使用服务器A的IP地址）

2. **Registration token**: 
   ```
   <从步骤3.2获取的token>
   ```

3. **Runner description**: 
   ```
   remote-runner
   ```
   （或自定义名称）

4. **Runner tags**: 
   ```
   docker
   ```
   （用于标识 Runner，在 `.gitlab-ci.yml` 中会用到）

5. **Executor**: 
   ```
   docker
   ```
   （选择 Docker executor）

6. **Default Docker image**: 
   ```
   docker:24
   ```
   （用于执行 CI/CD 任务的默认镜像）

---

### 方式 B：Docker 容器方式注册

如果你使用 Docker 方式运行 Runner（步骤 2 的方式 B），则需要在容器内注册：

#### B.1 在容器内注册 Runner

```bash
docker exec -it gitlab-runner gitlab-runner register
```

#### B.2 交互式配置

按提示输入以下信息（与方式 A 相同）：

1. **GitLab instance URL**: `http://192.168.4.102`
2. **Registration token**: `<从步骤3.2获取的token>`
3. **Runner description**: `remote-runner`
4. **Runner tags**: `docker`
5. **Executor**: `docker`
6. **Default Docker image**: `docker:24`

**注意**：注册后，配置文件会保存在 `/srv/gitlab-runner/config/config.toml`（主机路径）。

---

## 步骤 5: 配置 Runner 支持 Docker-in-Docker

### 方式 A：系统服务方式配置

#### A.1 编辑 Runner 配置文件

```bash
sudo nano /etc/gitlab-runner/config.toml
```

---

### 方式 B：Docker 容器方式配置

#### B.1 编辑 Runner 配置文件

配置文件在主机上的路径：`/srv/gitlab-runner/config/config.toml`

```bash
sudo nano /srv/gitlab-runner/config/config.toml
```

或者直接在容器内编辑：

```bash
docker exec -it gitlab-runner nano /etc/gitlab-runner/config.toml
```

**注意**：两种方式编辑的是同一个文件（通过 volume 挂载）。

---

### 5.2 关键配置（两种方式相同）

找到 `[[runners]]` 部分，确保配置如下：

```toml
[[runners]]
  name = "remote-runner"
  url = "http://192.168.4.102"
  token = "YOUR_TOKEN"
  executor = "docker"
  
  [runners.cache]
    MaxUploadedArchiveSize = 0
    
  [runners.docker]
    tls_verify = false
    image = "docker:24"
    privileged = true  # 重要：启用特权模式，支持 Docker-in-Docker
    disable_entrypoint_overwrite = false
    oom_kill_disable = false
    disable_cache = false
    volumes = ["/cache"]
    shm_size = 0
    network_mtu = 0
    # 重要：添加 DNS 解析，让容器能访问服务器A的 GitLab
    # 如果服务器B能直接访问 192.168.4.102，可以使用 IP
    # 如果需要使用域名，需要配置 extra_hosts
    extra_hosts = ["gitlab.iceymoss:192.168.4.102"]
```

**配置说明**：
- `privileged = true`: 启用特权模式，允许在容器中运行 Docker（Docker-in-Docker）
- `extra_hosts`: 让容器内能解析 GitLab 域名（如果 GitLab 使用域名访问）
- `url`: 指向服务器A的 GitLab 地址

### 5.3 如果 GitLab 使用 HTTP（非 HTTPS）

确保 `tls_verify = false`（已在上面配置中）

---

## 步骤 6: 启动并验证 Runner

### 方式 A：系统服务方式

#### A.1 启动 Runner

```bash
sudo gitlab-runner start
```

#### A.2 检查 Runner 状态

```bash
sudo gitlab-runner status
```

应该看到类似输出：
```
gitlab-runner: Service is running
```

#### A.3 查看 Runner 日志（如果遇到问题）

```bash
sudo gitlab-runner --debug run
```

或者查看系统日志：
```bash
journalctl -u gitlab-runner -f
```

---

### 方式 B：Docker 容器方式

#### B.1 启动/重启 Runner 容器

如果容器已经运行（使用 `--restart always` 会自动启动），修改配置后需要重启容器：

```bash
docker restart gitlab-runner
```

#### B.2 检查 Runner 容器状态

```bash
# 查看容器状态
docker ps | grep gitlab-runner

# 查看容器日志
docker logs gitlab-runner

# 查看实时日志
docker logs -f gitlab-runner
```

#### B.3 验证 Runner 在容器内运行

```bash
# 进入容器查看 Runner 状态
docker exec -it gitlab-runner gitlab-runner status

# 或者查看 Runner 进程
docker exec -it gitlab-runner ps aux | grep gitlab-runner
```

**注意**：使用 Docker 方式时，Runner 会自动启动（因为使用了 `--restart always`），无需手动启动服务。

---

## 步骤 7: 在 GitLab 中验证 Runner

1. 在 GitLab 项目页面，进入 **Settings > CI/CD > Runners**
2. 在 **Project runners** 部分，应该看到 Runner 显示为 **Online**（绿色圆点）
3. 如果显示为 **Offline**，检查：
   - Runner 服务是否运行：`sudo gitlab-runner status`
   - 网络连接：从服务器B能否访问 `http://192.168.4.102`
   - 防火墙设置：确保服务器A的 80 端口（或 GitLab 配置的端口）对服务器B开放

---

## 步骤 8: 测试 CI/CD Pipeline

### 8.1 确保项目有 `.gitlab-ci.yml`

如果还没有，可以参考 `docs/work/GitLab安装与CI_CD配置.md` 中的配置。

**关键配置点**（针对远程 GitLab）：

```yaml
variables:
  DOCKER_DRIVER: overlay2
  DOCKER_TLS_CERTDIR: ""
  # 使用服务器A的IP地址和端口（如果 Registry 在 5050 端口）
  CI_REGISTRY: "192.168.4.102:5050"
  CI_REGISTRY_IMAGE: "192.168.4.102:5050/root/hichat"
  IMAGE_TAG: $CI_REGISTRY_IMAGE:$CI_COMMIT_REF_SLUG
  IMAGE_TAG_LATEST: $CI_REGISTRY_IMAGE:latest
  IMAGE_TAG_COMMIT: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA

build:
  stage: build
  image: docker:24
  services:
    - name: docker:24-dind
      command: ["--insecure-registry=192.168.4.102:5050"]
  before_script:
    - |
      echo "登录 GitLab Container Registry: $CI_REGISTRY"
      docker login -u gitlab-ci-token -p $CI_JOB_TOKEN $CI_REGISTRY
  script:
    - docker build -t $IMAGE_TAG -t $IMAGE_TAG_LATEST -t $IMAGE_TAG_COMMIT .
    - docker push $IMAGE_TAG
    - docker push $IMAGE_TAG_LATEST
    - docker push $IMAGE_TAG_COMMIT
  tags:
    - docker
```

### 8.2 推送代码触发 Pipeline

```bash
git add .gitlab-ci.yml
git commit -m "Add CI/CD configuration"
git push origin master
```

### 8.3 查看 Pipeline 执行

1. 在 GitLab 项目页面，点击 **CI/CD > Pipelines**
2. 查看 Pipeline 状态和日志
3. 确认 Runner 成功执行任务

---

## 常见问题排查

### 问题 1: Runner 显示 Offline

**检查项**：
1. **方式 A**：Runner 服务是否运行：`sudo gitlab-runner status`
2. **方式 B**：Runner 容器是否运行：`docker ps | grep gitlab-runner`
3. 网络连通性：从服务器B测试 `curl http://192.168.4.102`
4. 防火墙：确保服务器A的端口对服务器B开放

**解决方案**：

**方式 A（系统服务）**：
```bash
# 重启 Runner
sudo gitlab-runner restart

# 检查网络连接
ping 192.168.4.102
curl -I http://192.168.4.102
```

**方式 B（Docker 容器）**：
```bash
# 重启 Runner 容器
docker restart gitlab-runner

# 查看容器日志
docker logs gitlab-runner

# 检查网络连接（在容器内）
docker exec -it gitlab-runner ping 192.168.4.102
docker exec -it gitlab-runner curl -I http://192.168.4.102
```

### 问题 2: Pipeline 失败 - 无法连接到 GitLab 或任务执行失败

**原因**：
1. 容器内无法访问服务器A的 GitLab（缺少 DNS 解析）
2. 任务需要访问 `gitlab.iceymoss:5050` 但容器内无法解析该域名
3. Docker Hub 登录失败（缺少环境变量）

**解决方案**：

**步骤 1：更新 Runner 配置文件**

编辑 `/srv/gitlab-runner/config/config.toml`，在 `[runners.docker]` 部分添加 `extra_hosts`：

```toml
[runners.docker]
  tls_verify = false
  image = "docker:24"
  privileged = true
  disable_entrypoint_overwrite = false
  oom_kill_disable = false
  disable_cache = false
  volumes = ["/cache"]
  shm_size = 0
  network_mtu = 0
  # 重要：添加 DNS 解析，让容器能访问服务器A的 GitLab
  extra_hosts = ["gitlab.iceymoss:192.168.4.102"]
```

**步骤 2：重启 Runner 容器使配置生效**

```bash
docker restart gitlab-runner
```

**步骤 3：检查 GitLab CI/CD 变量配置**

在 GitLab 项目页面（`http://192.168.4.102`），进入：
- **Settings > CI/CD > Variables**

确保配置了以下变量（如果 `.gitlab-ci.yml` 需要）：
- `DOCKERHUB_USERNAME`: Docker Hub 用户名
- `DOCKERHUB_PASSWORD`: Docker Hub 密码或访问令牌

**步骤 4：验证配置**

```bash
# 查看容器日志，确认配置已加载
docker logs gitlab-runner | tail -20

# 在容器内测试 DNS 解析
docker exec -it gitlab-runner ping -c 2 gitlab.iceymoss
```

**步骤 5：重新触发 Pipeline**

在 GitLab 项目页面：
- 进入 **CI/CD > Pipelines**
- 点击 **Retry** 重新运行失败的 Pipeline

### 问题 3: Docker-in-Docker 权限问题

**错误信息**：
```
mount: permission denied
```

**解决方案**：
确保 Runner 配置中 `privileged = true`：
```toml
[runners.docker]
  privileged = true
```

### 问题 4: Registry 登录失败

**原因**：Registry 地址配置错误或网络不通

**解决方案**：
1. 确认 Registry 地址：`http://192.168.4.102:5050`（根据实际配置调整）
2. 在 `.gitlab-ci.yml` 中使用 `--insecure-registry` 参数
3. 确保服务器B能访问服务器A的 Registry 端口

---

## 网络配置建议

### 如果服务器A和B在同一内网

- 直接使用 IP 地址：`192.168.4.102`
- 确保防火墙允许相关端口

### 如果需要通过跳板机访问

- 考虑在跳板机上配置端口转发
- 或者使用 VPN/专线连接

### 防火墙端口

确保以下端口开放：
- **GitLab Web**: 80（或配置的端口）
- **GitLab Registry**: 5050（或配置的端口）
- **GitLab SSH**: 22（如果需要）

---

## 完整配置检查清单

### 方式 A：系统服务方式
- [ ] Docker 已安装并运行
- [ ] GitLab Runner 已通过 `apt-get` 安装
- [ ] Runner 已注册到服务器A的 GitLab
- [ ] Runner 配置文件 `/etc/gitlab-runner/config.toml` 正确配置
- [ ] Runner 服务运行正常：`sudo gitlab-runner status`
- [ ] GitLab 页面显示 Runner 为 Online
- [ ] 测试 Pipeline 可以成功执行

### 方式 B：Docker 容器方式
- [ ] Docker 已安装并运行
- [ ] GitLab Runner 容器已创建并运行
- [ ] Runner 已注册到服务器A的 GitLab
- [ ] Runner 配置文件 `/srv/gitlab-runner/config/config.toml` 正确配置
- [ ] Runner 容器运行正常：`docker ps | grep gitlab-runner`
- [ ] GitLab 页面显示 Runner 为 Online
- [ ] 测试 Pipeline 可以成功执行

---

## 验证 Runner 运行方式

### 方式 A：系统服务方式验证

#### 检查 Runner 进程（主机上运行）

```bash
# 查看 Runner 进程（应该在主机上，不在容器中）
ps aux | grep gitlab-runner | grep -v grep

# 应该看到类似输出：
# gitlab-run+  1234  0.1  0.5  /usr/bin/gitlab-runner run --config /etc/gitlab-runner/config.toml
```

#### 检查 Runner 服务状态

```bash
# 查看系统服务状态
sudo systemctl status gitlab-runner

# 应该显示为 active (running)
```

---

### 方式 B：Docker 容器方式验证

#### 检查 Runner 容器

```bash
# 查看 Runner 容器（Runner 本身在容器中）
docker ps | grep gitlab-runner

# 应该看到类似：
# CONTAINER ID   IMAGE                        COMMAND                  CREATED         STATUS
# abc123def456   gitlab/gitlab-runner:latest  "/usr/bin/dentrypoint"   2 hours ago     Up 2 hours
```

#### 检查容器内 Runner 进程

```bash
# 进入容器查看 Runner 进程
docker exec -it gitlab-runner ps aux | grep gitlab-runner

# 或者查看容器日志
docker logs gitlab-runner
```

---

### 查看 CI/CD 任务容器（两种方式相同）

当 Pipeline 运行时，可以查看 Runner 创建的临时任务容器：

```bash
# 查看正在运行的容器（CI/CD 任务容器）
docker ps

# 应该看到类似：
# CONTAINER ID   IMAGE          COMMAND                  CREATED         STATUS
# xyz789ghi012   docker:24      "/usr/bin/docker-ent…"   2 minutes ago   Up 2 minutes
```

**总结**：
- **方式 A**：Runner 进程运行在**主机上**（系统服务），CI/CD 任务运行在**Docker 容器中**
- **方式 B**：Runner 进程运行在**Docker 容器中**，CI/CD 任务也运行在**Docker 容器中**（嵌套容器）

---

## 两种方式对比

| 特性 | 方式 A：系统服务 | 方式 B：Docker 容器 |
|------|-----------------|-------------------|
| **安装方式** | `apt-get install` | `docker run` |
| **配置文件位置** | `/etc/gitlab-runner/config.toml` | `/srv/gitlab-runner/config/config.toml` |
| **管理命令** | `systemctl` / `gitlab-runner` | `docker` |
| **隔离性** | 较低（直接运行在主机） | 较高（容器隔离） |
| **迁移性** | 需要重新安装配置 | 易于迁移（容器化） |
| **资源占用** | 略低 | 略高（容器开销） |
| **推荐场景** | 传统服务器环境 | 容器化环境、需要隔离 |

**推荐**：
- 如果服务器B已经是容器化环境，推荐使用**方式 B（Docker 容器方式）**
- 如果服务器B是传统服务器环境，两种方式都可以，**方式 A** 更简单直接

---

## 参考文档

- `docs/work/GitLab安装与CI_CD配置.md` - GitLab 完整安装配置
- `docs/GITLAB_CI的必要配置.md` - CI/CD 详细配置说明
- `docs/GITLAB_CI_CD.md` - CI/CD 使用指南

---

**文档版本**: 1.0  
**最后更新**: 2025-01-XX

