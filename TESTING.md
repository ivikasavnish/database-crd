# Testing Guide for Database Operator

This guide explains how to test the Database Operator both locally and in a Kubernetes cluster.

## Prerequisites

- Go 1.24+
- Docker (for building images)
- kubectl configured with a Kubernetes cluster
- kind or minikube (for local testing)

## Running Tests

### Unit Tests

Run the unit tests with:

```bash
make test
```

This will:
1. Generate code and manifests
2. Run `go vet` and `go fmt`
3. Set up envtest
4. Execute all unit tests with coverage

### E2E Tests

End-to-end tests require a running Kubernetes cluster:

```bash
# Start a local cluster (if needed)
kind create cluster --name database-operator-test

# Run E2E tests
make test-e2e
```

## Local Development

### Running the Operator Locally

1. **Install CRDs into the cluster:**

```bash
make install
```

This installs the Database CRD into your current kubectl context.

2. **Run the operator locally:**

```bash
make run
```

The operator will run on your local machine and connect to your Kubernetes cluster.

### Testing with Sample Databases

In a new terminal, apply sample databases:

#### PostgreSQL

```bash
# Create secret
kubectl apply -f config/samples/databases/secrets.yaml

# Create PostgreSQL database
kubectl apply -f config/samples/databases/postgresql.yaml

# Check status
kubectl get databases
kubectl describe database postgresql-sample

# Check created resources
kubectl get statefulsets
kubectl get services
kubectl get pvc
```

#### MongoDB

```bash
kubectl apply -f config/samples/databases/mongodb.yaml
kubectl get databases mongodb-sample -o yaml
```

#### Redis

```bash
kubectl apply -f config/samples/databases/redis.yaml
kubectl logs -l app=redis-sample
```

#### Elasticsearch

```bash
kubectl apply -f config/samples/databases/elasticsearch.yaml
kubectl get pods -l app=elasticsearch-sample
```

#### SQLite

```bash
kubectl apply -f config/samples/databases/sqlite.yaml
kubectl get deployments sqlite-sample
```

### Observing Reconciliation

Watch the operator logs to see reconciliation in action:

```bash
# In the terminal running 'make run'
# You'll see logs like:
# INFO    Reconciling Database    {"controller": "database", "name": "postgresql-sample"}
# INFO    Creating StatefulSet    {"controller": "database", "name": "postgresql-sample"}
```

### Testing Status Updates

Check the database status:

```bash
kubectl get databases
# Output:
# NAME                TYPE         VERSION   PHASE     READY   AGE
# postgresql-sample   PostgreSQL   16        Ready     1       2m

kubectl get database postgresql-sample -o jsonpath='{.status}' | jq .
```

### Testing Connection

Once a database is Ready, test connectivity:

#### PostgreSQL

```bash
# Get service name
SERVICE=$(kubectl get database postgresql-sample -o jsonpath='{.status.serviceName}')

# Port-forward
kubectl port-forward svc/$SERVICE 5432:5432

# Connect (in another terminal)
PGPASSWORD=changeme123 psql -h localhost -U appuser -d myapp
```

#### MongoDB

```bash
SERVICE=$(kubectl get database mongodb-sample -o jsonpath='{.status.serviceName}')
kubectl port-forward svc/$SERVICE 27017:27017

# Connect
mongosh "mongodb://appuser:changeme456@localhost:27017/myapp"
```

#### Redis

```bash
SERVICE=$(kubectl get database redis-sample -o jsonpath='{.status.serviceName}')
kubectl port-forward svc/$SERVICE 6379:6379

# Connect
redis-cli -a changeme789
```

### Testing Updates

Test updating a database:

```bash
# Edit the database
kubectl edit database postgresql-sample

# Change replicas from 1 to 2
# Save and exit

# Watch the reconciliation
kubectl get statefulsets -w
kubectl get databases postgresql-sample -w
```

### Testing Deletion

Test cleanup with finalizers:

```bash
# Delete a database
kubectl delete database postgresql-sample

# Watch resources being cleaned up
kubectl get all -l app=postgresql-sample -w
```

## In-Cluster Deployment

### Build and Deploy

1. **Build the Docker image:**

```bash
make docker-build IMG=your-registry/database-operator:v1.0.0
```

2. **Push to registry:**

```bash
make docker-push IMG=your-registry/database-operator:v1.0.0
```

3. **Deploy to cluster:**

```bash
make deploy IMG=your-registry/database-operator:v1.0.0
```

4. **Verify deployment:**

```bash
kubectl get deployment -n database-operator-system
kubectl get pods -n database-operator-system
kubectl logs -n database-operator-system -l control-plane=controller-manager
```

### Testing in Cluster

Once deployed, apply samples:

```bash
kubectl apply -f config/samples/databases/
kubectl get databases --all-namespaces
```

## Troubleshooting

### Check Operator Logs

```bash
# Local run
# Logs appear in the terminal running 'make run'

# In-cluster
kubectl logs -n database-operator-system -l control-plane=controller-manager --tail=100 -f
```

### Check Database Status

```bash
kubectl describe database <database-name>
kubectl get database <database-name> -o yaml
```

### Check Created Resources

```bash
# StatefulSets
kubectl get statefulsets -l database-type=PostgreSQL

# Services
kubectl get services -l app.kubernetes.io/managed-by=database-operator

# PVCs
kubectl get pvc -l app.kubernetes.io/managed-by=database-operator
```

### Common Issues

**Database stuck in Creating phase:**
- Check if storage class exists: `kubectl get storageclass`
- Check PVC status: `kubectl get pvc`
- Check pod logs: `kubectl logs <pod-name>`

**StatefulSet not created:**
- Check operator logs for errors
- Verify RBAC permissions: `kubectl auth can-i create statefulsets --as=system:serviceaccount:database-operator-system:database-operator-controller-manager`

**Connection issues:**
- Verify service exists: `kubectl get svc`
- Check pod status: `kubectl get pods`
- Check if secret exists: `kubectl get secret <secret-name>`

## Performance Testing

### Testing with Multiple Databases

Create multiple databases to test operator performance:

```bash
for i in {1..10}; do
  cat <<EOF | kubectl apply -f -
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: postgres-$i
spec:
  type: PostgreSQL
  version: "16"
  replicas: 1
  storage:
    size: 1Gi
EOF
done

# Watch reconciliation
kubectl get databases -w
```

### Load Testing

Monitor operator resource usage:

```bash
kubectl top pod -n database-operator-system
```

## Cleanup

### Delete Test Databases

```bash
kubectl delete databases --all
```

### Uninstall CRDs

```bash
make uninstall
```

### Delete Operator

```bash
# If running locally, just stop with Ctrl+C

# If deployed in-cluster
make undeploy
```

### Delete Test Cluster

```bash
kind delete cluster --name database-operator-test
```

## Continuous Integration

The operator includes GitHub Actions workflows:

- **Lint**: Runs on every push
- **Test**: Runs unit tests on PRs
- **E2E**: Runs end-to-end tests on main branch

View workflow results in the GitHub Actions tab of your repository.

## Best Practices

1. **Always test locally first** before deploying to production
2. **Use separate namespaces** for testing different configurations
3. **Monitor resource usage** when testing with multiple databases
4. **Test upgrade paths** by updating database versions
5. **Verify backups** if implementing backup functionality
6. **Test failure scenarios** (pod crashes, node failures, etc.)

## Next Steps

- Add webhook validation tests
- Implement integration tests with real database clients
- Add chaos engineering tests
- Set up performance benchmarks
- Create load testing scenarios
