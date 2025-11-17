# 本地 GitLab 搭建指南

在 WSL 环境中使用 GitLab Omnibus 方式搭建本地 GitLab，用于 CI/CD 测试。

## 前置条件

- WSL2 或 Linux 环境
- 至少 4GB 内存（推荐 8GB）
- 至少 10GB 磁盘空间
- 2 核以上 CPU

## 安装步骤

### 1. 安装依赖

```bash
# 更新系统
sudo apt-get update

# 安装必要的依赖
sudo apt-get install -y curl openssh-server ca-certificates tzdata perl postfix
```

**注意**: 安装 postfix 时如果提示配置，选择 "No configuration"（如果不需要邮件功能）

### 2. 安装 GitLab

```bash
# 下载并安装 GitLab 安装脚本
curl -sS https://packages.gitlab.com/install/repositories/gitlab/gitlab-ce/script.deb.sh | sudo bash

# 安装 GitLab CE
sudo EXTERNAL_URL="http://gitlab.iceymoss" apt-get install gitlab-ce
```

**说明**:
- `EXTERNAL_URL`: GitLab 访问地址，可以改为你的域名
- 安装过程可能需要 5-10 分钟

### 3. 配置 GitLab

```bash
# 编辑 GitLab 配置文件
sudo nano /etc/gitlab/gitlab.rb
```

**常用配置**（可选，如果内存较小）:
```ruby
# 外部访问地址
external_url 'http://gitlab.iceymoss'

# 减少资源使用（可选）
# puma['worker_processes'] = 2
# sidekiq['max_concurrency'] = 5
# prometheus_monitoring['enable'] = false
```

**重新配置 GitLab**:
```bash
sudo gitlab-ctl reconfigure
```

这个过程可能需要几分钟。

### 4. 获取初始 root 密码

```bash
# 查看初始 root 密码
sudo cat /etc/gitlab/initial_root_password
```

**重要**: 记录这个密码，首次登录需要使用

### 5. 配置 hosts 文件

**WSL hosts**:
```bash
echo "127.0.0.1 gitlab.iceymoss" | sudo tee -a /etc/hosts
```

**Windows hosts**（如果需要在浏览器访问）:
1. 以管理员身份打开记事本
2. 打开：`C:\Windows\System32\drivers\etc\hosts`
3. 添加：`127.0.0.1 gitlab.iceymoss`
4. 保存

### 6. 访问 GitLab

1. 打开浏览器访问：`http://gitlab.iceymoss`
2. 使用用户名 `root` 和步骤 4 获取的密码登录
3. 首次登录会要求修改密码

### 7. 验证 GitLab 运行状态

```bash
# 检查 GitLab 服务状态
sudo gitlab-ctl status

# 应该看到所有服务都是 "run"
```

## 配置 GitLab Runner

### 1. 安装 GitLab Runner

```bash
# 下载并安装 GitLab Runner
curl -L "https://packages.gitlab.com/install/repositories/runner/gitlab-runner/script.deb.sh" | sudo bash
sudo apt-get install gitlab-runner

# 将当前用户添加到 docker 组（如果使用 Docker executor）
sudo usermod -aG docker $USER
newgrp docker
```

### 2. 获取 Runner Registration Token

1. 在 GitLab 项目页面，进入 **Settings > CI/CD**
2. 展开 **Runners** 部分
3. 复制 **Registration token**

### 3. 注册 GitLab Runner

```bash
# 注册 Runner
sudo gitlab-runner register
```

**配置选项**:
- GitLab instance URL: `http://gitlab.iceymoss/`
- Registration token: 粘贴从网页复制的 token
- Description: `local-runner`
- Tags: `docker`（必须，与 .gitlab-ci.yml 中的 tags 匹配）
- Executor: `docker`
- Default Docker image: `docker:24`

### 4. 验证 Runner

```bash
# 检查 Runner 状态
sudo gitlab-runner status

# 查看 Runner 列表（在 GitLab 网页）
# Settings > CI/CD > Runners
# 应该看到 Runner 显示为 "Online"
```

## 推送项目到 GitLab

### 1. 在 GitLab 创建项目

1. 登录 GitLab 后，点击 **New project** 或 **Create a project**
2. 选择 **Create blank project**
3. 填写项目信息：
   - Project name: `HiChat`
   - Project slug: `hichat`（自动生成）
   - Visibility: 选择 Private 或 Internal
4. 点击 **Create project**

### 2. 配置 Git 远程仓库

```bash
# 查看当前远程仓库
git remote -v

# 方式 1: 更新现有的 origin（推荐）
git remote set-url origin http://gitlab.iceymoss/root/hichat.git

# 方式 2: 如果 origin 不存在，添加新的
# git remote add origin http://gitlab.iceymoss/root/hichat.git

# 验证
git remote -v
```

