package controller

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	corootv1 "github.io/coroot/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

func (r *CorootReconciler) validateCoroot(ctx context.Context, cr *corootv1.Coroot, configEnvs ConfigEnvs) []string {
	logger := log.FromContext(ctx)
	var errors []string
	logErr := func(msg string, args ...any) {
		e := fmt.Sprintf(msg, args...)
		errors = append(errors, e)
		logger.Error(fmt.Errorf("misconfigured"), e)
	}

	if cr.Spec.Replicas > 1 && cr.Spec.Postgres == nil {
		logErr("Coroot requires Postgres to run multiple replicas. Falling back to 1 replica.")
		cr.Spec.Replicas = 1
	}

	if cr.Spec.Service.Port == 0 {
		cr.Spec.Service.Port = 8080
	}

	var err error
	for _, p := range cr.Spec.Projects {
		for i, k := range p.ApiKeys {
			if k.KeySecret != nil {
				r.CreateOrUpdateSecret(ctx, cr, "coroot", k.KeySecret.Name, k.KeySecret.Key, 32)
				p.ApiKeys[i].Key = configEnvs.Add(k.KeySecret)
				p.ApiKeys[i].KeySecret = nil
			}
		}
		if p.NotificationIntegrations != nil {
			if slack := p.NotificationIntegrations.Slack; slack != nil {
				if slack.TokenSecret != nil {
					if _, err = r.GetSecret(ctx, cr, slack.TokenSecret); err != nil {
						logErr("Failed to get Slack Token: %s.", err.Error())
					} else {
						slack.Token = configEnvs.Add(slack.TokenSecret)
					}
					slack.TokenSecret = nil
				}
				if slack.Token == "" {
					p.NotificationIntegrations.Slack = nil
				}
			}
			if teams := p.NotificationIntegrations.Teams; teams != nil {
				if teams.WebhookURLSecret != nil {
					if _, err = r.GetSecret(ctx, cr, teams.WebhookURLSecret); err != nil {
						logErr("Failed to get MS Teams Webhook URL: %s.", err.Error())
					} else {
						teams.WebhookURL = configEnvs.Add(teams.WebhookURLSecret)
					}
					teams.WebhookURLSecret = nil
				}
				if teams.WebhookURL == "" {
					p.NotificationIntegrations.Teams = nil
				}
			}
			if pagerduty := p.NotificationIntegrations.Pagerduty; pagerduty != nil {
				if pagerduty.IntegrationKeySecret != nil {
					if _, err = r.GetSecret(ctx, cr, pagerduty.IntegrationKeySecret); err != nil {
						logErr("Failed to get PagerDuty Integration Key: %s.", err.Error())
					} else {
						pagerduty.IntegrationKey = configEnvs.Add(pagerduty.IntegrationKeySecret)
					}
					pagerduty.IntegrationKeySecret = nil
				}
				if pagerduty.IntegrationKey == "" {
					p.NotificationIntegrations.Pagerduty = nil
				}
			}
			if opsgenie := p.NotificationIntegrations.Opsgenie; opsgenie != nil {
				if opsgenie.ApiKeySecret != nil {
					if _, err = r.GetSecret(ctx, cr, opsgenie.ApiKeySecret); err != nil {
						logErr("Failed to get Opsgenie API Key: %s", err.Error())
					} else {
						opsgenie.ApiKey = configEnvs.Add(opsgenie.ApiKeySecret)
					}
					opsgenie.ApiKeySecret = nil
				}
				if opsgenie.ApiKey == "" {
					p.NotificationIntegrations.Opsgenie = nil
				}
			}
			if webhook := p.NotificationIntegrations.Webhook; webhook != nil {
				if basicAuth := webhook.BasicAuth; basicAuth != nil {
					if basicAuth.PasswordSecret != nil {
						if _, err = r.GetSecret(ctx, cr, basicAuth.PasswordSecret); err != nil {
							logErr("Failed to get Webhook Basic Auth password: %s", err.Error())
						} else {
							basicAuth.Password = configEnvs.Add(basicAuth.PasswordSecret)
						}
						basicAuth.PasswordSecret = nil
					}
					if basicAuth.Username == "" || basicAuth.Password == "" {
						webhook.BasicAuth = nil
					}
				}
				if webhook.Incidents && webhook.IncidentTemplate == "" {
					webhook.Incidents = false
				}
				if webhook.Deployments && webhook.DeploymentTemplate == "" {
					webhook.Deployments = false
				}
				if webhook.Url == "" {
					p.NotificationIntegrations.Webhook = nil
				}
			}
		}
	}

	if ee := cr.Spec.EnterpriseEdition; ee != nil {
		if ee.LicenseKeySecret != nil {
			if _, err = r.GetSecret(ctx, cr, ee.LicenseKeySecret); err != nil {
				logErr("Failed to get License Key: %s.", err.Error())
			}
		}
		if sso := cr.Spec.SSO; sso != nil && sso.Enabled {
			if saml := sso.SAML; saml != nil {
				metadata := saml.Metadata
				if saml.MetadataSecret != nil {
					if metadata, err = r.GetSecret(ctx, cr, saml.MetadataSecret); err != nil {
						logErr("Failed to get SAML Identity Provider Metadata: %s", err.Error())
					} else {
						saml.Metadata = configEnvs.Add(saml.MetadataSecret)
					}
					saml.MetadataSecret = nil
				}
				if metadata != "" {
					if err = ValidateSamlIdentityProviderMetadata(metadata); err != nil {
						logErr("Invalid SAML Identity Provider Metadata: %s", err.Error())
						saml.Metadata = ""
					}
				}
			}
			if sso.SAML == nil || sso.SAML.Metadata == "" {
				sso.Enabled = false
			}
		}
		if ai := cr.Spec.AI; ai != nil && ai.Provider != "" {
			switch ai.Provider {
			case "anthropic":
				if anthropic := ai.Anthropic; anthropic != nil {
					if anthropic.APIKeySecret != nil {
						if _, err = r.GetSecret(ctx, cr, anthropic.APIKeySecret); err != nil {
							logErr("Failed to get Anthropic API Key: %s", err.Error())
						} else {
							anthropic.APIKey = configEnvs.Add(anthropic.APIKeySecret)
						}
						anthropic.APIKeySecret = nil
					}
					if anthropic.APIKey == "" {
						ai.Anthropic = nil
					}
				}
				if ai.Anthropic == nil {
					ai.Provider = ""
				}
			case "openai":
				if openai := ai.OpenAI; openai != nil {
					if openai.APIKeySecret != nil {
						if _, err = r.GetSecret(ctx, cr, openai.APIKeySecret); err != nil {
							logErr("Failed to get OpenAI API Key: %s", err.Error())
						} else {
							openai.APIKey = configEnvs.Add(openai.APIKeySecret)
						}
						openai.APIKeySecret = nil
					}
					if openai.APIKey == "" {
						ai.OpenAI = nil
					}
				}
				if ai.OpenAI == nil {
					ai.Provider = ""
				}
			case "openai_compatible":
				if openaiCompatible := ai.OpenAICompatible; openaiCompatible != nil {
					if openaiCompatible.APIKeySecret != nil {
						if _, err = r.GetSecret(ctx, cr, openaiCompatible.APIKeySecret); err != nil {
							logErr("Failed to get API Key: %s", err.Error())
						} else {
							openaiCompatible.APIKey = configEnvs.Add(openaiCompatible.APIKeySecret)
						}
						openaiCompatible.APIKeySecret = nil
					}
					if openaiCompatible.APIKey == "" {
						ai.OpenAICompatible = nil
					}
				}
				if ai.OpenAICompatible == nil {
					ai.Provider = ""
				}
			default:
				logErr("Unknown AI model provider: %s", ai.Provider)
				ai.Provider = ""
			}
		}
	}

	return errors
}

