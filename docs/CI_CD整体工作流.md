# CI/CD + GitOps 部署工作流程

本文档详细描述 HiChat 项目的完整 CI/CD + GitOps 自动化部署流程，从代码提交到 Kubernetes 集群自动更新的全流程。

**文档版本**: 2.0  
**最后更新**: 2025-11-18  
**环境**: WSL2 (Windows Subsystem for Linux) + k3d Kubernetes 集群  
**分支策略**: 配置分支方案（代码分支 `master` + 配置分支 `k8s-config`）

---

## 目录

1. [系统架构](#系统架构)
2. [分支策略说明](#分支策略说明)
3. [完整流程概览](#完整流程概览)
4. [各阶段详细说明](#各阶段详细说明)
5. [关键配置文件](#关键配置文件)
6. [数据流向](#数据流向)
7. [初始配置步骤](#初始配置步骤)
8. [验证清单](#验证清单)

---

## 系统架构

### 架构组件

```
┌─────────────────────────────────────────────────────────────┐
│                    WSL2 环境                                │
│                                                             │
│  ┌──────────────────┐    ┌──────────────────┐             │
│  │   GitLab         │    │   GitLab Runner  │             │
│  │   - Web UI       │───▶│   - Docker Exec  │             │
│  │   - Registry     │    │   - 执行 CI/CD   │             │
│  │   - CI/CD        │    └──────────────────┘             │
│  └──────────────────┘                                      │
│         │                                                   │
│         │ 推送镜像                                          │
│         ▼                                                   │
│  ┌──────────────────┐                                      │
│  │ Container        │                                      │
│  │ Registry         │                                      │
│  │ :5050            │                                      │
│  └──────────────────┘                                      │
│                                                             │
│  ┌──────────────────┐    ┌──────────────────┐             │
│  │   k3d 集群       │    │   ArgoCD         │             │
│  │   - HiChat Pods  │◀───│   - GitOps 同步  │             │
│  │   - MySQL        │    │   - 自动部署     │             │
│  │   - Redis        │    │   - 监控 Git     │             │
│  └──────────────────┘    └──────────────────┘             │
└─────────────────────────────────────────────────────────────┘
```

### 组件说明

1. **GitLab** (`http://gitlab.iceymoss`)
   - Git 代码仓库
   - CI/CD Pipeline 管理
   - Container Registry (`http://gitlab.iceymoss:5050 或者其他的例如：docker hub`)

2. **GitLab Runner** (WSL 本地)
   - 执行 CI/CD 任务
   - 使用 Docker executor
   - 支持 Docker-in-Docker (dind)

3. **k3d Kubernetes 集群**
   - 命名空间: `hichat`
   - 运行组件: HiChat 应用、MySQL、Redis
   - Ingress: Traefik (k3d 默认)

4. **ArgoCD** (k3d 集群内)
   - 监控 Git 仓库: `http://172.21.0.1/root/hichat.git`
   - 监控分支: `k8s-config`（配置分支）
   - 监控路径: `k8s/`
   - 自动同步到 Kubernetes 集群

---

## 分支策略说明

### 配置分支方案（推荐）

为了保持代码仓库的 Git 历史干净，我们采用**配置分支方案**：

- **`master` 分支**：包含源代码和业务逻辑
  - 开发者推送代码到此分支
  - Git 历史只包含业务相关的 commit
  - 不包含配置更新的自动化 commit

- **`k8s-config` 分支**：包含 Kubernetes 配置文件
  - CI/CD 自动更新此分支的配置文件
  - 包含所有配置更新的 commit（如 `chore: update image to xxx [skip ci]`）
  - ArgoCD 监控此分支的变化

### 分支关系

```
master 分支（代码）
  │
  │ git push origin master
  ▼
GitLab CI/CD Pipeline
  │
  ├─▶ build 阶段：构建镜像
  │
  └─▶ update-config 阶段：更新 k8s-config 分支
        │
        │ git push origin k8s-config
        ▼
    k8s-config 分支（配置）
        │
        │ ArgoCD 监控
        ▼
    Kubernetes 集群部署
```

### 优势

✅ **Git 历史清晰**：master 分支只包含业务代码，不包含配置更新  
✅ **可追溯性**：所有配置变更都在 `k8s-config` 分支中记录  
✅ **可回滚**：可以通过 Git 回滚配置到任意版本  
✅ **符合最佳实践**：主流 GitOps 方案

---

## 完整流程概览

### 流程图

```
开发者
  │
  │ git push origin master
  ▼
GitLab 仓库 (master 分支)
  │
  │ 触发 Pipeline（检测到 .gitlab-ci.yml）
  ▼
GitLab CI/CD Pipeline
  │
  ├─▶ Stage 1: build ✅
  │     │
  │     ├─▶ Runner 启动 docker:24 容器
  │     ├─▶ 启动 docker:24-dind 服务
  │     ├─▶ 登录 GitLab Registry (这里也可以使用docker hub)
  │     ├─▶ 执行 Dockerfile 多阶段构建
  │     ├─▶ 打三个标签（master, latest, commit SHA）
  │     └─▶ 推送所有标签到 Registry
  │
  └─▶ Stage 2: update-config ✅
        │
        ├─▶ 检出 k8s-config 分支
        ├─▶ 更新 k8s/hichat.yaml 镜像标签
        ├─▶ git commit（使用 [skip ci]）
        └─▶ git push origin k8s-config
              │
              ▼
          GitLab 仓库 (k8s-config 分支更新)
              │
              │ ArgoCD 检测到变化（每 3 分钟轮询或 Webhook）
              ▼
          ArgoCD 自动同步
              │
              ▼
          Kubernetes 集群
              │
              ├─▶ 拉取新镜像（从 Registry）
              ├─▶ 更新 Deployment
              └─▶ 滚动更新 Pods
                    │
                    ▼
                新版本运行
```

### 关键时间点

| 时间点 | 操作 | 分支 | 说明 |
|--------|------|------|------|
| T0 | 开发者推送代码 | `master` | 触发 Pipeline |
| T1 | CI/CD 构建镜像 | - | 推送到 Registry |
| T2 | CI/CD 更新配置 | `k8s-config` | 更新镜像标签并推送 |
| T3 | ArgoCD 检测变化 | `k8s-config` | 轮询或 Webhook 触发 |
| T4 | ArgoCD 同步部署 | - | 更新 Kubernetes 集群 |
| T5 | Pod 滚动更新 | - | 新版本运行 |

### 流程特点

✅ **完全自动化**: 从代码推送到部署完全自动化  
✅ **可追溯**: 每个部署都有对应的 Git commit  
✅ **可回滚**: 可以通过 Git 回滚到之前的版本  
✅ **一致性**: Git 仓库是唯一的事实来源  
✅ **自愈**: ArgoCD 自动修复配置漂移

---

## 各阶段详细说明

### 阶段 1: 代码提交与触发

**触发条件**：
- 开发者执行 `git push origin master`
- GitLab 检测到 `.gitlab-ci.yml` 文件
- 自动创建 Pipeline

**当前状态**：✅ 已配置，正常工作

---

### 阶段 2: 构建阶段 (build)

**执行流程**：

1. **环境准备**
   - GitLab Runner 启动 Docker 容器（`docker:24`）
   - 启动 Docker-in-Docker 服务（`docker:24-dind`）
   - 配置 insecure-registry: `gitlab.iceymoss:5050`

2. **登录 Registry**
   - 使用 GitLab CI Token (`$CI_JOB_TOKEN`)
   - 或使用 GitLab 提供的认证信息

3. **构建 Docker 镜像**
   - **Builder 阶段**: 使用 `golang:1.17-alpine` 编译 Go 应用
     - 复制 `go.mod`, `go.sum`
     - 执行 `go mod download`
     - 复制源代码
     - 编译生成 `main` 二进制文件
   - **Runtime 阶段**: 使用 `alpine:latest` 准备运行环境
     - 安装 ca-certificates, tzdata
     - 复制二进制文件: `main`
     - 复制配置文件: `config-debug.yaml`
     - 复制静态资源: `index.html`, `views/`, `asset/`

4. **标记镜像**
   为镜像打上三个标签：
   - `$CI_REGISTRY_IMAGE:master` (分支名)
   - `$CI_REGISTRY_IMAGE:latest` (最新版本)
   - `$CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA` (commit SHA，如 `24f307d6`)

5. **推送镜像**
   依次推送到 GitLab Container Registry：
   ```
   gitlab.iceymoss:5050/root/hichat:master
   gitlab.iceymoss:5050/root/hichat:latest
   gitlab.iceymoss:5050/root/hichat:24f307d6
   ```

**当前状态**：✅ 已成功运行

**验证方法**：
- 在 GitLab 项目页面查看 **Packages & Registries > Container Registry**
- 应该看到三个标签的镜像

---

### 阶段 3: 更新配置阶段 (update-config)

**目标**：
- 更新 `k8s-config` 分支中的 `k8s/hichat.yaml` 镜像标签
- 从 `latest` 改为 commit SHA（如 `24f307d6`）
- 提交更改到配置分支
- 触发 ArgoCD 自动同步

**执行流程**：

1. **检出配置分支**
   ```bash
   git fetch origin k8s-config:k8s-config || git checkout -b k8s-config
   git checkout k8s-config
   ```
   确保配置分支存在，如果不存在则创建

2. **同步 master 分支的 k8s 目录**
   ```bash
   git checkout master -- k8s/
   ```
   确保配置分支的 k8s 目录与 master 分支同步（除了镜像标签）

3. **更新配置文件**
   ```bash
   sed -i "s|image:.*hichat.*|image: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA|g" k8s/hichat.yaml
   ```
   将镜像标签从 `latest` 更新为 commit SHA

4. **提交更改**
   ```bash
   git add k8s/hichat.yaml
   git commit -m "chore: update image to $CI_COMMIT_SHORT_SHA [skip ci]" || exit 0
   ```
   使用 `[skip ci]` 避免触发新的 Pipeline
   - 如果文件没有变化（镜像标签相同），`|| exit 0` 允许继续执行

5. **推送到配置分支**
   ```bash
   git push https://oauth2:$CI_JOB_TOKEN@gitlab.iceymoss/root/hichat.git HEAD:k8s-config || exit 0
   ```
   使用 CI Token 推送到 `k8s-config` 分支（不是 master）

**关键点**：
- ✅ 配置更新在 `k8s-config` 分支，不在 `master` 分支
- ✅ `master` 分支保持干净，只包含业务代码
- ✅ 使用 `[skip ci]` 避免循环触发 Pipeline
- ✅ 使用 `|| exit 0` 处理无变化的情况

**当前状态**：✅ 已实现（使用配置分支方案）

---

### 阶段 4: ArgoCD 自动同步

**执行流程**：

1. **监控 Git 仓库**
   - ArgoCD 每 3 分钟轮询 Git 仓库（默认）
   - 或通过 Webhook 立即触发（推荐配置）
   - 监控分支: `k8s-config`（配置分支）
   - 监控路径: `k8s/`

2. **检测变化**
   - 检测到 `k8s-config` 分支的 `k8s/hichat.yaml` 文件变化
   - 比较 Git 仓库与集群中的配置差异
   - 识别镜像标签变化（如 `latest` → `24f307d6`）

3. **自动同步**
   - 拉取新的配置（从 `k8s-config` 分支）
   - 更新 Kubernetes Deployment
   - 使用新的镜像标签（commit SHA）

4. **滚动更新**
   - Kubernetes 执行滚动更新策略
   - 从 Registry 拉取新镜像
   - 创建新 Pod（使用新镜像）
   - 等待新 Pod 就绪（健康检查通过）
   - 终止旧 Pod
   - 完成更新

**触发机制**：

- **默认方式**：每 3 分钟轮询一次
- **推荐方式**：配置 GitLab Webhook，实现立即触发
  - Webhook URL: `https://argocd-server.argocd.svc.cluster.local:443/api/webhook`
  - 触发事件: Push events
  - 分支: `k8s-config`

**当前状态**：✅ 已配置（监控 `k8s-config` 分支）

**配置位置**：`k8s/argocd-application.yaml`

**验证方法**：
- 在 ArgoCD UI (`https://localhost:8443`) 查看应用状态
- 应该看到自动同步和部署过程
- 查看 `k8s-config` 分支的 commit 历史

---

## 关键配置文件

### 1. `.gitlab-ci.yml`

**当前配置**：
- `stages`: `build`, `update-config`
- `build` job: ✅ 正常工作
- `update-config` ✅ 正常工作

**关键变量**：
```yaml
DOCKERHUB_PASSWORD: docker hub密码
DOCKERHUB_USERNAME: docker hub用户名
DOCKER_REGISTRY_IMAGE: docker hub镜像
PERSONAL_ACCESS_TOKEN: gitlab token
```

---

### 2. `Dockerfile`

**多阶段构建**：

```dockerfile
# Builder 阶段
FROM golang:1.17-alpine AS builder
- 编译 Go 应用
- 生成 main 二进制文件

# Runtime 阶段
FROM alpine:latest
- 安装运行时依赖
- 复制二进制文件、配置文件、静态资源
```

**输出镜像**：
- 轻量级 Alpine 基础镜像
- 包含完整的应用文件

---

### 3. `k8s/hichat.yaml`

**当前配置**：
```yaml
spec:
  containers:
  - name: hichat
    image: gitlab.iceymoss:5050/root/hichat:latest  # ⚠️ 固定为 latest,在 ci 中更新为 commit SHA
```

**更新后**（通过 update-config job）：
```yaml
image: gitlab.iceymoss:5050/root/hichat:24f307d6  # commit SHA
```

---

### 4. `k8s/argocd-application.yaml`

**配置要点**：
```yaml
spec:
  source:
    repoURL: http://172.21.0.1/root/hichat.git
    targetRevision: k8s-config  # ⚠️ 监控配置分支，不是 master
    path: k8s
  destination:
    server: https://kubernetes.default.svc
    namespace: hichat
  syncPolicy:
    automated:
      prune: true      # 自动删除 Git 中不存在的资源
      selfHeal: true   # 自动修复配置漂移
```

**功能**：
- 监控 Git 仓库的 `k8s-config` 分支（配置分支）
- 监控 `k8s/` 目录下的配置文件
- 自动同步到 `hichat` 命名空间
- 自动修复配置漂移

**重要**：
- ✅ `targetRevision: k8s-config` 确保 ArgoCD 监控配置分支
- ✅ 配置更新不会出现在 `master` 分支的 Git 历史中

---

### 5. 其他 k8s 配置文件

- **`k8s/namespace.yaml`**: 创建 `hichat` 命名空间
- **`k8s/mysql.yaml`**: MySQL 部署（包含 PVC、ConfigMap、初始化 SQL）
- **`k8s/redis.yaml`**: Redis 部署
- **`k8s/ingress.yaml`**: Traefik Ingress 配置（域名: `hichat.local`）

---

## 数据流向

### 完整数据流

```
1. 代码变更
   │
   │ git push
   ▼
2. GitLab 仓库
   │
   │ 触发 Pipeline
   ▼
3. GitLab CI/CD
   │
   ├─▶ 构建镜像 → Registry
   │
   └─▶ 更新 k8s/hichat.yaml → Git 仓库
         │
         │ ArgoCD 检测
         ▼
4. ArgoCD
   │
   │ 同步配置
   ▼
5. Kubernetes 集群
   │
   ├─▶ 拉取新镜像 (Registry)
   │
   └─▶ 更新 Deployment → 滚动更新 Pods
```

### 镜像流向

```
Dockerfile 构建
  │
  ▼
本地构建镜像
  │
  ▼
GitLab Container Registry
  docker hub
  │
  │ Kubernetes 拉取
  ▼
k3d 集群
  │
  ▼
HiChat Pods 运行
```

### 配置流向

```
k8s/hichat.yaml (k8s-config 分支)
  │
  │ ArgoCD 同步（监控 k8s-config 分支）
  ▼
Kubernetes Deployment
  │
  ▼
Pod 配置（镜像标签、环境变量等）
```

### 分支流向

```
master 分支（代码）
  │
  │ 开发者推送
  ▼
GitLab 仓库 (master)
  │
  │ CI/CD Pipeline 触发
  ▼
构建镜像 → Registry
  │
  │ update-config job
  ▼
k8s-config 分支（配置）
  │
  │ ArgoCD 监控
  ▼
Kubernetes 集群部署
```

---

## 初始配置步骤

### 步骤 1: 创建配置分支

```bash
# 1. 确保在 master 分支
git checkout master
git pull origin master

# 2. 创建并切换到配置分支
git checkout -b k8s-config

# 3. 推送配置分支到远程
git push origin k8s-config

# 4. 切换回 master 分支
git checkout master
```

### 步骤 2: 更新 ArgoCD Application 配置

```bash
# 编辑 ArgoCD Application 配置
# 修改 k8s/argocd-application.yaml 中的 targetRevision: k8s-config

# 应用配置
kubectl apply -f k8s/argocd-application.yaml

# 验证配置
argocd app get hichat
# 应该显示监控 k8s-config 分支
```

### 步骤 3: 更新 CI/CD 配置

```bash
# 编辑 .gitlab-ci.yml
# update-config job 已经配置为推送到 k8s-config 分支

# 提交并推送
git add .gitlab-ci.yml k8s/argocd-application.yaml
git commit -m "feat: 使用配置分支方案分离代码和配置"
git push origin master
```

### 步骤 4: 验证配置

```bash
# 1. 查看 master 分支（应该只包含业务代码）
git checkout master
git log --oneline
# 应该看不到配置更新的 commit

# 2. 查看配置分支（应该包含配置更新）
git checkout k8s-config
git log --oneline
# 应该看到配置更新的 commit（如果有的话）

# 3. 查看 ArgoCD 状态
argocd app get hichat
# 应该显示监控 k8s-config 分支
```

---

## 测试完整流程

### 步骤 1: 推送代码

```bash
# 修改代码
git add .
git commit -m "test: trigger CI/CD pipeline"
git push origin master
```

### 步骤 2: 观察 CI/CD Pipeline

1. 在 GitLab 项目页面，点击 **CI/CD > Pipelines**
2. 应该看到 Pipeline 正在运行
3. 查看 build 阶段的日志
4. 验证镜像已推送到 Registry

### 步骤 3: 验证镜像

```bash
# 查看 Registry 中的镜像
# 在 GitLab 项目页面: Packages & Registries > Container Registry
# 应该看到三个标签的镜像
```

### 步骤 4: 观察 ArgoCD 同步

1. 打开 ArgoCD UI：`https://localhost:8443`
2. 点击 `hichat` 应用
3. 应该看到：
   - 自动检测到 `k8s-config` 分支的变化
   - 自动同步部署
   - 新镜像部署到集群

**注意**：
- ArgoCD 监控的是 `k8s-config` 分支，不是 `master` 分支
- 配置更新在 `k8s-config` 分支中，可以通过 `git checkout k8s-config` 查看

### 步骤 5: 验证部署

```bash
# 查看 Pod 状态
kubectl get pods -n hichat

# 查看 Deployment 使用的镜像
kubectl get deployment hichat -n hichat -o jsonpath='{.spec.template.spec.containers[0].image}'

# 应该看到新的 commit SHA 标签，例如：
# gitlab.iceymoss:5050/root/hichat:24f307d6
```

---

## 故障排查

### 问题 1: Pipeline 构建失败

**检查**：
- GitLab Runner 是否在线
- Docker 服务是否运行
- Registry 是否可访问

**解决**：
```bash
# 检查 Runner 状态
sudo gitlab-runner status

# 检查 Docker
docker ps

# 检查 Registry
curl -I http://gitlab.iceymoss:5050/v2/
```

### 问题 2: 镜像推送失败

**检查**：
- Registry 是否启用
- 认证信息是否正确
- insecure-registry 配置是否正确

**解决**：
```bash
# 检查 Registry 服务
sudo gitlab-ctl status | grep registry

# 手动测试登录
docker login gitlab.iceymoss:5050
```

### 问题 3: ArgoCD 未同步

**检查**：
- ArgoCD Application 状态
- Git 仓库连接是否正常
- 同步策略是否正确

**解决**：
```bash
# 查看 Application 状态
argocd app get hichat

# 手动触发同步
argocd app sync hichat

# 查看同步日志
argocd app logs hichat
```

### 问题 4: Pod 无法拉取镜像

**检查**：
- 镜像是否存在于 Registry
- k3d 集群是否能访问 Registry
- imagePullPolicy 配置

**解决**：
```bash
# 检查镜像是否存在
# 在 GitLab Registry 页面查看

# 手动导入镜像到 k3d
k3d image import gitlab.iceymoss:5050/root/hichat:24f307d6 -c mycluster

# 检查 Pod 事件
kubectl describe pod <pod-name> -n hichat
```

---

## 最佳实践

### 1. 使用 commit SHA 作为镜像标签

- ✅ 可追溯：每个镜像对应一个 commit
- ✅ 唯一性：避免标签冲突
- ✅ 回滚：可以精确回滚到特定版本

### 2. 启用自动同步（仅测试环境）

- ✅ 测试环境：启用 `automated.syncPolicy`
- ❌ 生产环境：建议使用手动同步或审批流程

### 3. 使用 `[skip ci]` 避免循环触发

在 CI/CD 更新配置的 commit 消息中使用 `[skip ci]`，避免触发新的 Pipeline。

### 4. 定期清理旧镜像

定期清理 GitLab Registry 中的旧镜像，节省存储空间。

---

## 总结

通过以上配置，实现了完整的 GitOps 工作流：

1. ✅ **GitLab CI/CD**: 自动构建和推送镜像
2. ✅ **自动更新配置**: CI/CD 自动更新 `k8s-config` 分支的 Kubernetes 配置
3. ✅ **ArgoCD 自动同步**: 自动检测 `k8s-config` 分支变化并部署到集群
4. ✅ **可追溯和可回滚**: 所有配置变更都在 `k8s-config` 分支中记录
5. ✅ **Git 历史清晰**: `master` 分支保持干净，只包含业务代码

### 工作流特点

- **代码与配置分离**：使用配置分支方案，代码和配置在 Git 中分离
- **完全自动化**：从代码推送到部署完全自动化
- **可追溯性**：每个部署都有对应的 Git commit（在配置分支中）
- **可回滚**：可以通过 Git 回滚配置到任意版本
- **符合最佳实践**：采用主流 GitOps 配置分支方案

### 分支说明

- **`master` 分支**：包含源代码，Git 历史干净
- **`k8s-config` 分支**：包含 Kubernetes 配置，包含所有配置更新的 commit

### 下一步

1. 执行初始配置步骤，创建 `k8s-config` 分支
2. 更新 ArgoCD Application 监控配置分支
3. 测试完整的 GitOps 流程
4. （可选）配置 GitLab Webhook 实现立即触发

---

**文档版本**: 2.0  
**最后更新**: 2025-11-18  
**分支策略**: 配置分支方案（master + k8s-config）

