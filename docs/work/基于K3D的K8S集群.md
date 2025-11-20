## 基于 k3d 的 Kubernetes 集群最终实践指南

本指南在 `基于K3D构建K8S集群.md` 与 `HiChat Kubernetes 部署文档.md` 的基础上，结合当前 `k8s/` 目录的实际 YAML，整理出一套**从创建集群到跑通 HiChat + Ingress + ArgoCD** 的实践流程。

---

## 1. 整体架构与目标

- **宿主环境**: Windows + WSL2（Ubuntu）或原生 Linux
- **容器运行时**: Docker
- **K8s 发行版**: k3d（k3s in Docker）
- **目标**:
  - 创建一个名为 `mycluster` 的 k3d 集群；
  - 在 `hichat` 命名空间中部署：
    - MySQL（带 PVC + 初始化 SQL）；
    - Redis；
    - HiChat 应用（镜像 `docker.io/iceymoss/hichat:latest`）；
    - 通过 Ingress (`hichat.local`) 对外暴露；
  - 该集群后续可被 ArgoCD 托管（见 ArgoCD 最终版文档）。

---

## 2. 基础工具准备

### 2.1 安装 Docker

此处略，要求宿主机能正常运行：

```bash
docker version
```

### 2.2 安装 k3d

```bash
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

k3d version
```

### 2.3 安装 kubectl

Linux/WSL 推荐：

```bash
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

kubectl version --client
```

---

## 3. 创建 k3d 集群（带 Ingress 暴露）

考虑到后续要走 Ingress（Traefik），推荐直接在创建集群时映射 8000 端口：

```bash
k3d cluster create mycluster --port "8000:80@loadbalancer"
```

验证：

```bash
kubectl get nodes
kubectl get pods -n kube-system | grep traefik   # 默认 Ingress Controller
```

如需清理：

```bash
k3d cluster delete mycluster
```

---

## 4. 集群内基础组件部署（基于 k8s 目录）

当前仓库中 `k8s/` 目录的关键文件：

- `namespace.yaml`：创建 `hichat` 命名空间；
- `mysql.yaml`：PVC + MySQL Deployment + Service + 初始化 SQL 的 ConfigMap；
- `redis.yaml`：Redis Deployment + Service；
- `hichat.yaml`：HiChat Deployment + Service（`ClusterIP` 类型，供 Ingress 使用）；
- `ingress.yaml`：Ingress 规则（域名 `hichat.local`）。

### 4.1 创建命名空间

```bash
kubectl apply -f k8s/namespace.yaml
kubectl get ns
```

### 4.2 部署 MySQL（带持久化与初始化数据）

`mysql.yaml` 概览：

```1:66:k8s/mysql.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mysql-pvc
  namespace: hichat
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql
  namespace: hichat
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      containers:
      - name: mysql
        image: mysql:8.0
        env:
        - name: MYSQL_ROOT_PASSWORD
          value: "rootpassword"
        - name: MYSQL_DATABASE
          value: "hi_chat"
        - name: MYSQL_USER
          value: "iceymoss"
        - name: MYSQL_PASSWORD
          value: "Yk?123456"
        ports:
        - containerPort: 3306
        volumeMounts:
        - name: mysql-storage
          mountPath: /var/lib/mysql
        - name: mysql-init
          mountPath: /docker-entrypoint-initdb.d
      volumes:
      - name: mysql-storage
        persistentVolumeClaim:
          claimName: mysql-pvc
      - name: mysql-init
        configMap:
          name: mysql-init-script
---
apiVersion: v1
kind: Service
metadata:
  name: mysql
  namespace: hichat
spec:
  selector:
    app: mysql
  ports:
  - port: 3306
    targetPort: 3306
  type: ClusterIP
```

部署并等待就绪：

```bash
kubectl apply -f k8s/mysql.yaml
kubectl wait --for=condition=ready pod -l app=mysql -n hichat --timeout=180s
kubectl get pods,svc -n hichat | grep mysql
```

可选：验证初始化 SQL（已包含多张业务表与测试数据）：

```bash
kubectl exec -it deployment/mysql -n hichat -- \
  mysql -uiceymoss -p'Yk?123456' hi_chat -e "SHOW TABLES;"
```

### 4.3 部署 Redis

部署文件摘要：

```1:33:k8s/redis.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: hichat
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: hichat
spec:
  selector:
    app: redis
  ports:
  - port: 6379
    targetPort: 6379
  type: ClusterIP
```

部署：

```bash
kubectl apply -f k8s/redis.yaml
kubectl wait --for=condition=ready pod -l app=redis -n hichat --timeout=120s
kubectl get pods,svc -n hichat | grep redis
```

### 4.4 部署 HiChat 应用

`hichat.yaml` 中，应用容器镜像来自 Docker Hub，并通过环境变量指向上面部署的 MySQL、Redis：

```1:51:k8s/hichat.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hichat
  namespace: hichat
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hichat
  template:
    metadata:
      labels:
        app: hichat
    spec:
      containers:
      - name: hichat
        image: docker.io/iceymoss/hichat:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8000
        env:
        - name: MYSQL_HOST
          value: "mysql"
        - name: MYSQL_PORT
          value: "3306"
        - name: MYSQL_USER
          value: "iceymoss"
        - name: MYSQL_PASSWORD
          value: "Yk?123456"
        - name: MYSQL_DATABASE
          value: "hi_chat"
        - name: REDIS_HOST
          value: "redis"
        - name: REDIS_PORT
          value: "6379"
        - name: PORT
          value: "8000"
---
apiVersion: v1
kind: Service
metadata:
  name: hichat
  namespace: hichat
spec:
  selector:
    app: hichat
  ports:
  - port: 8000
    targetPort: 8000
  type: ClusterIP
```

