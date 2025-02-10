package controller

import (
	"bytes"
	"fmt"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/utils/ptr"
	"strings"
	"text/template"

	corootv1 "github.io/coroot/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *CorootReconciler) corootService(cr *corootv1.Coroot) *corev1.Service {
	ls := Labels(cr, "coroot")

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-coroot", cr.Name),
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}

	port := cr.Spec.Service.Port
	if port == 0 {
		port = 8080
	}
	s.Spec = corev1.ServiceSpec{
		Selector: ls,
		Type:     cr.Spec.Service.Type,
		Ports: []corev1.ServicePort{
			{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       port,
				TargetPort: intstr.FromString("http"),
				NodePort:   cr.Spec.Service.NodePort,
			},
		},
	}

	return s
}

func (r *CorootReconciler) corootPVCs(cr *corootv1.Coroot) []*corev1.PersistentVolumeClaim {
	ls := Labels(cr, "coroot")

	size := cr.Spec.Storage.Size
	if size.IsZero() {
		size, _ = resource.ParseQuantity("10Gi")
	}
	replicas := cr.Spec.Replicas
	if replicas == 0 {
		replicas = 1
	}

	var res []*corev1.PersistentVolumeClaim
	for replica := 0; replica < replicas; replica++ {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("data-%s-coroot-%d", cr.Name, replica),
				Namespace: cr.Namespace,
				Labels:    ls,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: size,
					},
				},
				StorageClassName: cr.Spec.Storage.ClassName,
			},
		}
		res = append(res, pvc)
	}
	return res
}

func (r *CorootReconciler) corootIngress(cr *corootv1.Coroot) *networkingv1.Ingress {
	ls := Labels(cr, "ingress")
	i := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}
	if cr.Spec.Ingress == nil {
		return i
	}
	i.Annotations = cr.Spec.Ingress.Annotations
	path := cr.Spec.Ingress.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	i.Spec = networkingv1.IngressSpec{
		IngressClassName: cr.Spec.Ingress.ClassName,
		Rules: []networkingv1.IngressRule{{
			Host: cr.Spec.Ingress.Host,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{{
						Path:     path,
						PathType: ptr.To(networkingv1.PathTypePrefix),
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: fmt.Sprintf("%s-coroot", cr.Name),
								Port: networkingv1.ServiceBackendPort{
									Name: "http",
								},
							},
						},
					}},
				},
			},
		}},
	}
	if cr.Spec.Ingress.TLS != nil {
		i.Spec.TLS = append(i.Spec.TLS, *cr.Spec.Ingress.TLS)
	}
	return i
}

func (r *CorootReconciler) corootDeployment(cr *corootv1.Coroot) *appsv1.Deployment {
	ls := Labels(cr, "coroot")
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-coroot",
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}
	return d
}

