# Quick Reference Guide

## Common Commands

### Development

```bash
# Generate code after modifying types
make generate

# Generate CRDs and RBAC
make manifests

# Run tests
make test

# Run linter
make lint

# Build the operator binary
make build

# Run operator locally
make run
```

### Installation

```bash
# Install CRDs into cluster
make install

# Deploy operator to cluster
make deploy IMG=your-registry/database-operator:tag

# Uninstall CRDs
make uninstall

# Undeploy operator
make undeploy
```

### Docker

```bash
# Build Docker image
make docker-build IMG=your-registry/database-operator:tag

# Push Docker image
make docker-push IMG=your-registry/database-operator:tag
```

## Database Resource Examples

### Minimal PostgreSQL

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: my-postgres
spec:
  type: PostgreSQL
  version: "16"
```

### Production PostgreSQL

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: prod-postgres
spec:
  type: PostgreSQL
  version: "16"
  replicas: 3
  storage:
    size: 100Gi
    storageClassName: fast-ssd
  resources:
    cpu: 2000m
    memory: 4Gi
    cpuLimit: 4000m
    memoryLimit: 8Gi
  postgresql:
    database: myapp
    username: appuser
    passwordSecret:
      name: postgres-secret
      key: password
```

### Minimal MongoDB

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: my-mongo
spec:
  type: MongoDB
  version: "7.0"
```

### Production MongoDB Replica Set

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: prod-mongo
spec:
  type: MongoDB
  version: "7.0"
  replicas: 3
  storage:
    size: 200Gi
    storageClassName: fast-ssd
  resources:
    cpu: 2000m
    memory: 4Gi
  mongodb:
    database: myapp
    username: appuser
    replicaSetName: rs0
    passwordSecret:
      name: mongo-secret
      key: password
```

### Minimal Redis

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: my-redis
spec:
  type: Redis
  version: "7.2"
```

### Production Redis with Auth

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: prod-redis
spec:
  type: Redis
  version: "7.2"
  replicas: 1
  storage:
    size: 10Gi
  resources:
    cpu: 1000m
    memory: 2Gi
  redis:
    mode: standalone
    passwordSecret:
      name: redis-secret
      key: password
    parameters:
      maxmemory: "1gb"
      maxmemory-policy: "allkeys-lru"
```

### Elasticsearch Cluster

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: my-elasticsearch
spec:
  type: Elasticsearch
  version: "8.11.0"
  replicas: 3
  storage:
    size: 100Gi
  resources:
    cpu: 2000m
    memory: 4Gi
  elasticsearch:
    clusterName: my-es-cluster
    nodeRoles:
      - master
      - data
      - ingest
  env:
    - name: ES_JAVA_OPTS
      value: "-Xms2g -Xmx2g"
```

### SQLite

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: my-sqlite
spec:
  type: SQLite
  version: "latest"
  storage:
    size: 5Gi
  sqlite:
    databaseFile: /data/app.db
```

## kubectl Commands

### List Databases

```bash
# All databases
kubectl get databases
# Short form
kubectl get db

# With details
kubectl get databases -o wide

# In all namespaces
kubectl get databases --all-namespaces
```

### Describe Database

```bash
kubectl describe database <name>
```

### Get Database YAML

```bash
kubectl get database <name> -o yaml
```

### Get Database Status

```bash
# Full status
kubectl get database <name> -o jsonpath='{.status}' | jq .

# Just phase
kubectl get database <name> -o jsonpath='{.status.phase}'

# Connection string
kubectl get database <name> -o jsonpath='{.status.connectionString}'

# Ready replicas
kubectl get database <name> -o jsonpath='{.status.readyReplicas}'
```

### Edit Database

```bash
kubectl edit database <name>
```

### Delete Database

```bash
kubectl delete database <name>

# Force delete (if stuck)
kubectl delete database <name> --grace-period=0 --force
```

### Watch Databases

```bash
kubectl get databases -w
```

## Troubleshooting Commands

### Check Operator Logs

```bash
# Local development
# Logs appear in terminal running 'make run'

# In-cluster
kubectl logs -n database-operator-system \
  -l control-plane=controller-manager \
  --tail=100 -f
```

### Check Database Resources

```bash
# StatefulSets
kubectl get statefulsets -l app=<database-name>
kubectl describe statefulset <name>

# Deployments
kubectl get deployments -l app=<database-name>
kubectl describe deployment <name>

# Services
kubectl get services -l app=<database-name>

# PVCs
kubectl get pvc -l app=<database-name>

# Pods
kubectl get pods -l app=<database-name>
kubectl logs <pod-name>
kubectl describe pod <pod-name>
```

