## HiChat 整体工作流程最终版（GitOps 视角）

本篇作为总览文档，整合 `CI_CD整体工作流.md`、`GITOPS的方案对比.md` 以及各专项文档（ArgoCD 最终版、k3d 集群实践、GitLab 最终版）的内容，从“开发者写代码”一直讲到“HiChat 在 k3d 集群自动滚动更新”，帮助你快速建立整体心智模型。

---

## 1. 系统角色与组件总览

从大局看，系统由四大块组成：

- **开发环境**：IDE + Git（本地）；
- **GitLab 平台**：Git 仓库 + CI/CD + Container Registry（`gitlab.iceymoss`）；
- **k3d Kubernetes 集群**：`mycluster`，业务命名空间 `hichat`；
- **ArgoCD**：部署在 `argocd` 命名空间，负责 GitOps 同步。

简化架构关系：

```text
开发者
  │ git push (master)
  ▼
GitLab (代码仓库 + CI/CD + Registry)
  │
  ├─ build: 构建并推送 Docker 镜像
  └─ update-config: 更新 k8s-config 分支中的 k8s/hichat.yaml
        │
        ▼
    Git 仓库 (k8s-config 分支)
        │
        ▼
ArgoCD (监控 k8s-config + k8s/ 目录)
        │
        ▼
k3d 集群 (mycluster, 命名空间 hichat)
        │
        ▼
HiChat + MySQL + Redis + Ingress
```

---

## 2. 分支与仓库策略（核心设计）

### 2.1 单仓库 + 配置分支方案

仓库：`root/hichat`，采用：

- **代码分支**：`master`
  - 只存业务代码与通用文件，如 `.gitlab-ci.yml`、`Dockerfile` 等；
  - 不存自动化更新的 K8s 配置 commit。
- **配置分支**：`k8s-config`
  - 存放 `k8s/` 目录下的所有 YAML（`namespace.yaml、mysql.yaml、redis.yaml、hichat.yaml、ingress.yaml、argocd-application.yaml` 等）；
  - GitLab CI 在这里更新镜像标签；
  - ArgoCD 监控该分支，作为集群“期望状态”的单一来源。

优势（相较“全在 master”）：

- master 历史干净，专注业务代码；
- 所有配置变更集中在 `k8s-config`，可关注、可回滚；
- 满足 GitOps “Git 即真相来源”的核心要求。

---

## 3. 从零搭建顺序（高层视角）

建议按以下顺序完成环境搭建（各步骤细节在对应“最终版”文档中）：

1. **搭建 k3d 集群**（见 `基于K3D的K8S集群_最终实践指南.md`）  
   - 安装 Docker / k3d / kubectl；  
   - `k3d cluster create mycluster --port "8000:80@loadbalancer"`；  
   - 部署 `k8s/namespace.yaml、mysql.yaml、redis.yaml、hichat.yaml、ingress.yaml`。

2. **搭建 GitLab + Runner + Registry**（见 `GitLab安装与CI_CD配置_最终版.md`）  
   - `gitlab.iceymoss` 安装与配置；  
   - 启用 Registry：`gitlab.iceymoss:5050`；  
   - 安装 Runner（Docker executor + privileged + extra_hosts）；  
   - 配置 `.gitlab-ci.yml` 的 `build` + `update-config`。

3. **安装 ArgoCD 并接入 Git 仓库**（见 `ARGOCD最终版_安装与GitOps使用指南.md`）  
   - 使用 Helm 在 `argocd` 命名空间安装 ArgoCD；  
   - 通过 CLI / UI 登录；  
   - 使用 k3d 网桥 IP 将 `http://172.21.0.1/root/hichat.git` 添加为仓库。

4. **应用 ArgoCD Application 管理 HiChat**  
   - 确认 `k8s/argocd-application.yaml` 中：
     - `repoURL: http://172.21.0.1/root/hichat.git`；
     - `targetRevision: k8s-config`；
     - `path: k8s`；
   - `kubectl apply -f k8s/argocd-application.yaml`；
   - `argocd app sync hichat` 完成首轮同步。

全部完成后，一个完整的 GitOps 环境就搭好了。

---

