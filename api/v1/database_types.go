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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatabaseEngine represents the type of database engine
// +kubebuilder:validation:Enum=PostgreSQL;MongoDB;Redis;Elasticsearch;SQLite
type DatabaseEngine string

const (
	// EnginePostgreSQL represents PostgreSQL database engine
	EnginePostgreSQL DatabaseEngine = "PostgreSQL"
	// EngineMongoDB represents MongoDB database engine
	EngineMongoDB DatabaseEngine = "MongoDB"
	// EngineRedis represents Redis database engine
	EngineRedis DatabaseEngine = "Redis"
	// EngineElasticsearch represents Elasticsearch database engine
	EngineElasticsearch DatabaseEngine = "Elasticsearch"
	// EngineSQLite represents SQLite database engine
	EngineSQLite DatabaseEngine = "SQLite"
)

// DatabasePhase represents the current phase of the database
// +kubebuilder:validation:Enum=Pending;Provisioning;Ready;Upgrading;Scaling;Healing;Failed;Paused;Deleting
type DatabasePhase string

const (
	// PhasePending indicates the database is pending provisioning
	PhasePending DatabasePhase = "Pending"
	// PhaseProvisioning indicates the database is being provisioned
	PhaseProvisioning DatabasePhase = "Provisioning"
	// PhaseReady indicates the database is ready
	PhaseReady DatabasePhase = "Ready"
	// PhaseUpgrading indicates the database is being upgraded
	PhaseUpgrading DatabasePhase = "Upgrading"
	// PhaseScaling indicates the database is being scaled
	PhaseScaling DatabasePhase = "Scaling"
	// PhaseHealing indicates the database is being healed
	PhaseHealing DatabasePhase = "Healing"
	// PhaseFailed indicates the database has failed
	PhaseFailed DatabasePhase = "Failed"
	// PhasePaused indicates reconciliation is paused
	PhasePaused DatabasePhase = "Paused"
	// PhaseDeleting indicates the database is being deleted
	PhaseDeleting DatabasePhase = "Deleting"
)

// DeletionPolicy defines how to handle database deletion
// +kubebuilder:validation:Enum=Retain;Snapshot;Delete
type DeletionPolicy string

const (
	// DeletionPolicyRetain retains the database data
	DeletionPolicyRetain DeletionPolicy = "Retain"
	// DeletionPolicySnapshot takes a snapshot before deletion
	DeletionPolicySnapshot DeletionPolicy = "Snapshot"
	// DeletionPolicyDelete deletes the database data
	DeletionPolicyDelete DeletionPolicy = "Delete"
)

// TopologyMode defines the deployment topology
// +kubebuilder:validation:Enum=Standalone;Replicated;Cluster;Sharded
type TopologyMode string

const (
	// TopologyStandalone is a single instance
	TopologyStandalone TopologyMode = "Standalone"
	// TopologyReplicated is a replicated setup
	TopologyReplicated TopologyMode = "Replicated"
	// TopologyCluster is a clustered setup
	TopologyCluster TopologyMode = "Cluster"
	// TopologySharded is a sharded setup
	TopologySharded TopologyMode = "Sharded"
)

// BackupMethod defines the backup method
// +kubebuilder:validation:Enum=Snapshot;Dump;WAL;Incremental
type BackupMethod string

const (
	// BackupMethodSnapshot uses volume snapshots
	BackupMethodSnapshot BackupMethod = "Snapshot"
	// BackupMethodDump uses logical dumps
	BackupMethodDump BackupMethod = "Dump"
	// BackupMethodWAL uses write-ahead logs
	BackupMethodWAL BackupMethod = "WAL"
	// BackupMethodIncremental uses incremental backups
	BackupMethodIncremental BackupMethod = "Incremental"
)

