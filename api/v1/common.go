package v1

import (
	"encoding/json"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ImageSpec defines the configuration for specifying an image repository, tag, pull policy, and pull secrets.
// If unspecified, the operator will automatically update its components to the latest versions from Coroot's public registry.
type ImageSpec struct {
	// Name specifies the full image reference, including registry, component, and tag.
	// E.g.: <private-registry>/<component-name>:<component-version>
	Name string `json:"name,omitempty"`
	// PullPolicy defines the image pull policy (e.g., Always, IfNotPresent, Never).
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
	// PullSecrets contains a list of references to Kubernetes secrets used for pulling the image from a private registry.
	PullSecrets []corev1.LocalObjectReference `json:"pullSecrets,omitempty"`
}

type StorageSpec struct {
	// Volume size
	Size resource.Quantity `json:"size,omitempty"`
	// If not set, the default storage class will be used.
	ClassName *string `json:"className,omitempty"`
	// Valid options are Retain (keep PVC), or Delete (default).
	ReclaimPolicy corev1.PersistentVolumeReclaimPolicy `json:"reclaimPolicy,omitempty"`
	// Annotations for PersistentVolumeClaim (PVC).
	Annotations map[string]string `json:"annotations,omitempty"`
}

type BasicAuthSpec struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	// Secret containing password. If specified, this takes precedence over the Password field.
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`
}

type HeaderSpec struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
}

// Percent represents a percentage value between 0 and 100 (inclusive).
// +kubebuilder:validation:Type=number
// +kubebuilder:validation:Minimum=0
// +kubebuilder:validation:Maximum=100
type Percent string

func (p *Percent) UnmarshalJSON(b []byte) error {
	var v float32
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*p = Percent(strconv.FormatFloat(float64(v), 'f', -1, 32))
	return nil
}

func (p Percent) MarshalJSON() ([]byte, error) {
	_, err := strconv.ParseFloat(string(p), 32)
	if err != nil {
		return nil, err
	}
	return []byte(string(p)), nil
}
