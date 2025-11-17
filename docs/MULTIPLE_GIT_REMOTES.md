# 配置多个 Git 远程仓库

本文档说明如何配置项目同时推送到多个 Git 远程仓库（如 GitHub 和 GitLab）。

---

## 使用场景

- **GitHub**: 作为主要的代码仓库（开源项目展示）
- **GitLab**: 用于 CI/CD 自动化构建和部署

---

## 配置步骤

### 方案 1: GitHub 作为主仓库（推荐）

#### 1. 删除当前 origin

```bash
git remote remove origin
```

#### 2. 添加 GitHub 作为 origin

```bash
git remote add origin https://github.com/iceymoss/HiChat.git
```

#### 3. 添加 GitLab 作为 gitlab

```bash
git remote add gitlab http://gitlab.iceymoss/root/hichat.git
```

#### 4. 验证配置

```bash
git remote -v
```

**预期输出**:
```
origin    https://github.com/iceymoss/HiChat.git (fetch)
origin    https://github.com/iceymoss/HiChat.git (push)
gitlab    http://gitlab.iceymoss/root/hichat.git (fetch)
gitlab    http://gitlab.iceymoss/root/hichat.git (push)
```

---

## 使用方法

### 推送到 GitHub（主仓库）

```bash
git push origin master
# 或
git push origin main  # 如果 GitHub 使用 main 分支
```

### 推送到 GitLab（CI/CD 仓库）

```bash
git push gitlab master
```

### 同时推送到两个仓库

```bash
# 方式 1: 分别推送
git push origin master && git push gitlab master

# 方式 2: 创建别名（一次性设置）
git config alias.pushall '!git push origin master && git push gitlab master'

# 之后只需要
git pushall
```

---

## 方案 2: GitLab 作为主仓库

如果 GitLab 是主仓库，GitHub 是备份：

```bash
# 1. 删除当前 origin
git remote remove origin

# 2. 添加 GitLab 作为 origin
git remote add origin http://gitlab.iceymoss/root/hichat.git

# 3. 添加 GitHub 作为 github
git remote add github https://github.com/iceymoss/HiChat.git

# 4. 验证
git remote -v
```

**使用方式**:
```bash
git push origin master      # 推送到 GitLab
git push github master      # 推送到 GitHub
```

---

## 方案 3: 使用同一个 remote 配置多个 URL

让 `git push origin` 同时推送到两个仓库：

```bash
# 1. 设置第一个 URL（GitHub）
git remote add origin https://github.com/iceymoss/HiChat.git

# 2. 添加第二个 URL（GitLab）到同一个 remote
git remote set-url --add --push origin http://gitlab.iceymoss/root/hichat.git

# 3. 验证
git remote -v
```

**预期输出**:
```
origin    https://github.com/iceymoss/HiChat.git (fetch)
origin    https://github.com/iceymoss/HiChat.git (push)
origin    http://gitlab.iceymoss/root/hichat.git (push)
```

**使用方式**:
```bash
# 一次推送，同时推送到两个仓库
git push origin master
```

---

## 常用命令

### 查看所有远程仓库

```bash
git remote -v
```

### 查看特定远程仓库信息

```bash
git remote show origin
git remote show gitlab
```

### 修改远程仓库 URL

```bash
# 修改 origin 的 URL
git remote set-url origin <new-url>

# 修改 gitlab 的 URL
git remote set-url gitlab <new-url>
```

### 删除远程仓库

```bash
git remote remove gitlab
```

### 重命名远程仓库

```bash
git remote rename gitlab gitlab-ci
```

---

## 注意事项

### 1. 分支名称差异

如果两个仓库使用不同的默认分支名称：

- **GitHub**: 通常使用 `main`
- **GitLab**: 可能使用 `master`

需要分别指定：

```bash
git push origin main      # GitHub
git push gitlab master    # GitLab
```

### 2. 认证配置

确保两个仓库都有正确的认证：

**GitHub**:
- 使用 Personal Access Token（推荐）
- 或配置 SSH 密钥

**GitLab**:
- 使用用户名密码
- 或使用 Personal Access Token

### 3. 首次推送

首次推送到新仓库时，使用 `-u` 参数设置上游分支：

```bash
git push -u origin master
git push -u gitlab master
```

### 4. 同步仓库

如果两个仓库内容不同，可能需要先同步：

```bash
# 从 GitHub 拉取
git pull origin master

# 推送到 GitLab
git push gitlab master
```

---

## 推荐配置

**推荐使用方案 1**（GitHub 作为 origin，GitLab 作为 gitlab），原因：

✅ **清晰明确**: 两个仓库分别管理，职责分明  
✅ **灵活可控**: 可以单独推送到某个仓库  
✅ **安全可靠**: 一个仓库失败不影响另一个  
✅ **易于维护**: 配置简单，易于理解

---

## 快速参考

```bash
# 完整配置（GitHub 作为主仓库）
git remote remove origin
git remote add origin https://github.com/iceymoss/HiChat.git
git remote add gitlab http://gitlab.iceymoss/root/hichat.git
git remote -v

# 日常使用
git push origin master      # 推送到 GitHub
git push gitlab master      # 推送到 GitLab
git push origin master && git push gitlab master  # 同时推送
```

---

**文档版本**: 1.0  
**最后更新**: 2025-11-17

