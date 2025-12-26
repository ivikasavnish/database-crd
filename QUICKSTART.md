# Database Operator - Quick Start Guide

This guide will help you get started with the Database Operator in minutes.

## Prerequisites

- Kubernetes cluster (v1.25+) - you can use kind, minikube, or a cloud provider
- kubectl configured to access your cluster
- Go 1.21+ (for local development)

## Installation

### Option 1: Install CRDs and Deploy Operator

```bash
# Clone the repository
git clone https://github.com/ivikasavnish/database-crd.git
cd database-crd

# Install CRDs
make install

# Build and run the operator locally
make run
```

### Option 2: Deploy to Kubernetes

```bash
# Build the Docker image
make docker-build IMG=myregistry/database-operator:v1.0.0

# Push the image
make docker-push IMG=myregistry/database-operator:v1.0.0

# Deploy to cluster
make deploy IMG=myregistry/database-operator:v1.0.0
```

## Creating Your First Database

### 1. PostgreSQL (Simple)

Create a file `my-postgres.yaml`:

```yaml
apiVersion: db.platform.io/v1
kind: Database
metadata:
  name: my-postgres
  namespace: default
spec:
  engine: PostgreSQL
  version: "15.0"
  topology:
    mode: Standalone
    replicas: 1
  storage:
    size: 10Gi
```

Apply it:

```bash
kubectl apply -f my-postgres.yaml
```

### 2. MongoDB with Backups

```yaml
apiVersion: db.platform.io/v1
kind: Database
metadata:
  name: my-mongodb
spec:
  engine: MongoDB
  version: "7.0"
  topology:
    mode: Replicated
    replicas: 3
  storage:
    size: 50Gi
  backup:
    enabled: true
    schedule: "0 2 * * *"
    method: Dump
    retention: 7
```

### 3. Redis Cluster

```yaml
apiVersion: db.platform.io/v1
kind: Database
metadata:
  name: my-redis
spec:
  engine: Redis
  version: "7.2"
  topology:
    mode: Cluster
    replicas: 6
    shards: 3
  storage:
    size: 20Gi
```

## Checking Database Status

```bash
# List all databases
kubectl get databases
# or short form:
kubectl get db

# Get detailed status
kubectl describe database my-postgres

# Watch status
kubectl get db -w
```

Example output:
```
NAME          ENGINE       VERSION   PHASE   READY   ENDPOINT                               AGE
my-postgres   PostgreSQL   15.0      Ready   1       my-postgres.default.svc.cluster.local  5m
```

## Working with Credentials

### Using Kubernetes Secrets

The operator automatically creates a secret with credentials:

```bash
# Get the password
kubectl get secret my-postgres-credentials -o jsonpath='{.data.password}' | base64 -d
```

### Using Consul for Credential Management

```yaml
apiVersion: db.platform.io/v1
kind: Database
metadata:
  name: my-postgres-consul
spec:
  engine: PostgreSQL
  version: "15.0"
  topology:
    mode: Standalone
    replicas: 1
  storage:
    size: 10Gi
  auth:
    consul:
      enabled: true
      address: consul.default.svc.cluster.local:8500
      path: database/credentials/my-postgres-consul
      tokenSecretRef:
        name: consul-token
        key: token
```

First, create the Consul token secret:

```bash
kubectl create secret generic consul-token \
  --from-literal=token=your-consul-token
```

## Enabling Automatic Credential Rotation

```yaml
spec:
  auth:
    rotationPolicy:
      enabled: true
      schedule: "0 0 1 * *"  # Monthly on the 1st
      strategy: TwoPhase
```

The operator will:
1. Create new credentials
2. Grant access with new credentials
3. Cutover applications
4. Revoke old credentials

## Configuring Backups

### Backup to S3

```yaml
spec:
  backup:
    enabled: true
    schedule: "0 2 * * *"
    method: Snapshot
    retention: 7
    destination:
      s3:
        bucket: my-backups
        region: us-east-1
        credentialsSecret: aws-credentials
```

