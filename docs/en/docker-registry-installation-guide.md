# Docker Registry Installation and Configuration Guide

**Server IP**: 192.168.4.104  
**Registry Port**: 5000  
**Registry UI Port**: 8080  
**Data Directory**: `/opt/docker-registry`

---

## Quick Install (Docker Compose)

> **⚠️ Important Note**: This setup uses an **HTTP** (non-HTTPS) registry. After installation, **every client host** (including CI/CD runners, Kubernetes nodes, etc.) must configure `insecure-registries`, otherwise image push/pull operations will fail. See the "Configure Clients" section below.

### 1. Install Docker Compose (if needed)

```bash
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

### 2. Create Directories and Authentication

```bash
# Create directories
sudo mkdir -p /opt/docker-registry/{data,auth}

# Install htpasswd
sudo apt-get install -y apache2-utils

# Create user (admin/admin123)
sudo htpasswd -Bbn admin admin123 > /opt/docker-registry/auth/htpasswd
```

### 3. Create `docker-compose.yml`

```bash
cd /opt/docker-registry
sudo nano docker-compose.yml
```

Contents:

```yaml
version: '3.8'

services:
  registry:
    image: registry:2
    container_name: docker-registry
    restart: always
    ports:
      - "5000:5000"
    environment:
      REGISTRY_AUTH: htpasswd
      REGISTRY_AUTH_HTPASSWD_PATH: /auth/htpasswd
      REGISTRY_AUTH_HTPASSWD_REALM: Registry Realm
    volumes:
      - ./data:/var/lib/registry
      - ./auth:/auth
    networks:
      - registry-net

  registry-ui:
    image: joxit/docker-registry-ui:latest
    container_name: registry-ui
    restart: always
    ports:
      - "8080:80"
    environment:
      REGISTRY_URL: http://registry:5000
      REGISTRY_SECURE: false
      DELETE_IMAGES: true
      REGISTRY_USERNAME: admin
      REGISTRY_PASSWORD: admin123
    depends_on:
      - registry
    networks:
      - registry-net

networks:
  registry-net:
    driver: bridge
```

### 4. Start the Services

```bash
sudo docker-compose up -d

# Tail logs
sudo docker-compose logs -f
```

### 5. Validate

```bash
# Check containers
docker ps | grep -E "docker-registry|registry-ui"

# Test API
curl -I http://192.168.4.104:5000/v2/
# Expect 401 Unauthorized (normal)

# Access the Web UI
# http://192.168.4.104:8080
```

---

## Common Operations

### Log In to the Registry

```bash
docker login 192.168.4.104:5000
# Username: admin
# Password: admin123
```

### Push an Image

```bash
# 1. Tag the image
docker tag <image-name>:<tag> 192.168.4.104:5000/<image-name>:<tag>

# 2. Push the image
docker push 192.168.4.104:5000/<image-name>:<tag>
```

### Pull an Image

```bash
docker pull 192.168.4.104:5000/<image-name>:<tag>
```

### Configure Clients (HTTP Registry)

**Critical**: Because this guide uses an HTTP (non-HTTPS) registry, **every client host must configure `insecure-registries`**, otherwise push/pull operations will fail.

#### Configure on Client Hosts

Edit `/etc/docker/daemon.json`:

```json
{
  "insecure-registries": ["192.168.4.104:5000"]
}
```

**Notes**:
- Create the file if it does not exist.
- Merge with existing settings instead of overwriting the file.
- You can list multiple registries, separated by commas.

Then restart Docker:

```bash
sudo systemctl restart docker
```

#### Verify the Configuration

```bash
# Confirm the setting took effect
docker info | grep -A 5 "Insecure Registries"

