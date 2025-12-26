# Contributing to Database Operator

Thank you for your interest in contributing to the Database Operator! This guide will help you get started.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Docker (for building images)
- kubectl
- Access to a Kubernetes cluster (kind, minikube, or cloud provider)

### Getting Started

1. **Fork and clone the repository**

```bash
git clone https://github.com/ivikasavnish/database-crd.git
cd database-crd
```

2. **Install dependencies**

```bash
go mod download
```

3. **Install controller-gen**

```bash
make controller-gen
```

4. **Run the operator locally**

```bash
make run
```

## Project Structure

```
database-crd/
├── api/v1/              # API type definitions (CRD spec)
├── controllers/         # Reconciliation logic
├── engines/             # Database engine implementations
│   ├── interface.go     # Engine interface definition
│   ├── factory.go       # Engine selection
│   └── postgres/        # PostgreSQL implementation
├── auth/                # Authentication and credential rotation
├── backup/              # Backup and restore logic
├── internal/utils/      # Shared utilities
├── config/              # Kubernetes manifests
└── test/                # Test scripts
```

## Adding a New Database Engine

To add support for a new database engine (e.g., MySQL):

### 1. Create engine package

```bash
mkdir -p engines/mysql
```

### 2. Implement the Engine interface

Create `engines/mysql/mysql.go`:

```go
package mysql

import (
    "context"
    // ... imports
)

type MySQLEngine struct{}

func NewMySQLEngine() *MySQLEngine {
    return &MySQLEngine{}
}

// Implement all Engine interface methods
func (e *MySQLEngine) Validate(ctx context.Context, db *dbv1.Database) error {
    // Validation logic
    return nil
}

func (e *MySQLEngine) EnsureStorage(ctx context.Context, db *dbv1.Database, c client.Client) (*corev1.PersistentVolumeClaim, error) {
    // Storage provisioning
    return nil, nil
}

// ... implement all other methods
```

### 3. Update the engine factory

Edit `engines/factory.go`:

```go
func (f *DefaultEngineFactory) GetEngine(db *dbv1.Database) (Engine, error) {
    switch db.Spec.Engine {
    case dbv1.EnginePostgreSQL:
        return postgres.NewPostgresEngine(), nil
    case dbv1.EngineMySQL:
        return mysql.NewMySQLEngine(), nil  // Add this
    // ... other cases
    }
}
```

### 4. Add to CRD enum

Edit `api/v1/database_types.go`:

```go
const (
    EnginePostgreSQL DatabaseEngine = "PostgreSQL"
    EngineMySQL DatabaseEngine = "MySQL"  // Add this
    // ... other engines
)
```

### 5. Regenerate CRD and code

```bash
make manifests generate
```

### 6. Test your implementation

Create a sample in `config/samples/mysql_sample.yaml`:

```yaml
apiVersion: db.platform.io/v1
kind: Database
metadata:
  name: mysql-sample
spec:
  engine: MySQL
  version: "8.0"
  topology:
    mode: Replicated
    replicas: 3
  storage:
    size: 50Gi
```

## Making Changes

### 1. Code Changes

- Follow Go best practices
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions small and focused

### 2. API Changes

When modifying the CRD:

1. Edit `api/v1/database_types.go`
2. Add kubebuilder markers for validation
3. Run `make manifests generate` to regenerate code
4. Update samples if needed

Example markers:
```go
// +kubebuilder:validation:Required
// +kubebuilder:validation:Pattern=`^[0-9]+\.[0-9]+(\.[0-9]+)?$`
// +kubebuilder:default="default"
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=100
```

### 3. Controller Changes

When modifying reconciliation logic:

1. Keep operations idempotent
2. Use `CreateOrUpdate` for resource management
3. Update status conditions appropriately
4. Handle errors gracefully
5. Add logging for debugging

Example:
```go
if err := r.someOperation(ctx, db); err != nil {
    log.Error(err, "Failed to perform operation")
    r.setCondition(db, ConditionType, metav1.ConditionFalse, "OperationFailed", err.Error())
    return ctrl.Result{RequeueAfter: 30 * time.Second}, err
}
r.setCondition(db, ConditionType, metav1.ConditionTrue, "OperationSucceeded", "Operation completed")
```

## Testing

### Running Tests