// DatabaseSpec defines the desired state of Database
type DatabaseSpec struct {
	// Engine specifies the database engine type
	// +kubebuilder:validation:Required
	Engine DatabaseEngine `json:"engine"`

	// Version specifies the database version
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[0-9]+\.[0-9]+(\.[0-9]+)?$`
	Version string `json:"version"`

	// Profile specifies the performance/resource profile (e.g., dev, prod, high-memory)
	// +optional
	// +kubebuilder:default="default"
	Profile string `json:"profile,omitempty"`

	// Topology defines the deployment topology
	// +optional
	Topology TopologySpec `json:"topology,omitempty"`

	// Storage defines storage configuration
	// +optional
	Storage StorageSpec `json:"storage,omitempty"`

	// Resources defines compute resources
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Networking defines network configuration
	// +optional
	Networking NetworkingSpec `json:"networking,omitempty"`

	// Backup defines backup configuration
	// +optional
	Backup BackupSpec `json:"backup,omitempty"`

	// Restore defines restore configuration
	// +optional
	Restore *RestoreSpec `json:"restore,omitempty"`

	// Auth defines authentication configuration
	// +optional
	Auth AuthSpec `json:"auth,omitempty"`

	// Maintenance defines maintenance window configuration
	// +optional
	Maintenance MaintenanceSpec `json:"maintenance,omitempty"`

	// Observability defines monitoring and metrics configuration
	// +optional
	Observability ObservabilitySpec `json:"observability,omitempty"`

	// Lifecycle defines lifecycle management configuration
	// +optional
	Lifecycle LifecycleSpec `json:"lifecycle,omitempty"`

	// EngineConfig provides engine-specific configuration as an opaque map
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	EngineConfig map[string]string `json:"engineConfig,omitempty"`
}

// TopologySpec defines the deployment topology
type TopologySpec struct {
	// Mode specifies the topology mode
	// +optional
	// +kubebuilder:default=Standalone
	Mode TopologyMode `json:"mode,omitempty"`

	// Replicas specifies the number of replicas
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	Replicas int32 `json:"replicas,omitempty"`

	// Shards specifies the number of shards (for sharded topology)
	// +optional
	// +kubebuilder:validation:Minimum=1
	Shards int32 `json:"shards,omitempty"`

	// AntiAffinity enables pod anti-affinity
	// +optional
	AntiAffinity bool `json:"antiAffinity,omitempty"`
}

// StorageSpec defines storage configuration
type StorageSpec struct {
	// StorageClassName specifies the storage class
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`

	// Size specifies the storage size
	// +optional
	// +kubebuilder:default="10Gi"
	Size string `json:"size,omitempty"`

	// VolumeMode specifies the volume mode (Filesystem or Block)
	// +optional
	// +kubebuilder:default=Filesystem
	VolumeMode corev1.PersistentVolumeMode `json:"volumeMode,omitempty"`

	// Snapshots enables volume snapshots
	// +optional
	Snapshots bool `json:"snapshots,omitempty"`
}

// NetworkingSpec defines network configuration
type NetworkingSpec struct {
	// ServiceType specifies the Kubernetes service type
	// +optional
	// +kubebuilder:default=ClusterIP
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// Port specifies the service port
	// +optional
	Port int32 `json:"port,omitempty"`

	// ExternalDNS specifies external DNS name
	// +optional
	ExternalDNS string `json:"externalDNS,omitempty"`

	// TLS enables TLS/SSL
	// +optional
	TLS *TLSSpec `json:"tls,omitempty"`
}

// TLSSpec defines TLS configuration
type TLSSpec struct {
	// Enabled indicates if TLS is enabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// SecretName specifies the secret containing TLS certificates
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// CertManager enables cert-manager integration
	// +optional
	CertManager bool `json:"certManager,omitempty"`
}

// BackupSpec defines backup configuration
type BackupSpec struct {
	// Enabled indicates if backups are enabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Schedule specifies the backup schedule in cron format
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// Method specifies the backup method
	// +optional
	// +kubebuilder:default=Snapshot
	Method BackupMethod `json:"method,omitempty"`

	// Retention specifies the backup retention policy
	// +optional
	// +kubebuilder:default=7
	Retention int32 `json:"retention,omitempty"`

	// Destination specifies the backup destination
	// +optional
	Destination BackupDestination `json:"destination,omitempty"`
}

// BackupDestination defines backup storage destination
type BackupDestination struct {
	// S3 specifies S3 storage configuration
	// +optional
	S3 *S3Spec `json:"s3,omitempty"`

	// PVC specifies PVC storage configuration
	// +optional
	PVC *PVCSpec `json:"pvc,omitempty"`
}

