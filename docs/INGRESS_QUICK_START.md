# HiChat Ingress 快速部署指南

使用 Ingress 方式在 k3d 集群中快速部署 HiChat。

## 前置条件

- WSL2 或 Linux 环境
- Docker、k3d、kubectl 已安装

## 快速部署

### 1. 创建集群（带端口映射）

```bash
k3d cluster create mycluster --port "8000:80@loadbalancer"
```

### 2. 构建并导入镜像

```bash
docker build -t hichat:latest .
k3d image import hichat:latest -c mycluster
```

### 3. 部署服务

```bash
# 命名空间
kubectl apply -f k8s/namespace.yaml

# MySQL
kubectl apply -f k8s/mysql.yaml
kubectl wait --for=condition=ready pod -l app=mysql -n hichat --timeout=120s

# Redis
kubectl apply -f k8s/redis.yaml
kubectl wait --for=condition=ready pod -l app=redis -n hichat --timeout=60s

# HiChat 应用
kubectl apply -f k8s/hichat.yaml
kubectl wait --for=condition=ready pod -l app=hichat -n hichat --timeout=60s

# Ingress
kubectl apply -f k8s/ingress.yaml
```

### 4. 配置 hosts

```bash
echo "127.0.0.1 hichat.local" | sudo tee -a /etc/hosts
```

### 5. 访问应用

```bash
# 测试
curl http://hichat.local:8000

# 浏览器访问
# http://hichat.local:8000
```

## 一键部署脚本

```bash
# 创建集群
k3d cluster create mycluster --port "8000:80@loadbalancer"

# 构建并导入镜像
docker build -t hichat:latest .
k3d image import hichat:latest -c mycluster

# 部署所有服务
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/mysql.yaml && kubectl wait --for=condition=ready pod -l app=mysql -n hichat --timeout=120s
kubectl apply -f k8s/redis.yaml && kubectl wait --for=condition=ready pod -l app=redis -n hichat --timeout=60s
kubectl apply -f k8s/hichat.yaml && kubectl wait --for=condition=ready pod -l app=hichat -n hichat --timeout=60s
kubectl apply -f k8s/ingress.yaml

# 配置 hosts
echo "127.0.0.1 hichat.local" | sudo tee -a /etc/hosts

# 验证
kubectl get all -n hichat
curl http://hichat.local:8000
```

## 常见问题

### Pod ImagePullBackOff

```bash
k3d image import hichat:latest -c mycluster
kubectl rollout restart deployment/hichat -n hichat
```

### 访问 404

```bash
# 检查 Pod 和 Endpoints
kubectl get pods -n hichat
kubectl get endpoints hichat -n hichat
```

### 清理旧 ReplicaSet

```bash
kubectl delete replicaset <replicaset-name> -n hichat
```

## 更新应用

```bash
docker build -t hichat:latest .
k3d image import hichat:latest -c mycluster
kubectl rollout restart deployment/hichat -n hichat
```

## 清理

```bash
k3d cluster delete mycluster
```

---

**访问地址**: http://hichat.local:8000
