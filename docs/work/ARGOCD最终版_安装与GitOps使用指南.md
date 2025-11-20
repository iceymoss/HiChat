## ArgoCD 最终版：安装与 GitOps 使用指南（基于 k3d集群）

本文是对现有 `ARGOCD安装及其使用.md` 与 CI/CD、GitOps 相关文档的精简整合版，结合当前 `k8s/argocd-application.yaml` 的实际配置，给出一条从“零安装”到“接管 HiChat 集群部署”的完整 ArgoCD 指南。

---

## 1. 场景与目标

- **集群环境**: 本地 k3d 集群（示例名：`mycluster`）
- **业务命名空间**: `hichat`（见 `k8s/namespace.yaml`）
- **Git 仓库**: `http://172.21.0.1/root/hichat.git`
- **配置分支**: `k8s-config`（ArgoCD 只监控这个分支）
- **k8s 配置目录**: `k8s/`

目标：

1. 在 k3d 集群安装 ArgoCD；
2. 暴露 ArgoCD UI（端口转发或 Ingress）；
3. 把 GitLab 仓库接入 ArgoCD；
4. 使用 `k8s/argocd-application.yaml` 管理 HiChat 的所有 K8s 资源；
5. 与 GitLab CI/CD 配合，实现完整 GitOps。

---

## 2. 安装前准备

### 2.1 检查 k3d 集群与 kubectl

```bash
kubectl cluster-info
kubectl get nodes
```

确保节点处于 `Ready` 状态；当前集群中业务命名空间为 `hichat`：

```bash
kubectl get ns
```

如未创建，可先应用：

```bash
kubectl apply -f k8s/namespace.yaml
```

### 2.2 安装 Helm（如未安装）

```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
helm version
```

---

## 3. 使用 Helm 安装 ArgoCD

### 3.1 添加 helm 仓库

```bash
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update
```

### 3.2 安装 ArgoCD（最小可用配置）

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

等待组件就绪：

```bash
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server -n argocd --timeout=300s
kubectl get pods -n argocd
kubectl get svc  -n argocd
```

---

## 4. 访问 ArgoCD 界面

### 4.1 本地端口转发（简单调试）

1）获取默认管理员密码：

```bash
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d && echo
```

2）端口转发：

```bash
kubectl port-forward svc/argocd-server -n argocd 8443:443
```

3）浏览器访问：

- 地址: `https://localhost:8443`
- 用户名: `admin`
- 密码: 上一步获取的密码（自签名证书需要在浏览器中手动信任）

### 4.2 使用 Ingress 暴露（接入 Ingress 体系时使用）

如希望与 HiChat、其他服务统一走 Ingress，可为 ArgoCD 单独写一个 Ingress 资源（示例）：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argocd-ingress
  namespace: argocd
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: web
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

应用并配置 hosts：

```bash
kubectl apply -f k8s/argocd-ingress.yaml   # 如使用该文件名
echo "127.0.0.1 argocd.local" | sudo tee -a /etc/hosts
```

然后访问 `http://argocd.local:8000`（取决于 k3d 暴露的 Traefik 端口）。

---

## 5. 将 GitLab 仓库接入 ArgoCD

HiChat 使用**本地 GitLab + 配置分支**模式：

- Git 仓库: `http://172.21.0.1/root/hichat.git`
- ArgoCD 在集群内访问 GitLab，不能使用 `gitlab.iceymoss` 这种宿主机域名，改用 k3d 网桥 IP。

### 5.1 安装 ArgoCD CLI（本地机）

```bash
curl -sSL -o argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
chmod +x argocd
sudo mv argocd /usr/local/bin/
argocd version
```

### 5.2 登录 ArgoCD（CLI）

```bash
kubectl port-forward svc/argocd-server -n argocd 8443:443 &

argocd login localhost:8443 \
  --username admin \
  --password <ADMIN_PASSWORD> \
  --insecure
```

### 5.3 添加 Git 仓库（使用 IP）

```bash
# 查看 k3d 网络网关（一般是 172.x 网段）
docker network inspect k3d-mycluster | grep Gateway

argocd repo add http://172.21.0.1/root/hichat.git \
  --username <GITLAB_USERNAME> \
  --password '<GITLAB_PASSWORD>' \
  --insecure

argocd repo list
```

如使用 UI，也应在 `Settings → Repositories` 中填 IP 形式的 URL。

---

## 6. 使用 Application 管理 HiChat 集群

### 6.1 Application 配置与当前 `k8s/argocd-application.yaml`

仓库中已有标准 Application 配置：