// S3Spec defines S3 configuration
type S3Spec struct {
	// Bucket specifies the S3 bucket name
	Bucket string `json:"bucket"`

	// Region specifies the S3 region
	// +optional
	Region string `json:"region,omitempty"`

	// Endpoint specifies the S3 endpoint
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// CredentialsSecret specifies the secret containing S3 credentials
	CredentialsSecret string `json:"credentialsSecret"`
}

// PVCSpec defines PVC configuration
type PVCSpec struct {
	// StorageClassName specifies the storage class
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`

	// Size specifies the PVC size
	Size string `json:"size"`
}

// RestoreSpec defines restore configuration
type RestoreSpec struct {
	// BackupName specifies the backup to restore from
	BackupName string `json:"backupName"`

	// PointInTime specifies point-in-time recovery timestamp
	// +optional
	PointInTime *metav1.Time `json:"pointInTime,omitempty"`
}

// AuthSpec defines authentication configuration
type AuthSpec struct {
	// SecretName specifies the secret containing credentials
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// Consul enables Consul integration for credential management
	// +optional
	Consul *ConsulSpec `json:"consul,omitempty"`

	// RotationPolicy defines credential rotation configuration
	// +optional
	RotationPolicy *RotationPolicy `json:"rotationPolicy,omitempty"`
}

// ConsulSpec defines Consul integration configuration
type ConsulSpec struct {
	// Enabled indicates if Consul integration is enabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Address specifies the Consul server address
	// +optional
	Address string `json:"address,omitempty"`

	// Path specifies the Consul KV path for credentials
	// +optional
	Path string `json:"path,omitempty"`

	// Token specifies the Consul token (should reference a secret)
	// +optional
	TokenSecretRef *corev1.SecretKeySelector `json:"tokenSecretRef,omitempty"`
}

// RotationPolicy defines credential rotation policy
type RotationPolicy struct {
	// Enabled indicates if automatic rotation is enabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Schedule specifies the rotation schedule in cron format
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// Strategy defines the rotation strategy (TwoPhase, Immediate)
	// +optional
	// +kubebuilder:default=TwoPhase
	Strategy string `json:"strategy,omitempty"`
}

// MaintenanceSpec defines maintenance window configuration
type MaintenanceSpec struct {
	// Windows specifies allowed maintenance windows
	// +optional
	Windows []MaintenanceWindow `json:"windows,omitempty"`

	// AutoUpgrade enables automatic upgrades during maintenance windows
	// +optional
	AutoUpgrade bool `json:"autoUpgrade,omitempty"`
}

// MaintenanceWindow defines a maintenance window
type MaintenanceWindow struct {
	// DayOfWeek specifies the day of week (0-6, 0 = Sunday)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=6
	DayOfWeek int `json:"dayOfWeek"`

	// StartTime specifies the start time (HH:MM format)
	// +kubebuilder:validation:Pattern=`^([0-1][0-9]|2[0-3]):[0-5][0-9]$`
	StartTime string `json:"startTime"`

	// Duration specifies the window duration
	Duration metav1.Duration `json:"duration"`
}

// ObservabilitySpec defines monitoring and metrics configuration
type ObservabilitySpec struct {
	// Metrics enables metrics export
	// +optional
	Metrics *MetricsSpec `json:"metrics,omitempty"`

	// Logging defines logging configuration
	// +optional
	Logging *LoggingSpec `json:"logging,omitempty"`

	// Tracing defines tracing configuration
	// +optional
	Tracing *TracingSpec `json:"tracing,omitempty"`
}

// MetricsSpec defines metrics configuration
type MetricsSpec struct {
	// Enabled indicates if metrics are enabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// ServiceMonitor enables ServiceMonitor creation
	// +optional
	ServiceMonitor bool `json:"serviceMonitor,omitempty"`

	// Port specifies the metrics port
	// +optional
	// +kubebuilder:default=9090
	Port int32 `json:"port,omitempty"`
}

// LoggingSpec defines logging configuration
type LoggingSpec struct {
	// Level specifies the log level
	// +optional
	// +kubebuilder:default=info
	Level string `json:"level,omitempty"`

	// Format specifies the log format (json, text)
	// +optional
	// +kubebuilder:default=json
	Format string `json:"format,omitempty"`
}

// TracingSpec defines tracing configuration
type TracingSpec struct {
	// Enabled indicates if tracing is enabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Endpoint specifies the tracing collector endpoint
	// +optional
	Endpoint string `json:"endpoint,omitempty"`
}

// LifecycleSpec defines lifecycle management configuration
type LifecycleSpec struct {
	// Paused indicates if reconciliation is paused
	// +optional
	Paused bool `json:"paused,omitempty"`

	// DeletionPolicy defines the deletion policy
	// +optional
	// +kubebuilder:default=Retain
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`

	// PreStopHook defines a pre-stop hook
	// +optional
	PreStopHook *LifecycleHook `json:"preStopHook,omitempty"`

	// PostStartHook defines a post-start hook
	// +optional
	PostStartHook *LifecycleHook `json:"postStartHook,omitempty"`
}