# You should see: 192.168.4.104:5000
```

#### Common Errors

**Error 1**: `http: server gave HTTP response to HTTPS client`

**Cause**: The client did not configure `insecure-registries`, so Docker defaults to HTTPS.

**Fix**: Follow the steps above to set `insecure-registries` and restart Docker.

**Error 2**: `Get "https://192.168.4.104:5000/v2/": dial tcp: lookup 192.168.4.104`

**Cause**: Network connectivity or DNS resolution failure.

**Fix**:
- Test network connectivity: `ping 192.168.4.104`
- Confirm that port 5000 is open in any firewalls
- If using a domain name, verify DNS settings

#### Server-Side Configuration Notes

**Current setup uses HTTP**:
- The `docker-compose.yml` registry service defaults to HTTP.
- `REGISTRY_SECURE: false` ensures the Registry UI talks to the registry over HTTP.

**If you need HTTPS** (recommended for production):

1. Obtain SSL certificates (`domain.crt` and `domain.key`).
2. Create the certificate directory: `mkdir -p /opt/docker-registry/certs`.
3. Place the certificate files inside that directory.
4. Update `docker-compose.yml` to mount the certificates:

```yaml
registry:
  # ... other settings ...
  volumes:
    - ./data:/var/lib/registry
    - ./auth:/auth
    - ./certs:/certs
  environment:
    # ... other environment variables ...
    REGISTRY_HTTP_TLS_CERTIFICATE: /certs/domain.crt
    REGISTRY_HTTP_TLS_KEY: /certs/domain.key
```

5. Configure clients to use HTTPS (no `insecure-registries` needed):

```bash
docker login https://192.168.4.104:5000
```

### Container Management

```bash
cd /opt/docker-registry

# Show status
docker-compose ps

# Tail logs
docker-compose logs -f

# Restart
docker-compose restart

# Stop
docker-compose stop

# Start
docker-compose start

# Stop and remove containers
docker-compose down
```

---

## Troubleshooting

### Image Push/Pull Failures

#### Error 1: `http: server gave HTTP response to HTTPS client`

**Cause**:
- The server uses HTTP, but the client is not configured with `insecure-registries`.
- Docker clients default to HTTPS when connecting to registries.

**Resolution**:
1. Edit `/etc/docker/daemon.json` on the client and add:
   ```json
   {
     "insecure-registries": ["192.168.4.104:5000"]
   }
   ```
2. Restart Docker: `sudo systemctl restart docker`
3. Verify: `docker info | grep "Insecure Registries"`
4. Retry your push/pull.

**Reminder**:
- **Every client that needs registry access must be configured.**
- This includes developer laptops, CI/CD runners, Kubernetes nodes, etc.
- Docker must be restarted for the change to take effect.

#### Error 2: `unauthorized: authentication required`

**Cause**: Authentication is enabled on the registry, but you did not log in.

**Resolution**:
```bash
# Log in first
docker login 192.168.4.104:5000
# Username: admin
# Password: admin123

# Then retry the push/pull
```

#### Error 3: `denied: requested access to the resource is denied`

**Cause**:
- Authentication failed (wrong username/password).
- Authentication file is misconfigured.

**Resolution**:
1. Double-check your credentials.
2. Log in again: `docker login 192.168.4.104:5000`
3. Inspect the server-side auth file: `cat /opt/docker-registry/auth/htpasswd`

### Registry UI Cannot Connect

**Checks**:
```bash
# View logs
docker-compose logs registry-ui

# Inspect REGISTRY_URL (should use the container name)
docker inspect registry-ui | grep REGISTRY_URL
```

**Fix**: Ensure `REGISTRY_URL=http://registry:5000` (use the container name, not the host IP).

### Data Loss

**Check volume mounts**:
```bash
docker inspect docker-registry | grep -A 10 Mounts
ls -la /opt/docker-registry/data
```

---

## Backup and Restore

### Backup

```bash
cd /opt/docker-registry
docker-compose stop
tar -czf registry-backup-$(date +%Y%m%d).tar.gz data auth
docker-compose start
```

### Restore

```bash
cd /opt/docker-registry
docker-compose stop
tar -xzf registry-backup-YYYYMMDD.tar.gz
docker-compose start
```

---

## Access URLs

- **Registry API**: `http://192.168.4.104:5000`
- **Registry UI**: `http://192.168.4.104:8080`
