# Database Operator

**Production-Grade Kubernetes Operator for Multi-Database Management**

Manage multiple database types (PostgreSQL, MongoDB, Redis, Elasticsearch, SQLite) using a single unified Custom Resource Definition (CRD).

## Quick Start

```bash
# Install CRDs
make install

# Run the operator
make run

# Apply a sample database
kubectl apply -f config/samples/databases/postgresql.yaml

# Check the database status
kubectl get databases
```

## Features

- ✅ **Unified CRD** - Single API for all database types
- ✅ **5 Database Types** - PostgreSQL, MongoDB, Redis, Elasticsearch, SQLite
- ✅ **Production Ready** - StatefulSets, persistent storage, resource management
- ✅ **Status Tracking** - Real-time status, conditions, and health monitoring
- ✅ **Flexible Configuration** - Database-specific settings and parameters
- ✅ **Security** - Secret management for credentials
- ✅ **High Availability** - Support for replicas and scaling

## Documentation

See [OPERATOR_README.md](OPERATOR_README.md) for comprehensive documentation including:
- Architecture overview
- Installation instructions
- API reference
- Usage examples
- Development guide
- Production considerations

## Example

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: my-postgres
spec:
  type: PostgreSQL
  version: "16"
  replicas: 1
  storage:
    size: 10Gi
  postgresql:
    database: myapp
    username: appuser
```

## License

Apache License 2.0 - See LICENSE file for details
