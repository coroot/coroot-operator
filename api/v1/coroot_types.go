package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	DefaultMetricRefreshInterval = "15s"
)

type CommunityEditionSpec struct {
	// If unspecified, the operator will automatically update Coroot CE to the latest version.
	Version string `json:"version,omitempty"`
}

type EnterpriseEditionSpec struct {
	// If unspecified, the operator will automatically update Coroot EE to the latest version.
	Version string `json:"version,omitempty"`
	// License key for Coroot Enterprise Edition.
	// You can get the Coroot Enterprise license and start a free trial anytime through the Coroot Customer Portal: https://coroot.com/account.
	LicenseKey string `json:"licenseKey,omitempty"`
}

type AgentsOnlySpec struct {
	// URL of the Coroot instance to which agents send metrics, logs, traces, and profiles.
	CorootURL string `json:"corootURL,omitempty"`
}

type ServiceSpec struct {
	// Service type (e.g., ClusterIP, NodePort, LoadBalancer).
	Type corev1.ServiceType `json:"type,omitempty"`
	// Service port number.
	Port int32 `json:"port,omitempty"`
	// NodePort number (if type is NodePort).
	NodePort int32 `json:"nodePort,omitempty"`
}

type StorageSpec struct {
	// Volume size
	Size resource.Quantity `json:"size,omitempty"`
	// If not set, the default storage class will be used.
	ClassName *string `json:"className,omitempty"`
}

type NodeAgentSpec struct {
	// If unspecified, the operator will automatically update the node-agent to the latest version.
	Version string `json:"version,omitempty"`

	// Priority class for the node-agent pods.
	PriorityClassName string                         `json:"priorityClassName,omitempty"`
	UpdateStrategy    appsv1.DaemonSetUpdateStrategy `json:"update_strategy,omitempty"`
	Affinity          *corev1.Affinity               `json:"affinity,omitempty"`
	Resources         corev1.ResourceRequirements    `json:"resources,omitempty"`
	Tolerations       []corev1.Toleration            `json:"tolerations,omitempty"`
	// Annotations for node-agent pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// Environment variables for the node-agent.
	Env []corev1.EnvVar `json:"env,omitempty"`
}

type ClusterAgentSpec struct {
	// If unspecified, the operator will automatically update the cluster-agent to the latest version.
	Version string `json:"version,omitempty"`

	Affinity    *corev1.Affinity            `json:"affinity,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration         `json:"tolerations,omitempty"`
	// Annotations for cluster-agent pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// Environment variables for the cluster-agent.
	Env []corev1.EnvVar `json:"env,omitempty"`
}

type PrometheusSpec struct {
	Affinity    *corev1.Affinity            `json:"affinity,omitempty"`
	Storage     StorageSpec                 `json:"storage,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration         `json:"tolerations,omitempty"`
	// Annotations for prometheus pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// Metrics retention time (e.g. 4h, 3d, 2w, 1y; default 2d)
	// +kubebuilder:validation:Pattern=^\d+[mhdwy]$
	Retention string `json:"retention,omitempty"`
}

type ClickhouseSpec struct {
	Shards   int `json:"shards,omitempty"`
	Replicas int `json:"replicas,omitempty"`

	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Storage configuration for clickhouse.
	Storage     StorageSpec                 `json:"storage,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration         `json:"tolerations,omitempty"`
	// Annotations for clickhouse pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	Keeper ClickhouseKeeperSpec `json:"keeper,omitempty"`
}

type ClickhouseKeeperSpec struct {
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Storage configuration for clickhouse-keeper.
	Storage     StorageSpec                 `json:"storage,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration         `json:"tolerations,omitempty"`
	// Annotations for clickhouse-keeper pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
}

type ExternalClickhouseSpec struct {
	// Address of the external ClickHouse instance.
	Address string `json:"address,omitempty"`
	// Username for accessing the external ClickHouse.
	User string `json:"user,omitempty"`
	// Name of the database to be used.
	Database string `json:"database,omitempty"`
	// Password for accessing the external ClickHouse (plain-text, not recommended).
	Password string `json:"password,omitempty"`
	// Secret containing password for accessing the external ClickHouse.
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`
}

type PostgresSpec struct {
	// Postgres host or service name.
	Host string `json:"host,omitempty"`
	// Postgres port (optional, default 5432).
	Port int32 `json:"port,omitempty"`
	// Username for accessing Postgres.
	User string `json:"user,omitempty"`
	// Name of the database.
	Database string `json:"database,omitempty"`
	// Password for accessing postgres (plain-text, not recommended).
	Password string `json:"password,omitempty"`
	// Secret containing password for accessing postgres.
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`
	// Extra parameters, e.g., sslmode and connect_timeout.
	Params map[string]string `json:"params,omitempty"`
}

