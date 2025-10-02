package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	DefaultMetricRefreshInterval = "15s"
)

type CommunityEditionSpec struct {
	Image ImageSpec `json:"image,omitempty"`
}

type EnterpriseEditionSpec struct {
	// License key for Coroot Enterprise Edition.
	// You can get the Coroot Enterprise license and start a free trial anytime through the Coroot Customer Portal: https://coroot.com/account.
	LicenseKey string `json:"licenseKey,omitempty"`
	// Secret containing the license key.
	LicenseKeySecret *corev1.SecretKeySelector `json:"licenseKeySecret,omitempty"`
	Image            ImageSpec                 `json:"image,omitempty"`
}

type AgentsOnlySpec struct {
	// URL of the Coroot instance to which agents send metrics, logs, traces, and profiles.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^https?://.+$"
	CorootURL string `json:"corootURL,omitempty"`
	// Whether to skip verification of the Coroot server's TLS certificate.
	TLSSkipVerify bool `json:"tlsSkipVerify,omitempty"`
}

type NodeAgentSpec struct {
	// Priority class for the node-agent pods.
	PriorityClassName string                         `json:"priorityClassName,omitempty"`
	UpdateStrategy    appsv1.DaemonSetUpdateStrategy `json:"update_strategy,omitempty"`
	// NodeSelector restricts scheduling to nodes matching the specified labels.
	NodeSelector map[string]string           `json:"nodeSelector,omitempty"`
	Affinity     *corev1.Affinity            `json:"affinity,omitempty"`
	Resources    corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations  []corev1.Toleration         `json:"tolerations,omitempty"`
	// Annotations for node-agent pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// Environment variables for the node-agent.
	Env   []corev1.EnvVar `json:"env,omitempty"`
	Image ImageSpec       `json:"image,omitempty"`

	LogCollector LogCollectorSpec `json:"logCollector,omitempty"`
	EbpfTracer   EbpfTracerSpec   `json:"ebpfTracer,omitempty"`
	EbpfProfiler EbpfProfilerSpec `json:"ebpfProfiler,omitempty"`

	// Allow track connections to the specified IP networks (e.g., Y.Y.Y.Y/mask, default: 0.0.0.0/0).
	TrackPublicNetworks []string `json:"trackPublicNetworks,omitempty"`
}

type LogCollectorSpec struct {
	// Collect log-based metrics (default: true).
	CollectLogBasedMetrics *bool `json:"collectLogBasedMetrics,omitempty"`
	// Collect log entries (default: true).
	CollectLogEntries *bool `json:"collectLogEntries,omitempty"`
}

type EbpfTracerSpec struct {
	// Enable or disable eBPF tracing (default: true).
	Enabled *bool `json:"enabled,omitempty"`
	// Trace sampling rate (0.0 to 1.0; default: 1.0).
	// +kubebuilder:validation:Pattern="^(0([.][0-9]+)?|1([.]0+)?)$"
	Sampling string `json:"sampling,omitempty"`
}

type EbpfProfilerSpec struct {
	// Enable or disable eBPF profiler (default: true).
	Enabled *bool `json:"enabled,omitempty"`
}

type ClusterAgentSpec struct {
	// NodeSelector restricts scheduling to nodes matching the specified labels.
	NodeSelector map[string]string           `json:"nodeSelector,omitempty"`
	Affinity     *corev1.Affinity            `json:"affinity,omitempty"`
	Resources    corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations  []corev1.Toleration         `json:"tolerations,omitempty"`
	// Annotations for cluster-agent pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// Environment variables for the cluster-agent.
	Env   []corev1.EnvVar `json:"env,omitempty"`
	Image ImageSpec       `json:"image,omitempty"`

	KubeStateMetrics KubeStateMetricsSpec `json:"kubeStateMetrics,omitempty"`
}

type KubeStateMetricsSpec struct {
	Image ImageSpec `json:"image,omitempty"`
}

