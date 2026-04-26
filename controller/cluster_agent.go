package controller

import (
	"cmp"
	"fmt"
	"os"
	"strings"

	corootv1 "github.io/coroot/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *CorootReconciler) clusterAgentClusterRoleBinding(cr *corootv1.Coroot) *rbacv1.ClusterRoleBinding {
	b := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cr.Name + "-cluster-agent",
			Labels: Labels(cr, "coroot-cluster-agent"),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      cr.Name + "-cluster-agent",
				Namespace: cr.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     cr.Name + "-cluster-agent",
		},
	}
	return b
}

func (r *CorootReconciler) clusterAgentClusterRole(cr *corootv1.Coroot) *rbacv1.ClusterRole {
	verbs := []string{"get", "list", "watch"}
	coreResources := []string{"namespaces", "nodes", "pods", "services", "endpoints", "persistentvolumeclaims", "persistentvolumes", "events"}
	if !strings.EqualFold(os.Getenv("DENY_GLOBAL_SECRETS"), "true") {
		coreResources = append(coreResources, "secrets")
	}
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cr.Name + "-cluster-agent",
			Labels: Labels(cr, "coroot-cluster-agent"),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: coreResources,
				Verbs:     verbs,
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "replicasets", "daemonsets", "statefulsets", "cronjobs"},
				Verbs:     verbs,
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"cronjobs", "jobs"},
				Verbs:     verbs,
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses", "volumeattachments"},
				Verbs:     verbs,
			},
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     verbs,
			},
			{
				APIGroups: []string{"source.toolkit.fluxcd.io", "kustomize.toolkit.fluxcd.io", "helm.toolkit.fluxcd.io", "notification.toolkit.fluxcd.io", "image.toolkit.fluxcd.io", "fluxcd.controlplane.io"},
				Resources: []string{"*"},
				Verbs:     verbs,
			},
		},
	}
	return role
}

func (r *CorootReconciler) clusterAgentDeployment(cr *corootv1.Coroot) *appsv1.Deployment {
	ls := Labels(cr, "coroot-cluster-agent")
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-cluster-agent",
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
		corootURL = cr.Spec.AgentsOnly.CorootURL
	}
	tlsSkipVerify := (cr.Spec.AgentsOnly != nil && cr.Spec.AgentsOnly.TLSSkipVerify) || (cr.Spec.ClusterAgent.TLS != nil && cr.Spec.ClusterAgent.TLS.TLSSkipVerify)
	var caSecret *corev1.SecretKeySelector
	if cr.Spec.ClusterAgent.TLS != nil {
		caSecret = cr.Spec.ClusterAgent.TLS.CASecret
	}
	scrapeInterval := cmp.Or(cr.Spec.MetricsRefreshInterval, corootv1.DefaultMetricRefreshInterval)
	env := []corev1.EnvVar{
		{Name: "COROOT_URL", Value: corootURL},
		{Name: "METRICS_SCRAPE_INTERVAL", Value: scrapeInterval},
		{Name: "KUBE_STATE_METRICS_ADDRESS", Value: "127.0.0.1:10302"},
	}

	env = append(env, envVarFromSecret("API_KEY", cr.Spec.ApiKeySecret, cr.Spec.ApiKey))

	if tlsSkipVerify {
		env = append(env, corev1.EnvVar{Name: "INSECURE_SKIP_VERIFY", Value: "true"})
	}
	if caSecret != nil {
		env = append(env, corev1.EnvVar{Name: "CA_FILE", Value: "/etc/coroot-ca/ca.crt"})
	}
	for _, e := range cr.Spec.ClusterAgent.Env {
		env = append(env, e)
	}
	image := r.getAppImage(cr, AppClusterAgent)
	ksmImage := r.getAppImage(cr, AppKubeStateMetrics)

	volumeMounts := []corev1.VolumeMount{
		{Name: "tmp", MountPath: "/tmp"},
	}
	volumes := []corev1.Volume{
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

	d.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: ls,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      ls,
				Annotations: cr.Spec.ClusterAgent.PodAnnotations,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: cr.Name + "-cluster-agent",
				SecurityContext:    nonRootSecurityContext,
				NodeSelector:       cr.Spec.ClusterAgent.NodeSelector,
				Affinity:           cr.Spec.ClusterAgent.Affinity,
				Tolerations:        cr.Spec.ClusterAgent.Tolerations,
				ImagePullSecrets:   image.PullSecrets,
				Containers: []corev1.Container{
					{
						Image:           image.Name,
						ImagePullPolicy: image.PullPolicy,
						Name:            "cluster-agent",
						Args: []string{
							"--listen=127.0.0.1:10301",
							"--metrics-wal-dir=/tmp",
						},
						Resources:    cr.Spec.ClusterAgent.Resources,
						VolumeMounts: volumeMounts,
						Env:          env,
					},
					{
						Image:           ksmImage.Name,
						ImagePullPolicy: ksmImage.PullPolicy,
						Name:            "kube-state-metrics",
						Args: []string{
							"--host=127.0.0.1",
							"--port=10302",
							"--resources=namespaces,nodes,daemonsets,deployments,cronjobs,jobs,persistentvolumeclaims,persistentvolumes,pods,replicasets,services,endpoints,statefulsets,storageclasses,volumeattachments",
							"--metric-labels-allowlist=pods=[*]",
							"--metric-annotations-allowlist=*=[coroot.com/application-category,coroot.com/custom-application-name]",
						},
						Resources: corev1.ResourceRequirements{
							Requests: cr.Spec.ClusterAgent.Resources.Requests,
							Limits:   cr.Spec.ClusterAgent.Resources.Limits,
						},
					},
				},
				Volumes: volumes,
			},
		},
	}

	return d
}
