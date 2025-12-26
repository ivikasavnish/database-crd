/*
Copyright 2025 Vikas Avnish.

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

package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasesv1alpha1 "github.com/ivikasavnish/database-crd/api/v1alpha1"
)

const (
	databaseFinalizer = "databases.database-operator.io/finalizer"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=databases.database-operator.io,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=databases.database-operator.io,resources=databases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=databases.database-operator.io,resources=databases/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Database instance
	database := &databasesv1alpha1.Database{}
	err := r.Get(ctx, req.NamespacedName, database)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Database resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Database")
		return ctrl.Result{}, err
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(database, databaseFinalizer) {
		controllerutil.AddFinalizer(database, databaseFinalizer)
		if err := r.Update(ctx, database); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check if the Database is marked to be deleted
	if !database.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(database, databaseFinalizer) {
			// Perform cleanup
			r.finalizeDatabase(ctx, database)

			// Remove finalizer
			controllerutil.RemoveFinalizer(database, databaseFinalizer)
			if err := r.Update(ctx, database); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Update status phase to Creating if it's empty
	if database.Status.Phase == "" {
		database.Status.Phase = databasesv1alpha1.DatabasePhaseCreating
		database.Status.ObservedGeneration = database.Generation
		if err := r.Status().Update(ctx, database); err != nil {
			log.Error(err, "Failed to update Database status")
			return ctrl.Result{}, err
		}
	}

	// Reconcile the database based on its type
	if err := r.reconcileDatabase(ctx, database); err != nil {
		log.Error(err, "Failed to reconcile database")
		r.updateStatusOnError(ctx, database, err)
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Update status to Ready
	if database.Status.Phase != databasesv1alpha1.DatabasePhaseReady {
		database.Status.Phase = databasesv1alpha1.DatabasePhaseReady
		database.Status.ObservedGeneration = database.Generation
		database.Status.Message = "Database is ready"

		condition := metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "DatabaseReady",
			Message:            "Database is successfully provisioned and ready",
			LastTransitionTime: metav1.NewTime(time.Now()),
			ObservedGeneration: database.Generation,
		}
		meta.SetStatusCondition(&database.Status.Conditions, condition)

		if err := r.Status().Update(ctx, database); err != nil {
			log.Error(err, "Failed to update Database status to Ready")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *DatabaseReconciler) reconcileDatabase(ctx context.Context, database *databasesv1alpha1.Database) error {
	log := log.FromContext(ctx)

	// Reconcile Service
	if err := r.reconcileService(ctx, database); err != nil {
		log.Error(err, "Failed to reconcile Service")
		return err
	}

	// Reconcile StatefulSet or Deployment based on database type
	switch database.Spec.Type {
	case databasesv1alpha1.DatabaseTypePostgreSQL:
		return r.reconcilePostgreSQL(ctx, database)
	case databasesv1alpha1.DatabaseTypeMongoDB:
		return r.reconcileMongoDB(ctx, database)
	case databasesv1alpha1.DatabaseTypeRedis:
		return r.reconcileRedis(ctx, database)
	case databasesv1alpha1.DatabaseTypeElasticsearch:
		return r.reconcileElasticsearch(ctx, database)
	case databasesv1alpha1.DatabaseTypeSQLite:
		return r.reconcileSQLite(ctx, database)
	default:
		return fmt.Errorf("unsupported database type: %s", database.Spec.Type)
	}
}

func (r *DatabaseReconciler) reconcileService(ctx context.Context, database *databasesv1alpha1.Database) error {
	service := &corev1.Service{}
	serviceName := database.Name + "-service"
	err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: database.Namespace}, service)

	ports := r.getServicePorts(database)

	if err != nil && errors.IsNotFound(err) {
		// Create the service
		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: database.Namespace,
				Labels:    r.getLabels(database),
			},
			Spec: corev1.ServiceSpec{
				Selector: r.getLabels(database),
				Ports:    ports,
				Type:     corev1.ServiceTypeClusterIP,
			},
		}

		if err := controllerutil.SetControllerReference(database, service, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, service); err != nil {
			return err
		}

		database.Status.ServiceName = serviceName
		database.Status.ConnectionString = r.getConnectionString(database, serviceName)
	}

	return nil
}

func (r *DatabaseReconciler) reconcilePostgreSQL(ctx context.Context, database *databasesv1alpha1.Database) error {
	statefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: database.Name, Namespace: database.Namespace}, statefulSet)

	replicas := int32(1)
	if database.Spec.Replicas != nil {
		replicas = *database.Spec.Replicas
	}

	env := r.getPostgreSQLEnv(database)

	if err != nil && errors.IsNotFound(err) {
		// Create StatefulSet
		statefulSet = r.createPostgreSQLStatefulSet(database, replicas, env)

		if err := controllerutil.SetControllerReference(database, statefulSet, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, statefulSet); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Update status
	database.Status.ReadyReplicas = statefulSet.Status.ReadyReplicas

	return nil
}

func (r *DatabaseReconciler) reconcileMongoDB(ctx context.Context, database *databasesv1alpha1.Database) error {
	statefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: database.Name, Namespace: database.Namespace}, statefulSet)

	replicas := int32(1)
	if database.Spec.Replicas != nil {
		replicas = *database.Spec.Replicas
	}

	env := r.getMongoDBEnv(database)

	if err != nil && errors.IsNotFound(err) {
		statefulSet = r.createMongoDBStatefulSet(database, replicas, env)

		if err := controllerutil.SetControllerReference(database, statefulSet, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, statefulSet); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	database.Status.ReadyReplicas = statefulSet.Status.ReadyReplicas
	return nil
}

func (r *DatabaseReconciler) reconcileRedis(ctx context.Context, database *databasesv1alpha1.Database) error {
	statefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: database.Name, Namespace: database.Namespace}, statefulSet)

	replicas := int32(1)
	if database.Spec.Replicas != nil {
		replicas = *database.Spec.Replicas
	}

	env := r.getRedisEnv(database)

	if err != nil && errors.IsNotFound(err) {
		statefulSet = r.createRedisStatefulSet(database, replicas, env)

		if err := controllerutil.SetControllerReference(database, statefulSet, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, statefulSet); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	database.Status.ReadyReplicas = statefulSet.Status.ReadyReplicas
	return nil
}

func (r *DatabaseReconciler) reconcileElasticsearch(ctx context.Context, database *databasesv1alpha1.Database) error {
	statefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: database.Name, Namespace: database.Namespace}, statefulSet)

	replicas := int32(1)
	if database.Spec.Replicas != nil {
		replicas = *database.Spec.Replicas
	}

	env := r.getElasticsearchEnv(database)

	if err != nil && errors.IsNotFound(err) {
		statefulSet = r.createElasticsearchStatefulSet(database, replicas, env)

		if err := controllerutil.SetControllerReference(database, statefulSet, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, statefulSet); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	database.Status.ReadyReplicas = statefulSet.Status.ReadyReplicas
	return nil
}

func (r *DatabaseReconciler) reconcileSQLite(ctx context.Context, database *databasesv1alpha1.Database) error {
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: database.Name, Namespace: database.Namespace}, deployment)

	replicas := int32(1)
	env := r.getSQLiteEnv(database)

	if err != nil && errors.IsNotFound(err) {
		deployment = r.createSQLiteDeployment(database, replicas, env)

		if err := controllerutil.SetControllerReference(database, deployment, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, deployment); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	database.Status.ReadyReplicas = deployment.Status.ReadyReplicas
	return nil
}

func (r *DatabaseReconciler) getLabels(database *databasesv1alpha1.Database) map[string]string {
	return map[string]string{
		"app":                          database.Name,
		"database-type":                string(database.Spec.Type),
		"app.kubernetes.io/name":       "database",
		"app.kubernetes.io/instance":   database.Name,
		"app.kubernetes.io/managed-by": "database-operator",
	}
}

func (r *DatabaseReconciler) getServicePorts(database *databasesv1alpha1.Database) []corev1.ServicePort {
	var port int32

	switch database.Spec.Type {
	case databasesv1alpha1.DatabaseTypePostgreSQL:
		port = 5432
	case databasesv1alpha1.DatabaseTypeMongoDB:
		port = 27017
	case databasesv1alpha1.DatabaseTypeRedis:
		port = 6379
	case databasesv1alpha1.DatabaseTypeElasticsearch:
		port = 9200
	case databasesv1alpha1.DatabaseTypeSQLite:
		port = 8080
	default:
		port = 8080
	}

	return []corev1.ServicePort{
		{
			Name:       "database",
			Port:       port,
			TargetPort: intstr.FromInt(int(port)),
			Protocol:   corev1.ProtocolTCP,
		},
	}
}

func (r *DatabaseReconciler) getConnectionString(database *databasesv1alpha1.Database, serviceName string) string {
	switch database.Spec.Type {
	case databasesv1alpha1.DatabaseTypePostgreSQL:
		dbName := "postgres"
		if database.Spec.PostgreSQL != nil && database.Spec.PostgreSQL.Database != "" {
			dbName = database.Spec.PostgreSQL.Database
		}
		return fmt.Sprintf("postgresql://<username>:<password>@%s.%s.svc.cluster.local:5432/%s",
			serviceName, database.Namespace, dbName)
	case databasesv1alpha1.DatabaseTypeMongoDB:
		dbName := "admin"
		if database.Spec.MongoDB != nil && database.Spec.MongoDB.Database != "" {
			dbName = database.Spec.MongoDB.Database
		}
		return fmt.Sprintf("mongodb://<username>:<password>@%s.%s.svc.cluster.local:27017/%s",
			serviceName, database.Namespace, dbName)
	case databasesv1alpha1.DatabaseTypeRedis:
		return fmt.Sprintf("redis://:%s@%s.%s.svc.cluster.local:6379",
			"<password>", serviceName, database.Namespace)
	case databasesv1alpha1.DatabaseTypeElasticsearch:
		return fmt.Sprintf("http://%s.%s.svc.cluster.local:9200",
			serviceName, database.Namespace)
	case databasesv1alpha1.DatabaseTypeSQLite:
		return fmt.Sprintf("http://%s.%s.svc.cluster.local:8080",
			serviceName, database.Namespace)
	default:
		return ""
	}
}

func (r *DatabaseReconciler) getPostgreSQLEnv(database *databasesv1alpha1.Database) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "POSTGRES_DB",
			Value: "postgres",
		},
		{
			Name:  "POSTGRES_USER",
			Value: "postgres",
		},
		{
			Name:  "POSTGRES_PASSWORD",
			Value: "postgres",
		},
	}

	if database.Spec.PostgreSQL != nil {
		if database.Spec.PostgreSQL.Database != "" {
			env[0].Value = database.Spec.PostgreSQL.Database
		}
		if database.Spec.PostgreSQL.Username != "" {
			env[1].Value = database.Spec.PostgreSQL.Username
		}
		if database.Spec.PostgreSQL.PasswordSecret != nil {
			env[2].ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: database.Spec.PostgreSQL.PasswordSecret.Name,
					},
					Key: database.Spec.PostgreSQL.PasswordSecret.Key,
				},
			}
			env[2].Value = ""
		}
	}

	env = append(env, r.convertEnvVars(database.Spec.Env)...)
	return env
}

func (r *DatabaseReconciler) getMongoDBEnv(database *databasesv1alpha1.Database) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "MONGO_INITDB_ROOT_USERNAME",
			Value: "root",
		},
		{
			Name:  "MONGO_INITDB_ROOT_PASSWORD",
			Value: "password",
		},
	}

	if database.Spec.MongoDB != nil {
		if database.Spec.MongoDB.Username != "" {
			env[0].Value = database.Spec.MongoDB.Username
		}
		if database.Spec.MongoDB.PasswordSecret != nil {
			env[1].ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: database.Spec.MongoDB.PasswordSecret.Name,
					},
					Key: database.Spec.MongoDB.PasswordSecret.Key,
				},
			}
			env[1].Value = ""
		}
	}

	env = append(env, r.convertEnvVars(database.Spec.Env)...)
	return env
}

func (r *DatabaseReconciler) getRedisEnv(database *databasesv1alpha1.Database) []corev1.EnvVar {
	env := []corev1.EnvVar{}

	if database.Spec.Redis != nil && database.Spec.Redis.PasswordSecret != nil {
		env = append(env, corev1.EnvVar{
			Name: "REDIS_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: database.Spec.Redis.PasswordSecret.Name,
					},
					Key: database.Spec.Redis.PasswordSecret.Key,
				},
			},
		})
	}

	env = append(env, r.convertEnvVars(database.Spec.Env)...)
	return env
}

func (r *DatabaseReconciler) getElasticsearchEnv(database *databasesv1alpha1.Database) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "discovery.type",
			Value: "single-node",
		},
		{
			Name:  "xpack.security.enabled",
			Value: "false",
		},
	}

	if database.Spec.Elasticsearch != nil && database.Spec.Elasticsearch.ClusterName != "" {
		env = append(env, corev1.EnvVar{
			Name:  "cluster.name",
			Value: database.Spec.Elasticsearch.ClusterName,
		})
	}

	env = append(env, r.convertEnvVars(database.Spec.Env)...)
	return env
}

func (r *DatabaseReconciler) getSQLiteEnv(database *databasesv1alpha1.Database) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "SQLITE_DATABASE",
			Value: "/data/database.db",
		},
	}

	if database.Spec.SQLite != nil && database.Spec.SQLite.DatabaseFile != "" {
		env[0].Value = database.Spec.SQLite.DatabaseFile
	}

	env = append(env, r.convertEnvVars(database.Spec.Env)...)
	return env
}

func (r *DatabaseReconciler) convertEnvVars(envVars []databasesv1alpha1.EnvVar) []corev1.EnvVar {
	result := make([]corev1.EnvVar, len(envVars))
	for i, ev := range envVars {
		result[i] = corev1.EnvVar{
			Name:  ev.Name,
			Value: ev.Value,
		}
		if ev.ValueFrom != nil && ev.ValueFrom.SecretKeyRef != nil {
			result[i].ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ev.ValueFrom.SecretKeyRef.Name,
					},
					Key: ev.ValueFrom.SecretKeyRef.Key,
				},
			}
			result[i].Value = ""
		}
	}
	return result
}

func (r *DatabaseReconciler) createPostgreSQLStatefulSet(database *databasesv1alpha1.Database, replicas int32, env []corev1.EnvVar) *appsv1.StatefulSet {
	labels := r.getLabels(database)

	volumeClaimTemplates := []corev1.PersistentVolumeClaim{}
	if database.Spec.Storage != nil {
		volumeClaimTemplates = append(volumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "data",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(database.Spec.Storage.Size),
					},
				},
				StorageClassName: database.Spec.Storage.StorageClass,
			},
		})
	}

	container := corev1.Container{
		Name:  "postgresql",
		Image: fmt.Sprintf("postgres:%s", database.Spec.Version),
		Ports: []corev1.ContainerPort{
			{
				Name:          "postgresql",
				ContainerPort: 5432,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: env,
	}

	if database.Spec.Storage != nil {
		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/var/lib/postgresql/data",
			},
		}
	}

	if database.Spec.Resources != nil {
		container.Resources = r.buildResourceRequirements(database.Spec.Resources)
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.Name,
			Namespace: database.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: database.Name + "-service",
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
			VolumeClaimTemplates: volumeClaimTemplates,
		},
	}
}

func (r *DatabaseReconciler) createMongoDBStatefulSet(database *databasesv1alpha1.Database, replicas int32, env []corev1.EnvVar) *appsv1.StatefulSet {
	labels := r.getLabels(database)

	volumeClaimTemplates := []corev1.PersistentVolumeClaim{}
	if database.Spec.Storage != nil {
		volumeClaimTemplates = append(volumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "data",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(database.Spec.Storage.Size),
					},
				},
				StorageClassName: database.Spec.Storage.StorageClass,
			},
		})
	}

	container := corev1.Container{
		Name:  "mongodb",
		Image: fmt.Sprintf("mongo:%s", database.Spec.Version),
		Ports: []corev1.ContainerPort{
			{
				Name:          "mongodb",
				ContainerPort: 27017,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: env,
	}

	if database.Spec.Storage != nil {
		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/data/db",
			},
		}
	}

	if database.Spec.Resources != nil {
		container.Resources = r.buildResourceRequirements(database.Spec.Resources)
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.Name,
			Namespace: database.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: database.Name + "-service",
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
			VolumeClaimTemplates: volumeClaimTemplates,
		},
	}
}

func (r *DatabaseReconciler) createRedisStatefulSet(database *databasesv1alpha1.Database, replicas int32, env []corev1.EnvVar) *appsv1.StatefulSet {
	labels := r.getLabels(database)

	volumeClaimTemplates := []corev1.PersistentVolumeClaim{}
	if database.Spec.Storage != nil {
		volumeClaimTemplates = append(volumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "data",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(database.Spec.Storage.Size),
					},
				},
				StorageClassName: database.Spec.Storage.StorageClass,
			},
		})
	}

	container := corev1.Container{
		Name:  "redis",
		Image: fmt.Sprintf("redis:%s", database.Spec.Version),
		Ports: []corev1.ContainerPort{
			{
				Name:          "redis",
				ContainerPort: 6379,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: env,
	}

	if database.Spec.Storage != nil {
		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/data",
			},
		}
	}

	if database.Spec.Resources != nil {
		container.Resources = r.buildResourceRequirements(database.Spec.Resources)
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.Name,
			Namespace: database.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: database.Name + "-service",
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
			VolumeClaimTemplates: volumeClaimTemplates,
		},
	}
}

func (r *DatabaseReconciler) createElasticsearchStatefulSet(database *databasesv1alpha1.Database, replicas int32, env []corev1.EnvVar) *appsv1.StatefulSet {
	labels := r.getLabels(database)

	volumeClaimTemplates := []corev1.PersistentVolumeClaim{}
	if database.Spec.Storage != nil {
		volumeClaimTemplates = append(volumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "data",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(database.Spec.Storage.Size),
					},
				},
				StorageClassName: database.Spec.Storage.StorageClass,
			},
		})
	}

	container := corev1.Container{
		Name:  "elasticsearch",
		Image: fmt.Sprintf("docker.elastic.co/elasticsearch/elasticsearch:%s", database.Spec.Version),
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 9200,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "transport",
				ContainerPort: 9300,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: env,
	}

	if database.Spec.Storage != nil {
		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/usr/share/elasticsearch/data",
			},
		}
	}

	if database.Spec.Resources != nil {
		container.Resources = r.buildResourceRequirements(database.Spec.Resources)
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.Name,
			Namespace: database.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: database.Name + "-service",
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
			VolumeClaimTemplates: volumeClaimTemplates,
		},
	}
}

func (r *DatabaseReconciler) createSQLiteDeployment(database *databasesv1alpha1.Database, replicas int32, env []corev1.EnvVar) *appsv1.Deployment {
	labels := r.getLabels(database)

	// For SQLite, use the version specified by the user
	// This allows flexibility for testing with "latest" or pinning to a specific version
	image := fmt.Sprintf("nouchka/sqlite3:%s", database.Spec.Version)

	container := corev1.Container{
		Name:  "sqlite",
		Image: image,
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 8080,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: env,
	}

	if database.Spec.Storage != nil {
		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/data",
			},
		}
	}

	if database.Spec.Resources != nil {
		container.Resources = r.buildResourceRequirements(database.Spec.Resources)
	}

	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{container},
	}

	if database.Spec.Storage != nil {
		podSpec.Volumes = []corev1.Volume{
			{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: database.Name + "-data",
					},
				},
			},
		}
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.Name,
			Namespace: database.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: podSpec,
			},
		},
	}
}

func (r *DatabaseReconciler) buildResourceRequirements(resources *databasesv1alpha1.ResourceRequirements) corev1.ResourceRequirements {
	requirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	if resources.CPU != "" {
		requirements.Requests[corev1.ResourceCPU] = resource.MustParse(resources.CPU)
	}
	if resources.Memory != "" {
		requirements.Requests[corev1.ResourceMemory] = resource.MustParse(resources.Memory)
	}
	if resources.CPULimit != "" {
		requirements.Limits[corev1.ResourceCPU] = resource.MustParse(resources.CPULimit)
	}
	if resources.MemoryLimit != "" {
		requirements.Limits[corev1.ResourceMemory] = resource.MustParse(resources.MemoryLimit)
	}

	return requirements
}

func (r *DatabaseReconciler) finalizeDatabase(ctx context.Context, database *databasesv1alpha1.Database) {
	log := log.FromContext(ctx)
	log.Info("Finalizing database", "name", database.Name)
	// Perform cleanup if needed
	// Kubernetes garbage collection will automatically clean up owned resources
	// (StatefulSets, Deployments, Services) due to controller references
}

func (r *DatabaseReconciler) updateStatusOnError(ctx context.Context, database *databasesv1alpha1.Database, err error) {
	database.Status.Phase = databasesv1alpha1.DatabasePhaseFailed
	database.Status.Message = err.Error()

	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "ReconciliationFailed",
		Message:            err.Error(),
		LastTransitionTime: metav1.NewTime(time.Now()),
		ObservedGeneration: database.Generation,
	}
	meta.SetStatusCondition(&database.Status.Conditions, condition)

	_ = r.Status().Update(ctx, database)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1alpha1.Database{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Named("database").
		Complete(r)
}
