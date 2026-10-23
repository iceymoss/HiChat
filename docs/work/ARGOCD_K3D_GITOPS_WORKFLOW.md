# GitOps Workflow Configuration Based on ArgoCD and K3D

## Prerequisites

You need to understand and install:

1. K3D K8S cluster setup model

2. ArgoCD installation and configuration

3. You need to first understand all other documents in this project:

```
ARGOCD_INSTALL.md DOCKER_INSTALL.md  DOCKER_REGISTRY.md  GITLAB-RUNNER.md  GITLAB_INSTALL.md  K3D_INSTALL.md  NETWORK.md  README.md  REFERENCES.md
```

## Environment Preparation

Server 1: Install ArgoCD to manage cluster applications

Server 2: Install K3D to run actual cluster applications

## Architecture Diagram

Architecture Relationship

```
┌─────────────────────────────────────────────────────────────┐
│  Server 1 (192.168.4.105)                                      │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  ArgoCD Service                                         │ │
│  │  - ArgoCD Server (Management Interface)                 │ │
│  │  - ArgoCD Application Controller (Sync Controller)      │ │
│  │  - Manage multiple clusters via kubectl/argocd CLI       │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  kubeconfig Configuration:                                   │
│  - context: k3d-argocd (Local Cluster)                       │
│  - context: k3d-hichat-cluster (Points to Server 2)         │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ Connect via 192.168.4.106:6444
                            │
┌───────────────────────────▼───────────────────────────────────┐
│  Server 2 (192.168.4.106)                                      │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  k3d Cluster (mycluster)                                │ │
│  │  - API Server (Listening on 0.0.0.0:6444)              │ │
│  │  - Actual Application Workloads (HiChat, etc.)          │ │
│  │  - ArgoCD will deploy applications here                 │ │
│  └─────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────┘
```

## Create K3D Cluster on Server 2

### Preparation

Since our image registry is private, k8s usually needs to use https. Here we need to configure:

```shell
vim /tmp/registry-config.yaml

# Input
mirrors:
  "192.168.4.104:5000":
    endpoint:
      - "http://192.168.4.104:5000"
```

### Create Cluster

Execute the following command to create the cluster:

```shell
k3d cluster create hichat-cluster   --api-port 0.0.0.0:6444   --port "8001:80@loadbalancer"   --k3s-arg "--tls-san=192.168.4.106@server:0"   --registry-config /tmp/registry-config.yaml
```

Explanation:

1. `--api-port 0.0.0.0:6444`: Specifies the cluster's API port. Here it's set to 0.0.0.0:6444, so when accessing the k8s cluster API interface from outside the cluster, you can use 192.168.4.106:6444. This is mainly used for ArgoCD management.

2. `--port "8001:80@loadbalancer"`: Specifies the cluster's web port. Here it's set to 8001:80@loadbalancer, so when accessing the k8s cluster's web interface from outside the cluster, you can use 192.168.4.106:8001.

3. `--k3s-arg "--tls-san=192.168.4.106@server:0"`: Specifies the cluster's tls-san. Here it's set to 192.168.4.106@server:0, so when accessing the k8s cluster's API interface from outside the cluster, you can use 192.168.4.106.

4. `--registry-config /tmp/registry-config.yaml`: Specifies the image registry configuration file. Here it's set to /tmp/registry-config.yaml, so the image registry in the k8s cluster can use http://192.168.4.104:5000.

You can use the following command to view cluster information:

```shell
kubectl config get-contexts
```

You should see:

```
CURRENT   NAME                 CLUSTER              AUTHINFO                   NAMESPACE
*         k3d-hichat-cluster   k3d-hichat-cluster   admin@k3d-hichat-cluster
```

### Copy Cluster Configuration

Copy the cluster's kubeconfig file to Server 1:

```shell
k3d kubeconfig get hichat-cluster > ~/hichat-cluster-server2.config
```

The content should look like:

