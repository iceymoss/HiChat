# HiChat Kubernetes 部署文档

本文档描述了如何在本地 k3d 集群中部署 HiChat 应用。

## 目录

- [前置条件](#前置条件)
- [项目结构](#项目结构)
- [配置说明](#配置说明)
- [构建步骤](#构建步骤)
- [部署步骤](#部署步骤)
- [验证部署](#验证部署)
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