func (r *CorootReconciler) corootService(cr *corootv1.Coroot) *corev1.Service {
	ls := Labels(cr, "coroot")

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-coroot", cr.Name),
			Namespace:   cr.Namespace,
			Labels:      ls,
			Annotations: cr.Spec.Service.Annotations,
		},
	}

	s.Spec = corev1.ServiceSpec{
		Selector: ls,
		Type:     cr.Spec.Service.Type,
		Ports: []corev1.ServicePort{
			{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       cr.Spec.Service.Port,
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

func (r *CorootReconciler) corootStatefulSet(cr *corootv1.Coroot, configEnvs ConfigEnvs, configHash string) *appsv1.StatefulSet {
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

	for name, selector := range configEnvs {
		env = append(env, corev1.EnvVar{Name: name, ValueFrom: &corev1.EnvVarSource{SecretKeyRef: selector}})
	}

	for _, e := range cr.Spec.Env {
		env = append(env, e)
	}

	podAnnotations := cr.Spec.PodAnnotations
	if podAnnotations == nil {
		podAnnotations = map[string]string{}
	}
	podAnnotations["checksum/config"] = configHash

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
				Annotations: podAnnotations,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: cr.Name + "-coroot",
				SecurityContext:    nonRootSecurityContext,
				NodeSelector:       cr.Spec.NodeSelector,
				Affinity:           cr.Spec.Affinity,
				Tolerations:        cr.Spec.Tolerations,
				ImagePullSecrets:   image.PullSecrets,
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
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: cr.Name + "-coroot",
								},
							},
						},
					},
				},
			},
		},
	}

	return ss
}

func (r *CorootReconciler) corootConfigMap(ctx context.Context, cr *corootv1.Coroot) (*corev1.ConfigMap, string) {
	ls := Labels(cr, "coroot")
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-coroot",
			Namespace: cr.Namespace,
			Labels:    ls,
		},
		BinaryData: map[string][]byte{},
	}

	var cfg = struct {
		Projects []corootv1.ProjectSpec `json:"projects,omitempty"`
		SSO      *corootv1.SSOSpec      `json:"sso,omitempty"`
		AI       *corootv1.AISpec       `json:"ai,omitempty"`
	}{
		Projects: cr.Spec.Projects,
		SSO:      cr.Spec.SSO,
		AI:       cr.Spec.AI,
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to marshal coroot config")
	}
	cm.BinaryData["config.yaml"] = data
	hash := sha256.New()
	hash.Write(data)
	return cm, hex.EncodeToString(hash.Sum(nil))
}