```
---
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkekNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdGMyVnkKZG1WeUxXTmhRREUzTmpNNE1EUXdNVE13SGhjTk1qVXhNVEl5TURrek16TXpXaGNOTXpVeE1USXdNRGt6TXpNegpXakFqTVNFd0h3WURWUVFEREJock0zTXRjMlZ5ZG1WeUxXTmhRREUzTmpNNE1EUXdNVE13V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFUMDl0QnBSNERBaExKZFhVZGhnUnM0TWlBLzlUT3M1OVA0VmRvR0tETnUKa2VqTVVidEMxeTkzUklWTFJSR01nTTBQMDY4b1VvamZKZ1JOaDg3VEZ5WTRvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVXl0YkNFZW12VFl0YjBkM0VPenZkCkpHeGtSNlV3Q2dZSUtvWkl6ajBFQXdJRFNBQXdSUUlnZEFBYWN1R2tLTzFHWlJYNWtLdVRCU25HM2FFTksyOG0KUUVvOGR6VDZ2dWdDSVFDcitMTHMvelExMDZtTlBsOG5kZExPU2JVelB6RlZWYUZQbUhnMFg1c1NZZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    server: https://0.0.0.0:6444
  name: k3d-hichat-cluster
contexts:
- context:
    cluster: k3d-hichat-cluster
    user: admin@k3d-hichat-cluster
  name: k3d-hichat-cluster
current-context: k3d-hichat-cluster
kind: Config
preferences: {}
users:
- name: admin@k3d-hichat-cluster
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrVENDQVRlZ0F3SUJBZ0lJTzVGT3B6Wmhtd2t3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOell6T0RBME1ERXpNQjRYRFRJMU1URXlNakE1TXpNek0xb1hEVEkyTVRFeQpNakE1TXpNek0xb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJKRjZWTmpKd2ZxR0NGRW4KaWdxeVRkSE84UVhJUWpNLythMG5mREh6akRPK1pNT1p5UWo0eG5MbkNBWldLQ0FqL1gyb2dPMERlSjRjZzJRSgpieEhmWGgralNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCUkx5bkJ5dVZqb21SUTNlUHc4Mjg1SWtvRGRyVEFLQmdncWhrak9QUVFEQWdOSUFEQkYKQWlFQTJMNUhFR080bHdsM2F2TmdzYmUxeFVhNWdFVzNxR0FtaWZqTDRRMmFOM3dDSUdBYlB6eU1jUG8vMjdUVQpjWnNabHZVTEJrYldQbW4vWWFIdXR3dnBwYUxZCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkakNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdFkyeHAKWlc1MExXTmhRREUzTmpNNE1EUXdNVE13SGhjTk1qVXhNVEl5TURrek16TXpXaGNOTXpVeE1USXdNRGt6TXpNegpXakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwWlc1MExXTmhRREUzTmpNNE1EUXdNVE13V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFSS3ROT2JKb0hISGN2Um1rSkFmUVgwMjJ5R0txNUlGd2xqS3E3MHJDQ1oKb0RpNzRSTUwxRnNsMGM1bnhhOVlnb1JIMVJtQ3d3eGFVWHQ0b1ZaN1dtS3ZvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVVM4cHdjcmxZNkprVU4zajhQTnZPClNKS0EzYTB3Q2dZSUtvWkl6ajBFQXdJRFJ3QXdSQUlnZGVXTEtybzVpWGY3N2pnbHEyeldITTBvK2JNdGlTa3EKLy9iYjhybHJCc2tDSUdMLzRxUzhWaXR0UmxpbTZhaVZ6U0lxb2RsRHZPYzNjMDZ3d0kzVlkrakoKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSVBLbmF6b08yZWxRdlJRWFkxdWRYck5BUUlBeVd6bUpNYXl3bmZwdW5qNzNvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFa1hwVTJNbkIrb1lJVVNlS0NySk4wYzd4QmNoQ016LzVyU2Q4TWZPTU03NWt3NW5KQ1BqRwpjdWNJQmxZb0lDUDlmYWlBN1FONG5oeURaQWx2RWQ5ZUh3PT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=
```

This content needs to be copied to Server 1.

## Managing K3D Application Cluster on Server 1

### Configuration

Copy the configuration file `hichat-cluster-server2.config` to Server 1:

```shell
vim hichat-cluster-server2.config
```

Note:

You need to change the IP address in the configuration file to Server 2's IP address, because it needs to manage Server 2's hichat-cluster:

```
server: https://192.168.4.106:6444
```

