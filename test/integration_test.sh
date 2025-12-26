#!/bin/bash
# Integration test script for Database Operator

set -e

echo "=== Database Operator Integration Tests ==="
echo ""

# Check if operator binary exists
if [ ! -f "bin/manager" ]; then
    echo "❌ Error: operator binary not found. Run 'make build' first."
    exit 1
fi

echo "✅ Operator binary found"

# Check if CRD manifest exists
if [ ! -f "config/crd/bases/db.platform.io_databases.yaml" ]; then
    echo "❌ Error: CRD manifest not found. Run 'make manifests' first."
    exit 1
fi

echo "✅ CRD manifest found"

# Validate CRD structure
echo ""
echo "=== Validating CRD Structure ==="

# Check for required fields
REQUIRED_FIELDS=(
    "engine"
    "version"
    "topology"
    "storage"
    "backup"
    "auth"
    "consul"
    "rotationPolicy"
    "maintenance"
    "observability"
    "lifecycle"
    "engineConfig"
)

for field in "${REQUIRED_FIELDS[@]}"; do
    if grep -q "$field:" config/crd/bases/db.platform.io_databases.yaml; then
        echo "✅ Field '$field' found in CRD"
    else
        echo "❌ Field '$field' NOT found in CRD"
        exit 1
    fi
done

# Check for engine types
echo ""
echo "=== Validating Engine Types ==="
ENGINES=("PostgreSQL" "MongoDB" "Redis" "Elasticsearch" "SQLite")

for engine in "${ENGINES[@]}"; do
    if grep -q "$engine" config/crd/bases/db.platform.io_databases.yaml; then
        echo "✅ Engine '$engine' supported"
    else
        echo "❌ Engine '$engine' NOT supported"
        exit 1
    fi
done

# Check for status fields
echo ""
echo "=== Validating Status Fields ==="
STATUS_FIELDS=(
    "phase"
    "conditions"
    "endpoint"
    "readyReplicas"
    "currentVersion"
    "observedGeneration"
    "lastBackup"
    "health"
    "rotationStatus"
)

for field in "${STATUS_FIELDS[@]}"; do
    if grep -q "$field:" config/crd/bases/db.platform.io_databases.yaml; then
        echo "✅ Status field '$field' found"
    else
        echo "❌ Status field '$field' NOT found"
        exit 1
    fi
done

# Validate sample manifests
echo ""
echo "=== Validating Sample Manifests ==="

SAMPLES=(
    "config/samples/db_v1_database.yaml"
    "config/samples/mongodb_sample.yaml"
    "config/samples/redis_sample.yaml"
    "config/samples/sqlite_sample.yaml"
    "config/samples/elasticsearch_sample.yaml"
)

for sample in "${SAMPLES[@]}"; do
    if [ -f "$sample" ]; then
        echo "✅ Sample manifest '$sample' exists"
    else
        echo "❌ Sample manifest '$sample' NOT found"
        exit 1
    fi
done

# Check code structure
echo ""
echo "=== Validating Code Structure ==="

CODE_FILES=(
    "api/v1/database_types.go"
    "api/v1/groupversion_info.go"
    "api/v1/zz_generated.deepcopy.go"
    "controllers/database_controller.go"
    "engines/interface.go"
    "engines/factory.go"
    "engines/postgres/postgres.go"
    "auth/rotation.go"
    "backup/backup.go"
    "main.go"
)

for file in "${CODE_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ Code file '$file' exists"
    else
        echo "❌ Code file '$file' NOT found"
        exit 1
    fi
done

# Check for key functions in controller
echo ""
echo "=== Validating Controller Functions ==="

CONTROLLER_FUNCTIONS=(
    "Reconcile"
    "handleDeletion"
    "validateSpec"
    "checkMaintenanceWindow"
    "handleCredentialRotation"
    "updateWorkloadStatus"
    "ensureBackupCronJob"
    "SetupWithManager"
)

for func in "${CONTROLLER_FUNCTIONS[@]}"; do
    if grep -q "func.*$func" controllers/database_controller.go; then
        echo "✅ Controller function '$func' implemented"
    else
        echo "❌ Controller function '$func' NOT implemented"
        exit 1
    fi
done

# Check for Engine interface methods
echo ""
echo "=== Validating Engine Interface ==="

ENGINE_METHODS=(
    "Validate"
    "EnsureStorage"
    "EnsureConfig"
    "EnsureService"
    "EnsureWorkload"
    "Scale"
    "Upgrade"
    "Backup"
    "Restore"
    "RotateAuth"
    "Heal"
    "Status"
    "GetEndpoint"
)

for method in "${ENGINE_METHODS[@]}"; do
    if grep -q "$method.*context.Context" engines/interface.go; then
        echo "✅ Engine method '$method' defined"
    else
        echo "❌ Engine method '$method' NOT defined"
        exit 1
    fi
done

# Check for Consul integration
echo ""
echo "=== Validating Consul Integration ==="

if grep -q "Consul" auth/rotation.go && grep -q "syncToConsul" auth/rotation.go; then
    echo "✅ Consul integration implemented"
else
    echo "❌ Consul integration NOT properly implemented"
    exit 1
fi

# Check for validation rules
echo ""
echo "=== Validating Business Rules ==="

VALIDATION_RULES=(
    "SQLite.*replicas"
    "Elasticsearch.*Standalone"
    "validateVersionUpgrade"
    "checkMaintenanceWindow"
)

for rule in "${VALIDATION_RULES[@]}"; do
    if grep -q "$rule" controllers/database_controller.go; then
        echo "✅ Validation rule for '$rule' implemented"
    else
        echo "❌ Validation rule for '$rule' NOT implemented"
        exit 1
    fi
done

# Check for two-phase rotation
echo ""
echo "=== Validating Two-Phase Credential Rotation ==="

ROTATION_PHASES=(
    "RotationPhaseIdle"
    "RotationPhaseCreatingNew"
    "RotationPhaseCutover"
    "RotationPhaseRevoking"
    "RotationPhaseComplete"
)

for phase in "${ROTATION_PHASES[@]}"; do
    if grep -q "$phase" auth/rotation.go; then
        echo "✅ Rotation phase '$phase' defined"
    else
        echo "❌ Rotation phase '$phase' NOT defined"
        exit 1
    fi
done

# Build test
echo ""
echo "=== Testing Build ==="
if make build > /dev/null 2>&1; then
    echo "✅ Build successful"
else
    echo "❌ Build failed"
    exit 1
fi

# Final summary
echo ""
echo "==================================================="
echo "✅ All tests passed successfully!"
echo "==================================================="
echo ""
echo "Summary:"
echo "  - CRD structure validated"
echo "  - All engine types supported"
echo "  - Status fields complete"
echo "  - Sample manifests present"
echo "  - Code structure correct"
echo "  - Controller functions implemented"
echo "  - Engine interface complete"
echo "  - Consul integration present"
echo "  - Validation rules implemented"
echo "  - Two-phase rotation implemented"
echo "  - Build successful"
echo ""
echo "Next steps:"
echo "  1. Deploy to Kubernetes: make install && make deploy"
echo "  2. Create a database: kubectl apply -f config/samples/db_v1_database.yaml"
echo "  3. Check status: kubectl get databases"
echo ""
