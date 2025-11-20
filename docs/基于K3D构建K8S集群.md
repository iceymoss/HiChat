# HiChat Kubernetes 部署文档

本文档描述了如何在本地 k3d 集群中部署 HiChat 应用。

## 目录

- [前置条件](#前置条件)
- [架构图](#架构图)
- [项目结构](#项目结构)
- [配置说明](#配置说明)
- [构建步骤](#构建步骤)
- [部署步骤](#部署步骤)
- [验证部署](#验证部署)
- [服务暴露方式](#服务暴露方式)
- [理解 kubectl get all 输出](#理解-kubectl-get-all-输出)
- [扩容和缩容](#扩容和缩容)
- [故障排查](#故障排查)

## 前置条件

### 1. 环境要求

- WSL2 (Windows Subsystem for Linux) 或 Linux 系统
- Docker 已安装并运行
- k3d 已安装（用于创建本地 Kubernetes 集群）
- kubectl 已安装并配置

### 2. 安装 kubectl

kubectl 是 Kubernetes 的命令行工具，需要单独安装。

#### Linux/WSL 安装方法

**方法 1: 使用 curl 安装（推荐）**

```bash
# 下载最新版本
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

# 安装到系统路径
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# 验证安装
kubectl version --client
```

**方法 2: 使用包管理器安装**

对于 Ubuntu/Debian：
```bash
sudo apt-get update
sudo apt-get install -y kubectl
```

对于 CentOS/RHEL：
```bash
sudo yum install -y kubectl
```

#### Windows 安装方法

**方法 1: 使用 Chocolatey**
```powershell
choco install kubernetes-cli
```

**方法 2: 使用 Scoop**
```powershell
scoop install kubectl
```

**方法 3: 手动下载**
1. 访问 https://kubernetes.io/docs/tasks/tools/install-kubectl-windows/
2. 下载 kubectl.exe
3. 添加到系统 PATH 环境变量

#### 验证安装

安装完成后，验证 kubectl 是否正常工作：

```bash
kubectl version --client
```

### 3. 安装 k3d

如果还没有安装 k3d，执行以下命令：

```bash
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
```

### 4. 创建 k3d 集群

```bash
k3d cluster create mycluster
```

这里我们使用ingress-nginx-controller对外提供服务，我们的集群最好使用(将宿主机端口8000映射k8s集群nginx的80)：
```bash
k3d cluster create mycluster --port "8000:80@loadbalancer"
```

验证集群创建成功：

```bash
kubectl get nodes
```

**注意**: k3d 创建集群后会自动配置 kubectl 的 kubeconfig，无需手动配置。

## 架构图

### 当前部署架构

```
┌─────────────────────────────────────────────────────────────────┐
│                      k3d Kubernetes Cluster                    │
│                      (Namespace: hichat)                        │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │                    LoadBalancer Service                    │ │
│  │                    (hichat:8000)                           │ │
│  │                    External IP: 172.21.0.2                │ │
│  └───────────────────────┬────────────────────────────────────┘ │
│                          │                                      │
│                          ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │              HiChat Application Deployment                 │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │ │
│  │  │   Pod: hichat│  │   Pod: hichat│  │   Pod: hichat│   │ │
│  │  │   (Port:8000)│  │   (Port:8000)│  │   (Port:8000)│   │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘   │ │
│  │         │                  │                  │          │ │
│  └─────────┼──────────────────┼──────────────────┼──────────┘ │
│            │                  │                  │             │
│            └──────────────────┼──────────────────┘             │
│                               │                                 │
│            ┌──────────────────┴──────────────────┐             │
│            │                                      │             │
│            ▼                                      ▼             │
│  ┌──────────────────┐              ┌──────────────────┐        │
│  │  MySQL Service   │              │  Redis Service    │        │
│  │  (ClusterIP)     │              │  (ClusterIP)     │        │
│  │  Port: 3306      │              │  Port: 6379      │        │
│  └────────┬─────────┘              └────────┬─────────┘        │
│           │                                  │                  │
│           ▼                                  ▼                  │
│  ┌──────────────────┐              ┌──────────────────┐        │
│  │ MySQL Deployment │              │ Redis Deployment  │        │
│  │  ┌────────────┐  │              │  ┌────────────┐  │        │
│  │  │ Pod: mysql│  │              │  │ Pod: redis │  │        │
│  │  │ Port:3306 │  │              │  │ Port:6379  │  │        │
│  │  └────────────┘  │              │  └────────────┘  │        │
│  │                  │              │                  │        │
│  │  PVC: mysql-pvc  │              │                  │        │
│  │  (Data Storage)   │              │                  │        │
│  └──────────────────┘              └──────────────────┘        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

外部访问:
  http://localhost:8000 (通过 port-forward)
  或
  http://localhost:32496 (通过 k3d LoadBalancer 映射)
```

### 组件说明

1. **LoadBalancer Service (hichat)**
   - 类型: LoadBalancer
   - 端口: 8000
   - 作用: 对外暴露 HiChat 应用服务
   - k3d 会自动将 LoadBalancer 映射到本地端口

2. **HiChat Deployment**
   - 当前副本数: 1 (可扩容)
   - 容器端口: 8000
   - 功能: 运行 HiChat 应用主程序

3. **MySQL Service & Deployment**
   - 类型: ClusterIP (集群内部访问)
   - 端口: 3306
   - 存储: 使用 PVC 持久化数据
   - 初始化: 通过 ConfigMap 自动执行 SQL 脚本

4. **Redis Service & Deployment**
   - 类型: ClusterIP (集群内部访问)
   - 端口: 6379
   - 功能: 缓存和消息队列

### 数据流向

```
用户请求 
  → LoadBalancer Service (hichat)
    → HiChat Pod(s) (负载均衡)
      → MySQL Service → MySQL Pod (数据持久化)
      → Redis Service → Redis Pod (缓存)
```

## 项目结构

```
HiChat/
├── Dockerfile              # Docker 镜像构建文件
├── k8s/                    # Kubernetes 资源文件
│   ├── namespace.yaml      # 命名空间定义
│   ├── mysql.yaml          # MySQL 部署配置（包含数据库初始化）
│   ├── redis.yaml          # Redis 部署配置
│   ├── hichat.yaml         # HiChat 应用部署配置
│   └── ingress.yaml        # Ingress 配置（域名路由）
├── config-debug.yaml       # 应用配置文件
├── hi_chat.sql             # 数据库初始化 SQL（已集成到 mysql.yaml）
├── .gitlab-ci.yml          # GitLab CI/CD 配置
└── ...                     # 其他项目文件
```

## 配置说明

### 1. 应用配置（config-debug.yaml）

应用支持通过配置文件或环境变量进行配置。环境变量优先级更高。

支持的环境变量：
- `PORT` - 服务端口（默认：8000）
- `MYSQL_HOST` - MySQL 主机地址
- `MYSQL_PORT` - MySQL 端口（默认：3306）
- `MYSQL_NAME` - 数据库名
- `MYSQL_USER` - MySQL 用户名
- `MYSQL_PASSWORD` - MySQL 密码
- `REDIS_HOST` - Redis 主机地址
- `REDIS_PORT` - Redis 端口（默认：6379）

### 2. Kubernetes 配置

所有 Kubernetes 资源文件位于 `k8s/` 目录：

- **namespace.yaml**: 创建 `hichat` 命名空间
- **mysql.yaml**: 
  - 创建 MySQL Deployment 和 Service
  - 创建 PersistentVolumeClaim 用于数据持久化
  - 包含 ConfigMap，自动执行数据库初始化 SQL
- **redis.yaml**: 创建 Redis Deployment 和 Service
- **hichat.yaml**: 创建 HiChat 应用 Deployment 和 ClusterIP Service
- **ingress.yaml**: 创建 Ingress 资源，通过域名路由访问服务（使用 Traefik）

## 构建步骤

### 1. 构建 Docker 镜像

在项目根目录执行：

```bash
docker build -t hichat:latest .
```

### 2. 将镜像导入到 k3d 集群

```bash
k3d image import hichat:latest -c mycluster
```

验证镜像导入成功：

```bash
k3d image list -c mycluster
```

## 部署步骤

### 步骤 1: 创建命名空间

```bash
kubectl apply -f k8s/namespace.yaml
```

### 步骤 2: 部署 MySQL

```bash
kubectl apply -f k8s/mysql.yaml
```

等待 MySQL Pod 就绪：

```bash
kubectl wait --for=condition=ready pod -l app=mysql -n hichat --timeout=120s
```

验证 MySQL 状态：

```bash
kubectl get pods -n hichat -l app=mysql
```

### 步骤 3: 验证数据库初始化（可选）

检查数据库表是否已创建：

```bash
kubectl exec -it -n hichat deployment/mysql -- mysql -uiceymoss -p'Yk?123456' hi_chat -e "SHOW TABLES;"
```

应该看到以下表：
- `communities`
- `group_infos`
- `messages`
- `relations`
- `user_basics`

### 步骤 4: 部署 Redis

```bash
kubectl apply -f k8s/redis.yaml
```

等待 Redis Pod 就绪：

```bash
kubectl wait --for=condition=ready pod -l app=redis -n hichat --timeout=60s
```

### 步骤 5: 部署 HiChat 应用
这里需要注意，如果使用的是本地直接构建的docker镜像，你需要在k8s/hichat.yaml中修改镜像名称为hichat:latest
```bash
kubectl apply -f k8s/hichat.yaml
```

等待应用 Pod 就绪：

```bash
kubectl wait --for=condition=ready pod -l app=hichat -n hichat --timeout=60s
```

### 步骤 6: 部署 Ingress（可选，推荐）

如果需要使用域名访问（推荐方式）：

```bash
kubectl apply -f k8s/ingress.yaml
```

验证 Ingress 部署：

```bash
kubectl get ingress -n hichat
```

配置本地 hosts 文件（WSL 环境）：

```bash
echo "127.0.0.1 hichat.local" | sudo tee -a /etc/hosts
```

详细配置步骤请参考 [Ingress 配置](#方式-3-ingress推荐用于生产环境) 章节。

### 步骤 7: 检查所有资源状态

```bash
kubectl get all -n hichat
```

预期输出应该显示所有 Pod 状态为 `Running`，所有服务都已创建。

## 验证部署

### 1. 查看应用日志

```bash
kubectl logs -f deployment/hichat -n hichat
```

成功启动的标志：
- 看到 `配置信息` 日志，显示正确的数据库和 Redis 连接信息
- 看到 `Loaded HTML Templates` 日志
- 看到 `Listening and serving HTTP on :8000` 日志

### 2. 获取服务访问地址

查看服务信息：

```bash
kubectl get svc hichat -n hichat
```

### 3. 访问应用

**方式 1: 使用 Ingress（推荐，已配置）**

如果已部署 Ingress，可以通过域名访问：

```bash
# 通过域名访问（需要配置 hosts 文件）
curl http://hichat.local:8000
```

在浏览器访问：`http://hichat.local:8000`

**方式 2: 使用 port-forward（临时调试）**

```bash
kubectl port-forward svc/hichat 8000:8000 -n hichat
```

然后在浏览器访问：`http://localhost:8000`

**方式 3: 使用 k3d LoadBalancer（如果 Service 类型为 LoadBalancer）**

k3d 会自动将 LoadBalancer 服务映射到本地端口。查看服务信息获取映射的端口：

```bash
kubectl get svc hichat -n hichat
```

访问：`http://localhost:<映射的端口>`

### 4. 测试 API（可选）

```bash
# 测试首页
curl http://localhost:8000

# 测试注册页面
curl http://localhost:8000/register
```

## 服务暴露方式

Kubernetes 提供了多种方式将集群内的服务暴露给外部访问。以下是常用的几种方式：

### 当前使用的方式：LoadBalancer

**配置位置**: `k8s/hichat.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: hichat
  namespace: hichat
spec:
  type: LoadBalancer  # LoadBalancer 类型
  selector:
    app: hichat
  ports:
  - port: 8000
    targetPort: 8000
```

**工作原理**:
- k3d 会自动将 LoadBalancer 服务映射到本地端口
- 查看映射端口：`kubectl get svc hichat -n hichat`
- 输出示例：`8000:32496/TCP` 表示可以通过 `localhost:32496` 访问

**访问方式**:
```bash
# 查看映射的端口
kubectl get svc hichat -n hichat

# 访问（假设映射到 32496 端口）
curl http://localhost:32496
```

**优点**:
- 简单直接，k3d 自动处理
- 适合本地开发和测试

**缺点**:
- 端口是动态分配的，可能每次不同
- 仅适合本地环境

### 方式 2: NodePort

NodePort 会在每个节点上开放一个固定端口（30000-32767），可以从集群外部访问。

**配置示例** (`k8s/hichat-nodeport.yaml`):

```yaml
apiVersion: v1
kind: Service
metadata:
  name: hichat-nodeport
  namespace: hichat
spec:
  type: NodePort
  selector:
    app: hichat
  ports:
  - port: 8000
    targetPort: 8000
    nodePort: 30080  # 可选：指定端口（30000-32767）
```

**部署**:
```bash
kubectl apply -f k8s/hichat-nodeport.yaml
```

**访问方式**:
```bash
# 查看 NodePort
kubectl get svc hichat-nodeport -n hichat

# 访问（假设 NodePort 是 30080）
curl http://localhost:30080
```

**优点**:
- 端口固定，便于配置
- 可以从集群外部直接访问

**缺点**:
- 端口范围受限（30000-32767）
- 需要知道节点 IP

### 方式 3: Ingress（推荐用于生产环境）

Ingress 提供基于 HTTP/HTTPS 的路由，支持域名、路径、SSL 等高级功能。

**前置条件**: k3d 默认使用 Traefik 作为 Ingress Controller

#### 完整配置步骤（WSL 环境）

**步骤 1: 检查 Traefik Ingress Controller**

```bash
# 检查 Traefik 是否运行
kubectl get pods -n kube-system | grep traefik
```

**预期输出**：应该看到 `traefik-xxx` Pod 状态为 `Running`

如果 Traefik 没有运行，需要重新创建 k3d 集群（见故障排查部分）。

**步骤 2: 更新 Service 类型为 ClusterIP**

编辑 `k8s/hichat.yaml`，将 Service 类型改为 `ClusterIP`：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: hichat
  namespace: hichat
spec:
  type: ClusterIP  # 改为 ClusterIP，通过 Ingress 暴露
  selector:
    app: hichat
  ports:
  - port: 8000
    targetPort: 8000
```

应用更新：

```bash
kubectl apply -f k8s/hichat.yaml
```

验证 Service 类型：

```bash
kubectl get svc hichat -n hichat
```

应该看到 `TYPE` 为 `ClusterIP`。

**步骤 3: 部署 Ingress**

Ingress 配置文件已创建在 `k8s/ingress.yaml`：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hichat-ingress
  namespace: hichat
  annotations:
    # Traefik 特定注解（k3d 默认使用 Traefik）
    traefik.ingress.kubernetes.io/router.entrypoints: web
spec:
  ingressClassName: traefik  # k3d 默认使用 traefik
  rules:
  - host: hichat.local  # 本地域名，需要在 hosts 文件中配置
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

部署 Ingress：

```bash
kubectl apply -f k8s/ingress.yaml
```

**步骤 4: 验证 Ingress 部署**

```bash
# 查看 Ingress 状态
kubectl get ingress -n hichat

# 查看详细信息
kubectl describe ingress hichat-ingress -n hichat
```

**预期输出**：
```
NAME             CLASS     HOSTS          ADDRESS      PORTS   AGE
hichat-ingress   traefik   hichat.local   172.21.0.2   80      10s
```

**步骤 5: 配置 WSL hosts 文件**

```bash
# 添加本地域名映射
echo "127.0.0.1 hichat.local" | sudo tee -a /etc/hosts
```

验证配置：

```bash
cat /etc/hosts | grep hichat.local
```

**步骤 6: 访问应用**

**注意**: k3d 的 Traefik 可能映射到 8000 端口而不是标准的 80 端口。

```bash
# 方式 1: 尝试 8000 端口（常见情况）
curl http://hichat.local:8000

# 方式 2: 如果 Traefik 映射到 80 端口
curl http://hichat.local

# 方式 3: 在浏览器中访问
# http://hichat.local:8000
```

**验证访问成功**：应该返回 HTML 内容（登录页面）。

#### 完整命令序列

```bash
# 1. 检查 Traefik
kubectl get pods -n kube-system | grep traefik

# 2. 更新 Service
kubectl apply -f k8s/hichat.yaml

# 3. 部署 Ingress
kubectl apply -f k8s/ingress.yaml

# 4. 验证 Ingress
kubectl get ingress -n hichat

# 5. 配置 hosts（需要输入密码）
echo "127.0.0.1 hichat.local" | sudo tee -a /etc/hosts

# 6. 测试访问
curl http://hichat.local:8000
```

#### Windows 浏览器访问（可选）

如果需要在 Windows 浏览器中访问，也需要配置 Windows hosts 文件：

1. 以**管理员身份**打开记事本
2. 打开文件：`C:\Windows\System32\drivers\etc\hosts`
3. 添加：`127.0.0.1 hichat.local`
4. 保存
5. 在浏览器访问：`http://hichat.local:8000`

#### 优点

- 支持域名和路径路由
- 支持 SSL/TLS（可配置）
- 适合生产环境
- 可以一个入口管理多个服务
- 使用域名访问，更专业

#### 缺点

- 需要安装 Ingress Controller（k3d 默认已安装 Traefik）
- 需要配置 hosts 文件
- 配置相对复杂

#### 故障排查

**问题 1: Ingress 没有 ADDRESS**

**原因**: Traefik 可能没有正确运行

**解决**:
```bash
# 检查 Traefik Pod
kubectl get pods -n kube-system | grep traefik

# 查看 Traefik 日志
kubectl logs -n kube-system -l app.kubernetes.io/name=traefik
```

**问题 2: 无法访问 hichat.local**

**检查**:
```bash
# 1. 检查 hosts 文件
cat /etc/hosts | grep hichat.local

# 2. 检查 Ingress
kubectl get ingress -n hichat

# 3. 检查 Service
kubectl get svc hichat -n hichat

# 4. 尝试不同端口
curl http://hichat.local:8000
curl http://hichat.local:80
```

**问题 3: 404 错误**

**原因**: Ingress 路由配置可能有问题

**解决**:
```bash
# 查看 Ingress 详细信息
kubectl describe ingress hichat-ingress -n hichat

# 检查 Service 名称是否匹配
kubectl get svc -n hichat
```

### 方式 4: port-forward（临时访问）

port-forward 是 kubectl 提供的临时端口转发功能，适合调试和临时访问。

**使用方式**:
```bash
# 转发 Service
kubectl port-forward svc/hichat 8000:8000 -n hichat

# 或直接转发 Pod
kubectl port-forward pod/<pod-name> 8000:8000 -n hichat
```

**访问**: `http://localhost:8000`

**优点**:
- 无需修改配置
- 适合临时访问和调试

**缺点**:
- 连接断开后需要重新执行
- 不适合生产环境

### 方式对比

| 方式 | 适用场景 | 端口 | 配置复杂度 | 生产环境 |
|------|---------|------|-----------|---------|
| LoadBalancer | 本地开发 | 动态 | 简单 | ❌ |
| NodePort | 简单暴露 | 30000-32767 | 简单 | ⚠️ |
| Ingress | 生产环境 | 80/443 | 中等 | ✅ |
| port-forward | 临时调试 | 任意 | 简单 | ❌ |

### 从外部网络访问（生产环境）

如果要从外部网络访问（非本地），需要：

1. **使用 Ingress + 域名**:
   - 配置真实的域名（如 `hichat.example.com`）
   - 配置 DNS 指向集群的 Ingress Controller IP
   - 配置 SSL 证书

2. **使用 NodePort + 公网 IP**:
   - 确保节点有公网 IP
   - 配置防火墙规则开放 NodePort 端口
   - 通过 `http://<公网IP>:<NodePort>` 访问

3. **使用云服务商的 LoadBalancer**:
   - 在云环境（如 AWS、Azure、GCP）中
   - LoadBalancer 会自动分配公网 IP
   - 通过公网 IP 访问

### 当前架构的服务暴露

#### 使用 Ingress 方式（推荐）

```
外部访问 (http://hichat.local:8000)
    ↓
Traefik Ingress Controller
    ↓
Ingress (hichat-ingress)
    ↓
ClusterIP Service (hichat)
    ↓
HiChat Pods (负载均衡)
```

**查看 Ingress 状态**:
```bash
kubectl get ingress -n hichat
```

**访问示例**:
```bash
# 通过域名访问（需要配置 hosts 文件）
curl http://hichat.local:8000

# 在浏览器中访问
# http://hichat.local:8000
```

#### 使用 LoadBalancer 方式（备选）

如果使用 LoadBalancer 方式：

```
外部访问
    ↓
LoadBalancer Service (hichat)
    ↓ (自动映射到本地端口，如 32496)
k3d 网络层
    ↓
HiChat Pods (负载均衡)
```

**查看当前暴露的端口**:
```bash
kubectl get svc hichat -n hichat
```

**访问示例**:
```bash
# 假设映射到 32496 端口
curl http://localhost:32496
```

## 故障排查

### 问题 1: Pod 一直处于 Pending 状态

**原因**: 可能是资源不足或节点问题

**解决**:
```bash
# 查看 Pod 详细信息
kubectl describe pod <pod-name> -n hichat

# 查看事件
kubectl get events -n hichat --sort-by='.lastTimestamp'
```

### 问题 2: Pod 一直处于 CrashLoopBackOff

**原因**: 应用启动失败

**解决**:
```bash
# 查看 Pod 日志
kubectl logs <pod-name> -n hichat

# 查看之前的日志（如果 Pod 重启了）
kubectl logs <pod-name> -n hichat --previous
```

### 问题 3: 应用返回 500 错误

**可能原因**:
- 数据库连接失败
- Redis 连接失败
- 模板文件缺失
- 静态资源文件缺失

**解决**:
```bash
# 查看应用日志
kubectl logs -f deployment/hichat -n hichat

# 检查数据库连接
kubectl exec -it -n hichat deployment/mysql -- mysql -uiceymoss -p'Yk?123456' hi_chat -e "SELECT 1;"

# 检查 Redis 连接
kubectl exec -it -n hichat deployment/redis -- redis-cli ping
```

### 问题 4: 找不到模板文件或静态资源

**原因**: Dockerfile 没有正确复制文件

**解决**:
1. 检查 Dockerfile 是否包含所有必要的 COPY 指令
2. 重新构建镜像：
   ```bash
   docker build -t hichat:latest .
   k3d image import hichat:latest -c mycluster
   kubectl delete deployment hichat -n hichat
   kubectl apply -f k8s/hichat.yaml
   ```

### 问题 5: 数据库初始化失败

**解决**:
```bash
# 查看 MySQL 日志
kubectl logs -f deployment/mysql -n hichat

# 手动执行 SQL（如果需要）
kubectl exec -it -n hichat deployment/mysql -- mysql -uiceymoss -p'Yk?123456' hi_chat
```

### 问题 6: 镜像拉取失败

**原因**: 镜像没有正确导入到 k3d 集群

**解决**:
```bash
# 重新导入镜像
k3d image import hichat:latest -c mycluster

# 验证镜像是否存在
k3d image list -c mycluster
```

## 常用命令速查

```bash
# 查看所有资源
kubectl get all -n hichat

# 查看 Pod 状态
kubectl get pods -n hichat

# 查看服务
kubectl get svc -n hichat

# 查看应用日志
kubectl logs -f deployment/hichat -n hichat

# 查看 MySQL 日志
kubectl logs -f deployment/mysql -n hichat

# 查看 Redis 日志
kubectl logs -f deployment/redis -n hichat

# 进入 Pod 调试
kubectl exec -it deployment/hichat -n hichat -- sh

# 删除所有资源（清理环境）
kubectl delete namespace hichat
```

## 理解 kubectl get all 输出

执行 `kubectl get all -n hichat` 会显示命名空间中的所有资源。以下是输出字段的详细说明：

### 示例输出

```bash
$ kubectl get all -n hichat

NAME                         READY   STATUS    RESTARTS   AGE
pod/hichat-9f7f97d6d-d7htm   1/1     Running   0          6m26s
pod/hichat-9f7f97d6d-fppff   1/1     Running   0          6m26s
pod/hichat-9f7f97d6d-vsdj9   1/1     Running   0          26m
pod/mysql-5fb77fcc9f-9ptp7   1/1     Running   0          39m
pod/redis-7786bddb49-9bgnm   1/1     Running   0          36m

NAME             TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
service/hichat   LoadBalancer   10.43.6.114     172.21.0.2    8000:32496/TCP   36m
service/mysql    ClusterIP      10.43.163.143   <none>        3306/TCP         39m
service/redis    ClusterIP      10.43.167.129   <none>        6379/TCP         36m

NAME                     READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/hichat   3/3     3            3           26m
deployment.apps/mysql    1/1     1            1           39m
deployment.apps/redis    1/1     1            1           36m

NAME                               DESIRED   CURRENT   READY   AGE
replicaset.apps/hichat-9f7f97d6d   3         3         3       26m
replicaset.apps/mysql-5fb77fcc9f   1         1         1       39m
replicaset.apps/redis-7786bddb49   1         1         1       36m
```

### 1. Pods（容器实例）

**字段说明：**

- **NAME**: Pod 名称
  - 格式：`<deployment名称>-<replicaset哈希>-<pod哈希>`
  - 例如：`hichat-9f7f97d6d-d7htm` 表示这是 hichat deployment 的一个 Pod 实例

- **READY**: 就绪状态
  - 格式：`就绪的容器数/总容器数`
  - `1/1` 表示 1 个容器已就绪，共 1 个容器
  - 如果是多容器 Pod，可能是 `2/2`、`1/2` 等

- **STATUS**: 运行状态
  - `Running`: Pod 正在运行（正常状态）
  - 其他常见状态：
    - `Pending`: 等待调度
    - `Error`: 启动错误
    - `CrashLoopBackOff`: 容器反复崩溃
    - `Terminating`: 正在终止

- **RESTARTS**: 重启次数
  - `0` 表示没有重启过（正常）
  - 如果数字持续增加，说明 Pod 可能存在问题

- **AGE**: 运行时长
  - `6m26s` = 6 分 26 秒
  - `26m` = 26 分钟
  - `39m` = 39 分钟

### 2. Services（服务）

**字段说明：**

- **NAME**: Service 名称

- **TYPE**: 服务类型
  - `LoadBalancer`: 对外暴露服务，k3d 会自动映射到本地端口
  - `ClusterIP`: 仅在集群内部访问（默认类型）
  - `NodePort`: 通过节点端口访问
  - `ExternalName`: 外部服务别名

- **CLUSTER-IP**: 集群内部 IP 地址
  - 集群内的 Pod 通过这个 IP 访问服务
  - 例如：`10.43.6.114` 是 hichat 服务的集群内 IP

- **EXTERNAL-IP**: 外部 IP 地址
  - `172.21.0.2`: LoadBalancer 类型服务的外部 IP（k3d 自动分配）
  - `<none>`: ClusterIP 类型服务没有外部 IP

- **PORT(S)**: 端口映射
  - `8000:32496/TCP`: Service 端口 8000，映射到本地端口 32496
  - `3306/TCP`: 仅集群内访问，端口 3306
  - `6379/TCP`: 仅集群内访问，端口 6379

- **AGE**: 创建时长

### 3. Deployments（部署）

**字段说明：**

- **NAME**: Deployment 名称

- **READY**: 就绪副本数/期望副本数
  - `3/3` 表示 3 个 Pod 已就绪，期望 3 个（全部就绪）
  - `1/1` 表示 1 个 Pod 已就绪，期望 1 个

- **UP-TO-DATE**: 已更新到最新配置的副本数
  - `3` 表示 3 个 Pod 使用了最新的配置

- **AVAILABLE**: 可用副本数
  - `3` 表示 3 个 Pod 当前可用
  - 如果小于期望值，说明有 Pod 不可用

- **AGE**: 创建时长

### 4. ReplicaSets（副本集）

**字段说明：**

- **NAME**: ReplicaSet 名称
  - 格式：`<deployment名称>-<哈希值>`
  - Deployment 管理 ReplicaSet，ReplicaSet 管理 Pod

- **DESIRED**: 期望的副本数
  - `3` 表示期望运行 3 个 Pod
  - `1` 表示期望运行 1 个 Pod

- **CURRENT**: 当前实际的副本数
  - `3` 表示当前有 3 个 Pod 在运行
  - 如果小于 DESIRED，说明正在创建或有问题

- **READY**: 就绪的副本数
  - `3` 表示 3 个 Pod 已就绪
  - 如果小于 CURRENT，说明有 Pod 未就绪

- **AGE**: 创建时长

### 资源层级关系

```
Deployment (hichat)
    ↓ 管理
ReplicaSet (hichat-9f7f97d6d)
    ↓ 管理
Pods (d7htm, fppff, vsdj9)
    ↓ 被访问
Service (hichat)
```

**说明：**
- Deployment 定义应用的期望状态（如副本数、镜像版本等）
- ReplicaSet 负责确保指定数量的 Pod 副本在运行
- Pod 是实际运行的容器实例
- Service 提供稳定的网络访问入口，将请求分发到 Pod

### 健康状态判断

**正常状态：**
- `READY` 等于期望值（如 `3/3`、`1/1`）
- `STATUS` 为 `Running`
- `RESTARTS` 为 `0` 或很小的数字
- `AVAILABLE` 等于期望值

**异常状态：**
- `READY` 小于期望值（如 `2/3`）
- `STATUS` 不是 `Running`（如 `Error`、`CrashLoopBackOff`）
- `RESTARTS` 持续增加
- `AVAILABLE` 小于期望值

### 相关命令

```bash
# 只看 Pods
kubectl get pods -n hichat

# 只看 Services
kubectl get svc -n hichat

# 只看 Deployments
kubectl get deployments -n hichat

# 只看 ReplicaSets
kubectl get rs -n hichat

# 查看详细信息
kubectl describe pod <pod-name> -n hichat
kubectl describe svc hichat -n hichat
kubectl describe deployment hichat -n hichat
```

## 扩容和缩容

### 扩容 HiChat 服务

当需要增加 HiChat 服务的 Pod 实例数量以提高可用性和处理能力时，可以使用以下方法：

#### 方法 1: 使用 kubectl scale 命令（推荐，快速）

```bash
# 扩容到 3 个 Pod 实例
kubectl scale deployment hichat --replicas=3 -n hichat

# 查看扩容进度
kubectl get pods -n hichat -l app=hichat -w
```

#### 方法 2: 修改 YAML 文件

编辑 `k8s/hichat.yaml` 文件，修改 `replicas` 字段：

```yaml
spec:
  replicas: 3  # 从 1 改为 3
```

然后应用更改：

```bash
kubectl apply -f k8s/hichat.yaml
```

#### 方法 3: 使用 kubectl edit（临时修改）

```bash
kubectl edit deployment hichat -n hichat
```

在编辑器中找到 `replicas` 字段并修改，保存后自动生效。

### 验证扩容

```bash
# 查看 Pod 数量
kubectl get pods -n hichat -l app=hichat

# 查看 Deployment 状态
kubectl get deployment hichat -n hichat

# 查看详细信息
kubectl describe deployment hichat -n hichat
```

预期输出示例：
```
NAME                     READY   STATUS    RESTARTS   AGE
hichat-9f7f97d6d-xxx1   1/1     Running   0          2m
hichat-9f7f97d6d-xxx2   1/1     Running   0          2m
hichat-9f7f97d6d-xxx3   1/1     Running   0          2m
```

### 负载均衡

扩容后，LoadBalancer Service 会自动在所有 Pod 之间进行负载均衡。请求会被分发到不同的 Pod 实例。

验证负载均衡：

```bash
# 查看 Service 的 Endpoints（可以看到所有 Pod 的 IP）
kubectl get endpoints hichat -n hichat

# 查看 Service 详细信息
kubectl describe svc hichat -n hichat
```

### 缩容（减少 Pod 数量）

```bash
# 缩容到 1 个 Pod
kubectl scale deployment hichat --replicas=1 -n hichat

# 或者缩容到 0（停止所有实例）
kubectl scale deployment hichat --replicas=0 -n hichat
```

### 自动扩容（HPA - Horizontal Pod Autoscaler）

如果需要根据 CPU 或内存使用率自动扩容，可以配置 HPA：

```bash
# 创建 HPA（需要 metrics-server）
kubectl autoscale deployment hichat --cpu-percent=70 --min=1 --max=5 -n hichat

# 查看 HPA 状态
kubectl get hpa -n hichat
```

**注意**: k3d 默认不包含 metrics-server，需要手动安装才能使用 HPA。

### 扩容注意事项

1. **资源限制**: k3d 集群资源有限，不建议扩容过多 Pod
2. **数据库连接**: 确保数据库连接池配置足够支持多个应用实例
3. **会话状态**: 如果应用有会话状态，考虑使用 Redis 存储会话
4. **文件上传**: 如果有文件上传功能，确保使用共享存储（如 NFS）或对象存储
5. **WebSocket**: WebSocket 连接是持久连接，扩容后新连接会分配到新 Pod

## 更新部署

如果需要更新应用：

1. 修改代码
2. 重新构建镜像：`docker build -t hichat:latest .`
3. 重新导入镜像：`k3d image import hichat:latest -c mycluster`
4. 删除旧 Deployment：`kubectl delete deployment hichat -n hichat`
5. 重新部署：`kubectl apply -f k8s/hichat.yaml`

## 清理环境

如果需要完全清理部署：

```bash
# 删除命名空间（会删除所有相关资源）
kubectl delete namespace hichat

# 或者逐个删除
kubectl delete -f k8s/hichat.yaml
kubectl delete -f k8s/redis.yaml
kubectl delete -f k8s/mysql.yaml
kubectl delete -f k8s/namespace.yaml
```

## 注意事项

1. **数据持久化**: MySQL 数据存储在 PersistentVolumeClaim 中，删除 PVC 会丢失数据
2. **配置文件**: 确保 `config-debug.yaml` 在项目根目录
3. **镜像更新**: 每次修改代码后需要重新构建和导入镜像
4. **端口冲突**: 确保本地 8000 端口没有被其他服务占用
5. **资源限制**: k3d 集群资源有限，不建议在生产环境使用

## 下一步

部署成功后，可以：
1. 配置 GitLab CI/CD 自动化构建和部署
2. 集成 ArgoCD 实现 GitOps 工作流
3. 添加健康检查和就绪探针
4. 配置资源限制和请求
5. 设置监控和日志收集

---

**文档版本**: 1.0  
**最后更新**: 2025-11-17