Create the AWS credentials secret:

```bash
kubectl create secret generic aws-credentials \
  --from-literal=access_key_id=YOUR_ACCESS_KEY \
  --from-literal=secret_access_key=YOUR_SECRET_KEY
```

### Backup to PVC

```yaml
spec:
  backup:
    enabled: true
    schedule: "0 2 * * *"
    method: Dump
    retention: 7
    destination:
      pvc:
        storageClassName: standard
        size: 100Gi
```

## Restoring from Backup

```yaml
apiVersion: db.platform.io/v1
kind: Database
metadata:
  name: restored-db
spec:
  engine: PostgreSQL
  version: "15.0"
  restore:
    backupName: my-postgres-backup-20240101
```

## Upgrading a Database

Simply update the version in the spec:

```yaml
spec:
  version: "16.0"  # Upgrade from 15.0 to 16.0
```

The operator will:
1. Check if upgrade is allowed
2. Wait for maintenance window (if configured)
3. Perform rolling upgrade
4. Update status

## Maintenance Windows

Configure maintenance windows for controlled upgrades:

```yaml
spec:
  maintenance:
    windows:
      - dayOfWeek: 0  # Sunday
        startTime: "02:00"
        duration: 4h
      - dayOfWeek: 3  # Wednesday
        startTime: "02:00"
        duration: 2h
    autoUpgrade: false
```

## Pausing Reconciliation

To temporarily stop the operator from making changes:

```yaml
spec:
  lifecycle:
    paused: true
```

## Scaling

Update the replicas field:

```yaml
spec:
  topology:
    replicas: 5  # Scale from 3 to 5
```

## Deletion Policies

Control what happens when you delete a Database resource:

```yaml
spec:
  lifecycle:
    deletionPolicy: Snapshot  # Options: Retain, Snapshot, Delete
```

- **Retain**: Keep all resources (data persists)
- **Snapshot**: Take a backup before deletion
- **Delete**: Delete everything including data

## Observability

### Enable Metrics

```yaml
spec:
  observability:
    metrics:
      enabled: true
      serviceMonitor: true
      port: 9187
```

### Configure Logging

```yaml
spec:
  observability:
    logging:
      level: info  # debug, info, warn, error
      format: json
```

## Troubleshooting

### Check operator logs

```bash
# If running locally
# Logs are shown in the terminal

# If deployed to cluster
kubectl logs -n database-operator-system deployment/database-operator-controller-manager
```

### Check database status

```bash
kubectl describe database my-postgres
```

Look at the `Conditions` section for detailed status.

### Common Issues

1. **Database stuck in Provisioning**
   - Check if storage class exists
   - Check if PVC can be created
   - Check operator logs

2. **Backup failing**
   - Verify S3 credentials
   - Check backup destination is accessible
   - Look at backup job logs: `kubectl logs job/my-postgres-backup-xxx`

3. **Credential rotation stuck**
   - Check rotation job logs
   - Verify database connectivity
   - Check rotation status: `kubectl get database my-postgres -o jsonpath='{.status.rotationStatus}'`

## Advanced Examples

See the `config/samples/` directory for more examples:

- `db_v1_database.yaml` - Full-featured PostgreSQL example
- `mongodb_sample.yaml` - MongoDB with replicas
- `redis_sample.yaml` - Redis cluster
- `sqlite_sample.yaml` - SQLite for development
- `elasticsearch_sample.yaml` - Elasticsearch cluster

## Next Steps

- Explore the [API Reference](api/v1/database_types.go)
- Read the [Architecture Documentation](README.md#architecture)
- Check out the [Engine Implementation Guide](engines/README.md)
- Join our community discussions

## Getting Help

- GitHub Issues: https://github.com/ivikasavnish/database-crd/issues
- Documentation: https://github.com/ivikasavnish/database-crd
