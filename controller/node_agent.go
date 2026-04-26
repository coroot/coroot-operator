package controller

import (
	"cmp"
	"fmt"
	"strings"

	corootv1 "github.io/coroot/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func (r *CorootReconciler) nodeAgentDaemonSet(cr *corootv1.Coroot) *appsv1.DaemonSet {
	ls := Labels(cr, "coroot-node-agent")
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-node-agent",
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}

	scheme := "http"
	port := cr.Spec.Service.Port
	if cr.Spec.TLS != nil || cr.Spec.HTTPDisabled {
		scheme = "https"
		port = cr.Spec.Service.HTTPSPort
	}
	corootURL := fmt.Sprintf("%s://%s-coroot.%s:%d", scheme, cr.Name, cr.Namespace, port)
	if cr.Spec.AgentsOnly != nil && cr.Spec.AgentsOnly.CorootURL != "" {
		corootURL = strings.TrimRight(cr.Spec.AgentsOnly.CorootURL, "/")
	}
	tlsSkipVerify := (cr.Spec.AgentsOnly != nil && cr.Spec.AgentsOnly.TLSSkipVerify) || (cr.Spec.NodeAgent.TLS != nil && cr.Spec.NodeAgent.TLS.TLSSkipVerify)
	var caSecret *corev1.SecretKeySelector
	if cr.Spec.NodeAgent.TLS != nil {
		caSecret = cr.Spec.NodeAgent.TLS.CASecret
	}
	scrapeInterval := cmp.Or(cr.Spec.MetricsRefreshInterval, corootv1.DefaultMetricRefreshInterval)
	env := []corev1.EnvVar{
		{Name: "SCRAPE_INTERVAL", Value: scrapeInterval},
	}

	env = append(env, envVarFromSecret("API_KEY", cr.Spec.ApiKeySecret, cr.Spec.ApiKey))

	if tlsSkipVerify {
		env = append(env, corev1.EnvVar{Name: "INSECURE_SKIP_VERIFY", Value: "true"})
	}
	if caSecret != nil {
		env = append(env, corev1.EnvVar{Name: "CA_FILE", Value: "/etc/coroot-ca/ca.crt"})
	}
	env = append(env, corev1.EnvVar{Name: "METRICS_ENDPOINT", Value: corootURL + "/v1/metrics"})
	if v := cr.Spec.NodeAgent.LogCollector.CollectLogBasedMetrics; v != nil && !*v {
		env = append(env, corev1.EnvVar{Name: "DISABLE_LOG_PARSING", Value: "true"})
	}
	if v := cr.Spec.NodeAgent.LogCollector.CollectLogEntries; v == nil || *v {
		env = append(env, corev1.EnvVar{Name: "LOGS_ENDPOINT", Value: corootURL + "/v1/logs"})
	}
	if v := cr.Spec.NodeAgent.EbpfTracer; v.Enabled == nil || *v.Enabled {
		env = append(env, corev1.EnvVar{Name: "TRACES_ENDPOINT", Value: corootURL + "/v1/traces"})
		if v.Sampling != "" {
			env = append(env, corev1.EnvVar{Name: "TRACES_SAMPLING", Value: v.Sampling})
		}
	}
	if v := cr.Spec.NodeAgent.EbpfProfiler.Enabled; v == nil || *v {
		env = append(env, corev1.EnvVar{Name: "PROFILES_ENDPOINT", Value: corootURL + "/v1/profiles"})
	}
	if v := cr.Spec.NodeAgent.TrackPublicNetworks; len(v) > 0 {
		env = append(env, corev1.EnvVar{Name: "TRACK_PUBLIC_NETWORK", Value: strings.Join(v, "\n")})
	}

	for _, e := range cr.Spec.NodeAgent.Env {
		env = append(env, e)
	}

	resources := cr.Spec.NodeAgent.Resources
	if resources.Requests == nil {
		resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("200Mi"),
		}
	}
	if resources.Limits == nil {
		resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		}
	}

	tolerations := cr.Spec.NodeAgent.Tolerations
	if len(tolerations) == 0 {
		tolerations = []corev1.Toleration{{Operator: corev1.TolerationOpExists}}
	}

	image := r.getAppImage(cr, AppNodeAgent)

	volumeMounts := []corev1.VolumeMount{
		{Name: "cgroupfs", MountPath: "/host/sys/fs/cgroup", ReadOnly: true},
		{Name: "tracefs", MountPath: "/sys/kernel/tracing"},
		{Name: "debugfs", MountPath: "/sys/kernel/debug"},
		{Name: "tmp", MountPath: "/tmp"},
	}
	volumes := []corev1.Volume{
		{
			Name: "cgroupfs",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/sys/fs/cgroup",
				},
			},
		},
		{
			Name: "tracefs",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/sys/kernel/tracing",
				},
			},
		},
		{
			Name: "debugfs",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/sys/kernel/debug",
				},
			},
		},
		{
			Name: "tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
	if caSecret != nil {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "ca", MountPath: "/etc/coroot-ca", ReadOnly: true})
		volumes = append(volumes, corev1.Volume{
			Name: "ca",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: caSecret.Name,
					Items:      []corev1.KeyToPath{{Key: caSecret.Key, Path: "ca.crt"}},
				},
			},
		})
	}

	ds.Spec = appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: ls,
		},
		UpdateStrategy: cr.Spec.NodeAgent.UpdateStrategy,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      ls,
				Annotations: cr.Spec.NodeAgent.PodAnnotations,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: cr.Name + "-node-agent",
				HostPID:            true,
				Tolerations:        tolerations,
				PriorityClassName:  cr.Spec.NodeAgent.PriorityClassName,
				NodeSelector:       cr.Spec.NodeAgent.NodeSelector,
				Affinity:           cr.Spec.NodeAgent.Affinity,
				ImagePullSecrets:   image.PullSecrets,
				Containers: []corev1.Container{
					{
						Name:            "node-agent",
						Image:           image.Name,
						ImagePullPolicy: image.PullPolicy,
						Args: []string{
							"--cgroupfs-root=/host/sys/fs/cgroup",
						},
						SecurityContext: &corev1.SecurityContext{Privileged: ptr.To(true)},
						Env:             env,
						Resources:       resources,
						VolumeMounts:    volumeMounts,
					},
				},
				Volumes: volumes,
			},
		},
	}

	return ds
}
