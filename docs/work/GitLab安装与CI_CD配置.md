## GitLab 安装与 CI/CD 配置最终版（本地 GitOps 基础设施）

本指南在 `LOCAL_GITLAB_SETUP.md`、`GITLAB_CI_CD.md`、`GITLAB_CI的必要配置.md` 等文档基础上，梳理出**从零安装本地 GitLab，到 Runner/Registry/CI/CD 全部打通**的一条主线，并刻意对与 ArgoCD 与 k3d 集群相关的关键参数做了收敛说明。

---

## 1. 目标与整体结构

- **GitLab 服务**：本地单机 GitLab CE（域名 `gitlab.iceymoss`）
- **仓库**：`root/hichat`（代码与配置同仓库，采用配置分支方案）
- **Runner**：Docker executor，支持 Docker-in-Docker（镜像构建+推 Registry）
- **Registry**：`http://gitlab.iceymoss:5050`（HTTP，非 HTTPS）
- **CI/CD 能力**：
  - 构建 HiChat Docker 镜像（多阶段构建）；
  - 推送至 Registry（多标签：分支、latest、commit SHA）；
  - 更新 `k8s-config` 分支中的 `k8s/hichat.yaml` 镜像标签；
  - 触发 ArgoCD 自动同步，实现 GitOps。

---

## 2. 安装本地 GitLab CE

### 2.1 系统与依赖

前置环境：

- WSL2 / 纯 Linux；
- 至少 4GB 内存（推荐 8GB）、10GB 磁盘。

安装依赖：

```bash
sudo apt-get update
sudo apt-get install -y curl openssh-server ca-certificates tzdata perl postfix
```

postfix 配置可选 “No configuration”。

### 2.2 安装 GitLab CE

```bash
curl -sS https://packages.gitlab.com/install/repositories/gitlab/gitlab-ce/script.deb.sh | sudo bash

sudo EXTERNAL_URL="http://gitlab.iceymoss" apt-get install gitlab-ce
```

可在 `/etc/gitlab/gitlab.rb` 中精简资源：

```ruby
external_url 'http://gitlab.iceymoss'

puma['worker_processes'] = 2
sidekiq['max_concurrency'] = 5
prometheus_monitoring['enable'] = false
```

应用配置：

```bash
sudo gitlab-ctl reconfigure
```

### 2.3 首次登录与 hosts 配置

获取初始 `root` 密码：

```bash
sudo cat /etc/gitlab/initial_root_password
```

WSL hosts：

```bash
echo "127.0.0.1 gitlab.iceymoss" | sudo tee -a /etc/hosts
```

如需在 Windows 浏览器访问，再在 Windows `hosts` 中添加同一条。

浏览器打开 `http://gitlab.iceymoss`，使用 `root` + 初始密码登录并修改密码。

---

## 3. 启用 Container Registry

### 3.1 配置 registry

编辑 `/etc/gitlab/gitlab.rb`：

```ruby
registry['enable'] = true
registry_external_url 'http://gitlab.iceymoss:5050'
gitlab_rails['registry_enabled'] = true
```

或直接追加：

```bash
sudo bash -c "cat >> /etc/gitlab/gitlab.rb << 'EOF'

registry['enable'] = true
registry_external_url 'http://gitlab.iceymoss:5050'
gitlab_rails['registry_enabled'] = true
EOF"
```

重新配置并重启：

```bash
sudo gitlab-ctl reconfigure
sudo gitlab-ctl restart
```

### 3.2 验证 Registry

```bash
sudo gitlab-ctl status | grep registry
curl -I http://gitlab.iceymoss:5050/v2/
```

返回 `401 Unauthorized` 即表示服务正常启动。

---

## 4. 创建 HiChat 项目并推送代码

### 4.1 GitLab 上创建项目

1. 登录 GitLab；
2. `New project` → `Create blank project`；
3. Project name: `hichat`（路径 `root/hichat`）；
4. Visibility 按需选择；
5. 创建完成。

### 4.2 本地仓库与远程绑定

在 HiChat 项目根目录：

```bash
git remote remove origin 2>/dev/null || true
git remote add origin http://gitlab.iceymoss/root/hichat.git

git push -u origin master   # 或 main, 根据本地分支名
```

如提示认证失败，可：

- 使用用户名 `root` + 密码；
- 或创建 Personal Access Token 作为密码。

---

## 5. 安装与注册 GitLab Runner（Docker executor）

