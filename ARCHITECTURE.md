# Architecture Overview

## System Architecture

The Database Operator follows the Kubernetes Operator pattern, consisting of a Custom Resource Definition (CRD) and a controller that continuously reconciles the desired state with the actual state.

```
┌─────────────────────────────────────────────────────────────────┐
│                         User / Developer                         │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ kubectl apply
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes API Server                        │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │           Database CRD (Custom Resource)                   │  │
│  │  - type: PostgreSQL|MongoDB|Redis|Elasticsearch|SQLite    │  │
│  │  - version: 16, 7.0, 7.2, etc.                            │  │
│  │  - replicas: 1-10                                          │  │
│  │  - storage: size, storageClass                            │  │
│  │  - resources: CPU, memory limits                          │  │
│  └───────────────────────────────────────────────────────────┘  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ Watch & Reconcile
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Database Operator Pod                         │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Database Controller                          │  │
│  │                                                            │  │
│  │  Reconcile Loop:                                          │  │
│  │  1. Fetch Database resource                               │  │
│  │  2. Determine database type                               │  │
│  │  3. Create/Update resources:                              │  │
│  │     - StatefulSet/Deployment                              │  │
│  │     - Service                                             │  │
│  │     - PersistentVolumeClaims                              │  │
│  │  4. Update status                                         │  │
│  └───────────────────────────────────────────────────────────┘  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ Create/Update
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Kubernetes Resources                           │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ StatefulSet  │  │   Service    │  │     PVC      │         │
│  │  (Postgres)  │  │  (ClusterIP) │  │   (Storage)  │         │
│  └──────┬───────┘  └──────────────┘  └──────────────┘         │
│         │                                                        │
│         ▼                                                        │
│  ┌──────────────────────────────────────────────┐              │
│  │              Database Pods                    │              │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐      │              │
│  │  │ Postgres│  │ MongoDB │  │  Redis  │      │              │
│  │  │  Pod-0  │  │  Pod-0  │  │  Pod-0  │      │              │
│  │  └─────────┘  └─────────┘  └─────────┘      │              │
│  └──────────────────────────────────────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

## Component Interactions

### 1. Database CRD Creation

```
User → kubectl apply → API Server → Etcd (stores Database CR)
                         ↓
                  Controller watches
```

### 2. Controller Reconciliation

```
Controller Detects Change
        ↓
Fetch Database CR
        ↓
┌───────┴────────────────────────────────────────────┐
│ Switch on database.Spec.Type                        │
├─────────────────────────────────────────────────────┤
│ PostgreSQL    → reconcilePostgreSQL()               │
│ MongoDB       → reconcileMongoDB()                  │
│ Redis         → reconcileRedis()                    │
│ Elasticsearch → reconcileElasticsearch()            │
│ SQLite        → reconcileSQLite()                   │
└─────────────────────────────────────────────────────┘
        ↓
Create/Update Resources
        ↓
Update Status
```

### 3. Resource Management

For each database type, the controller creates:

**StatefulSets (PostgreSQL, MongoDB, Redis, Elasticsearch):**
```
StatefulSet
├── PodTemplate
│   ├── Container (database-specific image)
│   ├── Environment Variables
│   ├── Volume Mounts
│   └── Resource Limits
└── VolumeClaimTemplates
    └── PersistentVolumeClaim
        ├── Storage Size
        └── StorageClass
```

**Deployment (SQLite):**
```
Deployment
├── PodTemplate
│   ├── Container (SQLite image)
│   ├── Volume Mounts
│   └── Resource Limits
└── Volumes
    └── PersistentVolumeClaim
```

**Service:**
```
Service (ClusterIP)
├── Selector: app=database-name
└── Ports:
    ├── PostgreSQL: 5432
    ├── MongoDB: 27017
    ├── Redis: 6379
    ├── Elasticsearch: 9200
    └── SQLite: 8080
```

## State Management

### Status Flow

```
Pending → Creating → Ready
                ↓
              Failed
                ↓
           (Retry/Fix)
                ↓
            Creating → Ready