Merge cluster configuration files:

```shell
export KUBECONFIG=~/.kube/config:hichat-cluster-server2.config
kubectl config view --flatten > ~/.kube/merged
mv ~/.kube/merged ~/.kube/config
```

Then we will see:

```shell
root@argocd:/srv/argocd# kubectl config get-contexts

CURRENT   NAME                 CLUSTER              AUTHINFO                   NAMESPACE
          k3d-argocd           k3d-argocd           admin@k3d-argocd
*         k3d-hichat-cluster   k3d-hichat-cluster   admin@k3d-hichat-cluster
```

`k3d-argocd` is the ArgoCD cluster. ArgoCD itself needs a cluster to run. Then we see `k3d-hichat-cluster` is the cluster we just connected to.

### Add Project Configuration on Server 1

We need to switch the current cluster context to the argocd cluster:

```shell
kubectl config use-context k3d-argocd
```

See:

```shell
CURRENT   NAME                 CLUSTER              AUTHINFO                   NAMESPACE
*         k3d-argocd           k3d-argocd           admin@k3d-argocd
          k3d-hichat-cluster   k3d-hichat-cluster   admin@k3d-hichat-cluster
```

We need to add project configuration so that argo-cd knows how to manage this cluster:

```shell
vim argocd-application-hichat-server2.yaml
```

The content should be approximately:

```shell
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hichat-hi-server2 # Application name
  namespace: argocd # Under argocd namespace
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: http://192.168.4.102/demo/hichat.git # Git repository address
    targetRevision: k8s-config # Branch
    path: k8s # Configuration file path
  destination:
    server: https://192.168.4.106:6444  # Server 2's cluster
    namespace: hichat # Server 2's cluster namespace
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true # Let argoCD create the namespace
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

Save and exit, then execute:

```shell
kubectl apply -f argocd-application-hichat-server2.yaml
```

There are several points to note:

We have configured this ArgoCD file above. In the project's k8s directory, you cannot have ArgoCD-related configurations, such as `argocd-application.yaml`, otherwise it will cause errors and prevent the entire k8s cluster from starting normally.

#### Check Configuration

Check the argocd namespace

```shell
kubectl get namespace argocd
```

Check ArgoCD Pods

```shell
kubectl get pods -n argocd
```

Check CRD

```shell
kubectl get crd applications.argoproj.io
```

### Execute apply yaml

```shell
kubectl apply -f argocd-application-hichat-server2.yaml
```

You will see:

```shell
root@argocd:/srv/argocd# kubectl apply -f argocd-application-hichat-server2.yaml
Warning: metadata.finalizers: "resources-finalizer.argocd.argoproj.io": prefer a domain-qualified finalizer name to avoid accidental conflicts with other finalizer writers
application.argoproj.io/hichat-hi-server2 created
root@argocd:/srv/argocd# Handling connection for 8443
```

Then we need to log in to ArgoCD:

```shell
argocd login 192.168.4.105:8443 --username admin --password 8wFKryBpD5lgcdOp
```

Output:

```
'admin:login' logged in successfully
Context '192.168.4.105:8443' updated
```

### Add Server 2's Cluster to Server 1's ArgoCD Management

Switch management cluster:

```shell
kubectl config use-context k3d-hichat-cluster
```

Add cluster:

```shell
argocd cluster add k3d-hichat-mycluster --name server2-hichat-cluster
```

After success, the output content is as follows:

```shell
 cluster add k3d-hichat-mycluster --name server2-hichat-cluster
{"level":"fatal","msg":"context k3d-hichat-mycluster does not exist in kubeconfig","time":"2025-11-22T17:50:37+08:00"}
root@argocd:/srv/argocd# argocd cluster add k3d-hichat-cluster --name server2-hichat-cluster
WARNING: This will create a service account `argocd-manager` on the cluster referenced by context `k3d-hichat-cluster` with full cluster level privileges. Do you want to continue [y/N]? y
{"level":"info","msg":"ServiceAccount \"argocd-manager\" created in namespace \"kube-system\"","time":"2025-11-22T17:51:06+08:00"}
{"level":"info","msg":"ClusterRole \"argocd-manager-role\" created","time":"2025-11-22T17:51:06+08:00"}
{"level":"info","msg":"ClusterRoleBinding \"argocd-manager-role-binding\" created","time":"2025-11-22T17:51:06+08:00"}
{"level":"info","msg":"Created bearer token secret \"argocd-manager-long-lived-token\" for ServiceAccount \"argocd-manager\"","time":"2025-11-22T17:51:06+08:00"}
Handling connection for 8443
Handling connection for 8443
Cluster 'https://192.168.4.106:6444' added
```

Switch back to the argocd cluster context and check the addition:

```shell
kubectl config use-context k3d-argocd
argocd cluster list
```

Output:

```shell
SERVER                          NAME                    VERSION  STATUS      MESSAGE  PROJECT
https://192.168.4.106:6444      server2-hichat-cluster  1.31     Successful
https://kubernetes.default.svc  in-cluster              1.31     Successful
```

## Manual Cluster Deployment

After completing the above process, our operations will automatically trigger and deploy automatically. Here we can also manually deploy:

```shell
argocd app sync hichat-hi-server2
```

You can see output like this:

```shell
root@argocd:/srv/argocd# argocd app sync hichat-hi-server2

