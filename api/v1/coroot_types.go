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
	Version string `json:"version,omitempty"`
}

type EnterpriseEditionSpec struct {
	Version    string `json:"version,omitempty"`
	LicenseKey string `json:"licenseKey,omitempty"`
}

type AgentsOnlySpec struct {
	CorootURL string `json:"corootURL,omitempty"`
}

type ServiceSpec struct {
	Type     corev1.ServiceType `json:"type,omitempty"`
	Port     int32              `json:"port,omitempty"`
	NodePort int32              `json:"nodePort,omitempty"`
}

type StorageSpec struct {
	Size      resource.Quantity `json:"size,omitempty"`
	ClassName *string           `json:"className,omitempty"`
}

type NodeAgentSpec struct {
	Version string `json:"version,omitempty"`

	PriorityClassName string                         `json:"priorityClassName,omitempty"`
	UpdateStrategy    appsv1.DaemonSetUpdateStrategy `json:"update_strategy,omitempty"`
	Affinity          *corev1.Affinity               `json:"affinity,omitempty"`
	Resources         corev1.ResourceRequirements    `json:"resources,omitempty"`
	Tolerations       []corev1.Toleration            `json:"tolerations,omitempty"`
	PodAnnotations    map[string]string              `json:"podAnnotations,omitempty"`
	Env               []corev1.EnvVar                `json:"env,omitempty"`
}

type ClusterAgentSpec struct {
	Version string `json:"version,omitempty"`

	Affinity       *corev1.Affinity            `json:"affinity,omitempty"`
	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations    []corev1.Toleration         `json:"tolerations,omitempty"`
	PodAnnotations map[string]string           `json:"podAnnotations,omitempty"`
	Env            []corev1.EnvVar             `json:"env,omitempty"`
}

type PrometheusSpec struct {
	Affinity       *corev1.Affinity            `json:"affinity,omitempty"`
	Storage        StorageSpec                 `json:"storage,omitempty"`
	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations    []corev1.Toleration         `json:"tolerations,omitempty"`
	PodAnnotations map[string]string           `json:"podAnnotations,omitempty"`
}

type ExternalPrometheusSpec struct {
	Address        string                    `json:"address,omitempty"`
	User           string                    `json:"user,omitempty"`
	Password       string                    `json:"password,omitempty"`
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`
}

type ClickhouseSpec struct {
	Shards   int `json:"shards,omitempty"`
	Replicas int `json:"replicas,omitempty"`

	Affinity       *corev1.Affinity            `json:"affinity,omitempty"`
	Storage        StorageSpec                 `json:"storage,omitempty"`
	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations    []corev1.Toleration         `json:"tolerations,omitempty"`
	PodAnnotations map[string]string           `json:"podAnnotations,omitempty"`

	Keeper ClickhouseKeeperSpec `json:"keeper,omitempty"`
}

type ClickhouseKeeperSpec struct {
	Affinity       *corev1.Affinity            `json:"affinity,omitempty"`
	Storage        StorageSpec                 `json:"storage,omitempty"`
	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations    []corev1.Toleration         `json:"tolerations,omitempty"`
	PodAnnotations map[string]string           `json:"podAnnotations,omitempty"`
}

type ExternalClickhouseSpec struct {
	Address        string                    `json:"address,omitempty"`
	User           string                    `json:"user,omitempty"`
	Database       string                    `json:"database,omitempty"`
	Password       string                    `json:"password,omitempty"`
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`
}

type PostgresSpec struct {
	Host           string                    `json:"host,omitempty"`
	Port           int32                     `json:"port,omitempty"`
	User           string                    `json:"user,omitempty"`
	Database       string                    `json:"database,omitempty"`
	Password       string                    `json:"password,omitempty"`
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`
	Params         map[string]string         `json:"params,omitempty"`
}

type IngressSpec struct {
	ClassName *string                  `json:"className,omitempty"`
	Host      string                   `json:"host,omitempty"`
	Path      string                   `json:"path,omitempty"`
	TLS       *networkingv1.IngressTLS `json:"tls,omitempty"`
}

type ProjectSpec struct {
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	ApiKeys []ApiKeySpec `json:"apiKeys,omitempty"`
}

type ApiKeySpec struct {
	// +kubebuilder:validation:Required
	Key         string `json:"key,omitempty"`
	Description string `json:"description,omitempty"`
}

type CorootSpec struct {
	MetricsRefreshInterval     metav1.Duration `json:"metricsRefreshInterval,omitempty"`
	CacheTTL                   metav1.Duration `json:"cacheTTL,omitempty"`
	AuthAnonymousRole          string          `json:"authAnonymousRole,omitempty"`
	AuthBootstrapAdminPassword string          `json:"authBootstrapAdminPassword,omitempty"`
	Projects                   []ProjectSpec   `json:"projects,omitempty"`
	Env                        []corev1.EnvVar `json:"env,omitempty"`

	CommunityEdition  CommunityEditionSpec   `json:"communityEdition,omitempty"`
	EnterpriseEdition *EnterpriseEditionSpec `json:"enterpriseEdition,omitempty"`
	AgentsOnly        *AgentsOnlySpec        `json:"agentsOnly,omitempty"`

	Replicas       int                         `json:"replicas,omitempty"`
	Service        ServiceSpec                 `json:"service,omitempty"`
	Affinity       *corev1.Affinity            `json:"affinity,omitempty"`
	Storage        StorageSpec                 `json:"storage,omitempty"`
	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations    []corev1.Toleration         `json:"tolerations,omitempty"`
	PodAnnotations map[string]string           `json:"podAnnotations,omitempty"`

	ApiKey       string           `json:"apiKey,omitempty"`
	NodeAgent    NodeAgentSpec    `json:"nodeAgent,omitempty"`
	ClusterAgent ClusterAgentSpec `json:"clusterAgent,omitempty"`

	Prometheus PrometheusSpec `json:"prometheus,omitempty"`
	ExternalPrometheus *ExternalPrometheusSpec `json:"externalPrometheus,omitempty"`

	Clickhouse         ClickhouseSpec          `json:"clickhouse,omitempty"`
	ExternalClickhouse *ExternalClickhouseSpec `json:"externalClickhouse,omitempty"`

	Postgres *PostgresSpec `json:"postgres,omitempty"`

	Ingress *IngressSpec `json:"ingress,omitempty"`
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