type IngressSpec struct {
	// Ingress class name (e.g., nginx, traefik; if not set the default IngressClass will be used).
	ClassName *string `json:"className,omitempty"`
	// Domain name for Coroot (e.g., coroot.company.com).
	Host string `json:"host,omitempty"`
	// Path prefix for Coroot (e.g., /coroot).
	Path string                   `json:"path,omitempty"`
	TLS  *networkingv1.IngressTLS `json:"tls,omitempty"`
	// Annotations for Ingress.
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ProjectSpec struct {
	// Project name (e.g., production, staging; required).
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// Project API keys, used by agents to send telemetry data (required).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	ApiKeys []ApiKeySpec `json:"apiKeys,omitempty"`
}

type ApiKeySpec struct {
	// Random string or UUID (must be unique; required).
	// +kubebuilder:validation:Required
	Key string `json:"key,omitempty"`
	// API key description (optional).
	Description string `json:"description,omitempty"`
}

type CorootSpec struct {
	// Specifies the metric resolution interval.
	MetricsRefreshInterval metav1.Duration `json:"metricsRefreshInterval,omitempty"`
	// Duration for which Coroot retains the metric cache.
	CacheTTL metav1.Duration `json:"cacheTTL,omitempty"`
	// Allows access to Coroot without authentication if set (one of Admin, Editor, or Viewer).
	AuthAnonymousRole string `json:"authAnonymousRole,omitempty"`
	// Initial admin password for bootstrapping.
	AuthBootstrapAdminPassword string `json:"authBootstrapAdminPassword,omitempty"`
	// Projects configuration (Coroot will create or update projects the specified projects).
	Projects []ProjectSpec `json:"projects,omitempty"`
	// Environment variables for Coroot.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Configurations for Coroot Community Edition.
	CommunityEdition CommunityEditionSpec `json:"communityEdition,omitempty"`
	// Configurations for Coroot Enterprise Edition.
	EnterpriseEdition *EnterpriseEditionSpec `json:"enterpriseEdition,omitempty"`
	// Configures the operator to install only the node-agent and cluster-agent.
	AgentsOnly *AgentsOnlySpec `json:"agentsOnly,omitempty"`

	// Number of Coroot StatefulSet pods.
	Replicas int `json:"replicas,omitempty"`
	// Service configuration for Coroot.
	Service ServiceSpec `json:"service,omitempty"`
	// Ingress configuration for Coroot.
	Ingress  *IngressSpec     `json:"ingress,omitempty"`
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Storage configuration for Coroot.
	Storage     StorageSpec                 `json:"storage,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration         `json:"tolerations,omitempty"`
	// Annotations for Coroot pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// The API key used by agents when sending telemetry to Coroot.
	ApiKey       string           `json:"apiKey,omitempty"`
	NodeAgent    NodeAgentSpec    `json:"nodeAgent,omitempty"`
	ClusterAgent ClusterAgentSpec `json:"clusterAgent,omitempty"`

	// Prometheus configuration.
	Prometheus PrometheusSpec `json:"prometheus,omitempty"`

	// Clickhouse configuration.
	Clickhouse ClickhouseSpec `json:"clickhouse,omitempty"`
	// Use an external ClickHouse instance instead of deploying one.
	ExternalClickhouse *ExternalClickhouseSpec `json:"externalClickhouse,omitempty"`

	// Store configuration in a Postgres DB instead of SQLite (required if replicas > 1).
	Postgres *PostgresSpec `json:"postgres,omitempty"`
}

type CorootStatus struct { // TODO
	// Represents the observations of a Coroot's current state.
	// Coroot.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// Coroot.status.conditions.status are one of True, False, Unknown.
	// Coroot.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// Coroot.status.conditions.Message is a human-readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type Coroot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CorootSpec   `json:"spec,omitempty"`
	Status CorootStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type CorootList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Coroot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Coroot{}, &CorootList{})
}
