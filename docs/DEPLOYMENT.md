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
│   └── hichat.yaml         # HiChat 应用部署配置
├── config-debug.yaml       # 应用配置文件
├── hi_chat.sql             # 数据库初始化 SQL（已集成到 mysql.yaml）
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
- **hichat.yaml**: 创建 HiChat 应用 Deployment 和 LoadBalancer Service

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

```bash
kubectl apply -f k8s/hichat.yaml
```

等待应用 Pod 就绪：

```bash
kubectl wait --for=condition=ready pod -l app=hichat -n hichat --timeout=60s
```

### 步骤 6: 检查所有资源状态

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

**方式 1: 使用 port-forward（推荐）**

```bash
kubectl port-forward svc/hichat 8000:8000 -n hichat
```

然后在浏览器访问：`http://localhost:8000`

**方式 2: 使用 k3d LoadBalancer**

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