### Check Events

```bash
kubectl get events --sort-by='.lastTimestamp' | grep <database-name>
```

### Debug Pod

```bash
kubectl exec -it <pod-name> -- /bin/bash
```

## Connection Examples

### PostgreSQL

```bash
# Port-forward
kubectl port-forward svc/<service-name> 5432:5432

# Connect with psql
PGPASSWORD=<password> psql -h localhost -U <username> -d <database>
```

### MongoDB

```bash
# Port-forward
kubectl port-forward svc/<service-name> 27017:27017

# Connect with mongosh
mongosh "mongodb://<username>:<password>@localhost:27017/<database>"
```

### Redis

```bash
# Port-forward
kubectl port-forward svc/<service-name> 6379:6379

# Connect with redis-cli
redis-cli -h localhost -p 6379 -a <password>
```

### Elasticsearch

```bash
# Port-forward
kubectl port-forward svc/<service-name> 9200:9200

# Test with curl
curl http://localhost:9200/
curl http://localhost:9200/_cluster/health
```

## Common Patterns

### Create Secret for Database

```bash
kubectl create secret generic <secret-name> \
  --from-literal=password=<your-password>
```

### Scale Database

```bash
# Edit to change replicas
kubectl edit database <name>

# Or patch
kubectl patch database <name> -p '{"spec":{"replicas":3}}'
```

### Update Database Version

```bash
kubectl patch database <name> -p '{"spec":{"version":"17"}}'
```

### Add Environment Variable

```bash
kubectl patch database <name> --type='json' -p='[
  {
    "op": "add",
    "path": "/spec/env/-",
    "value": {
      "name": "NEW_VAR",
      "value": "new_value"
    }
  }
]'
```

## Resource Cleanup

### Delete All Databases in Namespace

```bash
kubectl delete databases --all
```

### Delete Specific Database Type

```bash
kubectl delete databases -l database-type=PostgreSQL
```

### Clean Up Orphaned Resources

```bash
# Usually not needed due to owner references
# But can be used if needed
kubectl delete statefulsets -l app.kubernetes.io/managed-by=database-operator
kubectl delete services -l app.kubernetes.io/managed-by=database-operator
kubectl delete pvc -l app.kubernetes.io/managed-by=database-operator
```

## Monitoring

### Check Resource Usage

```bash
# Operator pods
kubectl top pod -n database-operator-system

# Database pods
kubectl top pod -l app=<database-name>
```

### Get Database Metrics

```bash
# If Prometheus is installed
kubectl port-forward -n database-operator-system \
  svc/database-operator-controller-manager-metrics-service 8443:8443

curl -k https://localhost:8443/metrics
```

## Useful Aliases

Add these to your `.bashrc` or `.zshrc`:

```bash
# Shortcut for databases
alias kgdb='kubectl get databases'
alias kddb='kubectl describe database'
alias kedb='kubectl edit database'
alias kdeld='kubectl delete database'

# Watch databases
alias kwdb='kubectl get databases -w'

# Database logs
alias kl-db='kubectl logs -l app.kubernetes.io/managed-by=database-operator'
```

## Environment Variables for Operator

When running locally:

```bash
# Set kubeconfig
export KUBECONFIG=~/.kube/config

# Enable debug logging
export ENABLE_WEBHOOKS=false
export METRICS_BIND_ADDRESS=:8080
```

## CRD Field Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `spec.type` | string | Yes | - | Database type (PostgreSQL, MongoDB, Redis, Elasticsearch, SQLite) |
| `spec.version` | string | Yes | - | Database version to deploy |
| `spec.replicas` | int32 | No | 1 | Number of replicas (0-10) |
| `spec.storage.size` | string | No | - | Storage size (e.g., "10Gi") |
| `spec.storage.storageClassName` | string | No | - | StorageClass name |
| `spec.resources.cpu` | string | No | - | CPU request (e.g., "500m") |
| `spec.resources.memory` | string | No | - | Memory request (e.g., "1Gi") |
| `spec.resources.cpuLimit` | string | No | - | CPU limit |
| `spec.resources.memoryLimit` | string | No | - | Memory limit |

## Status Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `status.phase` | string | Current phase (Pending, Creating, Ready, Failed, Deleting, Upgrading) |
| `status.readyReplicas` | int32 | Number of ready replicas |
| `status.serviceName` | string | Name of the created service |
| `status.connectionString` | string | Connection information (without credentials) |
| `status.observedGeneration` | int64 | Latest observed generation |
| `status.message` | string | Additional status information |
| `status.conditions` | []Condition | Detailed status conditions |
