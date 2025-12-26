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

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dbv1 "github.com/ivikasavnish/database-crd/api/v1"
)

const (
	// RotationPhaseIdle indicates no rotation is in progress
	RotationPhaseIdle = "Idle"
	// RotationPhaseCreatingNew indicates new credentials are being created
	RotationPhaseCreatingNew = "CreatingNew"
	// RotationPhaseCutover indicates cutover to new credentials
	RotationPhaseCutover = "Cutover"
	// RotationPhaseRevoking indicates old credentials are being revoked
	RotationPhaseRevoking = "Revoking"
	// RotationPhaseComplete indicates rotation is complete
	RotationPhaseComplete = "Complete"
)

// RotationManager handles credential rotation
type RotationManager struct {
	Client        client.Client
	ConsulEnabled bool
}

// NewRotationManager creates a new rotation manager
func NewRotationManager(client client.Client) *RotationManager {
	return &RotationManager{
		Client: client,
	}
}

// RotateCredentials performs two-phase credential rotation
// Phase 1: Create new credentials
// Phase 2: Cutover to new credentials
// Phase 3: Revoke old credentials
func (rm *RotationManager) RotateCredentials(ctx context.Context, db *dbv1.Database) error {
	// Check current rotation phase
	phase := RotationPhaseIdle
	if db.Status.RotationStatus != nil {
		phase = db.Status.RotationStatus.Phase
	}

	switch phase {
	case RotationPhaseIdle, "":
		return rm.startRotation(ctx, db)
	case RotationPhaseCreatingNew:
		return rm.checkCreationJob(ctx, db)
	case RotationPhaseCutover:
		return rm.performCutover(ctx, db)
	case RotationPhaseRevoking:
		return rm.revokeOldCredentials(ctx, db)
	case RotationPhaseComplete:
		return rm.completeRotation(ctx, db)
	default:
		return fmt.Errorf("unknown rotation phase: %s", phase)
	}
}

// startRotation starts the credential rotation process
func (rm *RotationManager) startRotation(ctx context.Context, db *dbv1.Database) error {
	// Initialize rotation status
	if db.Status.RotationStatus == nil {
		db.Status.RotationStatus = &dbv1.RotationStatus{}
	}
	db.Status.RotationStatus.Phase = RotationPhaseCreatingNew

	// Generate new credentials
	newPassword, err := generateSecurePassword(32)
	if err != nil {
		return fmt.Errorf("failed to generate new password: %w", err)
	}

	// Create secret for new credentials
	newSecretName := fmt.Sprintf("%s-credentials-new", db.Name)
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newSecretName,
			Namespace: db.Namespace,
			Labels: map[string]string{
				"app":      "database",
				"instance": db.Name,
				"rotation": "new",
			},
		},
		StringData: map[string]string{
			"password": newPassword,
			"username": "postgres", // TODO: Make engine-specific
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(db, newSecret, rm.Client.Scheme()); err != nil {
		return err
	}

	// Create the secret
	if err := rm.Client.Create(ctx, newSecret); err != nil {
		return fmt.Errorf("failed to create new credentials secret: %w", err)
	}

	// If Consul is enabled, sync to Consul
	if db.Spec.Auth.Consul != nil && db.Spec.Auth.Consul.Enabled {
		if err := rm.syncToConsul(ctx, db, newPassword); err != nil {
			return fmt.Errorf("failed to sync credentials to Consul: %w", err)
		}
	}

	// Create Job to update database with new credentials
	job := rm.createRotationJob(db, RotationPhaseCreatingNew)
	if err := controllerutil.SetControllerReference(db, job, rm.Client.Scheme()); err != nil {
		return err
	}

	if err := rm.Client.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create rotation job: %w", err)
	}

	db.Status.RotationStatus.JobName = job.Name

	return nil
}

// checkCreationJob checks if the credential creation job completed
func (rm *RotationManager) checkCreationJob(ctx context.Context, db *dbv1.Database) error {
	if db.Status.RotationStatus.JobName == "" {
		return fmt.Errorf("no job name in rotation status")
	}

	job := &batchv1.Job{}
	if err := rm.Client.Get(ctx, client.ObjectKey{
		Namespace: db.Namespace,
		Name:      db.Status.RotationStatus.JobName,
	}, job); err != nil {
		return err
	}

	// Check if job completed successfully
	if job.Status.Succeeded > 0 {
		db.Status.RotationStatus.Phase = RotationPhaseCutover
		return nil
	}

	// Check if job failed
	if job.Status.Failed > 0 {
		return fmt.Errorf("rotation job failed")
	}

	// Job still running
	return nil
}

