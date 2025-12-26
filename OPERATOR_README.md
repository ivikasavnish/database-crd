# Database Operator

A production-grade Kubernetes Operator for managing multiple database types using a single unified CRD (Custom Resource Definition). Built with Kubebuilder and Go.

## Supported Databases

- **PostgreSQL** - Relational database
- **MongoDB** - Document database
- **Redis** - In-memory data store
- **Elasticsearch** - Search and analytics engine
- **SQLite** - Lightweight embedded database

## Features

- ✅ Single CRD for all database types
- ✅ Automated provisioning and lifecycle management
- ✅ StatefulSet/Deployment management based on database type
- ✅ Persistent storage configuration
- ✅ Resource requests and limits
- ✅ Database-specific configurations
- ✅ Secret management for credentials
- ✅ Service discovery
- ✅ Status tracking and conditions
- ✅ Finalizers for cleanup

## Architecture

The operator uses the controller-runtime framework and follows Kubernetes operator patterns:

- **CRD**: Defines the `Database` custom resource with a unified schema
- **Controller**: Reconciles Database resources and manages workloads
- **Webhooks**: Validates and defaults Database specifications (future)

## Installation

### Prerequisites

- Kubernetes cluster (v1.31+)
- kubectl configured
- Go 1.24+ (for development)

### Deploy the Operator

```bash
# Install CRDs
make install

# Deploy the operator
make deploy

# Or run locally for development
make run
```

## Usage

### Creating a PostgreSQL Database

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: my-postgres
  namespace: default
spec:
  type: PostgreSQL
  version: "16"
  replicas: 1
  storage:
    size: 10Gi
    storageClassName: standard
  resources:
    cpu: 500m
    memory: 1Gi
  postgresql:
    database: myapp
    username: appuser
    passwordSecret:
      name: postgresql-secret
      key: password
```

### Creating a MongoDB Database

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: my-mongodb
  namespace: default
spec:
  type: MongoDB
  version: "7.0"
  replicas: 3
  storage:
    size: 20Gi
  mongodb:
    database: myapp
    username: appuser
    replicaSetName: rs0
    passwordSecret:
      name: mongodb-secret
      key: password
```

### Creating a Redis Database

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: my-redis
  namespace: default
spec:
  type: Redis
  version: "7.2"
  replicas: 1
  storage:
    size: 5Gi
  redis:
    mode: standalone
    passwordSecret:
      name: redis-secret
      key: password
```

### Creating an Elasticsearch Cluster

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: my-elasticsearch
  namespace: default
spec:
  type: Elasticsearch
  version: "8.11.0"
  replicas: 3
  storage:
    size: 50Gi
  elasticsearch:
    clusterName: my-elasticsearch
    nodeRoles:
      - master
      - data
      - ingest
```

### Creating a SQLite Database

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: my-sqlite
  namespace: default
spec:
  type: SQLite
  version: "latest"
  storage:
    size: 1Gi
  sqlite:
    databaseFile: /data/app.db
```

## API Reference

### Database Spec

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `type` | string | Database type (PostgreSQL, MongoDB, Redis, Elasticsearch, SQLite) | Yes |
| `version` | string | Database version to deploy | Yes |
| `replicas` | int32 | Number of replicas (default: 1) | No |
| `storage` | StorageSpec | Storage configuration | No |
| `resources` | ResourceRequirements | CPU and memory resources | No |
| `postgresql` | PostgreSQLConfig | PostgreSQL-specific config | No |
| `mongodb` | MongoDBConfig | MongoDB-specific config | No |
| `redis` | RedisConfig | Redis-specific config | No |
| `elasticsearch` | ElasticsearchConfig | Elasticsearch-specific config | No |
| `sqlite` | SQLiteConfig | SQLite-specific config | No |
| `env` | []EnvVar | Additional environment variables | No |

### Database Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current phase (Pending, Creating, Ready, Failed, Deleting, Upgrading) |
| `conditions` | []Condition | Detailed status conditions |
| `readyReplicas` | int32 | Number of ready replicas |
| `serviceName` | string | Name of the created service |
| `connectionString` | string | Connection information (without credentials) |
| `observedGeneration` | int64 | Latest observed generation |
| `message` | string | Additional status information |

## Examples

All example manifests are available in `config/samples/databases/`:

- `postgresql.yaml` - PostgreSQL database
- `mongodb.yaml` - MongoDB database
- `redis.yaml` - Redis database
- `elasticsearch.yaml` - Elasticsearch cluster
- `sqlite.yaml` - SQLite database
- `secrets.yaml` - Sample secrets for authentication

Apply examples:

```bash
# Create secrets first
kubectl apply -f config/samples/databases/secrets.yaml

# Create a database
kubectl apply -f config/samples/databases/postgresql.yaml

# Check status
kubectl get databases
kubectl describe database postgresql-sample
```

## Development

### Prerequisites

- Go 1.24+
- Docker
- kind or minikube (for local testing)
- Kubebuilder 4.5+

### Build and Test

```bash
# Install dependencies
go mod download

# Generate code and manifests
make generate
make manifests

# Run tests
make test

# Build the operator
make build

# Build Docker image
make docker-build IMG=your-registry/database-operator:tag

# Push Docker image
make docker-push IMG=your-registry/database-operator:tag
```

### Local Development

```bash
# Install CRDs into the cluster
make install

# Run the operator locally
make run

# In another terminal, apply sample resources
kubectl apply -f config/samples/databases/
```

## Production Considerations

### Security

- Always use Secrets for sensitive credentials
- Configure RBAC appropriately
- Use NetworkPolicies to restrict access
- Enable TLS/SSL for production databases
- Consider using cert-manager for certificate management

### High Availability

- Use replicas > 1 for production workloads
- Configure proper resource limits
- Use appropriate storage classes (e.g., SSD)
- Consider anti-affinity rules for pod distribution
- Set up monitoring and alerting

### Storage

- Choose appropriate storage classes based on performance needs
- Consider backup strategies
- Use volume snapshots for disaster recovery
- Monitor storage usage and capacity

### Monitoring

- Use Prometheus for metrics collection
- Set up alerts for database health
- Monitor resource usage (CPU, memory, storage, IOPS)
- Track reconciliation errors

## Roadmap

- [ ] Webhook validation and defaulting
- [ ] Database backup and restore
- [ ] Automated upgrades and migrations
- [ ] Multi-region support
- [ ] Custom metrics and monitoring
- [ ] Advanced security features (TLS, mTLS)
- [ ] Integration with service meshes
- [ ] Additional database types (MySQL, Cassandra, etc.)

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

Copyright 2025 Vikas Avnish.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Support

For issues and questions:
- GitHub Issues: https://github.com/ivikasavnish/database-crd/issues
- Documentation: See inline code documentation and examples

## Acknowledgments

Built with:
- [Kubebuilder](https://kubebuilder.io/) - SDK for building Kubernetes operators
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Kubernetes controller framework
- [Go](https://golang.org/) - Programming language
