# Hearth Deployment Options

Quick deployment options for Hearth - from single-server to production Kubernetes.

## 🚀 Docker Compose (Fastest)

### Development / Single Server

```bash
cd deploy/docker
cp env.example .env
# Edit .env with your values

# Start
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

### Production with SSL

```bash
cd deploy/docker

# Create SSL certificates
mkdir ssl
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout ssl/key.pem -out ssl/cert.pem

# Edit environment
cp env.example .env
# Fill in all values

# Start production stack
docker-compose -f docker-compose.prod.yml up -d
```

### Features
- One-command deployment
- PostgreSQL + Redis included
- Optional MinIO for S3-compatible storage
- Prometheus + Grafana monitoring
- Automated backups
- SSL/TLS support

## ☸️ Kubernetes Helm (Production)

### Prerequisites

```bash
# Install Helm
curl -fsSL https://get.helm.sh/helm-v3.12.0-linux-amd64.tar.gz | tar -xz
sudo mv linux-amd64/helm /usr/local/bin/helm

# Add dependencies
helm repo add bitnami https://charts.bitnami.com
helm repo update

# Install cert-manager (for TLS)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

### Quick Start

```bash
cd deploy/helm

# Create namespace
kubectl create namespace hearth

# Create secrets
kubectl create secret generic hearth-secrets \
  --namespace hearth \
  --from-literal=database-url="postgres://user:pass@host:5432/hearth" \
  --from-literal=redis-url="redis://:pass@host:6379" \
  --from-literal=jwt-secret="$(openssl rand -base64 64)"

# Install
helm install hearth ./hearth \
  --namespace hearth \
  --values ./hearth/values.yaml
```

### Production Deployment

```bash
# Edit values for your environment
vim hearth/values.yaml

# Install with custom values
helm install hearth ./hearth \
  --namespace hearth \
  --values production-values.yaml \
  --set backend.replicaCount=4 \
  --set backend.autoscaling.enabled=true

# Upgrade
helm upgrade hearth ./hearth \
  --namespace hearth \
  --values production-values.yaml
```

### Scaling

```bash
# Scale backend
kubectl scale deployment hearth-backend --namespace hearth --replicas=5

# Or use autoscaling
kubectl autoscale deployment hearth-backend \
  --namespace hearth \
  --cpu-percent=70 \
  --min=2 \
  --max=10
```

## 📊 Monitoring

### Prometheus + Grafana

Enable in values.yaml:
```yaml
monitoring:
  enabled: true
  prometheus:
    enabled: true
    retention: 30d
  grafana:
    enabled: true
```

Access:
```bash
# Port-forward for local access
kubectl port-forward -n hearth svc/prometheus 9090:9090 &
kubectl port-forward -n hearth svc/grafana 3001:3000 &
```

## 🔒 Security

### Network Policies
Enabled by default. Adjust in values.yaml:
```yaml
networkPolicy:
  enabled: true
  backendEgress:
    - to:
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: postgresql
```

### Pod Security
```yaml
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000

containerSecurityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
```

## 🗄️ Storage Options

### Local (Development)
```yaml
persistence:
  enabled: true
  storageClass: standard
```

### Cloud Storage Classes
```yaml
# AWS EBS
persistence:
  enabled: true
  storageClass: gp3

# GCE Persistent Disk
persistence:
  enabled: true
  storageClass: standard-rwo

# Azure Disk
persistence:
  enabled: true
  storageClass: managed-premium
```

### S3-Compatible External Storage
```yaml
minio:
  enabled: false  # Use external

backend:
  env:
    - name: S3_ENDPOINT
      value: "https://s3.example.com"
    - name: S3_BUCKET
      value: "hearth-attachments"
```

## 🌐 High Availability

### Multi-AZ Deployment
```yaml
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
```

### Pod Disruption Budget
```yaml
podDisruptionBudget:
  enabled: true
  minAvailable: 1
```

## 🔄 Updates

### Docker Compose
```bash
# Pull new images
docker-compose pull

# Restart with new version
docker-compose up -d
```

### Helm
```bash
# Update dependencies
helm dependency update

# Upgrade
helm upgrade hearth ./hearth \
  --namespace hearth \
  --values new-values.yaml
```

## 🗑️ Cleanup

### Docker Compose
```bash
docker-compose down -v  # Includes volumes
```

### Kubernetes
```bash
# Uninstall (keeps PVCs)
helm uninstall hearth --namespace hearth

# Uninstall with PVCs (deletes data!)
helm uninstall hearth --namespace hearth --purge
kubectl delete pvc --namespace hearth --all
```

## 📝 Common Issues

### Pods Not Starting
```bash
# Check events
kubectl describe pod -n hearth

# Check logs
kubectl logs -n hearth deployment/hearth-backend --previous
```

### Database Connection
```bash
# Verify secrets
kubectl get secret hearth-secrets -n hearth -o yaml

# Test connection
kubectl run -it --rm debug --image=postgres:latest \
  --namespace hearth \
  --restart=Never -- psql -h postgres -U hearth
```

### Ingress Not Working
```bash
# Check ingress status
kubectl describe ingress -n hearth

# Verify cert-manager
kubectl get certificate -n hearth
```

## 🔧 Configuration Reference

See `hearth/values.yaml` for all configuration options.

### Quick Reference

| Parameter | Default | Description |
|-----------|---------|-------------|
| `backend.replicaCount` | 2 | Number of backend pods |
| `backend.autoscaling.enabled` | true | Enable HPA |
| `postgresql.enabled` | false | Use embedded PostgreSQL |
| `redis.enabled` | false | Use embedded Redis |
| `monitoring.enabled` | true | Enable Prometheus/Grafana |
| `networkPolicy.enabled` | true | Enable network policies |
