# GitLab CI/CD 配置指南

本文档说明如何配置和使用 GitLab CI/CD 自动构建和推送 Docker 镜像。

## 功能说明

当前 CI/CD 流程实现：
- ✅ 自动构建 Docker 镜像
- ✅ 推送到 GitLab Container Registry
- ✅ 支持多标签（分支名、latest、commit SHA）

## 前置条件

1. **GitLab 项目已创建**
2. **GitLab Runner 已配置**（需要支持 Docker）
3. **项目已推送到 GitLab**

## 配置说明

### 1. GitLab Container Registry

GitLab 会自动提供 Container Registry，地址格式：
```
registry.gitlab.com/<group>/<project>
```

### 2. CI/CD 变量（通常自动配置）

GitLab 会自动提供以下变量：
- `CI_REGISTRY`: GitLab Container Registry 地址
- `CI_REGISTRY_USER`: Registry 用户名
- `CI_REGISTRY_PASSWORD`: Registry 密码
- `CI_REGISTRY_IMAGE`: 项目镜像地址

### 3. 镜像标签策略

- `$CI_COMMIT_REF_SLUG`: 分支名（如 main、develop）
- `latest`: 最新版本
- `$CI_COMMIT_SHORT_SHA`: commit SHA 前 8 位

## 使用流程

### 1. 推送代码到 GitLab

```bash
# 添加 GitLab 远程仓库（如果还没有）
git remote add origin <gitlab-repo-url>

# 推送代码
git add .
git commit -m "Add GitLab CI/CD"
git push origin main
```

### 2. 查看 CI/CD 流水线

1. 在 GitLab 项目页面，点击左侧菜单 **CI/CD > Pipelines**
2. 查看构建状态和日志

### 3. 查看构建的镜像

1. 在 GitLab 项目页面，点击左侧菜单 **Packages & Registries > Container Registry**
2. 可以看到所有构建的镜像

## 拉取和使用镜像

### 从 GitLab Registry 拉取镜像

```bash
# 登录 GitLab Container Registry
docker login registry.gitlab.com

# 拉取镜像
docker pull registry.gitlab.com/<group>/<project>:latest

# 导入到 k3d 集群
k3d image import registry.gitlab.com/<group>/<project>:latest -c mycluster
```

### 更新 k8s Deployment 使用新镜像

```bash
# 方式 1: 直接更新镜像
kubectl set image deployment/hichat hichat=registry.gitlab.com/<group>/<project>:latest -n hichat

# 方式 2: 编辑 Deployment
kubectl edit deployment hichat -n hichat
# 修改 image 字段为新的镜像地址
```

## 本地测试 CI/CD（可选）

如果想在本地测试 CI/CD 配置：

```bash
# 安装 gitlab-runner（如果还没有）
# Linux/WSL
curl -L "https://packages.gitlab.com/install/repositories/runner/gitlab-runner/script.deb.sh" | sudo bash
sudo apt-get install gitlab-runner

# 注册 Runner（需要从 GitLab 项目获取 token）
# Settings > CI/CD > Runners > Expand
sudo gitlab-runner register
```

## 下一步：集成 ArgoCD

配置好 CI/CD 后，可以：

1. **ArgoCD 从 GitLab Registry 拉取镜像**
2. **ArgoCD 监控 Git 仓库变化**
3. **自动同步部署到集群**

---

**文档版本**: 1.0  
**最后更新**: 2025-11-17

