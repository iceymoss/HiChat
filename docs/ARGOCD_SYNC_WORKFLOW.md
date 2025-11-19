# ArgoCD GitHub 手动同步流程

> 适用于当前 HiChat 项目：镜像已构建好，通过手动维护 GitHub `k8s-config` 分支 + ArgoCD 自动同步来完成部署。

---

## 1. 前置条件

1. k3d 集群已就绪（`k3d cluster list` 正常）。
2. ArgoCD 已通过 Helm 安装并在 `argocd` 命名空间运行。
3. 本地已安装 `kubectl`、`helm`、`argocd` CLI，并可访问集群。
4. GitHub 仓库：https://github.com/iceymoss/HiChat  
   - 主分支 `master` 存放代码  
   - `k8s-config` 分支存放 Kubernetes 清单（ArgoCD 监听目标）

---

## 2. 启动 ArgoCD 访问

```bash
# 端口转发（保持窗口挂起）
kubectl port-forward svc/argocd-server -n argocd 8443:443
```

在新的终端窗口登录 ArgoCD CLI：

```bash
# 首次需要管理员密码，可从 secret 获取
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d

argocd login localhost:8443 \
  --username admin \
  --password <ADMIN_PASSWORD> \
  --insecure
```

---

## 3. 配置仓库与 Application

1. **添加 GitHub 仓库**（公共仓库无需凭证）：
   ```bash
   argocd repo add https://github.com/iceymoss/HiChat.git
   ```

2. **应用 Application YAML**（repoURL 已指向 GitHub）：
   ```bash
   kubectl apply -f k8s/argocd-application.yaml
   ```

3. **验证应用状态**：
   ```bash
   argocd app list
   argocd app get hichat
   ```

若需要立即部署，执行 `argocd app sync hichat`；也可以在 Web UI 中点击 `SYNC`。

---

## 4. 手动更新 `k8s-config` 分支

当前不使用 CI/CD，直接手动推送：

```bash
git checkout master && git pull
git checkout k8s-config
git checkout master -- k8s/

# 修改 k8s/ 内的 YAML，例如更新镜像标签

git add k8s/
git commit -m "chore: update manifests"
git push origin k8s-config
```

ArgoCD 会自动检测到 `k8s-config` 分支变化并触发同步。如果想立即验证，可运行：

```bash
argocd app sync hichat
argocd app get hichat
```

---

## 5. 常见问题

| 问题 | 处理方式 |
| --- | --- |
| CLI 报 `exec format error` | 下载与芯片对应的 `argocd` 二进制（M2 → `argocd-darwin-arm64`）。 |
| `argocd version` 提示 `server address unspecified` | 先执行 `argocd login localhost:8443 ...`。 |
| Web UI `ComparisonError` 指向旧的 GitLab 仓库 | 重新应用 `k8s/argocd-application.yaml`，确保 `repoURL` 已更新为 GitHub，并在 CLI/UI 中查看 `Source`。 |
| 端口转发中断导致 CLI 连接失败 | 重新运行 `kubectl port-forward ...` 后再次登录。 |

---

## 6. 验证参考

- 运行 `argocd version` 可同时看到本地 CLI（darwin/arm64）和集群内 `argocd-server`（linux/arm64）版本。
- `argocd app sync hichat` 输出中所有资源状态为 `Synced/Healthy`，表示部署完成。Web UI 中也会显示绿色节点拓扑。

---

完成上述流程后，即可通过手动维护 GitHub `k8s-config` 分支 + ArgoCD 自动同步的方式，持续更新 HiChat 在 k3d 集群中的部署。若将来接入 CI/CD，只需让流水线自动修改并推送该分支即可。