```bash
# Run unit tests
go test ./...

# Run integration tests
./test/integration_test.sh

# Build to verify compilation
make build
```

### Manual Testing

1. Start the operator:
```bash
make run
```

2. In another terminal, apply a sample:
```bash
kubectl apply -f config/samples/db_v1_database.yaml
```

3. Watch the status:
```bash
kubectl get databases -w
```

4. Check logs in the operator terminal

## Code Style

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Run `gofmt` before committing
- Use `go vet` to catch common errors
- Prefer short variable names in small scopes
- Use descriptive names for package-level declarations

### Comments

- Add comments for all exported types and functions
- Use complete sentences
- Explain "why" not "what" for complex logic

Example:
```go
// EnsureStorage creates or updates the PersistentVolumeClaim for the database.
// It respects the storageClassName and size specified in the Database spec,
// and sets an owner reference for automatic cleanup.
func (e *PostgresEngine) EnsureStorage(ctx context.Context, db *dbv1.Database, c client.Client) (*corev1.PersistentVolumeClaim, error) {
    // Implementation
}
```

## Commit Messages

Use conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance tasks

Examples:
```
feat(engines): add MySQL engine implementation

Implement the Engine interface for MySQL with support for:
- Standalone and replicated topologies
- Backup via mysqldump
- Credential rotation

Closes #123

---

fix(controller): handle nil pointer in status update

Check if status.rotationStatus is nil before accessing fields
to prevent panic during reconciliation.

---

docs(readme): add MySQL to supported engines list
```

## Pull Request Process

1. **Create a branch**
```bash
git checkout -b feature/my-feature
```

2. **Make your changes**
- Write clean, documented code
- Add tests if applicable
- Update documentation

3. **Test your changes**
```bash
make build
make manifests generate
./test/integration_test.sh
```

4. **Commit your changes**
```bash
git add .
git commit -m "feat(scope): description"
```

5. **Push to your fork**
```bash
git push origin feature/my-feature
```

6. **Create a Pull Request**
- Provide a clear description
- Reference any related issues
- Include screenshots for UI changes
- Ensure CI passes

## Design Principles

When contributing, please adhere to these principles:

### 1. Idempotency
All reconciliation operations must be idempotent - calling them multiple times should have the same effect as calling once.

### 2. Level-based Logic
React to the current state, not events. The reconciler should be able to reconstruct the desired state from the spec alone.

### 3. No Blocking Calls
Long-running operations must be handled asynchronously via Jobs or other Kubernetes resources.

### 4. Status is Source of Truth
Always update status to reflect the current state. Other components should read status, not spec.

### 5. Engine Isolation
Keep engine-specific logic isolated behind the Engine interface. The controller should be engine-agnostic.

## Common Tasks

### Adding a New Validation Rule

1. Edit `controllers/database_controller.go`
2. Add logic to `validateSpec()`:
```go
func (r *DatabaseReconciler) validateSpec(ctx context.Context, db *dbv1.Database) error {
    // ... existing validation

    // Add new rule
    if db.Spec.Engine == dbv1.EngineMySQL && db.Spec.Topology.Shards > 0 {
        return fmt.Errorf("MySQL does not support sharding")
    }

    return nil
}
```

### Adding a New Status Condition

1. Edit `api/v1/database_types.go`:
```go
const (
    // ... existing conditions
    ConditionTypeMyNewCondition = "MyNewCondition"
)
```

2. Use in controller:
```go
r.setCondition(db, dbv1.ConditionTypeMyNewCondition, metav1.ConditionTrue, "ReasonHere", "Message here")
```

### Adding a New Backup Method

1. Edit `api/v1/database_types.go`:
```go
const (
    // ... existing methods
    BackupMethodMyMethod BackupMethod = "MyMethod"
)
```

2. Implement in `backup/backup.go`:
```go
func (bm *BackupManager) buildBackupCommand(db *dbv1.Database) []string {
    switch db.Spec.Backup.Method {
    case dbv1.BackupMethodMyMethod:
        return []string{"/bin/sh", "-c", "my-backup-command"}
    // ... other cases
    }
}
```

## Getting Help

- Check existing issues: https://github.com/ivikasavnish/database-crd/issues
- Read the documentation: README.md, QUICKSTART.md
- Look at existing implementations for reference
- Ask questions in pull requests or issues

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