**注意**: URL 中的 `root` 是用户名，`hichat` 是项目名，根据实际情况修改

### 3. 推送代码

```bash
# 查看当前分支
git branch

# 推送代码到 GitLab
git push -u origin main
# 或者
git push -u origin master
```

**如果遇到认证问题**:

```bash
# 方式 1: 使用用户名密码（会提示输入）
# 用户名：root
# 密码：你的 GitLab 密码

# 方式 2: 在 URL 中包含用户名
git remote set-url origin http://root@gitlab.iceymoss/root/hichat.git
git push -u origin main

# 方式 3: 使用 Personal Access Token（推荐）
# 1. GitLab: Settings > Access Tokens
# 2. 创建 Token，权限：api, read_repository, write_repository
# 3. 使用 Token 作为密码
```

## 验证 CI/CD

### 1. 查看 Pipeline

1. 在 GitLab 项目页面，点击 **CI/CD > Pipelines**
2. 应该看到正在运行或已完成的 Pipeline
3. 点击 Pipeline 查看详细日志

### 2. 查看构建的镜像

1. 在 GitLab 项目页面，点击 **Packages & Registries > Container Registry**
2. 应该看到构建的 Docker 镜像

### 3. 查看 Runner 执行情况

1. 在 GitLab 项目页面，点击 **CI/CD > Pipelines**
2. 点击 Pipeline，查看每个 Job 的执行日志

## 使用构建的镜像

### 从 GitLab Registry 拉取镜像

```bash
# 登录 GitLab Registry
docker login gitlab.iceymoss
# 用户名：root
# 密码：你的 GitLab 密码

# 拉取镜像
docker pull gitlab.iceymoss/root/hichat:latest

# 导入到 k3d 集群
k3d image import gitlab.iceymoss/root/hichat:latest -c mycluster

# 更新 Deployment
kubectl set image deployment/hichat hichat=gitlab.iceymoss/root/hichat:latest -n hichat
```

## 常用管理命令

### GitLab 管理

```bash
# 启动 GitLab
sudo gitlab-ctl start

# 停止 GitLab
sudo gitlab-ctl stop

# 重启 GitLab
sudo gitlab-ctl restart

# 查看 GitLab 状态
sudo gitlab-ctl status

# 查看 GitLab 日志
sudo gitlab-ctl tail

# 重新配置 GitLab
sudo gitlab-ctl reconfigure
```

### GitLab Runner 管理

```bash
# 启动 Runner
sudo gitlab-runner start

# 停止 Runner
sudo gitlab-runner stop

# 重启 Runner
sudo gitlab-runner restart

# 查看 Runner 状态
sudo gitlab-runner status

# 查看 Runner 列表
sudo gitlab-runner list

# 查看 Runner 日志
sudo gitlab-runner --debug run
```

## 故障排查

### GitLab 无法访问

```bash
# 检查 GitLab 服务状态
sudo gitlab-ctl status

# 检查端口是否被占用
netstat -tuln | grep 80

# 查看 GitLab 日志
sudo gitlab-ctl tail
```

### Runner 无法连接

```bash
# 检查 Runner 状态
sudo gitlab-runner status

# 查看 Runner 日志
sudo gitlab-runner --debug run

# 重新注册 Runner
sudo gitlab-runner unregister --name local-runner
sudo gitlab-runner register
```

### CI/CD 构建失败

1. 检查 Runner 是否在线（GitLab 网页：Settings > CI/CD > Runners）
2. 检查 Runner 标签是否匹配（需要 `docker` 标签）
3. 查看 Pipeline 日志定位错误
4. 检查 Docker 是否运行：`docker ps`

### 推送代码失败

```bash
# 检查远程仓库配置
git remote -v

# 测试连接
git ls-remote origin

# 如果认证失败，使用 Personal Access Token
```

## 卸载 GitLab（如果需要）

```bash
# 停止 GitLab
sudo gitlab-ctl stop

# 卸载 GitLab
sudo apt-get remove gitlab-ce

# 清理数据（可选，会删除所有数据）
sudo rm -rf /etc/gitlab
sudo rm -rf /var/opt/gitlab
sudo rm -rf /var/log/gitlab
```

## 验证清单

- [ ] GitLab 已安装并运行
- [ ] 可以通过浏览器访问 GitLab
- [ ] 已获取并修改 root 密码
- [ ] hosts 文件已配置
- [ ] GitLab Runner 已安装并注册
- [ ] Runner 在 GitLab 网页显示为 "Online"
- [ ] 项目已推送到 GitLab
- [ ] CI/CD Pipeline 可以正常运行
- [ ] 镜像已推送到 Container Registry

---

**文档版本**: 1.0  
**最后更新**: 2025-11-17

