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

// ClusterAgentDatabaseSpec is a database the cluster-agent collects metrics from.
// Exactly one of host, rds or elasticache must be set.
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