### 5.1 安装 Runner

```bash
curl -L "https://packages.gitlab.com/install/repositories/runner/gitlab-runner/script.deb.sh" | sudo bash
sudo apt-get install gitlab-runner

sudo usermod -aG docker $USER
newgrp docker
```

### 5.2 在项目页面获取注册 token

路径：**Project → Settings → CI/CD → Runners → New project runner**：

- Tags: 建议使用 `docker`；
- 复制注册命令与 token。

### 5.3 注册 Runner

```bash
sudo gitlab-runner register
```

关键选项：

- GitLab URL: `http://gitlab.iceymoss`
- Registration token: 来自上一步；
- Description: `local-runner`;
- Tags: `docker`;
- Executor: `docker`;
- Default Docker image: `docker:24`.

### 5.4 配置 Runner 支持 Docker-in-Docker

编辑 `/etc/gitlab-runner/config.toml`（或用户模式下的 `~/.gitlab-runner/config.toml`）：

```toml
[[runners]]
  name = "local-runner"
  url = "http://gitlab.iceymoss"
  token = "YOUR_TOKEN"
  executor = "docker"

  [runners.docker]
    image = "docker:24"
    privileged = true
    tls_verify = false
    volumes = ["/cache"]
    extra_hosts = ["gitlab.iceymoss:host-gateway"]
```

说明：

- `privileged = true`：允许 Runner 容器内使用 Docker-in-Docker；
- `extra_hosts`：让 CI 容器内可以通过 `gitlab.iceymoss` 访问宿主上的 GitLab/Registry。

启动或重启 Runner：

```bash
sudo gitlab-runner restart
sudo gitlab-runner status
```

页面 **Settings → CI/CD → Runners** 中应显示 Runner 为 Online。

---

## 6. 配置 `.gitlab-ci.yml`：构建 + 推送 + 更新配置

### 6.1 基础 build 阶段（构建并推送镜像）

核心模式（简化版）：

```yaml
stages:
  - build
  - update-config

variables:
  DOCKER_DRIVER: overlay2
  DOCKER_TLS_CERTDIR: ""
  CI_REGISTRY: "gitlab.iceymoss:5050"
  CI_REGISTRY_IMAGE: "gitlab.iceymoss:5050/root/hichat"
  IMAGE_TAG: $CI_REGISTRY_IMAGE:$CI_COMMIT_REF_SLUG
  IMAGE_TAG_LATEST: $CI_REGISTRY_IMAGE:latest
  IMAGE_TAG_COMMIT: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA

build:
  stage: build
  image: docker:24
  services:
    - name: docker:24-dind
      command: ["--insecure-registry=gitlab.iceymoss:5050"]
  before_script:
    - |
      echo "登录 GitLab Container Registry: $CI_REGISTRY"
      if [ -n "$CI_REGISTRY_USER" ] && [ -n "$CI_REGISTRY_PASSWORD" ]; then
        docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
      else
        docker login -u gitlab-ci-token -p $CI_JOB_TOKEN $CI_REGISTRY
      fi
  script:
    - docker build -t $IMAGE_TAG -t $IMAGE_TAG_LATEST -t $IMAGE_TAG_COMMIT .
    - docker push $IMAGE_TAG
    - docker push $IMAGE_TAG_LATEST
    - docker push $IMAGE_TAG_COMMIT
  only:
    - master
  tags:
    - docker
```

要点：

- 使用 `docker:24` + `docker:24-dind` 组合；
- `DOCKER_TLS_CERTDIR: ""` + `--insecure-registry` 允许 HTTP Registry；
- 推送三个标签：分支名、`latest`、commit SHA。

### 6.2 更新 `k8s-config` 分支配置（GitOps 关键）

推荐的配置分支方案已在 `CI_CD整体工作流.md` 与 `GITOPS的方案对比.md` 中详细描述，这里给出最终版 Job：

