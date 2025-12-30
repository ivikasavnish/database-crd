# Pod Error Reporting

## Overview

The database operator now includes enhanced error reporting for pods that are stuck in a pending or failed state. This feature helps administrators quickly identify and troubleshoot issues with database deployments.

## How It Works

During each reconciliation cycle, the operator:

1. Lists all pods associated with the database
2. Checks the status of each pod
3. Extracts detailed error information from:
   - Container statuses (waiting state)
   - Init container statuses (waiting state)
   - Pod conditions (false conditions)
4. Updates the Database status with error details

## Error Information Captured

The operator captures the following error details:

### Container Waiting States

When a container is waiting, the operator captures:
- Container name
- Reason (e.g., ImagePullBackOff, CrashLoopBackOff, ErrImagePull)
- Message with detailed error description

### Pod Conditions

The operator checks pod conditions including:
- **PodScheduled**: Whether the pod has been scheduled to a node
- **ContainersReady**: Whether all containers in the pod are ready
- **Initialized**: Whether all init containers have succeeded
- **Ready**: Whether the pod is ready to serve requests

### Common Error Scenarios

#### 1. Image Pull Errors

```yaml
status:
  phase: Failed
  message: "Pod issues detected: Pod postgresql-sample-0: Container postgresql is waiting - ErrImagePull: Failed to pull image 'postgres:invalid-version'"
```

**Cause**: Invalid image name or version, private registry access issues

**Solution**:
- Verify the database version is correct
- Check image registry credentials
- Ensure network connectivity to the image registry

#### 2. Insufficient Resources

```yaml
status:
  phase: Failed
  message: "Pod issues detected: Pod mongodb-sample-0: PodScheduled - Unschedulable: 0/3 nodes are available: 3 Insufficient memory"
```

**Cause**: Not enough resources (CPU/memory) available in the cluster

**Solution**:
- Reduce resource requests in the Database spec
- Add more nodes to the cluster
- Scale down other workloads

#### 3. Storage Issues

```yaml
status:
  phase: Failed
  message: "Pod issues detected: Pod redis-sample-0: PodScheduled - Unschedulable: pod has unbound immediate PersistentVolumeClaims"
```

**Cause**: No PersistentVolume available matching the PVC requirements

**Solution**:
- Create a PersistentVolume with matching specifications
- Install a storage provisioner (e.g., local-path-provisioner)
- Change the storageClassName to one that is available

#### 4. Container Crash

```yaml
status:
  phase: Failed
  message: "Pod issues detected: Pod elasticsearch-sample-0: Container elasticsearch is waiting - CrashLoopBackOff: Back-off restarting failed container"
```

**Cause**: Container keeps crashing and restarting

**Solution**:
- Check pod logs: `kubectl logs <pod-name>`
- Review container configuration
- Verify environment variables and secrets
- Check resource limits

## Checking Database Status

### Using kubectl

```bash
# Get database status
kubectl get database postgresql-sample -o yaml

# Check the status section
kubectl get database postgresql-sample -o jsonpath='{.status.phase}'
kubectl get database postgresql-sample -o jsonpath='{.status.message}'

# View all databases with their status
kubectl get databases -o custom-columns=NAME:.metadata.name,TYPE:.spec.type,PHASE:.status.phase,MESSAGE:.status.message
```

### Example Output

```bash
$ kubectl get database postgresql-sample -o yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: postgresql-sample
spec:
  type: PostgreSQL
  version: "16"
  replicas: 1
  # ... other spec fields
status:
  phase: Failed
  message: "Pod issues detected: Pod postgresql-sample-0: Container postgresql is waiting - ImagePullBackOff: Back-off pulling image 'postgres:invalid-version'"
  readyReplicas: 0
  conditions:
  - lastTransitionTime: "2025-12-30T09:00:00Z"
    message: "Pod issues detected: Pod postgresql-sample-0: Container postgresql is waiting - ImagePullBackOff: Back-off pulling image 'postgres:invalid-version'"
    observedGeneration: 1
    reason: ReconciliationFailed
    status: "False"
    type: Ready
```

## Troubleshooting Workflow

1. **Check Database Status**
   ```bash
   kubectl get database <name> -o jsonpath='{.status.message}'
   ```

2. **Inspect Pod Details**
   ```bash
   kubectl describe pod <database-name>-0
   ```

3. **Check Pod Logs**
   ```bash
   kubectl logs <database-name>-0
   ```

4. **Check Events**
   ```bash
   kubectl get events --field-selector involvedObject.name=<database-name>-0
   ```

5. **Fix the Issue** based on the error message

6. **Verify Resolution**
   ```bash
   # Wait for the operator to reconcile (typically within 1 minute)
   kubectl get database <name> -o jsonpath='{.status.phase}'
   ```

## Service Type Configuration

The operator now supports configuring the service type for database access. By default, services are created as `NodePort` for easier external access.

### Available Service Types

- **ClusterIP**: Internal access only (default for most Kubernetes services)
- **NodePort**: Accessible via node IP and a static port (default for database operator)
- **LoadBalancer**: Accessible via external load balancer (cloud providers)

### Configuration

Add the `serviceType` field to your Database spec:

```yaml
apiVersion: databases.database-operator.io/v1alpha1
kind: Database
metadata:
  name: postgresql-sample
spec:
  type: PostgreSQL
  version: "16"
  serviceType: LoadBalancer  # Options: ClusterIP, NodePort, LoadBalancer
  # ... other spec fields
```

### Default Behavior

If `serviceType` is not specified, the operator defaults to `NodePort` for easier external access during development and testing.

### Accessing the Database

#### NodePort (Default)

```bash
# Get the NodePort
kubectl get svc postgresql-sample-service -o jsonpath='{.spec.ports[0].nodePort}'

# Connect using any node IP
psql -h <node-ip> -p <node-port> -U postgres
```

#### LoadBalancer

```bash
# Get the external IP
kubectl get svc postgresql-sample-service -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

# Connect using the external IP
psql -h <external-ip> -p 5432 -U postgres
```

#### ClusterIP

```bash
# Port forward for local access
kubectl port-forward svc/postgresql-sample-service 5432:5432

# Connect to localhost
psql -h localhost -p 5432 -U postgres
```

## Best Practices

1. **Monitor Status Regularly**: Set up alerts for databases not in Ready state
2. **Check Logs**: Always check pod logs for detailed error information
3. **Resource Planning**: Ensure adequate resources are available before creating databases
4. **Storage Configuration**: Verify storage classes are properly configured
5. **Image Availability**: Test database versions before deploying to production
6. **Service Type Selection**: Choose appropriate service type based on access requirements
   - Use `ClusterIP` for internal-only access
   - Use `NodePort` for development and testing
   - Use `LoadBalancer` for production external access (with proper security)