TIMESTAMP                  GROUP                    KIND              NAMESPACE                  NAME    STATUS   HEALTH            HOOK  MESSAGE
2025-11-22T17:53:11+08:00                      ConfigMap                 hichat     mysql-init-script    Synced
2025-11-22T17:53:11+08:00                        Service                 hichat                hichat    Synced  Healthy
2025-11-22T17:53:11+08:00                        Service                 hichat                 redis    Synced  Healthy
2025-11-22T17:53:11+08:00  networking.k8s.io     Ingress                 hichat        hichat-ingress    Synced  Healthy
2025-11-22T17:53:11+08:00                     PersistentVolumeClaim      hichat             mysql-pvc    Synced  Healthy
2025-11-22T17:53:11+08:00                        Service                 hichat                 mysql    Synced  Healthy
2025-11-22T17:53:11+08:00   apps              Deployment                 hichat                hichat    Synced  Progressing
2025-11-22T17:53:11+08:00   apps              Deployment                 hichat                 mysql    Synced  Healthy
2025-11-22T17:53:11+08:00   apps              Deployment                 hichat                 redis    Synced  Healthy
2025-11-22T17:53:11+08:00   apps              Deployment                 hichat                 mysql    Synced  Healthy                  deployment.apps/mysql unchanged
2025-11-22T17:53:11+08:00  networking.k8s.io     Ingress                 hichat        hichat-ingress    Synced  Healthy                  ingress.networking.k8s.io/hichat-ingress unchanged
2025-11-22T17:53:11+08:00                      ConfigMap                 hichat     mysql-init-script    Synced                           configmap/mysql-init-script unchanged
2025-11-22T17:53:11+08:00                     PersistentVolumeClaim      hichat             mysql-pvc    Synced  Healthy                  persistentvolumeclaim/mysql-pvc unchanged
2025-11-22T17:53:11+08:00                        Service                 hichat                 redis    Synced  Healthy                  service/redis unchanged
2025-11-22T17:53:11+08:00   apps              Deployment                 hichat                 redis    Synced  Healthy                  deployment.apps/redis unchanged
2025-11-22T17:53:11+08:00                        Service                 hichat                hichat    Synced  Healthy                  service/hichat unchanged
2025-11-22T17:53:11+08:00                        Service                 hichat                 mysql    Synced  Healthy                  service/mysql unchanged
2025-11-22T17:53:11+08:00   apps              Deployment                 hichat                hichat    Synced  Progressing              deployment.apps/hichat unchanged

Handling connection for 8443

Name:               argocd/hichat-hi-server2
Project:            default
Server:             https://192.168.4.106:6444
Namespace:          hichat
URL:                https://argocd.example.com/applications/hichat-hi-server2
Source:
- Repo:             http://192.168.4.102/demo/hichat.git
  Target:           k8s-config
  Path:             k8s