```

### Database Lifecycle

```
┌─────────────────────────────────────────────────────┐
│ Create                                               │
│  1. User applies Database manifest                  │
│  2. Controller adds finalizer                        │
│  3. Controller creates resources                     │
│  4. Status updated to Ready                          │
└─────────────────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────────────────┐
│ Update                                               │
│  1. User modifies Database manifest                 │
│  2. Controller detects change                        │
│  3. Controller updates resources                     │
│  4. Status reflects new state                        │
└─────────────────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────────────────┐
│ Delete                                               │
│  1. User deletes Database resource                  │
│  2. Controller runs finalizer                        │
│  3. Kubernetes GC deletes owned resources           │
│  4. Controller removes finalizer                     │
│  5. Resource deleted                                 │
└─────────────────────────────────────────────────────┘
```

## Database Type Implementations

### PostgreSQL
- **Workload**: StatefulSet
- **Image**: `postgres:<version>`
- **Port**: 5432
- **Storage**: `/var/lib/postgresql/data`
- **Config**: Database name, username, password (from secret)

### MongoDB
- **Workload**: StatefulSet
- **Image**: `mongo:<version>`
- **Port**: 27017
- **Storage**: `/data/db`
- **Config**: Database name, username, password, replica set name

### Redis
- **Workload**: StatefulSet
- **Image**: `redis:<version>`
- **Port**: 6379
- **Storage**: `/data`
- **Config**: Mode (standalone/sentinel/cluster), password

### Elasticsearch
- **Workload**: StatefulSet
- **Image**: `docker.elastic.co/elasticsearch/elasticsearch:<version>`
- **Ports**: 9200 (HTTP), 9300 (transport)
- **Storage**: `/usr/share/elasticsearch/data`
- **Config**: Cluster name, node roles, discovery type

### SQLite
- **Workload**: Deployment (single replica)
- **Image**: `nouchka/sqlite3:latest`
- **Port**: 8080
- **Storage**: `/data`
- **Config**: Database file path

## Security Model

### RBAC Permissions

```
Database Operator Service Account
├── databases.database-operator.io/*
│   ├── get, list, watch, create, update, patch, delete
│   ├── status: get, update, patch
│   └── finalizers: update
├── apps/*
│   ├── statefulsets: get, list, watch, create, update, patch, delete
│   └── deployments: get, list, watch, create, update, patch, delete
└── core/*
    ├── services: get, list, watch, create, update, patch, delete
    ├── configmaps: get, list, watch, create, update, patch, delete
    ├── persistentvolumeclaims: get, list, watch, create, update, patch, delete
    └── secrets: get, list, watch (read-only)
```

### Secret Management

```
User Creates Secret
        ↓
Database CR references Secret
        ↓
Controller reads Secret
        ↓
Injects into Pod Environment Variables
```

## Extensibility

### Adding New Database Types

1. Add new constant in `DatabaseType` enum
2. Implement `reconcile<DatabaseType>()` function
3. Add database-specific configuration struct
4. Create helper functions for environment variables
5. Implement StatefulSet/Deployment creation logic
6. Update switch statement in `reconcileDatabase()`
7. Create sample manifest
8. Update documentation

### Adding Webhooks (Future)

```
API Server → Webhook Server
              ↓
        ┌─────┴─────┐
        │           │
    Validate    Default
        │           │
        └─────┬─────┘
              ↓
        Allow/Deny
```

## Monitoring & Observability

### Metrics (via controller-runtime)

- Reconciliation count
- Reconciliation errors
- Reconciliation duration
- Queue depth
- Worker goroutines

### Logs

- Structured logging (JSON)
- Log levels: Info, Error, Debug
- Context: database name, namespace, type

### Status Conditions

```
Database.Status.Conditions[]
├── Type: Ready
├── Status: True/False/Unknown
├── Reason: ReconciliationFailed/DatabaseReady
├── Message: Detailed error/success message
└── LastTransitionTime: timestamp
```

## High Availability

### Operator HA

```
Deployment (replicas: 2-3)
├── Pod 1 (Active - holds leader lock)
├── Pod 2 (Standby - waiting for lock)
└── Pod 3 (Standby - waiting for lock)
```

Leader election ensures only one controller instance is active.

### Database HA

```
StatefulSet (replicas: 3)
├── Pod 0 (Master/Primary)
├── Pod 1 (Replica/Secondary)
└── Pod 2 (Replica/Secondary)
```

## Performance Considerations

### Controller Settings

- **Reconciliation rate**: 5 minutes for periodic reconciliation
- **Concurrent reconcilers**: Configurable (default: 1 per controller)
- **Rate limiting**: Exponential backoff on errors

### Resource Management

- **CPU**: Operator typically uses <100m
- **Memory**: Operator typically uses <128Mi
- **Database resources**: User-defined per database

## Future Enhancements

1. **Webhooks**: Validation and mutation webhooks for CRD
2. **Backup/Restore**: Automated backup scheduling and restore operations
3. **Upgrades**: Blue-green or rolling upgrades for version changes
4. **Monitoring**: Built-in Prometheus metrics for databases
5. **Multi-region**: Cross-region replication support
6. **TLS/mTLS**: Automated certificate management
7. **Service Mesh**: Integration with Istio/Linkerd
8. **Autoscaling**: HPA integration based on database metrics
