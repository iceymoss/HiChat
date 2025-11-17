# Ingress 配置指南（WSL 环境）

## 前置检查

### 1. 检查 Traefik Ingress Controller

k3d 默认使用 Traefik 作为 Ingress Controller。检查是否已安装：

```bash
# 检查 Traefik Pod
kubectl get pods -n kube-system | grep traefik

# 检查 Traefik Service
kubectl get svc -n kube-system | grep traefik

# 检查 IngressClass
kubectl get ingressclass
```

**预期输出**：
- 应该看到 `traefik` 相关的 Pod 运行中
- 应该看到 `traefik` IngressClass

如果 Traefik 没有运行，需要重新创建 k3d 集群（见步骤 0）。

---

## 完整配置步骤

### 步骤 0: 确保 k3d 集群有 Traefik（如果需要）

如果 Traefik 没有运行，重新创建集群：

```bash
# 删除现有集群
k3d cluster delete mycluster

# 创建新集群（确保启用 Traefik）
k3d cluster create mycluster --port "80:80@loadbalancer"
```

### 步骤 1: 更新 Service 配置

Service 类型需要改为 `ClusterIP`（已完成，`k8s/hichat.yaml` 已更新）。

应用更新：

```bash
kubectl apply -f k8s/hichat.yaml
```

验证 Service 类型：

```bash
kubectl get svc hichat -n hichat
```

应该看到 `TYPE` 为 `ClusterIP`。

### 步骤 2: 部署 Ingress

```bash
kubectl apply -f k8s/ingress.yaml
```

### 步骤 3: 验证 Ingress 部署

```bash
# 查看 Ingress 状态
kubectl get ingress -n hichat

# 查看详细信息
kubectl describe ingress hichat-ingress -n hichat
```

**预期输出**：
```
NAME             CLASS    HOSTS           ADDRESS   PORTS   AGE
hichat-ingress   traefik  hichat.local              80      10s
```

### 步骤 4: 配置 WSL hosts 文件

在 WSL 中编辑 hosts 文件：

```bash
# 使用 sudo 编辑 hosts 文件
sudo nano /etc/hosts
```

或者：

```bash
# 直接添加（需要 sudo）
echo "127.0.0.1 hichat.local" | sudo tee -a /etc/hosts
```

**添加的内容**：
```
127.0.0.1 hichat.local
```

保存文件（nano: Ctrl+O, Enter, Ctrl+X）

### 步骤 5: 查找 Traefik 的访问端口

```bash
# 查看 Traefik Service 的端口映射
kubectl get svc -n kube-system -l app.kubernetes.io/name=traefik
```

或者：

```bash
# 查看所有 LoadBalancer 服务
kubectl get svc --all-namespaces | grep LoadBalancer
```

**常见情况**：
- k3d 通常将 Traefik 映射到本地 80 端口
- 也可能映射到其他端口（如 8080）

### 步骤 6: 测试访问

**方式 1: 如果 Traefik 在 80 端口**

```bash
# 直接访问
curl http://hichat.local

# 或者
curl -H "Host: hichat.local" http://localhost
```

**方式 2: 如果 Traefik 在其他端口（如 8080）**

```bash
curl http://hichat.local:8080
```

**方式 3: 在浏览器中访问**

在 Windows 浏览器中访问：`http://hichat.local`

**注意**: 如果浏览器无法访问，可能需要在 Windows 的 hosts 文件中也添加域名映射。

---

## Windows hosts 文件配置（可选）

如果需要在 Windows 浏览器中访问，也需要配置 Windows hosts 文件：

1. 以**管理员身份**打开记事本
2. 打开文件：`C:\Windows\System32\drivers\etc\hosts`
3. 添加：`127.0.0.1 hichat.local`
4. 保存

---

## 验证清单

- [ ] Traefik Pod 运行中
- [ ] Service 类型为 ClusterIP
- [ ] Ingress 已创建
- [ ] WSL hosts 文件已配置
- [ ] 可以访问 http://hichat.local

---

## 故障排查

### 问题 1: Ingress 没有 ADDRESS

**原因**: Traefik 可能没有正确运行

**解决**:
```bash
# 检查 Traefik Pod
kubectl get pods -n kube-system | grep traefik

# 查看 Traefik 日志
kubectl logs -n kube-system -l app.kubernetes.io/name=traefik
```

### 问题 2: 无法访问 hichat.local

**检查**:
```bash
# 1. 检查 hosts 文件
cat /etc/hosts | grep hichat.local

# 2. 检查 Ingress
kubectl get ingress -n hichat

# 3. 检查 Service
kubectl get svc hichat -n hichat

# 4. 检查 Pod
kubectl get pods -n hichat
```

### 问题 3: 404 错误

**原因**: Ingress 路由配置可能有问题

**解决**:
```bash
# 查看 Ingress 详细信息
kubectl describe ingress hichat-ingress -n hichat

# 检查 Service 名称是否匹配
kubectl get svc -n hichat
```

---

## 常用命令

```bash
# 查看 Ingress
kubectl get ingress -n hichat

# 查看 Ingress 详细信息
kubectl describe ingress hichat-ingress -n hichat

# 查看 Traefik 日志
kubectl logs -n kube-system -l app.kubernetes.io/name=traefik --tail=50

# 测试访问
curl -v http://hichat.local

# 删除 Ingress（如果需要）
kubectl delete ingress hichat-ingress -n hichat
```

