package v1

import corev1 "k8s.io/api/core/v1"

// ClusterAgentAWSSpec configures the AWS integration of the cluster-agent (discovery of RDS and ElastiCache instances).
// When set, it takes precedence over the AWS integration settings configured in the Coroot UI.
type ClusterAgentAWSSpec struct {
	// AWS region to discover instances in. Defaults to the region the cluster runs in.
	Region string `json:"region,omitempty"`
	// Secret with the static access key (keys: access_key_id, secret_access_key).
	// Leave empty to use the IAM role of the cluster-agent pod (EKS Pod Identity, IRSA) or the EC2 instance profile.
	AccessKeySecret *corev1.LocalObjectReference `json:"accessKeySecret,omitempty"`
	// Discover only RDS instances whose tags match (glob patterns are supported in values).
	RDSTagFilters map[string]string `json:"rdsTagFilters,omitempty"`
	// Discover only ElastiCache clusters whose tags match (glob patterns are supported in values).
	ElasticacheTagFilters map[string]string `json:"elasticacheTagFilters,omitempty"`
}

// ClusterAgentGCPSpec configures the GCP integration of the cluster-agent (discovery of Cloud SQL and Memorystore instances).
type ClusterAgentGCPSpec struct {
	// GCP project to discover instances in. Defaults to the project of the GKE cluster.
	ProjectId string `json:"projectId,omitempty"`
	// Region to discover instances in. Defaults to the region the cluster runs in; "all" scans every region of the project.
	Region string `json:"region,omitempty"`
	// Secret with a service account key (key: credentials.json).
	// Leave empty to use GKE Workload Identity or the service account of the node.
	CredentialsSecret *corev1.SecretKeySelector `json:"credentialsSecret,omitempty"`
	// Discover only Cloud SQL instances whose labels match (glob patterns are supported in values).
	CloudSQLLabelFilters map[string]string `json:"cloudsqlLabelFilters,omitempty"`
	// Discover only Memorystore instances whose labels match (glob patterns are supported in values).
	MemorystoreLabelFilters map[string]string `json:"memorystoreLabelFilters,omitempty"`
}

// ClusterAgentOCISpec configures the OCI integration of the cluster-agent (discovery of MySQL HeatWave and PostgreSQL DB systems and OCI Cache clusters).
type ClusterAgentOCISpec struct {
	// OCIDs of the compartments to discover instances in. Defaults to the compartment of the cluster when OKE Workload Identity is used.
	CompartmentIds []string `json:"compartmentIds,omitempty"`
	// Region to discover instances in. Defaults to the region the cluster runs in.
	Region string `json:"region,omitempty"`
	// Secret with an API key (keys: tenancy_id, user_id, fingerprint, private_key); leave unset to use OKE Workload Identity or the instance principal of the nodes.
	ApiKeySecret *corev1.LocalObjectReference `json:"apiKeySecret,omitempty"`
	// Discover only DB systems whose freeform tags match (glob patterns are supported in values).
	DBTagFilters map[string]string `json:"dbTagFilters,omitempty"`
	// Discover only OCI Cache clusters whose freeform tags match (glob patterns are supported in values).
	CacheTagFilters map[string]string `json:"cacheTagFilters,omitempty"`
}

// ClusterAgentDatabaseSpec is a database the cluster-agent collects metrics from.
// Exactly one of host, rds, elasticache, cloudsql, memorystore, ocidb or ocicache must be set.
type ClusterAgentDatabaseSpec struct {
	// Database type.
	// +kubebuilder:validation:Enum=postgres;mysql;redis;memcached;mongodb
	Type string `json:"type"`
	// Hostname or IP address. A hostname is re-resolved on every configuration update and every resolved address is monitored.
	Host string `json:"host,omitempty"`
	// Port. Required with host; defaults to the instance port for rds and elasticache.
	Port string `json:"port,omitempty"`
	// Identifier of an RDS instance discovered by the AWS integration.
	RDS string `json:"rds,omitempty"`
	// Id of an ElastiCache cluster discovered by the AWS integration; every node is monitored.
	Elasticache string `json:"elasticache,omitempty"`
	// Name of a Cloud SQL instance discovered by the GCP integration.
	CloudSQL string `json:"cloudsql,omitempty"`
	// Name of a Memorystore instance discovered by the GCP integration.
	Memorystore string `json:"memorystore,omitempty"`
	// Display name of a MySQL HeatWave or PostgreSQL DB system discovered by the OCI integration.
	OCIDB string `json:"ocidb,omitempty"`
	// Display name of an OCI Cache cluster discovered by the OCI integration.
	OCICache string `json:"ocicache,omitempty"`
	// Credentials.
	Credentials *ClusterAgentDatabaseCredentials `json:"credentials,omitempty"`
	// Type-specific parameters, e.g. sslmode: require for Postgres.
	Params map[string]string `json:"params,omitempty"`
}

type ClusterAgentDatabaseCredentials struct {
	// Plain-text username. Prefer usernameSecret.
	Username string `json:"username,omitempty"`
	// Secret with the username.
	UsernameSecret *corev1.SecretKeySelector `json:"usernameSecret,omitempty"`
	// Plain-text password. Prefer passwordSecret.
	Password string `json:"password,omitempty"`
	// Secret with the password.
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`
}