部署并检查：

```bash
kubectl apply -f k8s/hichat.yaml
kubectl wait --for=condition=ready pod -l app=hichat -n hichat --timeout=120s
kubectl get pods,svc -n hichat | grep hichat
```

---

## 5. Ingress 暴露 HiChat（基于 Traefik）

### 5.1 Ingress 资源

`k8s/ingress.yaml` 已给出标准配置：

```1:18:k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hichat-ingress
  namespace: hichat
spec:
  ingressClassName: traefik
  rules:
  - host: hichat.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hichat
            port:
              number: 8000
```

应用与验证：

```bash
kubectl apply -f k8s/ingress.yaml
kubectl get ingress -n hichat
kubectl describe ingress hichat-ingress -n hichat
```

### 5.2 配置 hosts 并访问

在 WSL 中：

```bash
echo "127.0.0.1 hichat.local" | sudo tee -a /etc/hosts
```

然后：

```bash
curl http://hichat.local:8000
```

如在 Windows 浏览器访问，再在 Windows 的 `hosts` 加同一条映射，访问 `http://hichat.local:8000` 即可。

---

## 6. 使用 kubectl 理解集群状态

### 6.1 `kubectl get all -n hichat` 的含义

典型输出与字段详解已在 `HiChat Kubernetes 部署文档.md` 中详细展开，这里给简版 checklist：

- **Pod**: `STATUS=Running`，`READY=1/1`，`RESTARTS` 不持续累加；
- **Service**:
  - `mysql / redis / hichat` 类型均为 `ClusterIP`；
  - `PORT(S)` 分别为 `3306 / 6379 / 8000`；
- **Deployment**:
  - `READY` 等于期望副本数（例如 `1/1`）；
  - `AVAILABLE` 与 `UP-TO-DATE` 均为 1；
- **ReplicaSet**:
  - `DESIRED / CURRENT / READY` 一致。

如以上任一异常，可通过：

```bash
kubectl describe pod <pod-name> -n hichat
kubectl logs <pod-name> -n hichat
```

排查错误原因。

---

## 7. 集群维护：扩容、缩容与更新

### 7.1 扩容 HiChat 实例

```bash
# 扩容到 3 个副本
kubectl scale deployment hichat --replicas=3 -n hichat

kubectl get deployment hichat -n hichat
kubectl get pods -n hichat -l app=hichat
```

Traefik + Service 会自动在所有 Pod 之间负载均衡，无需额外配置。

### 7.2 更新镜像版本（无 ArgoCD 时）

本地直接修改镜像并重启：

```bash
kubectl set image deployment/hichat \
  hichat=docker.io/iceymoss/hichat:<tag> \
  -n hichat
```

或更新 `k8s/hichat.yaml` 的 `image:` 后：

```bash
kubectl apply -f k8s/hichat.yaml
```

在启用 GitOps 后，建议只通过 GitLab CI/CD 更新 `k8s-config` 分支的 YAML，由 ArgoCD 负责同步。

---

## 8. 与 ArgoCD 与 GitLab 的衔接位置

这一节只指出**边界与衔接点**，具体细节见对应文档：

- **本文件**：完成 k3d 集群 + `hichat` 命名空间 + MySQL/Redis/应用/Ingress 的部署模型设计；
- **`ARGOCD最终版_安装与GitOps使用指南.md`**：
  - 将本地 GitLab 仓库与 ArgoCD 连接；
  - 使用 `k8s/argocd-application.yaml` 托管 `k8s/` 目录资源；
- **GitLab 最终版文档**：
  - 如何安装本地 GitLab；
  - 如何配置 Runner、Registry 与 `.gitlab-ci.yml`；
  - 如何通过 `update-config` Job 自动更新 `k8s-config` 分支；
- **CI/CD + GitOps 流程最终版文档**：
  - 把“开发者推代码”到“集群滚动升级”的完整链路串成一张大图。

---

## 9. 快速全流程命令清单

从空环境到 HiChat+Ingress 跑通的最小命令集（假设 Docker 已安装）：

```bash
# 1) 安装工具
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# 2) 创建集群
k3d cluster create mycluster --port "8000:80@loadbalancer"

# 3) 部署命名空间与基础组件
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/mysql.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/hichat.yaml
kubectl apply -f k8s/ingress.yaml

# 4) 等待所有 Pod ready
kubectl wait --for=condition=ready pod -l app=mysql  -n hichat --timeout=180s
kubectl wait --for=condition=ready pod -l app=redis  -n hichat --timeout=120s
kubectl wait --for=condition=ready pod -l app=hichat -n hichat --timeout=120s

# 5) 配置主机名
echo "127.0.0.1 hichat.local" | sudo tee -a /etc/hosts

# 6) 访问
curl http://hichat.local:8000
```

执行无误后，即可继续参考 ArgoCD 与 GitLab 文档，把这个本地 k3d 集群正式纳入 CI/CD + GitOps 体系。