## 4. 代码提交到上线的全链路（时序）

### 4.1 时序表

| 时间点 | 动作 | 发生位置 | 说明 |
|-------|------|----------|------|
| T0 | 开发者 `git push origin master` | 本地 → GitLab | 推送代码到 `master` |
| T1 | GitLab 创建 Pipeline | GitLab | 发现 `.gitlab-ci.yml` |
| T2 | Runner 执行 `build` Job | Runner + Docker-in-Docker | 构建并推送镜像 |
| T3 | Runner 执行 `update-config` Job | Runner + Git | 更新 `k8s-config` 分支 YAML |
| T4 | Git 上 `k8s-config` 分支变更 | GitLab | `k8s/hichat.yaml` 镜像标签被更新 |
| T5 | ArgoCD 检测到变更 | ArgoCD | 同步 `k8s/` 目录到集群 |
| T6 | K8s 执行滚动更新 | k3d 集群 | 新 Pod 上线，旧 Pod 逐步下线 |
| T7 | 用户通过 `hichat.local:8000` 访问新版 | 浏览器/客户端 | 流量自动被路由到新版本 |

### 4.2 各阶段简要说明

1. **代码提交**  
   - 规范：所有业务变更提交到 `master`；  
   - 不直接手改 `k8s-config` 分支（除非特别运维场景）。

2. **CI 构建镜像（build 阶段）**  
   - 使用 `Dockerfile` 多阶段构建；  
   - 将镜像推送到 `gitlab.iceymoss:5050/root/hichat`，并打上 `master/latest/<commit-sha>` 三类标签。

3. **CI 更新配置分支（update-config 阶段）**  
   - 从 `master` 拿 `k8s/` 目录；  
   - 更新 `k8s/hichat.yaml` 中的镜像为当前 commit SHA；  
   - 提交并推送到 `k8s-config` 分支（commit 信息带 `[skip ci]`，防止循环）。

4. **ArgoCD GitOps 同步**  
   - ArgoCD 监控 `k8s-config` 分支的 `k8s/` 目录；  
   - 发现 YAML 中的镜像标签发生变化 → 创建新 ReplicaSet，执行 Deployment 滚动更新；  
   - 自动对齐集群状态与 Git 记录的“期望配置”。

5. **集群滚动升级**  
   - 新 Pod 启动通过健康检查后加入 Service；  
   - 旧 Pod 优雅下线；  
   - Ingress / Service 持续提供稳定访问。

---

## 5. 关键文件与它们在流程中的位置

### 5.1 `.gitlab-ci.yml`

作用：定义从构建镜像到更新配置分支的自动化流程。

关键内容：

- `build`：  
  - 镜像构建 + 推送（使用 Docker-in-Docker 和 Registry）；
- `update-config`：  
  - 检出 / 创建 `k8s-config` 分支；  
  - 更新 `k8s/hichat.yaml` 中的镜像地址；  
  - 提交 `[skip ci]` commit 并推送至 `k8s-config`。

### 5.2 `k8s/hichat.yaml`

作用：定义 HiChat Deployment 与 Service。

在流程中的角色：

- 在 `master` 分支：模板版本（通常镜像可使用 `latest` 或任意固定值）；
- 在 `k8s-config` 分支：CI 根据 commit SHA 生成的真实部署版本。

### 5.3 `k8s/argocd-application.yaml`

作用：告诉 ArgoCD 要监控哪个仓库/分支/路径、往哪个集群/namespace 部署、使用何种同步策略。

在流程中的角色：

- 把 GitLab 仓库的 `k8s-config` 分支的 `k8s/` 目录与 k3d 集群中的 `hichat` 命名空间绑定起来；
- 实现“Git 配置即集群真相”的封装。

---

## 6. 常见操作场景与建议流程

### 6.1 开发新功能

1. 在本地新建/修改代码；
2. 提交并推送至 `master`；
3. 观察 GitLab Pipeline（build + update-config）；
4. 打开 ArgoCD UI 查看 `hichat` 应用是否自动同步；  
5. 在浏览器访问 `http://hichat.local:8000` 验证功能。

### 6.2 回滚到某个已知版本

有两种推荐方式：

