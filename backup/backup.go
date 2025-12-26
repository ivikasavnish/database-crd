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

package backup

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dbv1 "github.com/ivikasavnish/database-crd/api/v1"
)

// BackupManager handles database backups
type BackupManager struct {
	Client client.Client
}

// NewBackupManager creates a new backup manager
func NewBackupManager(client client.Client) *BackupManager {
	return &BackupManager{
		Client: client,
	}
}

// CreateBackupJob creates a Kubernetes Job to perform a backup
func (bm *BackupManager) CreateBackupJob(ctx context.Context, db *dbv1.Database) (*batchv1.Job, error) {
	jobName := fmt.Sprintf("%s-backup-%d", db.Name, metav1.Now().Unix())

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: db.Namespace,
			Labels: map[string]string{
				"app":      "database",
				"instance": db.Name,
				"job-type": "backup",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":      "database",
						"instance": db.Name,
						"job-type": "backup",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "backup",
							Image:   bm.getBackupImage(db),
							Command: bm.buildBackupCommand(db),
							Env:     bm.buildBackupEnv(db),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "backup",
									MountPath: "/backup",
								},
							},
						},
					},
					Volumes: bm.buildBackupVolumes(db),
				},
			},
			BackoffLimit: int32Ptr(3),
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(db, job, bm.Client.Scheme()); err != nil {
		return nil, err
	}

	if err := bm.Client.Create(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}

// CreateBackupCronJob creates a CronJob for scheduled backups
func (bm *BackupManager) CreateBackupCronJob(ctx context.Context, db *dbv1.Database) (*batchv1.CronJob, error) {
	cronJobName := fmt.Sprintf("%s-backup", db.Name)

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronJobName,
			Namespace: db.Namespace,
			Labels: map[string]string{
				"app":      "database",
				"instance": db.Name,
				"job-type": "backup",
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule: db.Spec.Backup.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":      "database",
						"instance": db.Name,
						"job-type": "backup",
					},
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app":      "database",
								"instance": db.Name,
								"job-type": "backup",
							},
						},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{
									Name:    "backup",
									Image:   bm.getBackupImage(db),
									Command: bm.buildBackupCommand(db),
									Env:     bm.buildBackupEnv(db),
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "backup",
											MountPath: "/backup",
										},
									},
								},
							},
							Volumes: bm.buildBackupVolumes(db),
						},
					},
					BackoffLimit: int32Ptr(3),
				},
			},
			SuccessfulJobsHistoryLimit: int32Ptr(int32(db.Spec.Backup.Retention)),
			FailedJobsHistoryLimit:     int32Ptr(3),
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(db, cronJob, bm.Client.Scheme()); err != nil {
		return nil, err
	}

	if err := bm.Client.Create(ctx, cronJob); err != nil {
		return nil, err
	}

	return cronJob, nil
}

// getBackupImage returns the image to use for backup jobs
func (bm *BackupManager) getBackupImage(db *dbv1.Database) string {
	switch db.Spec.Engine {
	case dbv1.EnginePostgreSQL:
		return fmt.Sprintf("postgres:%s", db.Spec.Version)
	case dbv1.EngineMongoDB:
		return fmt.Sprintf("mongo:%s", db.Spec.Version)
	case dbv1.EngineRedis:
		return fmt.Sprintf("redis:%s", db.Spec.Version)
	case dbv1.EngineElasticsearch:
		return fmt.Sprintf("elasticsearch:%s", db.Spec.Version)
	default:
		return "busybox:latest"
	}
}