SyncWindow:         Sync Allowed
Sync Policy:        Automated (Prune)
Sync Status:        Synced to k8s-config (062a43a)
Health Status:      Progressing
Operation:          Sync
Sync Revision:      062a43ab37a287794e8d5413254527998ba54971
Phase:              Succeeded
Start:              2025-11-22 17:53:11 +0800 HKT
Finished:           2025-11-22 17:53:11 +0800 HKT
Duration:           0s
Message:            successfully synced (all tasks run)

GROUP              KIND                   NAMESPACE  NAME               STATUS  HEALTH       HOOK  MESSAGE
                   ConfigMap              hichat     mysql-init-script  Synced                     configmap/mysql-init-script unchanged
                   PersistentVolumeClaim  hichat     mysql-pvc          Synced  Healthy            persistentvolumeclaim/mysql-pvc unchanged
                   Service                hichat     hichat             Synced  Healthy            service/hichat unchanged
                   Service                hichat     redis              Synced  Healthy            service/redis unchanged
                   Service                hichat     mysql              Synced  Healthy            service/mysql unchanged
apps               Deployment             hichat     hichat             Synced  Progressing        deployment.apps/hichat unchanged
apps               Deployment             hichat     mysql              Synced  Healthy            deployment.apps/mysql unchanged
apps               Deployment             hichat     redis              Synced  Healthy            deployment.apps/redis unchanged
networking.k8s.io  Ingress                hichat     hichat-ingress     Synced  Healthy            ingress.networking.k8s.io/hichat-ingress unchanged

root@argocd:/srv/argocd# argocd app get hichat-hi-server2
```

Use the following command to view application details:

```shell
argocd app get hichat-hi-server2
```

Output result:

```shell
Name:               argocd/hichat-hi-server2
Project:            default
Server:             https://192.168.4.106:6444
Namespace:          hichat
URL:                https://argocd.example.com/applications/hichat-hi-server2
Source:
- Repo:             http://192.168.4.102/demo/hichat.git
  Target:           k8s-config
  Path:             k8s
SyncWindow:         Sync Allowed
Sync Policy:        Automated (Prune)
Sync Status:        Synced to k8s-config (062a43a)
Health Status:      Progressing
GROUP              KIND                   NAMESPACE  NAME               STATUS  HEALTH       HOOK  MESSAGE
                   ConfigMap              hichat     mysql-init-script  Synced                     configmap/mysql-init-script unchanged
                   PersistentVolumeClaim  hichat     mysql-pvc          Synced  Healthy            persistentvolumeclaim/mysql-pvc unchanged
                   Service                hichat     hichat             Synced  Healthy            service/hichat unchanged
                   Service                hichat     redis              Synced  Healthy            service/redis unchanged
                   Service                hichat     mysql              Synced  Healthy            service/mysql unchanged
apps               Deployment             hichat     hichat             Synced  Progressing        deployment.apps/hichat unchanged
apps               Deployment             hichat     mysql              Synced  Healthy            deployment.apps/mysql unchanged
apps               Deployment             hichat     redis              Synced  Healthy            deployment.apps/redis unchanged
networking.k8s.io  Ingress                hichat     hichat-ingress     Synced  Healthy            ingress.networking.k8s.io/hichat-ingress unchanged

root@argocd:/srv/argocd# E1122 17:53:53.521069   44057 portforward.go:391] "Unhandled Error" err="error copying from remote stream to local connection: readfrom tcp4 192.168.4.105:8443->192.168.4.105:60370: write tcp4 192.168.4.105:8443->192.168.4.105:60370: write: broken pipe"
```

Normal display indicates that the application has been successfully deployed.

Congratulations, you have successfully deployed the Hichat application.

## Troubleshooting

From the above, we can see that there may be exceptions in some places. The hichat pod has been in a Progressing state. We need to check the logs:

```shell
# Switch context
kubectl config use-context k3d-hichat-cluster

# View pod logs
kubectl get pods -n hichat
kubectl logs -f deployment/hichat -n hichat
kubectl describe deployment hichat -n hichat
```

Core issue output:

```shell
root@argocd:/srv/argocd# kubectl describe deployment hichat -n hichat

Name:                   hichat
Namespace:              hichat
CreationTimestamp:      Sat, 22 Nov 2025 17:51:38 +0800
Labels:                 <none>
Annotations:            argocd.argoproj.io/tracking-id: hichat-hi-server2:apps/Deployment:hichat/hichat
                        deployment.kubernetes.io/revision: 1
