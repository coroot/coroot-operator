package controller

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"strings"
	"text/template"

	corootv1 "github.io/coroot/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *CorootReconciler) corootValidate(ctx context.Context, cr *corootv1.Coroot) {
	logger := ctrl.Log.WithValues("namespace", cr.Namespace, "name", cr.Name)

	if cr.Spec.Replicas > 1 && cr.Spec.Postgres == nil {
		logger.Error(fmt.Errorf("postgres not configured"), "Coroot requires Postgres to run multiple replicas (will run only one replica)")
		cr.Spec.Replicas = 1
	}

	for _, p := range cr.Spec.Projects {
		for i, k := range p.ApiKeys {
			if k.KeySecret != nil {
				p.ApiKeys[i].Key = r.CreateOrUpdateSecret(ctx, cr, "coroot", k.KeySecret.Name, k.KeySecret.Key, 32)
			}
		}
	}
}

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
		Annotations: cr.Spec.Service.Annotations,
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

	refreshInterval := cmp.Or(cr.Spec.MetricsRefreshInterval, corootv1.DefaultMetricRefreshInterval)

	env := []corev1.EnvVar{
		{Name: "GLOBAL_REFRESH_INTERVAL", Value: refreshInterval},
		{Name: "INSTALLATION_TYPE", Value: "k8s-operator"},
	}
	if cr.Spec.CacheTTL != "" {
		env = append(env, corev1.EnvVar{Name: "CACHE_TTL", Value: cr.Spec.CacheTTL})
	}
	if cr.Spec.TracesTTL != "" {
		env = append(env, corev1.EnvVar{Name: "TRACES_TTL", Value: cr.Spec.TracesTTL})
	}
	if cr.Spec.LogsTTL != "" {
		env = append(env, corev1.EnvVar{Name: "LOGS_TTL", Value: cr.Spec.LogsTTL})
	}
	if cr.Spec.ProfilesTTL != "" {
		env = append(env, corev1.EnvVar{Name: "PROFILES_TTL", Value: cr.Spec.ProfilesTTL})
	}
	if cr.Spec.AuthAnonymousRole != "" {
		env = append(env, corev1.EnvVar{Name: "AUTH_ANONYMOUS_ROLE", Value: cr.Spec.AuthAnonymousRole})
	}
	if cr.Spec.AuthBootstrapAdminPassword != "" {
		env = append(env, corev1.EnvVar{Name: "AUTH_BOOTSTRAP_ADMIN_PASSWORD", Value: cr.Spec.AuthBootstrapAdminPassword})
	}

	image := r.getAppImage(cr, AppCorootCE)
	if ee := cr.Spec.EnterpriseEdition; ee != nil {
		image = r.getAppImage(cr, AppCorootEE)
		licenseKey := corev1.EnvVar{Name: "LICENSE_KEY"}
		if ee.LicenseKeySecret != nil {
			licenseKey.ValueFrom = &corev1.EnvVarSource{SecretKeyRef: ee.LicenseKeySecret}
		} else {
			licenseKey.Value = ee.LicenseKey
		}
		env = append(env, licenseKey)
	}

	if ep := cr.Spec.ExternalPrometheus; ep != nil {
		env = append(env,
			corev1.EnvVar{Name: "GLOBAL_PROMETHEUS_URL", Value: ep.URL},
		)
		if ep.TLSSkipVerify {
			env = append(env, corev1.EnvVar{Name: "GLOBAL_PROMETHEUS_TLS_SKIP_VERIFY", Value: "true"})
		}
		if customHeaders := ep.CustomHeaders; len(customHeaders) > 0 {
			var headers []string
			for name, value := range customHeaders {
				headers = append(headers, fmt.Sprintf("%s=%s", name, value))
			}
			env = append(env, corev1.EnvVar{Name: "GLOBAL_PROMETHEUS_CUSTOM_HEADERS", Value: strings.Join(headers, "\n")})
		}
		if basicAuth := ep.BasicAuth; basicAuth != nil {
			env = append(env, corev1.EnvVar{Name: "GLOBAL_PROMETHEUS_USER", Value: basicAuth.Username})
			password := corev1.EnvVar{Name: "GLOBAL_PROMETHEUS_PASSWORD"}
			if basicAuth.PasswordSecret != nil {
				password.ValueFrom = &corev1.EnvVarSource{SecretKeyRef: basicAuth.PasswordSecret}
			} else {
				password.Value = basicAuth.Password
			}
			env = append(env, password)
		}
		if ep.RemoteWriteUrl != "" {
			env = append(env, corev1.EnvVar{Name: "GLOBAL_PROMETHEUS_REMOTE_WRITE_URL", Value: ep.RemoteWriteUrl})
		}
	} else {
		env = append(env,
			corev1.EnvVar{Name: "GLOBAL_PROMETHEUS_URL", Value: fmt.Sprintf("http://%s-prometheus.%s:9090", cr.Name, cr.Namespace)},
		)
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

	for _, e := range cr.Spec.Env {
		env = append(env, e)
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
				NodeSelector:       cr.Spec.NodeSelector,
				Affinity:           cr.Spec.Affinity,
				Tolerations:        cr.Spec.Tolerations,
				ImagePullSecrets:   image.PullSecrets,
				InitContainers: []corev1.Container{
					{
						Image:           image.Name,
						ImagePullPolicy: image.PullPolicy,
						Name:            "config",
						Command:         []string{"/bin/sh", "-c"},
						Args:            []string{corootConfigCmd("/config/config.yaml", cr)},
						VolumeMounts:    []corev1.VolumeMount{{Name: "config", MountPath: "/config"}},
					},
				},
				Containers: []corev1.Container{
					{
						Image:           image.Name,
						ImagePullPolicy: image.PullPolicy,
						Name:            "coroot",
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