// LifecycleHook defines a lifecycle hook
type LifecycleHook struct {
	// Exec specifies a command to execute
	// +optional
	Exec *corev1.ExecAction `json:"exec,omitempty"`

	// HTTPGet specifies an HTTP request
	// +optional
	HTTPGet *corev1.HTTPGetAction `json:"httpGet,omitempty"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// Phase represents the current phase of the database
	// +optional
	Phase DatabasePhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the database's state
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// Endpoint represents the connection endpoint
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// ReadyReplicas is the number of ready replicas
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// CurrentVersion is the currently running database version
	// +optional
	CurrentVersion string `json:"currentVersion,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed Database spec
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastBackup represents the last successful backup timestamp
	// +optional
	LastBackup *metav1.Time `json:"lastBackup,omitempty"`

	// Health represents the health status of the database
	// +optional
	Health HealthStatus `json:"health,omitempty"`

	// LastReconcileTime is the last time reconciliation was performed
	// +optional
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`

	// RotationStatus tracks the status of credential rotation
	// +optional
	RotationStatus *RotationStatus `json:"rotationStatus,omitempty"`
}

// HealthStatus represents the health status of the database
type HealthStatus struct {
	// Status indicates the health status (Healthy, Degraded, Unhealthy, Unknown)
	// +optional
	Status string `json:"status,omitempty"`

	// Message provides additional health information
	// +optional
	Message string `json:"message,omitempty"`

	// LastCheckTime is the last time health was checked
	// +optional
	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`
}

// RotationStatus tracks credential rotation progress
type RotationStatus struct {
	// Phase indicates the rotation phase (Idle, CreatingNew, Cutover, Revoking, Complete)
	// +optional
	Phase string `json:"phase,omitempty"`

	// LastRotation is the timestamp of the last successful rotation
	// +optional
	LastRotation *metav1.Time `json:"lastRotation,omitempty"`

	// NextRotation is the timestamp of the next scheduled rotation
	// +optional
	NextRotation *metav1.Time `json:"nextRotation,omitempty"`

	// JobName is the name of the rotation job
	// +optional
	JobName string `json:"jobName,omitempty"`
}

// Condition types for Database
const (
	// ConditionTypeReady indicates the database is ready
	ConditionTypeReady = "Ready"
	// ConditionTypeProvisioned indicates the database is provisioned
	ConditionTypeProvisioned = "Provisioned"
	// ConditionTypeStorageReady indicates storage is ready
	ConditionTypeStorageReady = "StorageReady"
	// ConditionTypeBackupConfigured indicates backup is configured
	ConditionTypeBackupConfigured = "BackupConfigured"
	// ConditionTypeUpgrading indicates the database is upgrading
	ConditionTypeUpgrading = "Upgrading"
	// ConditionTypeScaling indicates the database is scaling
	ConditionTypeScaling = "Scaling"
	// ConditionTypeValidated indicates the spec is validated
	ConditionTypeValidated = "Validated"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=db;dbs
// +kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.spec.engine`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.endpoint`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Database is the Schema for the databases API
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatabaseList contains a list of Database
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}
