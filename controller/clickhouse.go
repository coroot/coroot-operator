package controller

import (
	"bytes"
	"fmt"
	"text/template"

	corootv1 "github.io/coroot/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func clickhousePasswordSecret(cr *corootv1.Coroot) *corev1.SecretKeySelector {
	return &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: fmt.Sprintf("%s-clickhouse", cr.Name),
		},
		Key: "password",
	}
}

func (r *CorootReconciler) clickhouseService(cr *corootv1.Coroot) *corev1.Service {
	ls := Labels(cr, "clickhouse")
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-clickhouse", cr.Name),
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
				Port:       8123,
				TargetPort: intstr.FromString("http"),
			},
			{
				Name:       "tcp",
				Protocol:   corev1.ProtocolTCP,
				Port:       9000,
				TargetPort: intstr.FromString("tcp"),
			},
		},
	}

	return s
}

func (r *CorootReconciler) clickhouseServiceHeadless(cr *corootv1.Coroot) *corev1.Service {
	ls := Labels(cr, "clickhouse")
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-clickhouse-headless", cr.Name),
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}

	s.Spec = corev1.ServiceSpec{
		Selector:                 ls,
		ClusterIP:                corev1.ClusterIPNone,
		Type:                     corev1.ServiceTypeClusterIP,
		PublishNotReadyAddresses: true,
		Ports: []corev1.ServicePort{
			{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       8123,
				TargetPort: intstr.FromString("http"),
			},
			{
				Name:       "tcp",
				Protocol:   corev1.ProtocolTCP,
				Port:       9000,
				TargetPort: intstr.FromString("tcp"),
			},
		},
	}

	return s
}

func (r *CorootReconciler) clickhousePVCs(cr *corootv1.Coroot) []*corev1.PersistentVolumeClaim {
	ls := Labels(cr, "clickhouse")
	shards := cr.Spec.Clickhouse.Shards
	if shards == 0 {
		shards = 1
	}
	replicas := cr.Spec.Clickhouse.Replicas
	if replicas == 0 {
		replicas = 1
	}
	size := cr.Spec.Clickhouse.Storage.Size
	if size.IsZero() {
		size, _ = resource.ParseQuantity("100Gi")
	}

	var res []*corev1.PersistentVolumeClaim
	for shard := 0; shard < shards; shard++ {
		for replica := 0; replica < replicas; replica++ {
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:        fmt.Sprintf("data-%s-clickhouse-shard-%d-%d", cr.Name, shard, replica),
					Namespace:   cr.Namespace,
					Labels:      ls,
					Annotations: cr.Spec.Clickhouse.Storage.Annotations,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: size,
						},
					},
					StorageClassName: cr.Spec.Clickhouse.Storage.ClassName,
				},
			}
			res = append(res, pvc)
		}
	}
	return res
}

func (r *CorootReconciler) clickhouseStatefulSets(cr *corootv1.Coroot) []*appsv1.StatefulSet {
	ls := Labels(cr, "clickhouse")

	shards := cr.Spec.Clickhouse.Shards
	if shards <= 0 {
		shards = 1
	}
	replicas := int32(cr.Spec.Clickhouse.Replicas)
	if replicas <= 0 {
		replicas = 1
	}

	image := r.getAppImage(cr, AppClickhouse)

	var res []*appsv1.StatefulSet
	for shard := 0; shard < shards; shard++ {
		ss := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-clickhouse-shard-%d", cr.Name, shard),
				Namespace: cr.Namespace,
				Labels:    ls,
			},
		}

		ss.Spec = appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Replicas:    &replicas,
			ServiceName: fmt.Sprintf("%s-clickhouse-headless", cr.Name),
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "data",
						Namespace: cr.Namespace,
					},
					Spec: r.clickhousePVCs(cr)[0].Spec,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      ls,
					Annotations: cr.Spec.Clickhouse.PodAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: cr.Name + "-clickhouse",
					SecurityContext:    nonRootSecurityContext,
					NodeSelector:       cr.Spec.Clickhouse.NodeSelector,
					Affinity:           cr.Spec.Clickhouse.Affinity,
					Tolerations:        cr.Spec.Clickhouse.Tolerations,
					ImagePullSecrets:   image.PullSecrets,
					InitContainers: []corev1.Container{
						{
							Image:           image.Name,
							ImagePullPolicy: image.PullPolicy,
							Name:            "config",
							Command:         []string{"/bin/sh", "-c"},
							Args:            []string{clickhouseConfigCmd("/config/config.xml", cr, shards, int(replicas), clickhouseKeeperReplicas(cr))},
							VolumeMounts:    []corev1.VolumeMount{{Name: "config", MountPath: "/config"}},
							Resources:       cr.Spec.Clickhouse.Resources,
						},
					},
					Containers: []corev1.Container{
						{
							Image:           image.Name,
							ImagePullPolicy: image.PullPolicy,
							Name:            "clickhouse-server",
							Command:         []string{"clickhouse-server"},
							Args: []string{
								"--config-file=/config/config.xml",
							},
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: 8123, Protocol: corev1.ProtocolTCP},
								{Name: "tcp", ContainerPort: 9000, Protocol: corev1.ProtocolTCP},
							},
							Resources: cr.Spec.Clickhouse.Resources,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "config", MountPath: "/config"},
								{Name: "data", MountPath: "/var/lib/clickhouse"},
							},
							Env: []corev1.EnvVar{
								{Name: "CLICKHOUSE_SHARD_ID", Value: fmt.Sprintf("shard-%d", shard)},
								{Name: "CLICKHOUSE_REPLICA_ID", ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								}},
								{Name: "CLICKHOUSE_PASSWORD", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: clickhousePasswordSecret(cr)}},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{Path: "/ping", Port: intstr.FromString("http")},
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
					},
				},
			},
		}

		res = append(res, ss)
	}
	return res
}