type PrometheusSpec struct {
	// NodeSelector restricts scheduling to nodes matching the specified labels.
	NodeSelector map[string]string           `json:"nodeSelector,omitempty"`
	Affinity     *corev1.Affinity            `json:"affinity,omitempty"`
	Storage      StorageSpec                 `json:"storage,omitempty"`
	Resources    corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations  []corev1.Toleration         `json:"tolerations,omitempty"`
	// Annotations for prometheus pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	Image          ImageSpec         `json:"image,omitempty"`
	// Metrics retention time (e.g. 4h, 3d, 2w, 1y; default 2d).
	// +kubebuilder:validation:Pattern="^[0-9]+[mhdwy]$"
	Retention string `json:"retention,omitempty"`
	// Out-of-order time window (e.g. 30s, 10m, 2h; default 1h).
	// +kubebuilder:validation:Pattern="^[0-9]+[smhdwy]$"
	OutOfOrderTimeWindow string `json:"outOfOrderTimeWindow,omitempty"`
}

type ExternalPrometheusSpec struct {
	// http(s)://IP:Port or http(s)://Domain:Port or http(s)://ServiceName:Port
	// +kubebuilder:validation:Pattern="^https?://.+$"
	URL string `json:"url,omitempty"`
	// Whether to skip verification of the Prometheus server's TLS certificate.
	TLSSkipVerify bool `json:"tlsSkipVerify,omitempty"`
	// Basic auth credentials.
	BasicAuth *BasicAuthSpec `json:"basicAuth,omitempty"`
	// Custom headers to include in requests to the Prometheus server.
	CustomHeaders map[string]string `json:"customHeaders,omitempty"`
	// The URL for metric ingestion though the Prometheus Remote Write protocol (optional).
	// +kubebuilder:validation:Pattern="^https?://.+$"
	RemoteWriteUrl string `json:"remoteWriteURL,omitempty"`
}