// buildBackupCommand builds the backup command based on engine and method
func (bm *BackupManager) buildBackupCommand(db *dbv1.Database) []string {
	backupFile := fmt.Sprintf("/backup/%s-%s.backup", db.Name, "$(date +%%Y%%m%%d-%%H%%M%%S)")

	switch db.Spec.Engine {
	case dbv1.EnginePostgreSQL:
		switch db.Spec.Backup.Method {
		case dbv1.BackupMethodDump:
			return []string{
				"/bin/sh",
				"-c",
				fmt.Sprintf("pg_dump -h $DB_HOST -U $DB_USER -Fc -f %s", backupFile),
			}
		case dbv1.BackupMethodSnapshot:
			// For snapshot, use pg_basebackup
			return []string{
				"/bin/sh",
				"-c",
				fmt.Sprintf("pg_basebackup -h $DB_HOST -U $DB_USER -D %s -Ft -z", backupFile),
			}
		default:
			return []string{
				"/bin/sh",
				"-c",
				fmt.Sprintf("pg_dump -h $DB_HOST -U $DB_USER -Fc -f %s", backupFile),
			}
		}
	case dbv1.EngineMongoDB:
		return []string{
			"/bin/sh",
			"-c",
			fmt.Sprintf("mongodump --host=$DB_HOST --username=$DB_USER --password=$DB_PASSWORD --out=%s", backupFile),
		}
	case dbv1.EngineRedis:
		return []string{
			"/bin/sh",
			"-c",
			"redis-cli -h $DB_HOST --rdb /backup/dump.rdb",
		}
	default:
		return []string{"/bin/sh", "-c", "echo 'Backup not implemented for this engine'"}
	}
}

// buildBackupEnv builds environment variables for the backup job
func (bm *BackupManager) buildBackupEnv(db *dbv1.Database) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "DB_HOST",
			Value: db.Status.Endpoint,
		},
		{
			Name: "DB_USER",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-credentials", db.Name),
					},
					Key: "username",
				},
			},
		},
		{
			Name: "DB_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-credentials", db.Name),
					},
					Key: "password",
				},
			},
		},
	}

	// Add S3 credentials if configured
	if db.Spec.Backup.Destination.S3 != nil {
		env = append(env,
			corev1.EnvVar{
				Name: "AWS_ACCESS_KEY_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: db.Spec.Backup.Destination.S3.CredentialsSecret,
						},
						Key: "access_key_id",
					},
				},
			},
			corev1.EnvVar{
				Name: "AWS_SECRET_ACCESS_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: db.Spec.Backup.Destination.S3.CredentialsSecret,
						},
						Key: "secret_access_key",
					},
				},
			},
			corev1.EnvVar{
				Name:  "S3_BUCKET",
				Value: db.Spec.Backup.Destination.S3.Bucket,
			},
		)

		if db.Spec.Backup.Destination.S3.Region != "" {
			env = append(env, corev1.EnvVar{
				Name:  "AWS_REGION",
				Value: db.Spec.Backup.Destination.S3.Region,
			})
		}

		if db.Spec.Backup.Destination.S3.Endpoint != "" {
			env = append(env, corev1.EnvVar{
				Name:  "S3_ENDPOINT",
				Value: db.Spec.Backup.Destination.S3.Endpoint,
			})
		}
	}

	return env
}

// buildBackupVolumes builds volumes for the backup job
func (bm *BackupManager) buildBackupVolumes(db *dbv1.Database) []corev1.Volume {
	volumes := []corev1.Volume{}

	if db.Spec.Backup.Destination.PVC != nil {
		// Use PVC for backup storage
		volumes = append(volumes, corev1.Volume{
			Name: "backup",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf("%s-backup", db.Name),
				},
			},
		})
	} else {
		// Use emptyDir for temporary storage (will upload to S3)
		volumes = append(volumes, corev1.Volume{
			Name: "backup",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	return volumes
}

// CreateRestoreJob creates a Kubernetes Job to restore from a backup
func (bm *BackupManager) CreateRestoreJob(ctx context.Context, db *dbv1.Database) (*batchv1.Job, error) {
	jobName := fmt.Sprintf("%s-restore-%d", db.Name, metav1.Now().Unix())

	// TODO: Implement restore job similar to backup job
	// - Download backup from destination
	// - Restore database from backup
	// - Handle point-in-time recovery if specified

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: db.Namespace,
			Labels: map[string]string{
				"app":      "database",
				"instance": db.Name,
				"job-type": "restore",
			},
		},
	}

	return job, fmt.Errorf("restore not yet implemented")
}

// int32Ptr returns a pointer to an int32
func int32Ptr(i int32) *int32 {
	return &i
}
