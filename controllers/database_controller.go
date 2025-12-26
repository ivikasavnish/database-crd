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

package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dbv1 "github.com/ivikasavnish/database-crd/api/v1"
	"github.com/ivikasavnish/database-crd/engines"
)

const (
	databaseFinalizer = "db.platform.io/finalizer"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EngineFactory engines.EngineFactory
}

// +kubebuilder:rbac:groups=db.platform.io,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=db.platform.io,resources=databases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=db.platform.io,resources=databases/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling Database", "namespace", req.Namespace, "name", req.Name)

	// Fetch the Database instance
	db := &dbv1.Database{}
	if err := r.Get(ctx, req.NamespacedName, db); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Database resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Database")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !db.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, db)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(db, databaseFinalizer) {
		controllerutil.AddFinalizer(db, databaseFinalizer)
		if err := r.Update(ctx, db); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if reconciliation is paused
	if db.Spec.Lifecycle.Paused {
		log.Info("Reconciliation is paused", "namespace", db.Namespace, "name", db.Name)
		r.updatePhase(ctx, db, dbv1.PhasePaused)
		return ctrl.Result{}, nil
	}

	// Validate the database spec
	if err := r.validateSpec(ctx, db); err != nil {
		log.Error(err, "Validation failed")
		r.setCondition(db, dbv1.ConditionTypeValidated, metav1.ConditionFalse, "ValidationFailed", err.Error())
		r.updatePhase(ctx, db, dbv1.PhaseFailed)
		if statusErr := r.Status().Update(ctx, db); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}
	r.setCondition(db, dbv1.ConditionTypeValidated, metav1.ConditionTrue, "ValidationSucceeded", "Spec validation passed")

	// Get the engine for this database
	engine, err := r.EngineFactory.GetEngine(db)
	if err != nil {
		log.Error(err, "Failed to get engine")
		r.updatePhase(ctx, db, dbv1.PhaseFailed)
		if statusErr := r.Status().Update(ctx, db); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Validate using engine-specific validation
	if err := engine.Validate(ctx, db); err != nil {
		log.Error(err, "Engine validation failed")
		r.setCondition(db, dbv1.ConditionTypeValidated, metav1.ConditionFalse, "EngineValidationFailed", err.Error())
		r.updatePhase(ctx, db, dbv1.PhaseFailed)
		if statusErr := r.Status().Update(ctx, db); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Update phase to provisioning if this is a new database
	if db.Status.Phase == "" || db.Status.Phase == dbv1.PhasePending {
		r.updatePhase(ctx, db, dbv1.PhaseProvisioning)
	}

	// Ensure storage
	if _, err := engine.EnsureStorage(ctx, db, r.Client); err != nil {
		log.Error(err, "Failed to ensure storage")
		r.setCondition(db, dbv1.ConditionTypeStorageReady, metav1.ConditionFalse, "StorageFailed", err.Error())
		if statusErr := r.Status().Update(ctx, db); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}
	r.setCondition(db, dbv1.ConditionTypeStorageReady, metav1.ConditionTrue, "StorageReady", "Storage is ready")

	// Ensure configuration
	if err := engine.EnsureConfig(ctx, db, r.Client); err != nil {
		log.Error(err, "Failed to ensure config")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Ensure service
	_, err = engine.EnsureService(ctx, db, r.Client)
	if err != nil {
		log.Error(err, "Failed to ensure service")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Update endpoint in status
	endpoint, err := engine.GetEndpoint(ctx, db, r.Client)
	if err != nil {
		log.Error(err, "Failed to get endpoint")
	} else {
		db.Status.Endpoint = endpoint
	}

	// Handle restore if specified
	if db.Spec.Restore != nil && db.Status.Phase != dbv1.PhaseReady {
		log.Info("Restoring database from backup", "backupName", db.Spec.Restore.BackupName)
		if err := engine.Restore(ctx, db, r.Client); err != nil {
			log.Error(err, "Failed to restore database")
			return ctrl.Result{RequeueAfter: 60 * time.Second}, err
		}
	}

	// Ensure workload
	if err := engine.EnsureWorkload(ctx, db, r.Client); err != nil {
		log.Error(err, "Failed to ensure workload")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}
	r.setCondition(db, dbv1.ConditionTypeProvisioned, metav1.ConditionTrue, "Provisioned", "Database is provisioned")

	// Check if we need to upgrade
	if db.Status.CurrentVersion != "" && db.Status.CurrentVersion != db.Spec.Version {
		log.Info("Version mismatch detected, upgrading", "current", db.Status.CurrentVersion, "desired", db.Spec.Version)

		// Check if upgrade is allowed
		if err := r.checkMaintenanceWindow(db); err != nil {
			log.Info("Upgrade deferred due to maintenance window", "error", err)
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
		}

		r.updatePhase(ctx, db, dbv1.PhaseUpgrading)
		r.setCondition(db, dbv1.ConditionTypeUpgrading, metav1.ConditionTrue, "Upgrading", "Database is being upgraded")

		if err := engine.Upgrade(ctx, db, r.Client); err != nil {
			log.Error(err, "Failed to upgrade database")
			r.setCondition(db, dbv1.ConditionTypeUpgrading, metav1.ConditionFalse, "UpgradeFailed", err.Error())
			return ctrl.Result{RequeueAfter: 60 * time.Second}, err
		}

		r.setCondition(db, dbv1.ConditionTypeUpgrading, metav1.ConditionFalse, "UpgradeComplete", "Upgrade completed successfully")
	}

	// Update status with workload information
	if err := r.updateWorkloadStatus(ctx, db); err != nil {
		log.Error(err, "Failed to update workload status")
	}

	// Handle backup configuration
	if db.Spec.Backup.Enabled {
		if err := r.ensureBackupCronJob(ctx, db); err != nil {
			log.Error(err, "Failed to ensure backup CronJob")
			r.setCondition(db, dbv1.ConditionTypeBackupConfigured, metav1.ConditionFalse, "BackupFailed", err.Error())
		} else {
			r.setCondition(db, dbv1.ConditionTypeBackupConfigured, metav1.ConditionTrue, "BackupConfigured", "Backup is configured")
		}
	}

	// Handle credential rotation if enabled
	if db.Spec.Auth.RotationPolicy != nil && db.Spec.Auth.RotationPolicy.Enabled {
		if err := r.handleCredentialRotation(ctx, db, engine); err != nil {
			log.Error(err, "Failed to handle credential rotation")
		}
	}

	// Perform self-healing
	if err := engine.Heal(ctx, db, r.Client); err != nil {
		log.Error(err, "Failed to perform self-healing")
	}

	// Get health status
	health, err := engine.Status(ctx, db, r.Client)
	if err != nil {
		log.Error(err, "Failed to get health status")
	} else {
		db.Status.Health = *health
	}

	// Update current version
	db.Status.CurrentVersion = db.Spec.Version

	// Update observed generation
	db.Status.ObservedGeneration = db.Generation

	// Update last reconcile time
	now := metav1.Now()
	db.Status.LastReconcileTime = &now

	// Set phase to Ready if everything is successful
	if db.Status.ReadyReplicas == db.Spec.Topology.Replicas {
		r.updatePhase(ctx, db, dbv1.PhaseReady)
		r.setCondition(db, dbv1.ConditionTypeReady, metav1.ConditionTrue, "Ready", "Database is ready")
	}

	// Update status
	if err := r.Status().Update(ctx, db); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled Database", "namespace", db.Namespace, "name", db.Name, "phase", db.Status.Phase)

	// Requeue after some time to check for maintenance windows, backups, etc.
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// handleDeletion handles the deletion of a Database resource
func (r *DatabaseReconciler) handleDeletion(ctx context.Context, db *dbv1.Database) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling deletion", "namespace", db.Namespace, "name", db.Name, "policy", db.Spec.Lifecycle.DeletionPolicy)

	if controllerutil.ContainsFinalizer(db, databaseFinalizer) {
		// Update phase
		r.updatePhase(ctx, db, dbv1.PhaseDeleting)
		if err := r.Status().Update(ctx, db); err != nil {
			log.Error(err, "Failed to update status during deletion")
		}

		// Handle deletion policy
		switch db.Spec.Lifecycle.DeletionPolicy {
		case dbv1.DeletionPolicySnapshot:
			log.Info("Taking snapshot before deletion")
			engine, err := r.EngineFactory.GetEngine(db)
			if err != nil {
				log.Error(err, "Failed to get engine for snapshot")
			} else {
				if _, err := engine.Backup(ctx, db, r.Client); err != nil {
					log.Error(err, "Failed to create snapshot before deletion")
					// Continue with deletion even if snapshot fails
				}
			}
		case dbv1.DeletionPolicyRetain:
			log.Info("Retaining database resources")
			// Remove finalizer but keep resources
			controllerutil.RemoveFinalizer(db, databaseFinalizer)
			return ctrl.Result{}, r.Update(ctx, db)
		case dbv1.DeletionPolicyDelete:
			log.Info("Deleting database resources")
			// Resources will be deleted automatically due to owner references
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(db, databaseFinalizer)
		if err := r.Update(ctx, db); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// validateSpec validates the database specification
func (r *DatabaseReconciler) validateSpec(ctx context.Context, db *dbv1.Database) error {
	// Validate SQLite cannot have replicas > 1
	if db.Spec.Engine == dbv1.EngineSQLite && db.Spec.Topology.Replicas > 1 {
		return fmt.Errorf("SQLite does not support multiple replicas")
	}

	// Validate Elasticsearch cannot run in single mode
	if db.Spec.Engine == dbv1.EngineElasticsearch && db.Spec.Topology.Mode == dbv1.TopologyStandalone {
		return fmt.Errorf("Elasticsearch requires at least 3 nodes for production use")
	}

	// Validate version downgrade prevention
	if db.Status.CurrentVersion != "" {
		if err := r.validateVersionUpgrade(db.Status.CurrentVersion, db.Spec.Version); err != nil {
			return err
		}
	}

	// Validate topology changes
	if db.Status.ObservedGeneration > 0 {
		// TODO: Add logic to prevent incompatible topology changes
	}

	return nil
}

// validateVersionUpgrade validates version upgrade
func (r *DatabaseReconciler) validateVersionUpgrade(currentVersion, desiredVersion string) error {
	// TODO: Implement version comparison logic
	// For now, allow all version changes
	return nil
}

// checkMaintenanceWindow checks if we're in a maintenance window
func (r *DatabaseReconciler) checkMaintenanceWindow(db *dbv1.Database) error {
	if len(db.Spec.Maintenance.Windows) == 0 {
		return nil // No maintenance window restrictions
	}

	now := time.Now()
	for _, window := range db.Spec.Maintenance.Windows {
		if int(now.Weekday()) == window.DayOfWeek {
			// Parse start time
			startTime, err := time.Parse("15:04", window.StartTime)
			if err != nil {
				continue
			}

			// Create today's window
			windowStart := time.Date(now.Year(), now.Month(), now.Day(),
				startTime.Hour(), startTime.Minute(), 0, 0, now.Location())
			windowEnd := windowStart.Add(window.Duration.Duration)

			if now.After(windowStart) && now.Before(windowEnd) {
				return nil // We're in a maintenance window
			}
		}
	}

	return fmt.Errorf("not in maintenance window")
}

// updateWorkloadStatus updates the status with workload information
func (r *DatabaseReconciler) updateWorkloadStatus(ctx context.Context, db *dbv1.Database) error {
	// Try StatefulSet first
	sts := &appsv1.StatefulSet{}
	err := r.Get(ctx, client.ObjectKey{Namespace: db.Namespace, Name: db.Name}, sts)
	if err == nil {
		db.Status.ReadyReplicas = sts.Status.ReadyReplicas
		return nil
	}

	// Try Deployment
	deploy := &appsv1.Deployment{}
	err = r.Get(ctx, client.ObjectKey{Namespace: db.Namespace, Name: db.Name}, deploy)
	if err == nil {
		db.Status.ReadyReplicas = deploy.Status.ReadyReplicas
		return nil
	}

	return err
}

// ensureBackupCronJob ensures a CronJob exists for backups
func (r *DatabaseReconciler) ensureBackupCronJob(ctx context.Context, db *dbv1.Database) error {
	// TODO: Implement backup CronJob creation
	// - Create CronJob with configured schedule
	// - Use engine-specific backup commands
	// - Store backups to configured destination
	return nil
}

// handleCredentialRotation handles credential rotation
func (r *DatabaseReconciler) handleCredentialRotation(ctx context.Context, db *dbv1.Database, engine engines.Engine) error {
	log := log.FromContext(ctx)

	// Check if rotation is due
	if db.Status.RotationStatus != nil && db.Status.RotationStatus.NextRotation != nil {
		if time.Now().Before(db.Status.RotationStatus.NextRotation.Time) {
			return nil // Not time for rotation yet
		}
	}

	log.Info("Initiating credential rotation")
	if err := engine.RotateAuth(ctx, db, r.Client); err != nil {
		return err
	}

	// Update rotation status
	now := metav1.Now()
	if db.Status.RotationStatus == nil {
		db.Status.RotationStatus = &dbv1.RotationStatus{}
	}
	db.Status.RotationStatus.LastRotation = &now

	// TODO: Calculate next rotation based on schedule
	// nextRotation := calculateNextRotation(db.Spec.Auth.RotationPolicy.Schedule)
	// db.Status.RotationStatus.NextRotation = &nextRotation

	return nil
}

// updatePhase updates the database phase
func (r *DatabaseReconciler) updatePhase(ctx context.Context, db *dbv1.Database, phase dbv1.DatabasePhase) {
	db.Status.Phase = phase
}

// setCondition sets a condition in the database status
func (r *DatabaseReconciler) setCondition(db *dbv1.Database, conditionType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: db.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Find existing condition
	for i, c := range db.Status.Conditions {
		if c.Type == conditionType {
			// Update if status changed
			if c.Status != status {
				db.Status.Conditions[i] = condition
			}
			return
		}
	}

	// Add new condition
	db.Status.Conditions = append(db.Status.Conditions, condition)
}

// SetupWithManager sets up the controller with the Manager
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dbv1.Database{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