type ClickhouseSpec struct {
	Shards   int `json:"shards,omitempty"`
	Replicas int `json:"replicas,omitempty"`

	// NodeSelector restricts scheduling to nodes matching the specified labels.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	Affinity     *corev1.Affinity  `json:"affinity,omitempty"`
	// Storage configuration for clickhouse.
	Storage     StorageSpec                 `json:"storage,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration         `json:"tolerations,omitempty"`
	// Annotations for clickhouse pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	Image          ImageSpec         `json:"image,omitempty"`
	// Log level (fatal, critical, error, warning, notice, information, debug, trace, test, or none; default: warning).
	// +kubebuilder:validation:Enum="none";"fatal";"critical";"error";"warning";"notice";"information";"debug";"trace";"test"
	LogLevel string `json:"logLevel,omitempty"`

	Keeper ClickhouseKeeperSpec `json:"keeper,omitempty"`
}

type ClickhouseKeeperSpec struct {
	Replicas int `json:"replicas,omitempty"`
	// NodeSelector restricts scheduling to nodes matching the specified labels.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	Affinity     *corev1.Affinity  `json:"affinity,omitempty"`
	// Storage configuration for clickhouse-keeper.
	Storage     StorageSpec                 `json:"storage,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration         `json:"tolerations,omitempty"`
	// Annotations for clickhouse-keeper pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	Image          ImageSpec         `json:"image,omitempty"`
	// Log level (fatal, critical, error, warning, notice, information, debug, trace, test, or none; default: warning).
	// +kubebuilder:validation:Enum="none";"fatal";"critical";"error";"warning";"notice";"information";"debug";"trace";"test"
	LogLevel string `json:"logLevel,omitempty"`
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
	// Secret containing a password for accessing the external ClickHouse.
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`
	// Whether to enable TLS for the connection to ClickHouse.
	TLSEnabled bool `json:"tlsEnabled,omitempty"`
	// Whether to skip verification of the ClickHouse server's TLS certificate.
	TLSSkipVerify bool `json:"tlsSkipVerify,omitempty"`
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
	// Secret containing a password for accessing postgres.
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

type ServiceSpec struct {
	// Service type (e.g., ClusterIP, NodePort, LoadBalancer).
	Type corev1.ServiceType `json:"type,omitempty"`
	// Service port (default 8080).
	Port int32 `json:"port,omitempty"`
	// Service nodePort (if type is NodePort).
	NodePort int32 `json:"nodePort,omitempty"`
	// gRPC port (default 4317).
	GRPCPort int32 `json:"grpcPort,omitempty"`
	// gRPC nodePort (if type is NodePort).
	GRPCNodePort int32 `json:"grpcNodePort,omitempty"`
	// Annotations for the service.
	Annotations map[string]string `json:"annotations,omitempty"`
}

type GRPCSpec struct {
	// Disables gRPC server.
	Disabled bool `json:"disabled,omitempty"`
}

type TLSSpec struct {
	// Secret containing TLS certificate.
	// +kubebuilder:validation:Required
	CertSecret *corev1.SecretKeySelector `json:"certSecret,omitempty"`
	// Secret containing a TLS private key.
	// +kubebuilder:validation:Required
	KeySecret *corev1.SecretKeySelector `json:"keySecret,omitempty"`
}

type CorootSpec struct {
	// Specifies the metric resolution interval.
	// +kubebuilder:validation:Pattern="^[0-9]+[sm]$"
	MetricsRefreshInterval string `json:"metricsRefreshInterval,omitempty"`
	// Metric cache retention time (e.g. 4h, 3d, 2w, 1y; default 30d).
	// +kubebuilder:validation:Pattern="^[0-9]+[mhdwy]$"
	CacheTTL string `json:"cacheTTL,omitempty"`
	// Traces retention time (e.g. 4h, 3d, 2w, 1y; default 7d).
	// +kubebuilder:validation:Pattern="^[0-9]+[mhdwy]$"
	TracesTTL string `json:"tracesTTL,omitempty"`
	// Logs retention time (e.g. 4h, 3d, 2w, 1y; default 7d).
	// +kubebuilder:validation:Pattern="^[0-9]+[mhdwy]$"
	LogsTTL string `json:"logsTTL,omitempty"`
	// Profiles retention time (e.g. 4h, 3d, 2w, 1y; default 7d).
	// +kubebuilder:validation:Pattern="^[0-9]+[mhdwy]$"
	ProfilesTTL string `json:"profilesTTL,omitempty"`
	// Allows access to Coroot without authentication if set (one of Admin, Editor, or Viewer).
	AuthAnonymousRole string `json:"authAnonymousRole,omitempty"`
	// Initial admin password for bootstrapping.
	AuthBootstrapAdminPassword string `json:"authBootstrapAdminPassword,omitempty"`
	// Secret containing the initial admin password.
	AuthBootstrapAdminPasswordSecret *corev1.SecretKeySelector `json:"authBootstrapAdminPasswordSecret,omitempty"`
	// gRPC settings.
	GRPC GRPCSpec `json:"grpc,omitempty"`
	// TLS settings (enables TLS for gRPC if defined).
	TLS *TLSSpec `json:"tls,omitempty"`
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
	Ingress *IngressSpec `json:"ingress,omitempty"`
	// NodeSelector restricts scheduling to nodes matching the specified labels.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	Affinity     *corev1.Affinity  `json:"affinity,omitempty"`
	// Storage configuration for Coroot.
	Storage     StorageSpec                 `json:"storage,omitempty"`
	Resources   corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration         `json:"tolerations,omitempty"`
	// Annotations for Coroot pods.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// The API key used by agents when sending telemetry to Coroot.
	ApiKey string `json:"apiKey,omitempty"`
	// Secret containing API key.
	ApiKeySecret *corev1.SecretKeySelector `json:"apiKeySecret,omitempty"`
	NodeAgent    NodeAgentSpec             `json:"nodeAgent,omitempty"`
	ClusterAgent ClusterAgentSpec          `json:"clusterAgent,omitempty"`

	// Prometheus configuration.
	Prometheus PrometheusSpec `json:"prometheus,omitempty"`
	// Use an external Prometheus instance instead of deploying one.
	ExternalPrometheus *ExternalPrometheusSpec `json:"externalPrometheus,omitempty"`

	// Clickhouse configuration.
	Clickhouse ClickhouseSpec `json:"clickhouse,omitempty"`
	// Use an external ClickHouse instance instead of deploying one.
	ExternalClickhouse *ExternalClickhouseSpec `json:"externalClickhouse,omitempty"`

	// Store configuration in a Postgres DB instead of SQLite (required if replicas > 1).
	Postgres *PostgresSpec `json:"postgres,omitempty"`

	// Coroot Cloud integration.
	CorootCloud *CorootCloudSpec `json:"corootCloud,omitempty"`
	// Projects configuration (Coroot will create or update the specified projects).
	Projects []ProjectSpec `json:"projects,omitempty"`
	// Single Sign-On configuration (Coroot Enterprise Edition only).
	SSO *SSOSpec `json:"sso,omitempty"`
	// AI configuration (Coroot Enterprise Edition only).
	AI *AISpec `json:"ai,omitempty"`
}

type CorootStatus struct {
	Status string   `json:"status"`
	Errors []string `json:"errors,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:metadata:annotations=argocd.argoproj.io/sync-options=Replace=true

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
