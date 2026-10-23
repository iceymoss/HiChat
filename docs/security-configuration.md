# Security configuration

HiChat requires a deployment-specific JWT signing secret. The application
refuses to start when the secret is missing or shorter than 32 bytes.

Generate a secret without committing it to the repository:

```shell
openssl rand -base64 48
```

For local development, export it before starting the application:

```shell
export JWT_SECRET='<generated value>'
go run .
```

`HICHAT_JWT_SECRET` is also supported by the Viper environment configuration.

For Kubernetes, create the Secret referenced by `k8s/hichat.yaml`:

```shell
kubectl -n hichat create secret generic hichat-secrets \
  --from-literal=jwt-secret='<generated value>'
```

The previously hardcoded `ice_moss` key is compromised and must not be reused.
Deploying this change invalidates all tokens signed with that key, so users must
sign in again.