```yaml
update-config:
  stage: update-config
  image: alpine/git:latest
  before_script:
    - git config --global user.email "ci@gitlab.iceymoss"
    - git config --global user.name "GitLab CI"
  script:
    - echo "准备切换到 k8s-config 分支..."
    - git fetch origin
    - git checkout -B k8s-config origin/k8s-config || git checkout -b k8s-config
    - git checkout master -- k8s/
    - |
      sed -i "s|image:.*hichat.*|image: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA|g" k8s/hichat.yaml
    - echo "更新后的镜像配置："
    - grep "image:" k8s/hichat.yaml
    - git add k8s/hichat.yaml
    - git commit -m "chore: update image to $CI_COMMIT_SHORT_SHA [skip ci]" || echo "no changes"
    - |
      git push https://oauth2:${CI_JOB_TOKEN}@gitlab.iceymoss/root/hichat.git \
        HEAD:k8s-config || echo "push failed or no changes"
  only:
    - master
  when: on_success
  tags:
    - docker
```

说明：

- **分支角色**：
  - `master`：代码分支（CI 从这里构建镜像）；
  - `k8s-config`：配置分支（ArgoCD 监控此分支）；
- **更新逻辑**：
  - 先确保本地 `k8s-config` 分支与远程同步；
  - 从 `master` 拿最新的 `k8s/` 目录；
  - 使用 `sed` 把 `k8s/hichat.yaml` 中镜像更新为 `$CI_COMMIT_SHORT_SHA`；
  - 提交带 `[skip ci]` 的 commit（避免循环）；
  - 推送到 `k8s-config` 分支。

此 Job 成功后，ArgoCD 会检测到配置分支变更，并自动同步到 k3d 集群。

---

## 7. 关键变量与环境配置一览

- **GitLab 侧**：
  - 域名：`gitlab.iceymoss`；
  - Web：`http://gitlab.iceymoss`；
  - Registry：`http://gitlab.iceymoss:5050`；
  - 默认用户：`root` + 初始密码（`/etc/gitlab/initial_root_password`）。

- **Runner 配置**（`config.toml`）：
  - `executor = "docker"`；
  - `runners.docker.image = "docker:24"`；
  - `runners.docker.privileged = true`；
  - `runners.docker.extra_hosts = ["gitlab.iceymoss:host-gateway"]`。

- **CI 变量**（隐式）：
  - `CI_REGISTRY`、`CI_REGISTRY_USER`、`CI_REGISTRY_PASSWORD`；
  - `CI_JOB_TOKEN`；
  - `CI_COMMIT_REF_SLUG`、`CI_COMMIT_SHORT_SHA`。

- **K8s/ArgoCD 侧**（由其他文档详细说明，这里只给出 GitLab 相关值）：
  - ArgoCD Application `spec.source.repoURL`: `http://172.21.0.1/root/hichat.git`（k3d 网桥 IP，而非 `gitlab.iceymoss`）；
  - `targetRevision: k8s-config`。

---

## 8. CI/CD + GitOps 流程回顾（从 GitLab 视角）

1. 开发者向 `master` 推送代码；
2. GitLab 检测到 `.gitlab-ci.yml`：
   - 触发 `build`：在 Runner 容器中，通过 Docker-in-Docker 构建镜像并推送至 Registry；
   - 触发 `update-config`：更新 `k8s-config` 分支的 `k8s/hichat.yaml` 镜像标签并推送；
3. ArgoCD（在 k3d 集群内）监控 `k8s-config` 分支：
   - 检测到配置变更（镜像标签变化）；
   - 自动同步 Deployment；
   - K8s 执行滚动更新，最终 HiChat Pod 使用最新镜像。

整个链路中，GitLab 负责：

- 源代码托管；
- 镜像构建 & Registry；
- 配置分支更新。

K8s + ArgoCD 负责：

- 集群运行；
- GitOps 同步与回滚。

---

## 9. 快速验证清单

- **GitLab**：
  - `sudo gitlab-ctl status` 全部 `run`；
  - `http://gitlab.iceymoss` 能登录；
  - `Packages & Registries → Container Registry` 能看到构建出的镜像。

- **Runner**：
  - 页面 Runners 区域显示 `Online`；
  - Pipeline 能被分配到指定 Runner 执行。

- **Registry**：
  - `curl -I http://gitlab.iceymoss:5050/v2/` 返回 401；
  - 本地能 `docker login gitlab.iceymoss:5050` 并 `docker pull/push`。

- **CI/CD Pipeline**：
  - `build` Job 日志中能看到 `docker push` 成功；
  - `update-config` Job 能看到 `k8s/hichat.yaml` 中 `image:` 被替换为最新 commit SHA，并成功推送到 `k8s-config`。

通过以上检查后，GitLab 侧的基础设施就准备好，可以与 ArgoCD 与 k3d 集群共同组成完整的 GitOps 环境。


