# Database Operator

A production-grade Kubernetes Operator for managing multiple database engines using a unified Custom Resource Definition (CRD).

## Overview

The Database Operator provides a declarative API for deploying, managing, and operating databases on Kubernetes with enterprise features including:

- **Multi-Engine Support**: PostgreSQL, MongoDB, Redis, Elasticsearch, SQLite
- **Pluggable Architecture**: Single controller with engine-specific implementations
- **Production Features**: Backup/restore, credential rotation, self-healing, observability
- **Consul Integration**: Full credential management through Consul KV store
- **Safe Operations**: Finalizers, maintenance windows, validation, deletion policies

## Features

### Core Capabilities
- ✅ Declarative database lifecycle management (install, scale, upgrade, heal)
- ✅ Multiple topology modes (Standalone, Replicated, Cluster, Sharded)
- ✅ Automated backups with CronJobs (Snapshot, Dump, WAL, Incremental)
- ✅ Point-in-time restore support
- ✅ Two-phase credential rotation with zero downtime
- ✅ Consul integration for credential management
- ✅ Maintenance windows for controlled upgrades
- ✅ Pause/resume reconciliation
- ✅ Safe deletion policies (Retain, Snapshot, Delete)
- ✅ Comprehensive status reporting with conditions
- ✅ Observability hooks (metrics, logging, tracing)

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  Database Controller                      │
│  ┌──────────────────────────────────────────────────┐   │
│  │            Reconciliation Loop                   │   │
│  │  - Validate Spec                                 │   │
│  │  - Check Lifecycle State (Paused?)               │   │
│  │  - Get Engine (Pluggable)                        │   │
│  │  - EnsureStorage / Config / Service / Workload   │   │
│  │  - Handle Scale / Upgrade / Backup / Rotation    │   │
│  │  - Update Status & Conditions                    │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            │               │               │
    ┌───────▼──────┐ ┌─────▼──────┐ ┌─────▼──────┐
    │  PostgreSQL  │ │  MongoDB   │ │   Redis    │
    │    Engine    │ │   Engine   │ │   Engine   │
    └──────────────┘ └────────────┘ └────────────┘
```

## Installation

### Prerequisites
- Kubernetes cluster (v1.25+)
- kubectl configured
- Go 1.21+ (for development)

### Quick Start

1. **Install CRDs**:
```bash
make install
```

2. **Run the operator locally**:
```bash
make run
```

3. **Deploy a database**:
```bash
kubectl apply -f config/samples/db_v1_database.yaml
```

4. **Check status**:
```bash
kubectl get databases
kubectl describe database postgres-sample
```

## Usage

### Basic PostgreSQL Database

```yaml
apiVersion: db.platform.io/v1
kind: Database
metadata:
  name: my-postgres
spec:
  engine: PostgreSQL
  version: "15.0"
  topology:
    mode: Standalone
    replicas: 1
  storage:
    size: 10Gi
```

### Production PostgreSQL with All Features

```yaml
apiVersion: db.platform.io/v1
kind: Database
metadata:
  name: prod-postgres
spec:
  engine: PostgreSQL
  version: "15.0"
  profile: prod
  
  topology:
    mode: Replicated
    replicas: 3
    antiAffinity: true
  
  storage:
    storageClassName: fast-ssd
    size: 50Gi
    snapshots: true
  
  resources:
    requests:
      memory: "2Gi"
      cpu: "1000m"
    limits:
      memory: "4Gi"
      cpu: "2000m"
  
  networking:
    serviceType: ClusterIP
    port: 5432
    tls:
      enabled: true
      certManager: true
  
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
  
  auth:
    consul:
      enabled: true
      address: consul.default.svc.cluster.local:8500
      path: database/credentials/prod-postgres
      tokenSecretRef:
        name: consul-token
        key: token
    rotationPolicy:
      enabled: true
      schedule: "0 0 1 * *"
      strategy: TwoPhase
  
  maintenance:
    windows:
      - dayOfWeek: 0
        startTime: "02:00"
        duration: 4h
    autoUpgrade: false
  
  observability:
    metrics:
      enabled: true
      serviceMonitor: true
      port: 9187
    logging:
      level: info
      format: json
  
  lifecycle:
    paused: false
    deletionPolicy: Snapshot
  
  engineConfig:
    max_connections: "200"
    shared_buffers: "256MB"