func clickhouseConfigCmd(filename string, cr *corootv1.Coroot, shards, replicas, keepers int) string {
	params := struct {
		Namespace string
		Name      string
		Shards    []int
		Replicas  []int
		Keepers   []int
		LogLevel  string
	}{
		Namespace: cr.Namespace,
		Name:      cr.Name,
		LogLevel:  cr.Spec.Clickhouse.LogLevel,
	}
	for i := 0; i < shards; i++ {
		params.Shards = append(params.Shards, i)
	}
	for i := 0; i < replicas; i++ {
		params.Replicas = append(params.Replicas, i)
	}
	for i := 0; i < keepers; i++ {
		params.Keepers = append(params.Keepers, i)
	}
	if params.LogLevel == "" {
		params.LogLevel = "warning"
	}
	var out bytes.Buffer
	_ = clickhouseConfigTemplate.Execute(&out, params)
	return "cat <<EOF > " + filename + out.String() + "EOF"
}

var clickhouseConfigTemplate = template.Must(template.New("").Parse(`
<clickhouse>

<display_name from_env="CLICKHOUSE_REPLICA_ID"/>

<profiles>
    <default>
    </default>
</profiles>

<users>
    <default>
        <password from_env="CLICKHOUSE_PASSWORD"/>
    </default>
</users>

<logger>
    <console>1</console>
    <level>{{ .LogLevel }}</level>
</logger>

<listen_host>::</listen_host>
<listen_host>0.0.0.0</listen_host>
<listen_try>1</listen_try>

<http_port>8123</http_port>
<tcp_port>9000</tcp_port>
<interserver_http_port>9009</interserver_http_port>

<concurrent_threads_soft_limit_num>0</concurrent_threads_soft_limit_num>
<concurrent_threads_soft_limit_ratio_to_cores>2</concurrent_threads_soft_limit_ratio_to_cores>
<max_concurrent_queries>1000</max_concurrent_queries>
<max_server_memory_usage>0</max_server_memory_usage>
<max_thread_pool_size>10000</max_thread_pool_size>
<async_load_databases>true</async_load_databases>
<mlock_executable>true</mlock_executable>

<macros>
    <shard from_env="CLICKHOUSE_SHARD_ID"/>
    <replica from_env="CLICKHOUSE_REPLICA_ID"/>
</macros>

<remote_servers>
    <default>
        {{- range $shard := .Shards }}
        <shard>
            <internal_replication>true</internal_replication>
            {{- range $replica := $.Replicas }}
            <replica>
                <host>{{$.Name}}-clickhouse-shard-{{$shard}}-{{$replica}}.{{$.Name}}-clickhouse-headless.{{$.Namespace}}</host>
                <port>9000</port>
                <user>default</user>
                <password from_env="CLICKHOUSE_PASSWORD"/>
            </replica>
            {{- end }}
        </shard>
        {{- end }}
    </default>
</remote_servers>

<zookeeper>
    {{- range $keeper := .Keepers }}
    <node>
        <host>{{$.Name}}-clickhouse-keeper-{{$keeper}}.{{$.Name}}-clickhouse-keeper-headless.{{$.Namespace}}</host>
        <port>9181</port>
    </node>
    {{- end }}
</zookeeper>

<distributed_ddl>
    <path>/clickhouse/task_queue/ddl</path>
</distributed_ddl>

</clickhouse>
`))