```10:38:k8s/argocd-application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hichat
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: http://172.21.0.1/root/hichat.git
    targetRevision: k8s-config
    path: k8s
  destination:
    server: https://kubernetes.default.svc
    namespace: hichat
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

关键点：

- **repoURL** 指向 GitLab 仓库（使用 IP）；
- **targetRevision: `k8s-config`**：ArgoCD 只看配置分支；
- **path: `k8s`**：同步整个 `k8s/` 目录（包含 `namespace.yaml、mysql.yaml、redis.yaml、hichat.yaml、ingress.yaml`）；
- **destination.namespace: `hichat`**：所有资源落在业务命名空间；
- **automated.prune + selfHeal**：自动删除多余资源、自动纠偏漂移；
- **CreateNamespace=true**：如 `hichat` 不存在会自动创建。

### 6.2 应用 Application

```bash
kubectl apply -f k8s/argocd-application.yaml

argocd app list
argocd app get hichat
```

若仓库与集群初次不同步，可手动触发一次：

```bash
argocd app sync hichat --wait
argocd app resources hichat
```

此时 ArgoCD 会部署：

- `hichat` 命名空间；
- `mysql` Deployment + Service + PVC + 初始化 SQL（见 `k8s/mysql.yaml`）；
- `redis` Deployment + Service（见 `k8s/redis.yaml`）；
- `hichat` Deployment + Service（镜像来自 Docker Hub: `docker.io/iceymoss/hichat:latest`，见 `k8s/hichat.yaml`）；
- `hichat-ingress`（`hichat.local` 域名，见 `k8s/ingress.yaml`）。

---

## 7. 与 GitLab CI/CD 的 GitOps 集成要点

### 7.1 配置分支模式（推荐）

在 `CI_CD整体工作流.md` 与 `GITOPS的方案对比.md` 中已经完成的最佳实践：

- 代码分支：`master`（只放业务代码）；
- 配置分支：`k8s-config`（只放 `k8s/` 下 YAML，含镜像标签更新 commit）；
- `.gitlab-ci.yml` 中 `update-config` Job：
  - 从 `master` 拿最新的 `k8s/` 目录；
  - 使用 `sed` 更新 `k8s/hichat.yaml` 中镜像为 `$CI_COMMIT_SHORT_SHA`；
  - 提交到 `k8s-config` 分支，并带 `[skip ci]` 避免循环；
  - 推送到远程 `k8s-config`。

ArgoCD 只监控 `k8s-config`，因此：

1. **开发者推代码 → CI 构建镜像**；
2. **CI 更新 `k8s-config` 的 YAML**；
3. **ArgoCD 检测到配置分支变化 → 自动同步到集群**。

### 7.2 ArgoCD 侧只需要保证的配置

1. `spec.source.targetRevision: k8s-config`；
2. 开启 `spec.syncPolicy.automated` 的 `prune` 和 `selfHeal`；
3. 确保 Application 使用的 repoURL 与 CI 中推送的仓库一致。

---

## 8. 常用 ArgoCD 运维命令（围绕 HiChat）

```bash
# 查看所有应用
argocd app list

# 查看 HiChat 应用详情
argocd app get hichat

# 手动同步 / 强制同步
argocd app sync hichat
argocd app sync hichat --force

# 查看 HiChat 资源编排情况
argocd app resources hichat

# 查看历史与回滚
argocd app history hichat
argocd app rollback hichat <revision>
```

---

## 9. 常见问题与排查路径

- **CrashLoopBackOff / Redis 连接失败**  
  - 检查 `argocd-redis` Pod 与 Service 是否正常：`kubectl get pods,svc -n argocd | grep redis`  
  - 如多次实验导致脏数据，可直接 `kubectl delete namespace argocd` 后重新安装。

- **无法添加 Git 仓库（DNS 解析失败）**  
  - 现象：`lookup gitlab.iceymoss ... no such host`  
  - 解决：统一改用 `http://172.21.0.1/root/hichat.git` 这种 IP 形式。

- **Application 同步失败**  
  - 检查：`argocd app get hichat`、`argocd app logs hichat`；  
  - 确认 `k8s-config` 分支存在且包含完整 `k8s/` 目录。

- **镜像拉取失败**  
  - 若后续改回从 GitLab Registry 拉镜像，需要配置 imagePullSecret，并在 `k8s/hichat.yaml` 中引用；  
  - 当前示例直接从 Docker Hub 拉 `docker.io/iceymoss/hichat:latest`，避免本地 Registry 复杂度。

---

## 10. 一次性安装 + 接管 HiChat 的推荐步骤回顾

1. 创建 / 确认 k3d 集群，`kubectl` 可正常访问；
2. 按本指南用 Helm 安装 ArgoCD；
3. 通过端口转发或 Ingress 访问 ArgoCD UI，并修改默认密码；
4. 使用 IP 地址把 GitLab 仓库 `hichat.git` 接入 ArgoCD；
5. 确认 `.gitlab-ci.yml` 中的 `update-config` Job 能更新 `k8s-config` 分支；
6. 应用 `k8s/argocd-application.yaml`，让 ArgoCD 托管 `k8s/` 全部资源；
7. 之后只要推代码到 `master`，就会经由 CI → 配置分支 → ArgoCD → K8s，完成一次完整的 GitOps 部署。