// performCutover performs the cutover to new credentials
func (rm *RotationManager) performCutover(ctx context.Context, db *dbv1.Database) error {
	// Get the new credentials secret
	newSecretName := fmt.Sprintf("%s-credentials-new", db.Name)
	newSecret := &corev1.Secret{}
	if err := rm.Client.Get(ctx, client.ObjectKey{
		Namespace: db.Namespace,
		Name:      newSecretName,
	}, newSecret); err != nil {
		return fmt.Errorf("failed to get new credentials: %w", err)
	}

	// Backup old credentials
	oldSecretName := fmt.Sprintf("%s-credentials", db.Name)
	oldSecret := &corev1.Secret{}
	if err := rm.Client.Get(ctx, client.ObjectKey{
		Namespace: db.Namespace,
		Name:      oldSecretName,
	}, oldSecret); err == nil {
		// Rename old secret
		backupSecretName := fmt.Sprintf("%s-credentials-old", db.Name)
		backupSecret := oldSecret.DeepCopy()
		backupSecret.Name = backupSecretName
		backupSecret.ResourceVersion = ""
		backupSecret.UID = ""
		backupSecret.Labels["rotation"] = "old"

		if err := rm.Client.Create(ctx, backupSecret); err != nil {
			return fmt.Errorf("failed to backup old credentials: %w", err)
		}

		// Delete old secret
		if err := rm.Client.Delete(ctx, oldSecret); err != nil {
			return fmt.Errorf("failed to delete old credentials: %w", err)
		}
	}

	// Rename new secret to primary
	primarySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oldSecretName,
			Namespace: db.Namespace,
			Labels: map[string]string{
				"app":      "database",
				"instance": db.Name,
				"rotation": "current",
			},
		},
		Data: newSecret.Data,
	}

	if err := controllerutil.SetControllerReference(db, primarySecret, rm.Client.Scheme()); err != nil {
		return err
	}

	if err := rm.Client.Create(ctx, primarySecret); err != nil {
		return fmt.Errorf("failed to create primary credentials: %w", err)
	}

	// Delete new secret
	if err := rm.Client.Delete(ctx, newSecret); err != nil {
		return fmt.Errorf("failed to delete new credentials secret: %w", err)
	}

	// Move to revoking phase
	db.Status.RotationStatus.Phase = RotationPhaseRevoking

	// Create job to revoke old credentials
	job := rm.createRotationJob(db, RotationPhaseRevoking)
	if err := controllerutil.SetControllerReference(db, job, rm.Client.Scheme()); err != nil {
		return err
	}

	if err := rm.Client.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create revocation job: %w", err)
	}

	db.Status.RotationStatus.JobName = job.Name

	return nil
}

// revokeOldCredentials revokes the old credentials
func (rm *RotationManager) revokeOldCredentials(ctx context.Context, db *dbv1.Database) error {
	if db.Status.RotationStatus.JobName == "" {
		return fmt.Errorf("no job name in rotation status")
	}

	job := &batchv1.Job{}
	if err := rm.Client.Get(ctx, client.ObjectKey{
		Namespace: db.Namespace,
		Name:      db.Status.RotationStatus.JobName,
	}, job); err != nil {
		return err
	}

	// Check if job completed successfully
	if job.Status.Succeeded > 0 {
		// Delete old credentials backup
		backupSecretName := fmt.Sprintf("%s-credentials-old", db.Name)
		backupSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backupSecretName,
				Namespace: db.Namespace,
			},
		}
		if err := rm.Client.Delete(ctx, backupSecret); err != nil {
			// Log error but continue
		}

		db.Status.RotationStatus.Phase = RotationPhaseComplete
		return nil
	}

	// Check if job failed
	if job.Status.Failed > 0 {
		return fmt.Errorf("revocation job failed")
	}

	// Job still running
	return nil
}

// completeRotation completes the rotation process
func (rm *RotationManager) completeRotation(ctx context.Context, db *dbv1.Database) error {
	now := metav1.Now()
	db.Status.RotationStatus.Phase = RotationPhaseIdle
	db.Status.RotationStatus.LastRotation = &now
	db.Status.RotationStatus.JobName = ""

	return nil
}