func (r *CorootReconciler) corootStatefulSet(cr *corootv1.Coroot) *appsv1.StatefulSet {
	ls := Labels(cr, "coroot")
	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-coroot",
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}

	refreshInterval := cr.Spec.MetricsRefreshInterval.Duration.String()
	if cr.Spec.MetricsRefreshInterval.Duration == 0 {
		refreshInterval = corootv1.DefaultMetricRefreshInterval
	}

	env := []corev1.EnvVar{
		{Name: "GLOBAL_REFRESH_INTERVAL", Value: refreshInterval},
		{Name: "GLOBAL_PROMETHEUS_URL", Value: fmt.Sprintf("http://%s-prometheus.%s:9090", cr.Name, cr.Namespace)},
		{Name: "INSTALLATION_TYPE", Value: "k8s-operator"},
	}
	if cr.Spec.CacheTTL.Duration > 0 {
		env = append(env, corev1.EnvVar{Name: "CACHE_TTL", Value: cr.Spec.CacheTTL.Duration.String()})
	}
	if cr.Spec.AuthAnonymousRole != "" {
		env = append(env, corev1.EnvVar{Name: "AUTH_ANONYMOUS_ROLE", Value: cr.Spec.AuthAnonymousRole})
	}
	if cr.Spec.AuthBootstrapAdminPassword != "" {
		env = append(env, corev1.EnvVar{Name: "AUTH_BOOTSTRAP_ADMIN_PASSWORD", Value: cr.Spec.AuthBootstrapAdminPassword})
	}
	for _, e := range cr.Spec.Env {
		env = append(env, e)
	}

	var image string
	if cr.Spec.EnterpriseEdition != nil {
		image = r.getAppImage(cr, AppCorootEE)
		env = append(env, corev1.EnvVar{Name: "LICENSE_KEY", Value: cr.Spec.EnterpriseEdition.LicenseKey})
	} else {
		image = r.getAppImage(cr, AppCorootCE)
	}

	if ec := cr.Spec.ExternalClickhouse; ec != nil {
		env = append(env,
			corev1.EnvVar{Name: "GLOBAL_CLICKHOUSE_ADDRESS", Value: ec.Address},
			corev1.EnvVar{Name: "GLOBAL_CLICKHOUSE_USER", Value: ec.User},
			corev1.EnvVar{Name: "GLOBAL_CLICKHOUSE_INITIAL_DATABASE", Value: ec.Database},
		)
		password := corev1.EnvVar{Name: "GLOBAL_CLICKHOUSE_PASSWORD"}
		if ec.PasswordSecret != nil {
			password.ValueFrom = &corev1.EnvVarSource{SecretKeyRef: ec.PasswordSecret}
		} else {
			password.Value = ec.Password
		}
		env = append(env, password)
	} else {
		env = append(env,
			corev1.EnvVar{
				Name:  "GLOBAL_CLICKHOUSE_ADDRESS",
				Value: fmt.Sprintf("%s-clickhouse.%s:9000", cr.Name, cr.Namespace),
			},
			corev1.EnvVar{Name: "GLOBAL_CLICKHOUSE_USER", Value: "default"},
			corev1.EnvVar{Name: "GLOBAL_CLICKHOUSE_PASSWORD", ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: secretKeySelector(fmt.Sprintf("%s-clickhouse", cr.Name), "password")}},
			corev1.EnvVar{Name: "GLOBAL_CLICKHOUSE_INITIAL_DATABASE", Value: "default"},
		)
	}

	if p := cr.Spec.Postgres; p != nil {
		password := corev1.EnvVar{Name: "PG_PASSWORD"}
		if p.PasswordSecret != nil {
			password.ValueFrom = &corev1.EnvVarSource{SecretKeyRef: p.PasswordSecret}
		} else {
			password.Value = p.Password
		}
		env = append(env, password)
		env = append(env, corev1.EnvVar{Name: "PG_CONNECTION_STRING", Value: postgresConnectionString(*p, "PG_PASSWORD")})
	}

	if cr.Spec.Ingress != nil && cr.Spec.Ingress.Path != "" {
		env = append(env, corev1.EnvVar{Name: "URL_BASE_PATH", Value: cr.Spec.Ingress.Path})
	}

	replicas := int32(cr.Spec.Replicas)
	if replicas <= 0 {
		replicas = 1
	}

	ss.Spec = appsv1.StatefulSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: ls,
		},
		Replicas: &replicas,
		VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "data",
				Namespace: cr.Namespace,
			},
		}},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      ls,
				Annotations: cr.Spec.PodAnnotations,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: cr.Name + "-coroot",
				SecurityContext:    nonRootSecurityContext,
				Affinity:           cr.Spec.Affinity,
				Tolerations:        cr.Spec.Tolerations,
				InitContainers: []corev1.Container{
					{
						Image:        image,
						Name:         "config",
						Command:      []string{"/bin/sh", "-c"},
						Args:         []string{corootConfigCmd("/config/config.yaml", cr)},
						VolumeMounts: []corev1.VolumeMount{{Name: "config", MountPath: "/config"}},
					},
				},
				Containers: []corev1.Container{
					{
						Image: image,
						Name:  "coroot",
						Args: []string{
							"--config=/config/config.yaml",
							"--listen=:8080",
							"--data-dir=/data",
						},
						Env: env,
						Ports: []corev1.ContainerPort{
							{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "config", MountPath: "/config"},
							{Name: "data", MountPath: "/data"},
						},
						Resources: cr.Spec.Resources,
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromString("http")},
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
	}

	return ss
}

func corootConfigCmd(filename string, cr *corootv1.Coroot) string {
	var out bytes.Buffer
	_ = corootConfigTemplate.Execute(&out, cr.Spec)
	return "cat <<EOF > " + filename + out.String() + "EOF"
}

var corootConfigTemplate = template.Must(template.New("").Parse(`
projects:
{{- range $project := .Projects }}
- name: {{ $project.Name }}
  api_keys:
  {{- range $key := $project.ApiKeys }}
  - key: {{ $key.Key }}
    description: {{ $key.Description }}
  {{- end }}
{{- end }}
`))
