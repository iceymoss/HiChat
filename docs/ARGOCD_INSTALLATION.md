# ArgoCD 安装与配置指南

本文档详细记录在 k3d Kubernetes 集群中安装和配置 ArgoCD 的完整过程。

**文档版本**: 1.0  
**最后更新**: 2025-11-17  
**环境**: k3d (Kubernetes in Docker)

---

## 目录

1. [前置条件](#前置条件)
2. [安装 Helm](#安装-helm)
3. [安装 ArgoCD](#安装-argocd)
4. [访问 ArgoCD UI](#访问-argocd-ui)
5. [常见问题排查](#常见问题排查)
6. [下一步配置](#下一步配置)

---

## 前置条件

### 系统要求

- **Kubernetes 集群**: k3d 集群已创建并运行
- **kubectl**: 已安装并配置连接到集群
- **Helm**: 用于安装 ArgoCD（如果未安装，见下方安装步骤）

### 验证集群状态

```bash
# 检查集群连接
kubectl cluster-info

# 检查节点状态
kubectl get nodes

# 应该看到 k3d 节点运行中
```

---

## 安装 Helm

Helm 是 Kubernetes 的包管理器，用于安装 ArgoCD。

### 检查 Helm 是否已安装

```bash
helm version
```

如果已安装，会显示版本信息。如果未安装，继续以下步骤。

### 安装 Helm

```bash
# 下载 Helm 安装脚本
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# 验证安装
helm version
```

**预期输出**:
```
version.BuildInfo{Version:"v3.x.x", ...}
```

---

## 安装 ArgoCD

### 步骤 1: 添加 Argo Helm 仓库

```bash
# 添加 Argo Helm 仓库
helm repo add argo https://argoproj.github.io/argo-helm

# 更新仓库
helm repo update
```

### 步骤 2: 安装 ArgoCD

使用 Helm 安装 ArgoCD，配置适合 k3d 环境的资源设置：

```bash
helm install argocd argo/argo-cd \
  --namespace argocd \
  --create-namespace \
  --set controller.replicas=1 \
  --set server.replicas=1 \
  --set repoServer.replicas=1 \
  --set applicationSet.replicas=1 \
  --set notifications.replicas=1 \
  --set redis.replicas=1
```

**配置说明**:
- `--namespace argocd`: 在 `argocd` namespace 中安装
- `--create-namespace`: 如果 namespace 不存在则创建
- `--set controller.replicas=1`: 设置控制器副本数为 1（适合测试环境）
- `--set server.replicas=1`: 设置服务器副本数为 1
- `--set repoServer.replicas=1`: 设置仓库服务器副本数为 1
- `--set applicationSet.replicas=1`: 设置应用集控制器副本数为 1
- `--set notifications.replicas=1`: 设置通知控制器副本数为 1
- `--set redis.replicas=1`: 设置 Redis 副本数为 1

### 步骤 3: 等待 Pod 就绪

```bash
# 等待 ArgoCD Server 就绪（最多等待 5 分钟）
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server -n argocd --timeout=300s

# 检查所有 Pod 状态
kubectl get pods -n argocd
```

**预期输出**:
```
NAME                                                READY   STATUS      RESTARTS   AGE
argocd-application-controller-0                     1/1     Running     0          36s
argocd-applicationset-controller-xxx                1/1     Running     0          36s
argocd-dex-server-xxx                               1/1     Running     0          36s
argocd-notifications-controller-xxx                 1/1     Running     0          36s
argocd-redis-xxx                                    1/1     Running     0          36s
argocd-redis-secret-init-xxx                        0/1     Completed   0          41s
argocd-repo-server-xxx                              1/1     Running     0          36s
argocd-server-xxx                                    1/1     Running     0          36s
```

**注意**: `argocd-redis-secret-init-xxx` 显示 `Completed` 是正常的，这是初始化容器，任务完成后会自动退出。

### 步骤 4: 验证安装

```bash
# 检查所有服务
kubectl get svc -n argocd

# 应该看到以下服务：
# - argocd-server (主服务)
# - argocd-redis
# - argocd-repo-server
# - argocd-dex-server
```

---

## 访问 ArgoCD UI

### 方法 1: 端口转发（推荐用于本地测试）

#### 步骤 1: 获取管理员密码

```bash
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d && echo
```

**重要**: 记录这个密码，首次登录需要使用。

#### 步骤 2: 端口转发

```bash
# 使用 8443 端口（避免与 GitLab 的 8080 端口冲突）
kubectl port-forward svc/argocd-server -n argocd 8443:443
```

**注意**: 
- 如果 8443 端口被占用，可以使用其他端口，如 `9999:443`
- 端口转发会占用终端，需要保持运行

#### 步骤 3: 访问 ArgoCD UI

1. 打开浏览器访问：`https://localhost:8443`
2. 接受证书警告（ArgoCD 使用自签名证书）
3. 登录信息：
   - **用户名**: `admin`
   - **密码**: 步骤 1 获取的密码

### 方法 2: 使用 Ingress（推荐用于生产环境）

如果使用 k3d 的 Traefik Ingress，可以通过 Ingress 暴露 ArgoCD：

#### 创建 Ingress 配置

```yaml
# k8s/argocd-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argocd-ingress
  namespace: argocd
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: web
    # 如果需要 SSL 直通
    # traefik.ingress.kubernetes.io/router.tls: "true"
spec:
  ingressClassName: traefik
  rules:
  - host: argocd.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: argocd-server
            port:
              number: 80
```

#### 应用配置

```bash
# 应用 Ingress 配置
kubectl apply -f k8s/argocd-ingress.yaml

# 添加 hosts（如果使用本地域名）
echo "127.0.0.1 argocd.local" | sudo tee -a /etc/hosts

# 访问：http://argocd.local:8000（如果 k3d 映射了 8000 端口）
```

---

## 常见问题排查

### 问题 1: Pod 处于 CrashLoopBackOff 状态

**症状**:
```
NAME                                                READY   STATUS             RESTARTS   AGE
argocd-application-controller-0                     1/2     CrashLoopBackOff   6          5m
```

**可能原因**:
1. 重复安装导致配置冲突
2. Redis 连接失败
3. 资源不足

**解决方案**:

```bash
# 1. 查看 Pod 日志
kubectl logs argocd-application-controller-0 -n argocd --tail=50

# 2. 如果发现 Redis 连接问题，检查 Redis
kubectl get pods -n argocd | grep redis
kubectl get svc -n argocd | grep redis

# 3. 如果问题持续，清理并重新安装
kubectl delete namespace argocd
# 然后重新执行安装步骤
```

### 问题 2: Redis 连接失败

**错误信息**:
```
Failed to save cluster info: dial tcp 10.43.244.82:6379: connect: connection refused
```

**解决方案**:

```bash
# 1. 检查 Redis Pod 状态
kubectl get pods -n argocd | grep redis

# 2. 检查 Redis Service
kubectl get svc -n argocd | grep redis

# 3. 如果 Redis 正常，可能是网络问题，重启相关 Pod
kubectl delete pod -l app.kubernetes.io/name=argocd-application-controller -n argocd
```

### 问题 3: 端口冲突

**症状**: 端口转发失败或访问到错误的服务

**解决方案**:

```bash
# 1. 检查端口占用
netstat -tulpn | grep 8443
# 或
ss -tulpn | grep 8443

# 2. 停止旧的端口转发
pkill -f "port-forward.*argocd"

# 3. 使用不同的端口
kubectl port-forward svc/argocd-server -n argocd 9999:443
# 然后访问：https://localhost:9999
```

### 问题 4: 无法访问 UI

**检查清单**:

1. **确认端口转发正在运行**:
   ```bash
   ps aux | grep "port-forward.*argocd"
   ```

2. **确认使用 HTTPS**:
   - ArgoCD 默认使用 HTTPS
   - 访问地址应该是 `https://localhost:8443`（不是 http）

3. **接受证书警告**:
   - ArgoCD 使用自签名证书
   - 浏览器会显示安全警告，需要点击"高级" -> "继续访问"

4. **检查 Pod 状态**:
   ```bash
   kubectl get pods -n argocd
   # 确保所有 Pod 都是 Running 状态
   ```

### 问题 5: 忘记管理员密码

**解决方案**:

```bash
# 方法 1: 重新获取初始密码（如果 secret 还存在）
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d && echo

# 方法 2: 通过 ArgoCD CLI 重置密码
# 首先安装 ArgoCD CLI
curl -sSL -o argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
chmod +x argocd
sudo mv argocd /usr/local/bin/

# 登录并更新密码
argocd login localhost:8443 --insecure
argocd account update-password
```

---

## ArgoCD 配置与使用

成功安装并登录 ArgoCD 后，需要配置 Git 仓库和 Application 以实现 GitOps 工作流。

---

## 添加 Git 仓库到 ArgoCD

### 方法 1: 使用 ArgoCD CLI（推荐）

#### 步骤 1: 安装 ArgoCD CLI

```bash
# 下载 ArgoCD CLI
curl -sSL -o argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64

# 添加执行权限
chmod +x argocd

# 移动到 PATH
sudo mv argocd /usr/local/bin/

# 验证安装
argocd version
```

#### 步骤 2: 登录 ArgoCD

```bash
# 确保端口转发正在运行
kubectl port-forward svc/argocd-server -n argocd 8443:443 &

# 登录 ArgoCD（使用 --insecure 因为使用自签名证书）
argocd login localhost:8443 --username admin --password <ADMIN_PASSWORD> --insecure
```

#### 步骤 3: 添加 Git 仓库

目前部署使用 GitHub 上的公共仓库 `iceymoss/HiChat`，ArgoCD 可以直接通过 HTTPS 访问，无需再解析本地域名或指定网关 IP。

```bash
# 直接添加 GitHub 仓库（公开仓库无需凭证）
argocd repo add https://github.com/iceymoss/HiChat.git

# 如果后续改为私有仓库，可改用以下命令并提供 PAT
# argocd repo add https://github.com/iceymoss/HiChat.git \
#   --username <GITHUB_USER> \
#   --password '<PERSONAL_ACCESS_TOKEN>'
```

#### 步骤 4: 验证仓库添加成功

```bash
# 列出所有仓库
argocd repo list

# 应该看到类似输出：
# TYPE  NAME  REPO                                INSECURE  STATUS      MESSAGE
# git         https://github.com/iceymoss/HiChat.git  false  Successful
```

### 方法 2: 在 ArgoCD UI 中添加

1. 打开 ArgoCD UI：`https://localhost:8443`
2. 点击左侧菜单 **Settings** → **Repositories**
3. 点击 **Connect Repo**
4. 填写仓库信息：
   - **Type**: Git
   - **Repository URL**: `https://github.com/iceymoss/HiChat.git`
   - **Username/Password**: 公开仓库可留空；若改为私有，则填入 GitHub 账号与 PAT
5. 点击 **Connect**

---

## 创建 ArgoCD Application

ArgoCD Application 定义了要部署的应用、源仓库和目标集群。

### 创建 Application 配置文件

创建 `k8s/argocd-application.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hichat
  namespace: argocd
  # 添加 finalizers 以支持级联删除
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  
  # 源仓库配置
  source:
    repoURL: https://github.com/iceymoss/HiChat.git
    targetRevision: master  # 监控 master 分支
    path: k8s  # k8s 配置文件所在目录
  
  # 目标集群和命名空间
  destination:
    server: https://kubernetes.default.svc
    namespace: hichat
  
  # 同步策略
  syncPolicy:
    # 自动同步（当 Git 仓库有变化时自动同步）
    automated:
      prune: true      # 自动删除 Git 中不存在的资源
      selfHeal: true   # 自动修复集群中的配置漂移
    # 同步选项
    syncOptions:
      - CreateNamespace=true  # 如果命名空间不存在则创建
    # 保留历史记录（用于回滚）
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

### 应用 Application 配置

```bash
# 应用 ArgoCD Application
kubectl apply -f k8s/argocd-application.yaml

# 查看 Application 状态
argocd app get hichat

# 或者在 ArgoCD UI 中查看
# 访问 https://localhost:8443，点击 Applications
```

### 验证 Application 同步

```bash
# 使用 CLI 查看
argocd app list
argocd app get hichat

# 查看同步状态
argocd app sync hichat

# 查看应用资源
argocd app resources hichat
```

**预期输出**:
```
Name:               argocd/hichat
Project:            default
Server:             https://kubernetes.default.svc
Namespace:          hichat
Sync Status:        Synced to master (xxxxxxx)
Health Status:      Healthy

GROUP              KIND                   STATUS   HEALTH
                   Namespace              Synced
apps               Deployment             Synced   Healthy
                   Service                Synced   Healthy
...
```

---

## 手动推送 GitHub `k8s-config` 分支（无需 CI/CD）

目前我们直接手动维护 GitHub 仓库 `iceymoss/HiChat` 的 `k8s-config` 分支即可触发 ArgoCD，同步流程如下：

1. **确认远程为 GitHub**  
   ```bash
   git remote -v
   # origin  https://github.com/iceymoss/HiChat.git
   ```
2. **切换到配置分支并同步最新内容**  
   ```bash
   git checkout master && git pull
   git checkout k8s-config
   git checkout master -- k8s/
   ```
3. **编辑需要变更的清单**  
   - 例如把 `k8s/hichat.yaml` 的镜像标签手动改成最新版本；
   - 或者新增/删除其它 Kubernetes 资源。
4. **提交并推送到 GitHub**  
   ```bash
   git add k8s/
   git commit -m "chore: update manifests"
   git push origin k8s-config
   ```
5. **等待 ArgoCD 自动同步**（也可以执行 `argocd app sync hichat` 手动立即同步）。

> 只要 `k8s-config` 分支成功推送到 GitHub，ArgoCD 会把变更部署到集群，因此即使没有 CI/CD 流水线也能维持完整的 GitOps 回路。

---

## 配置 CI/CD 自动更新镜像标签

为了实现完整的 GitOps 流程，需要在 CI/CD Pipeline 中添加自动更新 Kubernetes 配置的步骤。

### 更新 `.gitlab-ci.yml`

在现有的 `build` 阶段后添加 `update-config` 阶段：

```yaml
stages:
  - build
  - update-config  # 新增：更新 Kubernetes 配置阶段

# ... build 阶段保持不变 ...

# 更新 Kubernetes 配置中的镜像标签
update-config:
  stage: update-config
  image: alpine/git:latest
  before_script:
    - git config --global user.email "ci@gitlab.iceymoss"
    - git config --global user.name "GitLab CI"
  script:
    - echo "更新 Kubernetes 配置文件中的镜像标签..."
    # 更新 k8s/hichat.yaml 中的镜像标签为 commit SHA
    - |
      sed -i "s|image:.*hichat.*|image: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA|g" k8s/hichat.yaml
    # 显示更新后的内容（用于调试）
    - echo "更新后的镜像配置:"
    - grep "image:" k8s/hichat.yaml
    # 提交更改
    - git add k8s/hichat.yaml
    - git commit -m "chore: update image to $CI_COMMIT_SHORT_SHA [skip ci]" || echo "没有变化，跳过提交"
    # 推送到 Git 仓库（使用 CI Token）
    - |
      git push https://oauth2:${CI_JOB_TOKEN}@gitlab.iceymoss/root/hichat.git HEAD:master || \
      echo "推送失败或没有变化"
    - echo "配置更新完成，ArgoCD 将自动同步部署"
  only:
    - master  # 只在 master 分支执行
  when: on_success  # 只在 build 阶段成功后才执行
  tags:
    - docker
```

### 配置说明

- **stage**: `update-config` - 在构建完成后执行
- **sed 命令**: 更新 `k8s/hichat.yaml` 中的镜像标签为 commit SHA
- **git commit**: 提交更改，使用 `[skip ci]` 避免触发新的 Pipeline
- **git push**: 使用 CI Token 推送到 Git 仓库
- **when: on_success**: 只在构建成功后才执行

---

## 完整的 GitOps 工作流

### 流程概览

```
开发者推送代码到 GitLab
    ↓
GitLab CI/CD 触发 Pipeline
    ↓
构建阶段 (build):
  - 构建 Docker 镜像
  - 推送到 GitLab Container Registry
    ↓
更新配置阶段 (update-config):
  - 更新 k8s/hichat.yaml 中的镜像标签
  - 提交更改到 Git 仓库
    ↓
ArgoCD 检测到 Git 仓库变化
    ↓
ArgoCD 自动同步到 Kubernetes 集群
    ↓
应用自动更新部署（滚动更新）
```

### 工作流特点

✅ **自动化**: 从代码推送到部署完全自动化  
✅ **可追溯**: 每个部署都有对应的 Git commit  
✅ **可回滚**: 可以通过 Git 回滚到之前的版本  
✅ **一致性**: Git 仓库是唯一的事实来源  
✅ **自愈**: ArgoCD 自动修复配置漂移

---

## 测试完整 GitOps 流程

### 步骤 1: 推送代码

```bash
# 提交更新的 CI/CD 配置
git add .gitlab-ci.yml k8s/argocd-application.yaml
git commit -m "Add GitOps: ArgoCD integration"
git push origin master
```

### 步骤 2: 观察 CI/CD Pipeline

1. 在 GitLab 网页查看 **CI/CD > Pipelines**
2. 应该看到两个阶段：
   - `build`: 构建并推送镜像
   - `update-config`: 更新 Kubernetes 配置

### 步骤 3: 观察 ArgoCD 自动同步

1. 打开 ArgoCD UI：`https://localhost:8443`
2. 点击 `hichat` 应用
3. 应该看到：
   - 自动检测到 Git 变化
   - 自动同步部署
   - 新镜像部署到集群

### 步骤 4: 验证部署

```bash
# 查看 Pod 状态
kubectl get pods -n hichat

# 查看 Deployment 使用的镜像
kubectl get deployment hichat -n hichat -o jsonpath='{.spec.template.spec.containers[0].image}'

# 应该看到新的 commit SHA 标签，例如：
# gitlab.iceymoss:5050/root/hichat:832ee3e
```

---

## ArgoCD 常用命令

### Application 管理

```bash
# 列出所有应用
argocd app list

# 获取应用详情
argocd app get <app-name>

# 同步应用
argocd app sync <app-name>

# 查看应用资源
argocd app resources <app-name>

# 查看应用历史
argocd app history <app-name>

# 回滚应用
argocd app rollback <app-name> <revision>
```

### Repository 管理

```bash
# 列出所有仓库
argocd repo list

# 添加仓库
argocd repo add <repo-url> --username <user> --password <pass> --insecure

# 删除仓库
argocd repo rm <repo-url>
```

### 同步操作

```bash
# 手动同步应用
argocd app sync hichat

# 同步并等待完成
argocd app sync hichat --wait

# 强制同步（忽略差异）
argocd app sync hichat --force
```

---

## ArgoCD 配置常见问题

### 问题 1: 无法添加 Git 仓库 - DNS 解析失败

**错误信息示例**:
```
lookup gitlab.iceymoss on 10.43.0.10:53: no such host
```

**原因**: 集群内部无法解析本地域名，或无法访问内网 Git 服务。

**解决方案**:

- 优先使用公共可访问的 GitHub 仓库（默认已切换为 `https://github.com/iceymoss/HiChat.git`）：
  ```bash
  argocd repo add https://github.com/iceymoss/HiChat.git
  ```
- 如果必须使用内网 GitLab/Gitea，可以按需获取网关 IP 并使用 `http://<IP>/<repo>.git` 方式添加。

### 问题 2: Application 同步失败

**检查清单**:

1. **确认仓库已添加**:
   ```bash
   argocd repo list
   ```

2. **检查仓库连接**:
   ```bash
   argocd repo get https://github.com/iceymoss/HiChat.git
   ```

3. **查看 Application 详情**:
   ```bash
   argocd app get hichat
   ```

4. **查看同步日志**:
   ```bash
   argocd app logs hichat
   ```

### 问题 3: CI/CD 更新配置失败

**可能原因**:
- Git 推送权限不足
- 仓库地址错误
- CI Token 无效

**解决方案**:

```bash
# 检查 .gitlab-ci.yml 中的推送命令
# 确保使用正确的仓库地址和 CI Token

# 如果使用域名，确保 CI/CD Runner 能解析
# 如果使用 IP，确保 IP 地址正确
```

### 问题 4: 镜像拉取失败

**错误信息**: `ImagePullBackOff` 或 `ErrImagePull`

**解决方案**:

```bash
# 1. 确保镜像已推送到 Registry
docker pull gitlab.iceymoss:5050/root/hichat:latest

# 2. 导入镜像到 k3d 集群
k3d image import gitlab.iceymoss:5050/root/hichat:latest -c mycluster

# 3. 或者配置镜像拉取密钥（如果 Registry 需要认证）
kubectl create secret docker-registry gitlab-registry-secret \
  --docker-server=gitlab.iceymoss:5050 \
  --docker-username=root \
  --docker-password='PASSWORD' \
  -n hichat
```

---

## GitOps 最佳实践

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

通过以上配置，你已经实现了完整的 GitOps 工作流：

1. ✅ **GitLab CI/CD**: 自动构建和推送镜像
2. ✅ **自动更新配置**: CI/CD 自动更新 Kubernetes 配置
3. ✅ **ArgoCD 自动同步**: 自动检测并部署到集群
4. ✅ **可追溯和可回滚**: 所有变更都在 Git 中记录

现在，每次推送代码到 master 分支，整个流程都会自动执行，实现真正的 GitOps！

---

## 完整安装脚本

一键安装脚本：

```bash
#!/bin/bash

echo "=== 安装 ArgoCD ==="

# 1. 检查 Helm
if ! command -v helm &> /dev/null; then
    echo "安装 Helm..."
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

# 2. 添加仓库
echo "添加 Argo Helm 仓库..."
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update

# 3. 安装 ArgoCD
echo "安装 ArgoCD..."
helm install argocd argo/argo-cd \
  --namespace argocd \
  --create-namespace \
  --set controller.replicas=1 \
  --set server.replicas=1 \
  --set repoServer.replicas=1 \
  --set applicationSet.replicas=1 \
  --set notifications.replicas=1 \
  --set redis.replicas=1

# 4. 等待就绪
echo "等待 ArgoCD 启动（这可能需要 1-2 分钟）..."
sleep 60

# 5. 检查状态
echo ""
echo "=== ArgoCD Pod 状态 ==="
kubectl get pods -n argocd

# 6. 获取密码
echo ""
echo "=== ArgoCD 管理员密码 ==="
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d && echo

# 7. 访问说明
echo ""
echo "=== 访问 ArgoCD ==="
echo "执行以下命令进行端口转发："
echo "kubectl port-forward svc/argocd-server -n argocd 8443:443"
echo ""
echo "然后访问：https://localhost:8443"
echo "用户名：admin"
echo "密码：上面显示的密码"
```

---

## 验证安装

安装完成后，验证以下内容：

### 1. 所有 Pod 运行正常

```bash
kubectl get pods -n argocd
# 所有 Pod 应该都是 Running 状态（除了 Completed 的 init 容器）
```

### 2. 服务正常

```bash
kubectl get svc -n argocd
# 应该看到所有 ArgoCD 服务
```

### 3. 可以访问 UI

- 能够通过端口转发访问 ArgoCD UI
- 能够使用 admin 账号登录
- UI 界面正常显示

---

## 常用命令

### 管理 ArgoCD

```bash
# 查看 ArgoCD 状态
kubectl get pods -n argocd

# 查看 ArgoCD 服务
kubectl get svc -n argocd

# 查看 ArgoCD 日志
kubectl logs -f deployment/argocd-server -n argocd

# 重启 ArgoCD Server
kubectl rollout restart deployment/argocd-server -n argocd
```

### 卸载 ArgoCD

```bash
# 卸载 ArgoCD
helm uninstall argocd -n argocd

# 删除 namespace（可选，会删除所有数据）
kubectl delete namespace argocd
```

---

## 参考资源

- [ArgoCD 官方文档](https://argo-cd.readthedocs.io/)
- [ArgoCD Helm Chart](https://github.com/argoproj/argo-helm)
- [GitOps 最佳实践](https://argo-cd.readthedocs.io/en/stable/user-guide/best_practices/)

---

**文档维护**: 如有问题或更新，请及时更新本文档。