1. **GitOps 原生回滚（推荐）**：  
   - 在 ArgoCD UI → `hichat` → History → 选择目标 revision → Rollback；  
   - ArgoCD 会把集群状态回退到某个历史配置版本（即某个 Git commit）。

2. **通过 Git 回滚配置分支**：  
   - 在本地 `git checkout k8s-config`；  
   - `git revert` 或 `git reset` 到某个历史 commit；  
   - 推送到远程 `k8s-config`；  
   - ArgoCD 检测到变更后会自动同步。

### 6.3 临时手工改配置（紧急修复）

不建议在集群上直接手改 `kubectl edit`，但如必须：

1. 先改 `k8s-config` 分支下对应 YAML（例如 `hichat.yaml`），提交并推送；  
2. 由 ArgoCD 同步下发到集群；  
3. 后续再通过正常 CI 流程把代码侧修复补齐。

---

## 7. 典型故障点与排查入口（按链路分段）

### 7.1 CI 未触发 / 构建失败

排查：

- `.gitlab-ci.yml` 是否在 `master` 分支中存在且语法正确；  
- GitLab 项目下 **CI/CD → Pipelines** 是否有新流水线产生；  
- Runner 是否 Online：**Settings → CI/CD → Runners**；
- 构建日志中是否出现 Registry 登录失败、镜像构建错误等信息。

### 7.2 Registry 相关问题

症状：

- `docker login gitlab.iceymoss:5050` 失败；
- CI 日志提示无法访问 `https://gitlab.iceymoss:5050`。

排查：

- /etc/gitlab/gitlab.rb 中 Registry 是否正确启用并使用 HTTP；  
- CI 服务部分是否带 `--insecure-registry=gitlab.iceymoss:5050`；  
- `DOCKER_TLS_CERTDIR: ""` 是否设置。

### 7.3 ArgoCD 无法拉取 Git 仓库

症状：

- ArgoCD Application 状态中提示 DNS 解析失败 `gitlab.iceymoss`；

排查与修复：

- Application `spec.source.repoURL` 中使用 k3d 网络 IP：`http://172.21.0.1/root/hichat.git`；
- 确认 GitLab 容器与集群容器在同一 Docker 网络上并可互访。

### 7.4 Application 同步失败 / 状态 OutOfSync

排查：

- `argocd app get hichat` 查看详细差异与错误；  
- `argocd app logs hichat` 查看同步日志；  
- 确认 `k8s-config` 分支中 `k8s/` 目录结构完整，并与集群版本兼容。

### 7.5 集群侧 Pod 崩溃 / 服务 500

常见原因：

- MySQL / Redis 未准备好；  
- 环境变量错误（例如 `MYSQL_HOST`、`REDIS_HOST`）；  
- Docker 镜像构建时缺少静态资源/模板。

排查：

- 查看各 Pod 日志：`kubectl logs -n hichat deployment/hichat` / `mysql` / `redis`；  
- 检查 Service 与 Endpoints：`kubectl get svc,endpoints -n hichat`；  
- 根据错误栈对应到代码与 Dockerfile。

---

## 8. 总结：如何理解这套体系

- **k3d 集群**：提供一个本地轻量级的 Kubernetes 实验场，运行 HiChat + MySQL + Redis + Ingress 等全部业务组件；
- **GitLab**：集成代码托管、CI/CD 与 Registry，承担“构建 + 制品管理 + 配置分支更新”职责；
- **ArgoCD**：只关心 Git 仓库中 `k8s-config` 分支下的 YAML，把它们转换成集群中的真实资源，并持续对齐；
- **GitOps 思想**：  
  - 所有“想要的集群状态”都写在 Git 里；  
  - 改 Git → CI 构建与改 YAML → ArgoCD 同步 → 集群自动收敛；  
  - 集群不是手工运维对象，而是 Git 状态的投影。

通过前面几篇“最终版文档”配合本篇，你可以：

1. 从零把环境搭起来；  
2. 对任何阶段的行为有清晰的 mental model；  
3. 在出问题时迅速定位是在“GitLab/CI/Registry/ArgoCD/K8s”哪个环节；  
4. 有信心在此基础上扩展更多服务或迁移到云上生产环境。


