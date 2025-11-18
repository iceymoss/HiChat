# GitOps 最佳实践：代码与配置分离

本文档说明 GitOps 中代码和配置分离的主流最佳实践，以及为什么拉取 master 分支时看不到配置更新的 commit。

**文档版本**: 1.0  
**最后更新**: 2025-11-17

---

## 目录

1. [为什么看不到配置更新的 commit？](#为什么看不到配置更新的-commit)
2. [主流最佳实践方案](#主流最佳实践方案)
3. [方案对比](#方案对比)
4. [推荐实现：配置分支方案](#推荐实现配置分支方案)
5. [替代方案：GitLab API 更新文件](#替代方案gitlab-api-更新文件)

---

## 为什么看不到配置更新的 commit？

### 现象

在 GitLab Web UI 的 master 分支可以看到这些 commit：
```
commit jkl012 "chore: update image to 789abc12 [skip ci]" (CI/CD 自动生成)
```

但是当你 `git pull origin master` 时，却看不到这些 commit。

### 原因分析

**最可能的原因：使用了配置分支**

1. **代码在 `master` 分支**：包含源代码
2. **配置在 `k8s-config` 分支**：包含 Kubernetes 配置文件
3. **ArgoCD 监控配置分支**：`targetRevision: k8s-config`
4. **CI/CD 更新配置分支**：`update-config` job 推送到 `k8s-config` 分支

因此：
- GitLab Web UI 显示的是配置分支的 commit（如果切换到配置分支查看）
- 或者 GitLab Web UI 显示的是 master 分支，但配置更新实际在配置分支
- 当你拉取 master 分支时，自然看不到配置分支的 commit

**其他可能原因**：

1. **使用了单独的配置仓库**：配置更新在另一个 Git 仓库中
2. **使用了 GitLab Repository Files API**：直接更新文件内容，不产生 commit（但 ArgoCD 仍可通过轮询检测到变化）

---

## 主流最佳实践方案

### 方案 1：分离代码和配置仓库 ⭐⭐⭐⭐⭐（最推荐）

**架构**：
- **代码仓库**：`hichat` - 只包含源代码
- **配置仓库**：`hichat-k8s-config` - 只包含 k8s 配置文件

**工作流**：
```
开发者推送代码到 hichat 仓库
    ↓
GitLab CI/CD 构建镜像
    ↓
CI/CD 更新 hichat-k8s-config 仓库的配置文件
    ↓
ArgoCD 监控 hichat-k8s-config 仓库
    ↓
自动部署到集群
```

**优点**：
- ✅ 职责清晰：代码和配置完全分离
- ✅ 权限分离：可以给不同团队不同的仓库权限
- ✅ Git 历史干净：代码仓库不包含配置更新
- ✅ 符合 GitOps 最佳实践

**缺点**：
- ❌ 需要管理两个仓库
- ❌ 需要配置两个仓库的 CI/CD

**适用场景**：
- 大型项目
- 多团队协作
- 需要严格的权限控制

---

### 方案 2：使用配置分支 ⭐⭐⭐⭐（次推荐）

**架构**：
- **代码分支**：`master` - 包含源代码
- **配置分支**：`k8s-config` - 包含 k8s 配置文件

**工作流**：
```
开发者推送代码到 master 分支
    ↓
GitLab CI/CD 构建镜像
    ↓
CI/CD 更新 k8s-config 分支的配置文件
    ↓
ArgoCD 监控 k8s-config 分支
    ↓
自动部署到集群
```

**优点**：
- ✅ 单仓库管理：代码和配置在同一个仓库
- ✅ Git 历史清晰：master 分支不包含配置更新
- ✅ 易于切换：可以轻松查看不同分支
- ✅ 符合 GitOps 原则

**缺点**：
- ❌ 需要切换分支查看配置
- ❌ 需要维护两个分支

**适用场景**：
- 中小型项目
- 单团队开发
- 希望保持单仓库

---

### 方案 3：在 master 分支直接更新 ⭐⭐⭐（简单但不推荐）

**架构**：
- **所有内容在 `master` 分支**：代码和配置都在同一个分支

**工作流**：
```
开发者推送代码到 master 分支
    ↓
GitLab CI/CD 构建镜像
    ↓
CI/CD 更新 master 分支的配置文件（产生 commit）
    ↓
ArgoCD 监控 master 分支
    ↓
自动部署到集群
```

**优点**：
- ✅ 简单直接：不需要管理多个分支或仓库
- ✅ 所有变更都在一个地方

**缺点**：
- ❌ Git 历史包含自动化 commit：`chore: update image to xxx [skip ci]`
- ❌ 代码和配置混合：不够清晰
- ❌ 不符合大型项目的最佳实践

**适用场景**：
- 小型项目
- 个人项目
- 快速原型

---

### 方案 4：使用 GitLab API 更新文件 ⭐⭐（不推荐用于 GitOps）

**架构**：
- 使用 GitLab Repository Files API 直接更新文件内容
- 不产生 commit，但文件内容会变化

**工作流**：
```
开发者推送代码到 master 分支
    ↓
GitLab CI/CD 构建镜像
    ↓
CI/CD 使用 API 更新文件内容（不产生 commit）
    ↓
ArgoCD 通过轮询检测到文件内容变化
    ↓
自动部署到集群
```

**优点**：
- ✅ Git 历史干净：不产生 commit

**缺点**：
- ❌ 失去可追溯性：无法通过 Git 历史查看配置变更
- ❌ 不符合 GitOps 原则：Git 不是唯一的事实来源
- ❌ 难以回滚：无法通过 Git 回滚配置

**适用场景**：
- 不推荐用于生产环境
- 仅用于特殊场景

---

## 方案对比

| 方案 | Git 历史干净 | 可追溯性 | 可回滚 | 复杂度 | 推荐度 |
|------|-------------|---------|--------|--------|--------|
| 分离仓库 | ✅✅✅ | ✅✅✅ | ✅✅✅ | 中 | ⭐⭐⭐⭐⭐ |
| 配置分支 | ✅✅ | ✅✅✅ | ✅✅✅ | 低 | ⭐⭐⭐⭐ |
| master 直接更新 | ❌ | ✅✅✅ | ✅✅✅ | 极低 | ⭐⭐⭐ |
| API 更新文件 | ✅✅✅ | ❌ | ❌ | 中 | ⭐⭐ |

---

## 推荐实现：配置分支方案

基于你的情况，推荐使用**配置分支方案**，这样：
- ✅ master 分支保持干净（只包含源代码）
- ✅ 配置更新在 `k8s-config` 分支
- ✅ ArgoCD 监控配置分支
- ✅ 符合主流最佳实践

### 实现步骤

#### 步骤 1: 创建配置分支

```bash
# 从 master 分支创建配置分支
git checkout master
git checkout -b k8s-config

# 推送配置分支到远程
git push origin k8s-config
```

#### 步骤 2: 修改 ArgoCD 监控配置分支

修改 `k8s/argocd-application.yaml`：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hichat
  namespace: argocd
spec:
  source:
    repoURL: http://172.21.0.1/root/hichat.git
    targetRevision: k8s-config  # 改为监控配置分支
    path: k8s
  # ... 其他配置保持不变
```

#### 步骤 3: 修改 CI/CD 更新配置分支

修改 `.gitlab-ci.yml` 中的 `update-config` job：

```yaml
update-config:
  stage: update-config
  image: alpine/git:latest
  before_script:
    - git config --global user.email "ci@gitlab.iceymoss"
    - git config --global user.name "GitLab CI"
  script:
    - echo "更新 Kubernetes 配置文件中的镜像标签..."
    # 更新 k8s/hichat.yaml 中的镜像标签
    - sed -i "s|image:.*hichat.*|image: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA|g" k8s/hichat.yaml
    # 显示更新后的内容
    - echo "更新后的镜像配置:"
    - grep "image:" k8s/hichat.yaml
    # 提交更改到配置分支
    - git add k8s/hichat.yaml
    - git commit -m "chore: update image to $CI_COMMIT_SHORT_SHA [skip ci]" || exit 0
    # 推送到配置分支（不是 master）
    - git push https://oauth2:$CI_JOB_TOKEN@gitlab.iceymoss/root/hichat.git HEAD:k8s-config || exit 0
    - echo "配置更新完成，ArgoCD 将自动同步部署"
  only:
    - master  # 只在 master 分支的代码推送时触发
  when: on_success
  tags:
    - docker
```

#### 步骤 4: 应用配置

```bash
# 提交配置分支的更改
git add k8s/argocd-application.yaml .gitlab-ci.yml
git commit -m "feat: 使用配置分支分离代码和配置"
git push origin master

# 更新 ArgoCD Application
kubectl apply -f k8s/argocd-application.yaml
```

### 工作流说明

```
开发者推送代码到 master 分支
    ↓
GitLab CI/CD 触发 Pipeline（在 master 分支上）
    ↓
build 阶段：构建并推送镜像
    ↓
update-config 阶段：
  - 更新 k8s/hichat.yaml 中的镜像标签
  - 提交到 k8s-config 分支（不是 master）
  - 推送到远程 k8s-config 分支
    ↓
ArgoCD 检测到 k8s-config 分支的变化
    ↓
自动同步部署到集群
```

### 验证

```bash
# 1. 查看 master 分支（应该看不到配置更新的 commit）
git checkout master
git log --oneline
# 应该只看到你的正常提交，没有 "chore: update image" 的 commit

# 2. 查看配置分支（应该看到配置更新的 commit）
git checkout k8s-config
git log --oneline
# 应该看到 "chore: update image" 的 commit

# 3. 查看 ArgoCD 状态
argocd app get hichat
# 应该显示监控 k8s-config 分支
```

---

## 替代方案：GitLab API 更新文件

如果你确实希望 Git 历史完全干净（不推荐），可以使用 GitLab Repository Files API：

### 实现方式

```yaml
update-config:
  stage: update-config
  image: curlimages/curl:latest
  script:
    - |
      # 读取当前文件内容
      CURRENT_CONTENT=$(curl -s --header "PRIVATE-TOKEN: $CI_JOB_TOKEN" \
        "http://gitlab.iceymoss/api/v4/projects/$CI_PROJECT_ID/repository/files/k8s%2Fhichat.yaml/raw?ref=master")
      
      # 更新镜像标签
      NEW_CONTENT=$(echo "$CURRENT_CONTENT" | \
        sed "s|image:.*hichat.*|image: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA|g")
      
      # 使用 API 更新文件（不产生 commit）
      curl -X PUT \
        --header "PRIVATE-TOKEN: $CI_JOB_TOKEN" \
        --header "Content-Type: application/json" \
        --data "{\"branch\":\"master\",\"content\":\"$NEW_CONTENT\",\"commit_message\":\"[skip ci]\"}" \
        "http://gitlab.iceymoss/api/v4/projects/$CI_PROJECT_ID/repository/files/k8s%2Fhichat.yaml"
  only:
    - master
  when: on_success
```

**注意**：这种方式不产生 commit，ArgoCD 需要通过轮询检测文件内容变化。

---

## 总结

### 推荐方案

根据你的情况，**推荐使用配置分支方案**：

1. ✅ **master 分支保持干净**：只包含源代码，不包含配置更新
2. ✅ **配置分支包含所有配置更新**：可追溯、可回滚
3. ✅ **符合主流最佳实践**：大多数公司采用这种方式
4. ✅ **易于实现**：只需要创建分支并修改配置

### 为什么看不到配置更新的 commit？

**答案**：因为配置更新在配置分支（`k8s-config`），而不是 master 分支。当你拉取 master 分支时，自然看不到配置分支的 commit。

### 如何查看配置更新？

```bash
# 切换到配置分支
git checkout k8s-config
git pull origin k8s-config

# 查看配置更新的 commit
git log --oneline

# 查看配置文件
cat k8s/hichat.yaml
```

---

**文档版本**: 1.0  
**最后更新**: 2025-11-17

