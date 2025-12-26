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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatabaseType defines the type of database to create
// +kubebuilder:validation:Enum=PostgreSQL;MongoDB;Redis;Elasticsearch;SQLite
type DatabaseType string

const (
	DatabaseTypePostgreSQL    DatabaseType = "PostgreSQL"
	DatabaseTypeMongoDB       DatabaseType = "MongoDB"
	DatabaseTypeRedis         DatabaseType = "Redis"
	DatabaseTypeElasticsearch DatabaseType = "Elasticsearch"
	DatabaseTypeSQLite        DatabaseType = "SQLite"
)

// DatabaseSpec defines the desired state of Database.
type DatabaseSpec struct {
	// Type specifies the database type (PostgreSQL, MongoDB, Redis, Elasticsearch, SQLite)
	// +kubebuilder:validation:Required
	Type DatabaseType `json:"type"`

	// Version specifies the version of the database to deploy
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Version string `json:"version"`

	// Replicas specifies the number of database replicas
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Storage defines the storage configuration for the database
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// Resources defines the compute resources for the database
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// PostgreSQL specific configuration
	// +optional
	PostgreSQL *PostgreSQLConfig `json:"postgresql,omitempty"`

	// MongoDB specific configuration
	// +optional
	MongoDB *MongoDBConfig `json:"mongodb,omitempty"`

	// Redis specific configuration
	// +optional
	Redis *RedisConfig `json:"redis,omitempty"`

	// Elasticsearch specific configuration
	// +optional
	Elasticsearch *ElasticsearchConfig `json:"elasticsearch,omitempty"`

	// SQLite specific configuration
	// +optional
	SQLite *SQLiteConfig `json:"sqlite,omitempty"`

	// Environment variables to set in the database container
	// +optional
	Env []EnvVar `json:"env,omitempty"`
}

// StorageSpec defines the storage configuration
type StorageSpec struct {
	// Size specifies the size of the persistent volume
	// +kubebuilder:validation:Required
	Size string `json:"size"`

	// StorageClass specifies the storage class to use
	// +optional
	StorageClass *string `json:"storageClassName,omitempty"`

	// AccessMode specifies the access mode for the volume
	// +kubebuilder:default=ReadWriteOnce
	// +optional
	AccessMode string `json:"accessMode,omitempty"`
}

// ResourceRequirements defines the compute resources
type ResourceRequirements struct {
	// CPU resource request
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Memory resource request
	// +optional
	Memory string `json:"memory,omitempty"`

	// CPU resource limit
	// +optional
	CPULimit string `json:"cpuLimit,omitempty"`

	// Memory resource limit
	// +optional
	MemoryLimit string `json:"memoryLimit,omitempty"`
}

// PostgreSQLConfig defines PostgreSQL-specific configuration
type PostgreSQLConfig struct {
	// Database name to create
	// +optional
	Database string `json:"database,omitempty"`

	// Username for the database
	// +optional
	Username string `json:"username,omitempty"`

	// Password secret reference
	// +optional
	PasswordSecret *SecretReference `json:"passwordSecret,omitempty"`

	// Additional PostgreSQL configuration parameters
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// MongoDBConfig defines MongoDB-specific configuration
type MongoDBConfig struct {
	// Database name to create
	// +optional
	Database string `json:"database,omitempty"`

	// Username for the database
	// +optional
	Username string `json:"username,omitempty"`

	// Password secret reference
	// +optional
	PasswordSecret *SecretReference `json:"passwordSecret,omitempty"`

	// ReplicaSet name for MongoDB replica set
	// +optional
	ReplicaSetName string `json:"replicaSetName,omitempty"`

	// Additional MongoDB configuration parameters
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// RedisConfig defines Redis-specific configuration
type RedisConfig struct {
	// Password secret reference
	// +optional
	PasswordSecret *SecretReference `json:"passwordSecret,omitempty"`

	// Mode specifies Redis mode (standalone, sentinel, cluster)
	// +kubebuilder:validation:Enum=standalone;sentinel;cluster
	// +kubebuilder:default=standalone
	// +optional
	Mode string `json:"mode,omitempty"`

	// Additional Redis configuration parameters
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// ElasticsearchConfig defines Elasticsearch-specific configuration
type ElasticsearchConfig struct {
	// ClusterName specifies the Elasticsearch cluster name
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// NodeRoles specifies the roles for this node (master, data, ingest)
	// +optional
	NodeRoles []string `json:"nodeRoles,omitempty"`

	// Additional Elasticsearch configuration parameters
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// SQLiteConfig defines SQLite-specific configuration
type SQLiteConfig struct {
	// DatabaseFile specifies the SQLite database file path
	// +optional
	DatabaseFile string `json:"databaseFile,omitempty"`

	// Additional SQLite configuration parameters
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// SecretReference defines a reference to a Kubernetes Secret
type SecretReference struct {
	// Name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key in the secret to use
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// EnvVar defines an environment variable
type EnvVar struct {
	// Name of the environment variable
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Value of the environment variable
	// +optional
	Value string `json:"value,omitempty"`

	// ValueFrom specifies a source for the environment variable's value
	// +optional
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty"`
}

// EnvVarSource defines a source for an environment variable
type EnvVarSource struct {
	// SecretKeyRef selects a key from a secret
	// +optional
	SecretKeyRef *SecretReference `json:"secretKeyRef,omitempty"`
}

// DatabasePhase defines the phase of the database
type DatabasePhase string

const (
	DatabasePhasePending   DatabasePhase = "Pending"
	DatabasePhaseCreating  DatabasePhase = "Creating"
	DatabasePhaseReady     DatabasePhase = "Ready"
	DatabasePhaseFailed    DatabasePhase = "Failed"
	DatabasePhaseDeleting  DatabasePhase = "Deleting"
	DatabasePhaseUpgrading DatabasePhase = "Upgrading"
)

// DatabaseStatus defines the observed state of Database.
type DatabaseStatus struct {
	// Phase represents the current phase of the database
	// +optional
	Phase DatabasePhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the database's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ReadyReplicas is the number of ready database replicas
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// ServiceName is the name of the service created for the database
	// +optional
	ServiceName string `json:"serviceName,omitempty"`

	// ConnectionString provides connection information (without credentials)
	// +optional
	ConnectionString string `json:"connectionString,omitempty"`

	// ObservedGeneration is the most recent generation observed for this database
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Message provides additional information about the current state
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=db
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Database is the Schema for the databases API.
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatabaseList contains a list of Database.
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}