Selector:               app=hichat
Replicas:               1 desired | 1 updated | 1 total | 0 available | 1 unavailable
StrategyType:           RollingUpdate
MinReadySeconds:        0
RollingUpdateStrategy:  25% max unavailable, 25% max surge
Pod Template:
  Labels:  app=hichat
  Containers:
   hichat:
    Image:      192.168.4.104:5000/admin/hichat:3e73bce2
    Port:       8000/TCP
    Host Port:  0/TCP
    Environment:
      MYSQL_HOST:      mysql
      MYSQL_PORT:      3306
      MYSQL_USER:      iceymoss
      MYSQL_PASSWORD:  Yk?123456
      MYSQL_DATABASE:  hi_chat
      REDIS_HOST:      redis
      REDIS_PORT:      6379
      PORT:            8000
    Mounts:            <none>
  Volumes:             <none>
  Node-Selectors:      <none>
  Tolerations:         <none>
Conditions:
  Type           Status  Reason
  ----           ------  ------
  Available      False   MinimumReplicasUnavailable
  Progressing    False   ProgressDeadlineExceeded
OldReplicaSets:  <none>
NewReplicaSet:   hichat-6ff967cf55 (1/1 replicas created)
Events:
  Type    Reason             Age   From                   Message
  ----    ------             ----  ----                   -------
  Normal  ScalingReplicaSet  10m   deployment-controller  Scaled up replica set hichat-6ff967cf55 to 1
root@argocd:/srv/argocd# kubectl describe pod -l app=hichat -n hichat

Name:             hichat-6ff967cf55-cmvhn
Namespace:        hichat
Priority:         0
Service Account:  default
Node:             k3d-hichat-cluster-server-0/172.19.0.2
Start Time:       Sat, 22 Nov 2025 17:51:38 +0800
Labels:           app=hichat
                  pod-template-hash=6ff967cf55
Annotations:      <none>
Status:           Pending
IP:               10.42.0.9
IPs:
  IP:           10.42.0.9
Controlled By:  ReplicaSet/hichat-6ff967cf55
Containers:
  hichat:
    Container ID:
    Image:          192.168.4.104:5000/admin/hichat:3e73bce2
    Image ID:
    Port:           8000/TCP
    Host Port:      0/TCP
    State:          Waiting
      Reason:       ImagePullBackOff
    Ready:          False
    Restart Count:  0
    Environment:
      MYSQL_HOST:      mysql
      MYSQL_PORT:      3306
      MYSQL_USER:      iceymoss
      MYSQL_PASSWORD:  Yk?123456
      MYSQL_DATABASE:  hi_chat
      REDIS_HOST:      redis
      REDIS_PORT:      6379
      PORT:            8000
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-cc8s6 (ro)
Conditions:
  Type                        Status
  PodReadyToStartContainers   True
  Initialized                 True
  Ready                       False
  ContainersReady             False
  PodScheduled                True
Volumes:
  kube-api-access-cc8s6:
    Type:                    Projected (a volume that contains injected data from multiple sources)
    TokenExpirationSeconds:  3607
    ConfigMapName:           kube-root-ca.crt
    Optional:                false
    DownwardAPI:              true
QoS Class:                   BestEffort
Node-Selectors:              <none>
Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
Events:
  Type     Reason     Age                  From               Message
  ----     ------     ----                 ----               -------
  Normal   Scheduled  10m                  default-scheduler  Successfully assigned hichat/hichat-6ff967cf55-cmvhn to k3d-hichat-cluster-server-0
  Normal   Pulling    9m23s (x4 over 10m)  kubelet            Pulling image "192.168.4.104:5000/admin/hichat:3e73bce2"
  Warning  Failed     9m23s (x4 over 10m)  kubelet            Failed to pull image "192.168.4.104:5000/admin/hichat:3e73bce2": failed to pull and unpack image "192.168.4.104:5000/admin/hichat:3e73bce2": failed to resolve reference "192.168.4.104:5000/admin/hichat:3e73bce2": pull access denied, repository does not exist or may require authorization: authorization failed: no basic auth credentials
  Warning  Failed     9m23s (x4 over 10m)  kubelet            Error: ErrImagePull
  Warning  Failed     9m8s (x6 over 10m)   kubelet            Error: ImagePullBackOff
  Normal   BackOff    33s (x43 over 10m)   kubelet            Back-off pulling image "192.168.4.104:5000/admin/hichat:3e73bce2"
