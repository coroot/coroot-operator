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

func clickhouseKeeperReplicas(cr *corootv1.Coroot) int {
	if cr.Spec.Clickhouse.Keeper.Replicas > 0 {
		return cr.Spec.Clickhouse.Keeper.Replicas
	}
	return 3
}

func (r *CorootReconciler) clickhouseKeeperServiceHeadless(cr *corootv1.Coroot) *corev1.Service {
	ls := Labels(cr, "clickhouse-keeper")
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-clickhouse-keeper-headless", cr.Name),
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
				Name:       "client",
				Protocol:   corev1.ProtocolTCP,
				Port:       9181,
				TargetPort: intstr.FromString("client"),
			},
			{
				Name:       "inter",
				Protocol:   corev1.ProtocolTCP,
				Port:       9234,
				TargetPort: intstr.FromString("inter"),
			},
		},
	}

	return s
}

func (r *CorootReconciler) clickhouseKeeperPVCs(cr *corootv1.Coroot) []*corev1.PersistentVolumeClaim {
	ls := Labels(cr, "clickhouse-keeper")
	size := cr.Spec.Clickhouse.Keeper.Storage.Size
	if size.IsZero() {
		size, _ = resource.ParseQuantity("10Gi")
	}

	var res []*corev1.PersistentVolumeClaim
	for replica := 0; replica < clickhouseKeeperReplicas(cr); replica++ {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("data-%s-clickhouse-keeper-%d", cr.Name, replica),
				Namespace:   cr.Namespace,
				Labels:      ls,
				Annotations: cr.Spec.Clickhouse.Keeper.Storage.Annotations,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: size,
					},
				},
				StorageClassName: cr.Spec.Clickhouse.Keeper.Storage.ClassName,
			},
		}
		res = append(res, pvc)
	}
	return res
}

func (r *CorootReconciler) clickhouseKeeperStatefulSet(cr *corootv1.Coroot) *appsv1.StatefulSet {
	ls := Labels(cr, "clickhouse-keeper")
	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-clickhouse-keeper",
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}

	replicas := int32(clickhouseKeeperReplicas(cr))

	image := r.getAppImage(cr, AppClickhouseKeeper)

	ss.Spec = appsv1.StatefulSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: ls,
		},
		Replicas:    &replicas,
		ServiceName: fmt.Sprintf("%s-clickhouse-keeper-headless", cr.Name),
		VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "data",
				Namespace: cr.Namespace,
			},
		}},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      ls,
				Annotations: cr.Spec.Clickhouse.Keeper.PodAnnotations,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: cr.Name + "-clickhouse-keeper",
				SecurityContext:    nonRootSecurityContext,
				NodeSelector:       cr.Spec.Clickhouse.Keeper.NodeSelector,
				Affinity:           cr.Spec.Clickhouse.Keeper.Affinity,
				Tolerations:        cr.Spec.Clickhouse.Keeper.Tolerations,
				ImagePullSecrets:   image.PullSecrets,
				InitContainers: []corev1.Container{
					{
						Image:           image.Name,
						ImagePullPolicy: image.PullPolicy,
						Name:            "config",
						Command:         []string{"/bin/sh", "-c"},
						Args:            []string{clickhouseKeeperConfigCmd("/config/config.xml", cr, int(replicas))},
						VolumeMounts:    []corev1.VolumeMount{{Name: "config", MountPath: "/config"}},
						Resources:       cr.Spec.Clickhouse.Keeper.Resources,
					},
				},
				Containers: []corev1.Container{
					{
						Image:           image.Name,
						ImagePullPolicy: image.PullPolicy,
						Name:            "clickhouse-keeper",
						Command:         []string{"clickhouse-keeper"},
						Args: []string{
							"--config-file=/config/config.xml",
						},
						Ports: []corev1.ContainerPort{
							{Name: "client", ContainerPort: 9181, Protocol: corev1.ProtocolTCP},
							{Name: "control", ContainerPort: 9182, Protocol: corev1.ProtocolTCP},
							{Name: "inter", ContainerPort: 9234, Protocol: corev1.ProtocolTCP},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "config", MountPath: "/config"},
							{Name: "data", MountPath: "/var/lib/clickhouse-keeper"},
						},
						Resources: cr.Spec.Clickhouse.Keeper.Resources,
						//ReadinessProbe: &corev1.Probe{
						//	ProbeHandler: corev1.ProbeHandler{
						//		//HTTPGet: &corev1.HTTPGetAction{Path: "/ready", Port: intstr.FromString("control")},
						//		//TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("client")},
						//	},
						//	TimeoutSeconds: 10,
						//},
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

func clickhouseKeeperConfigCmd(filename string, cr *corootv1.Coroot, replicas int) string {
	params := struct {
		Namespace string
		Name      string
		Ids       []int
	}{
		Namespace: cr.Namespace,
		Name:      cr.Name,
	}
	for id := 0; id < replicas; id++ {
		params.Ids = append(params.Ids, id)
	}
	var out bytes.Buffer
	_ = clickhouseKeeperConfigTemplate.Execute(&out, params)
	return "cat <<EOF | sed s/SERVER_ID/$(echo $HOSTNAME | sed -E 's/.*-([0-9]+)$/\\1/')/ > " + filename + out.String() + "EOF"
}

var clickhouseKeeperConfigTemplate = template.Must(template.New("").Parse(`
<clickhouse>
<logger>
    <console>1</console>
    <level>information</level>
</logger>

<listen_host>::</listen_host>
<listen_host>0.0.0.0</listen_host>
<listen_try>1</listen_try>

<keeper_server>
    <tcp_port>9181</tcp_port>
    <server_id>SERVER_ID</server_id>
    <log_storage_path>/var/lib/clickhouse-keeper/coordination/log</log_storage_path>
    <snapshot_storage_path>/var/lib/clickhouse-keeper/coordination/snapshots</snapshot_storage_path>

    <coordination_settings>
        <operation_timeout_ms>10000</operation_timeout_ms>
        <session_timeout_ms>30000</session_timeout_ms>
        <raft_logs_level>trace</raft_logs_level>
    </coordination_settings>

    <feature_flags>
        <check_not_exists>0</check_not_exists>
        <create_if_not_exists>0</create_if_not_exists>
    </feature_flags>

    <http_control>
        <port>9182</port>
        <readiness>
            <endpoint>/ready</endpoint>
        </readiness>
    </http_control>

    <raft_configuration>
        {{- range $id := .Ids }}
        <server>
            <id>{{$id}}</id>
            <hostname>{{$.Name}}-clickhouse-keeper-{{$id}}.{{$.Name}}-clickhouse-keeper-headless.{{$.Namespace}}</hostname>
            <port>9234</port>
        </server>
        {{- end }}
    </raft_configuration>
</keeper_server>
</clickhouse>
`))
