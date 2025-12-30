# Usage Endpoint Documentation

## Overview

The database operator now provides a `/usage` endpoint that exposes database usage statistics. This endpoint is available on the metrics server and provides information about all databases managed by the operator.

## Accessing the Usage Endpoint

The usage endpoint is served on the same port as the metrics endpoint (default is port 8443 for HTTPS or 8080 for HTTP).

### Example Request

```bash
# If metrics are served over HTTP (--metrics-secure=false)
curl http://localhost:8080/usage

# If metrics are served over HTTPS (default)
curl -k https://localhost:8443/usage
```

### With Authentication (Production)

In production, the metrics endpoint is secured with authentication. You'll need proper credentials:

```bash
# Create a service account token
kubectl create serviceaccount metrics-reader -n database-operator-system
kubectl create clusterrolebinding metrics-reader --clusterrole=metrics-reader --serviceaccount=database-operator-system:metrics-reader

# Get the token
TOKEN=$(kubectl create token metrics-reader -n database-operator-system)

# Access the endpoint
kubectl port-forward -n database-operator-system svc/database-operator-controller-manager-metrics-service 8443:8443
curl -k -H "Authorization: Bearer $TOKEN" https://localhost:8443/usage
```

## Response Format

The endpoint returns a JSON response with the following structure:

```json
{
  "total_databases": 3,
  "by_type": {
    "PostgreSQL": 1,
    "MongoDB": 1,
    "Redis": 1
  },
  "by_phase": {
    "Ready": 2,
    "Pending": 1
  },
  "databases": [
    {
      "name": "postgresql-sample",
      "namespace": "default",
      "type": "PostgreSQL",
      "version": "16",
      "phase": "Ready",
      "replicas": 1,
      "ready": 1
    },
    {
      "name": "mongodb-sample",
      "namespace": "default",
      "type": "MongoDB",
      "version": "7.0",
      "phase": "Ready",
      "replicas": 3,
      "ready": 3
    },
    {
      "name": "redis-sample",
      "namespace": "default",
      "type": "Redis",
      "version": "7.2",
      "phase": "Pending",
      "replicas": 1,
      "ready": 0
    }
  ]
}
```

## Fields Description

- **total_databases**: Total number of database instances managed by the operator
- **by_type**: Count of databases grouped by type (PostgreSQL, MongoDB, Redis, Elasticsearch, SQLite)
- **by_phase**: Count of databases grouped by phase (Pending, Creating, Ready, Failed, Deleting, Upgrading)
- **databases**: Array of all database instances with their details:
  - **name**: Name of the database resource
  - **namespace**: Kubernetes namespace
  - **type**: Database type
  - **version**: Database version
  - **phase**: Current phase of the database
  - **replicas**: Desired number of replicas
  - **ready**: Number of ready replicas

## Use Cases

### Monitoring and Alerting

You can use this endpoint to:
- Monitor the total number of databases across your cluster
- Track the distribution of database types
- Identify databases that are not in Ready state
- Create alerts for databases with mismatched replica counts

### Dashboard Integration

The endpoint can be integrated with:
- Prometheus for scraping metrics
- Grafana for visualization
- Custom monitoring dashboards
- CI/CD pipelines for health checks

### Example: Check for Unhealthy Databases

```bash
# Get all databases not in Ready state
curl -s http://localhost:8080/usage | jq '.databases[] | select(.phase != "Ready")'

# Count databases with replica mismatch
curl -s http://localhost:8080/usage | jq '.databases[] | select(.replicas != .ready) | .name'
```

## Security Considerations

1. **Access Control**: The usage endpoint is served on the metrics server, which should be protected with proper RBAC in production
2. **Network Policy**: Consider using Kubernetes Network Policies to restrict access to the metrics service
3. **TLS**: Enable TLS for the metrics server in production (default behavior)
4. **Authentication**: Use service account tokens for authentication in production environments