```

It can be seen that the image registry requires authentication. Let's first create a secret:

1. Add secret in the hichat.yaml configuration under the monitored branch:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hichat
  namespace: hichat
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hichat
  template:
    metadata:
      labels:
        app: hichat
    spec:
      imagePullSecrets:
      - name: registry-secret # Add secret
      containers:
      - name: hichat
        image: 192.168.4.104:5000/admin/hichat:3e73bce2
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8000
        env:
        - name: MYSQL_HOST
          value: "mysql"
        - name: MYSQL_PORT
          value: "3306"
        - name: MYSQL_USER
          value: "iceymoss"
        - name: MYSQL_PASSWORD
          value: "Yk?123456"
        - name: MYSQL_DATABASE
          value: "hi_chat"
        - name: REDIS_HOST
          value: "redis"
        - name: REDIS_PORT
          value: "6379"
        - name: PORT
          value: "8000"
---
apiVersion: v1
kind: Service
metadata:
  name: hichat
  namespace: hichat
spec:
  selector:
    app: hichat
  ports:
  - port: 8000
    targetPort: 8000
  type: ClusterIP  # Use ClusterIP, expose service through Ingress
```

2. Add secret in the cluster:

```shell
root@argocd:/srv/argocd# kubectl config get-contexts

CURRENT   NAME                 CLUSTER              AUTHINFO                   NAMESPACE
          k3d-argocd           k3d-argocd           admin@k3d-argocd
*         k3d-hichat-cluster   k3d-hichat-cluster   admin@k3d-hichat-cluster
```

Create secret:

```shell
 kubectl create secret docker-registry registry-secret \
  --docker-server=192.168.4.104:5000 \
  --docker-username=username \
  --docker-password=password \
  -n hichat
```

After committing the code:

```shell
root@argocd:/srv/argocd# cat argocd-application-hichat-server2.yaml

apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hichat-hi-server2
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: http://192.168.4.102/demo/hichat.git
    targetRevision: k8s-config
    path: k8s
  destination:
    server: https://192.168.4.106:6444  # Server 2's cluster
    namespace: hichat
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m

root@argocd:/srv/argocd#
root@argocd:/srv/argocd# argocd app get hichat-hi-server2

Handling connection for 8443
Handling connection for 8443
Handling connection for 8443
Handling connection for 8443
Handling connection for 8443

Name:               argocd/hichat-hi-server2
Project:            default
Server:             https://192.168.4.106:6444
Namespace:          hichat
URL:                https://argocd.example.com/applications/hichat-hi-server2
Source:
- Repo:             http://192.168.4.102/demo/hichat.git
  Target:           k8s-config
  Path:             k8s
SyncWindow:         Sync Allowed
Sync Policy:        Automated (Prune)
Sync Status:        Synced to k8s-config (e624d6c)
Health Status:      Healthy

GROUP              KIND                   NAMESPACE  NAME               STATUS  HEALTH   HOOK  MESSAGE
                   ConfigMap              hichat     mysql-init-script  Synced                 configmap/mysql-init-script unchanged
                   PersistentVolumeClaim  hichat     mysql-pvc          Synced  Healthy        persistentvolumeclaim/mysql-pvc unchanged
                   Service                hichat     mysql              Synced  Healthy        service/mysql unchanged
                   Service                hichat     hichat             Synced  Healthy        service/hichat unchanged
                   Service                hichat     redis              Synced  Healthy        service/redis unchanged
apps               Deployment             hichat     mysql              Synced  Healthy        deployment.apps/mysql unchanged
apps               Deployment             hichat     hichat             Synced  Healthy        deployment.apps/hichat configured
apps               Deployment             hichat     redis              Synced  Healthy        deployment.apps/redis unchanged
networking.k8s.io  Ingress                hichat     hichat-ingress     Synced  Healthy        ingress.networking.k8s.io/hichat-ingress unchanged
```

