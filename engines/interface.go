/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package engines

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dbv1 "github.com/ivikasavnish/database-crd/api/v1"
)

// Engine defines the interface for database lifecycle management
// All engines must implement this interface to support the operator's functionality
type Engine interface {
	// Validate validates the database specification
	// Returns an error if the spec is invalid for this engine
	Validate(ctx context.Context, db *dbv1.Database) error

	// EnsureStorage ensures storage resources (PVCs) are created and configured
	// Returns the PVC reference or error
	EnsureStorage(ctx context.Context, db *dbv1.Database, client client.Client) (*corev1.PersistentVolumeClaim, error)

	// EnsureConfig ensures configuration resources (ConfigMaps, Secrets) are created
	// Returns error if configuration cannot be ensured
	EnsureConfig(ctx context.Context, db *dbv1.Database, client client.Client) error

	// EnsureService ensures Kubernetes Service is created and configured
	// Returns the Service reference or error
	EnsureService(ctx context.Context, db *dbv1.Database, client client.Client) (*corev1.Service, error)

	// EnsureWorkload ensures the database workload (StatefulSet, Deployment) is created
	// Returns error if workload cannot be ensured
	EnsureWorkload(ctx context.Context, db *dbv1.Database, client client.Client) error

	// Scale scales the database to the desired number of replicas
	// Must be idempotent and handle scaling up/down safely
	Scale(ctx context.Context, db *dbv1.Database, client client.Client) error

	// Upgrade upgrades the database to the desired version
	// Must handle rolling upgrades and version compatibility
	Upgrade(ctx context.Context, db *dbv1.Database, client client.Client) error

	// Backup initiates a backup operation
	// Returns the backup job name or error
	Backup(ctx context.Context, db *dbv1.Database, client client.Client) (string, error)

	// Restore restores the database from a backup
	// Returns error if restore cannot be initiated
	Restore(ctx context.Context, db *dbv1.Database, client client.Client) error

	// RotateAuth rotates database credentials
	// Must implement two-phase rotation for zero-downtime
	RotateAuth(ctx context.Context, db *dbv1.Database, client client.Client) error

	// Heal performs self-healing operations
	// Detects and fixes common issues automatically
	Heal(ctx context.Context, db *dbv1.Database, client client.Client) error

	// Status retrieves the current status of the database
	// Returns the health status and any relevant metrics
	Status(ctx context.Context, db *dbv1.Database, client client.Client) (*dbv1.HealthStatus, error)

	// GetEndpoint returns the connection endpoint for the database
	GetEndpoint(ctx context.Context, db *dbv1.Database, client client.Client) (string, error)
}

// EngineFactory is a factory for creating engine instances
type EngineFactory interface {
	// GetEngine returns an engine implementation for the specified database
	GetEngine(db *dbv1.Database) (Engine, error)
}