// createRotationJob creates a Kubernetes Job for credential rotation
func (rm *RotationManager) createRotationJob(db *dbv1.Database, phase string) *batchv1.Job {
	jobName := fmt.Sprintf("%s-rotation-%s-%d", db.Name, phase, metav1.Now().Unix())

	// Build the command based on engine and phase
	command := rm.buildRotationCommand(db, phase)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: db.Namespace,
			Labels: map[string]string{
				"app":      "database",
				"instance": db.Name,
				"rotation": phase,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":      "database",
						"instance": db.Name,
						"rotation": phase,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "rotation",
							Image:   rm.getRotationImage(db),
							Command: command,
							Env:     rm.buildRotationEnv(db, phase),
						},
					},
				},
			},
			BackoffLimit: int32Ptr(3),
		},
	}

	return job
}

// buildRotationCommand builds the command for the rotation job
func (rm *RotationManager) buildRotationCommand(db *dbv1.Database, phase string) []string {
	// TODO: Make engine-specific
	switch db.Spec.Engine {
	case dbv1.EnginePostgreSQL:
		if phase == RotationPhaseCreatingNew {
			return []string{
				"/bin/sh",
				"-c",
				"psql -h $DB_HOST -U $OLD_USERNAME -c \"CREATE USER $NEW_USERNAME WITH PASSWORD '$NEW_PASSWORD'; GRANT ALL PRIVILEGES ON DATABASE postgres TO $NEW_USERNAME;\"",
			}
		} else if phase == RotationPhaseRevoking {
			return []string{
				"/bin/sh",
				"-c",
				"psql -h $DB_HOST -U $NEW_USERNAME -c \"REVOKE ALL PRIVILEGES ON DATABASE postgres FROM $OLD_USERNAME; DROP USER $OLD_USERNAME;\"",
			}
		}
	}

	return []string{"/bin/sh", "-c", "echo 'Rotation not implemented'"}
}

// buildRotationEnv builds environment variables for the rotation job
func (rm *RotationManager) buildRotationEnv(db *dbv1.Database, phase string) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "DB_HOST",
			Value: db.Status.Endpoint,
		},
	}

	if phase == RotationPhaseCreatingNew {
		env = append(env,
			corev1.EnvVar{
				Name: "OLD_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: fmt.Sprintf("%s-credentials", db.Name),
						},
						Key: "username",
					},
				},
			},
			corev1.EnvVar{
				Name: "NEW_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: fmt.Sprintf("%s-credentials-new", db.Name),
						},
						Key: "username",
					},
				},
			},
			corev1.EnvVar{
				Name: "NEW_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: fmt.Sprintf("%s-credentials-new", db.Name),
						},
						Key: "password",
					},
				},
			},
		)
	} else if phase == RotationPhaseRevoking {
		env = append(env,
			corev1.EnvVar{
				Name: "OLD_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: fmt.Sprintf("%s-credentials-old", db.Name),
						},
						Key: "username",
					},
				},
			},
			corev1.EnvVar{
				Name: "NEW_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: fmt.Sprintf("%s-credentials", db.Name),
						},
						Key: "username",
					},
				},
			},
		)
	}

	return env
}

// getRotationImage returns the image to use for rotation jobs
func (rm *RotationManager) getRotationImage(db *dbv1.Database) string {
	// TODO: Make engine-specific
	switch db.Spec.Engine {
	case dbv1.EnginePostgreSQL:
		return fmt.Sprintf("postgres:%s", db.Spec.Version)
	case dbv1.EngineMongoDB:
		return fmt.Sprintf("mongo:%s", db.Spec.Version)
	case dbv1.EngineRedis:
		return fmt.Sprintf("redis:%s", db.Spec.Version)
	default:
		return "busybox:latest"
	}
}

// syncToConsul syncs credentials to Consul KV store
func (rm *RotationManager) syncToConsul(ctx context.Context, db *dbv1.Database, password string) error {
	// TODO: Implement Consul integration
	// - Connect to Consul using address and token from spec
	// - Store credentials at the configured path
	// - Use Consul SDK or API calls

	return fmt.Errorf("Consul integration not yet implemented")
}

// generateSecurePassword generates a secure random password
func generateSecurePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// int32Ptr returns a pointer to an int32
func int32Ptr(i int32) *int32 {
	return &i
}
