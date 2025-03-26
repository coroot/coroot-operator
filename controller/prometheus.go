package controller

import (
	"fmt"

	corootv1 "github.io/coroot/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	PrometheusDefaultRetention = "2d"
)

func (r *CorootReconciler) prometheusService(cr *corootv1.Coroot) *corev1.Service {
	ls := Labels(cr, "prometheus")
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-prometheus", cr.Name),
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}

	s.Spec = corev1.ServiceSpec{
		Selector: ls,
		Type:     corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{
			{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       9090,
				TargetPort: intstr.FromString("http"),
			},
		},
	}

	return s
}

func (r *CorootReconciler) prometheusPVC(cr *corootv1.Coroot) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "data-" + cr.Name + "-prometheus",
			Namespace: cr.Namespace,
			Labels:    Labels(cr, "prometheus"),
		},
	}

	size := cr.Spec.Prometheus.Storage.Size
	if size.IsZero() {
		size, _ = resource.ParseQuantity("10Gi")
	}
	pvc.Spec = corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: size,
			},
		},
		StorageClassName: cr.Spec.Prometheus.Storage.ClassName,
	}

	return pvc
}

func (r *CorootReconciler) prometheusDeployment(cr *corootv1.Coroot) *appsv1.Deployment {
	ls := Labels(cr, "prometheus")
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-prometheus",
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}

	retention := PrometheusDefaultRetention
	if cr.Spec.Prometheus.Retention != "" {
		retention = cr.Spec.Prometheus.Retention
	}

	image := r.getAppImage(cr, AppPrometheus)

	d.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: ls,
		},
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      ls,
				Annotations: cr.Spec.Prometheus.PodAnnotations,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: cr.Name + "-prometheus",
				SecurityContext:    nonRootSecurityContext,
				NodeSelector:       cr.Spec.Prometheus.NodeSelector,
				Affinity:           cr.Spec.Prometheus.Affinity,
				Tolerations:        cr.Spec.Prometheus.Tolerations,
				ImagePullSecrets:   image.PullSecrets,
				Containers: []corev1.Container{
					{
						Image:           image.Name,
						ImagePullPolicy: image.PullPolicy,
						Name:            "prometheus",
						Command:         []string{"prometheus"},
						Args: []string{
							"--config.file=/etc/prometheus/prometheus.yml",
							"--web.listen-address=0.0.0.0:9090",
							"--storage.tsdb.path=/data",
							"--storage.tsdb.out-of-order-time-window=5m",
							"--storage.tsdb.retention.time=" + retention,
							"--web.enable-remote-write-receiver",
							"--query.max-samples=100000000",
						},
						Ports: []corev1.ContainerPort{
							{Name: "http", ContainerPort: 9090, Protocol: corev1.ProtocolTCP},
						},
						Resources: cr.Spec.Prometheus.Resources,
						VolumeMounts: []corev1.VolumeMount{
							{Name: "config", MountPath: "/config"},
							{Name: "data", MountPath: "/data"},
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/-/healthy", Port: intstr.FromString("http")},
							},
							TimeoutSeconds: 10,
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
					{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "data-" + cr.Name + "-prometheus",
							},
						},
					},
				},
			},
		},
	}

	return d
}