```

## Consul Integration

The operator supports full integration with HashiCorp Consul for credential management:

### Setup

1. **Create Consul token secret**:
```bash
kubectl create secret generic consul-token \
  --from-literal=token=your-consul-token
```

2. **Configure database with Consul**:
```yaml
spec:
  auth:
    consul:
      enabled: true
      address: consul.default.svc.cluster.local:8500
      path: database/credentials/my-db
      tokenSecretRef:
        name: consul-token
        key: token
```

3. **Credentials are automatically synced**:
   - New credentials → Consul KV → Kubernetes Secret
   - Rotation → Updates both Consul and Secret
   - Applications read from either source

## Credential Rotation

The operator implements two-phase credential rotation for zero-downtime:

### Phase 1: Create New Credentials
- Generate new secure credentials
- Store in `{db-name}-credentials-new` secret
- Sync to Consul if enabled
- Create database user with new credentials

### Phase 2: Cutover
- Promote new credentials to primary
- Backup old credentials temporarily
- Update all services to use new credentials

### Phase 3: Revoke Old
- Revoke database access for old credentials
- Delete old credentials
- Complete rotation cycle

### Example Configuration

```yaml
spec:
  auth:
    rotationPolicy:
      enabled: true
      schedule: "0 0 1 * *"  # Monthly on the 1st at midnight
      strategy: TwoPhase
```

## Validation Rules

The operator enforces several validation rules:

- ✅ SQLite cannot have `replicas > 1`
- ✅ Elasticsearch requires at least 3 nodes (no Standalone mode)
- ✅ Version downgrades are prevented
- ✅ Upgrades respect maintenance windows
- ✅ Incompatible topology changes are blocked

## Status and Conditions

The operator maintains comprehensive status:

### Status Fields
- `phase`: Current phase (Pending, Provisioning, Ready, Upgrading, Scaling, Failed, etc.)
- `conditions`: Array of condition objects
- `endpoint`: Connection endpoint
- `readyReplicas`: Number of ready replicas
- `currentVersion`: Currently running version
- `observedGeneration`: Last observed spec generation
- `lastBackup`: Timestamp of last successful backup
- `health`: Health status object
- `rotationStatus`: Credential rotation status

### Condition Types
- `Ready`: Database is ready for use
- `Provisioned`: Database resources are provisioned
- `StorageReady`: Storage is ready
- `BackupConfigured`: Backup is configured
- `Validated`: Spec is valid

## Development

### Building

```bash
# Build binary
make build

# Run tests
make test

# Generate manifests and code
make manifests generate

# Run locally
make run
```

### Project Structure

```
database-crd/
├── api/v1/              # API type definitions
│   ├── groupversion_info.go
│   └── database_types.go
├── controllers/         # Controllers
│   └── database_controller.go
├── engines/             # Database engine implementations
│   ├── interface.go
│   ├── factory.go
│   └── postgres/
│       └── postgres.go
├── auth/                # Authentication and rotation
│   └── rotation.go
├── backup/              # Backup management
│   └── backup.go
├── config/              # Kubernetes manifests
│   ├── crd/
│   ├── rbac/
│   ├── manager/
│   └── samples/
├── hack/                # Build scripts
├── main.go              # Entry point
├── Makefile             # Build automation
├── PROJECT              # Kubebuilder metadata
└── README.md            # This file
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Design Principles

- **Idempotent Reconciliation**: All operations can be safely retried
- **Level-Based Logic**: React to current state, not events
- **Status as First-Class API**: Rich status reporting
- **Finalizers for Safety**: Controlled deletion with policies
- **Engine Isolation**: Engine-specific logic behind interfaces
- **Future-Proof CRD**: Extensible without breaking changes
- **No Blocking Calls**: Async operations via Jobs

## Supported Engines

| Engine         | Status      | Notes                          |
|---------------|-------------|--------------------------------|
| PostgreSQL    | ✅ Implemented | Full feature support         |
| MongoDB       | 🚧 Planned   | Engine stub created           |
| Redis         | 🚧 Planned   | Engine stub created           |
| Elasticsearch | 🚧 Planned   | Engine stub created           |
| SQLite        | 🚧 Planned   | Engine stub created           |

## License

Licensed under the Apache License, Version 2.0.
