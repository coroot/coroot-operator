package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	corootv1 "github.io/coroot/operator/api/v1"
	"golang.org/x/exp/maps"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	AppVersionsUpdateInterval = time.Hour
)

type CorootReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	instances     map[ctrl.Request]bool
	instancesLock sync.Mutex

	versions     map[App]string
	versionsLock sync.Mutex

	deploymentDeleted bool
}

func NewCorootReconciler(mgr ctrl.Manager) *CorootReconciler {
	r := &CorootReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),

		instances: map[ctrl.Request]bool{},
		versions:  map[App]string{},
	}

	r.fetchAppVersions()
	go func() {
		for range time.Tick(AppVersionsUpdateInterval) {
			r.fetchAppVersions()
			r.instancesLock.Lock()
			instances := maps.Keys(r.instances)
			r.instancesLock.Unlock()
			for _, i := range instances {
				ctx, cancel := context.WithTimeout(context.Background(), AppVersionsUpdateInterval)
				_, _ = r.Reconcile(ctx, i)
				cancel()
			}
		}
	}()

	return r
}

// +kubebuilder:rbac:groups=coroot.com,resources=coroots,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coroot.com,resources=coroots/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coroot.com,resources=coroots/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces;nodes;pods;endpoints;persistentvolumes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services;persistentvolumeclaims;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments;replicasets;daemonsets;statefulsets;cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs;jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses;volumeattachments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=use

func (r *CorootReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.Log.WithValues("namespace", req.Namespace, "name", req.Name)

	cr := &corootv1.Coroot{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			r.instancesLock.Lock()
			if r.instances[req] {
				logger.Info("Coroot has been deleted")
				delete(r.instances, req)
			}
			if len(r.instances) == 0 {
				cr = &corootv1.Coroot{}
				cr.Name = req.Name
				cr.Namespace = req.Namespace
				_ = r.Delete(ctx, r.clusterAgentClusterRoleBinding(cr))
				_ = r.Delete(ctx, r.clusterAgentClusterRole(cr))
			}
			r.instancesLock.Unlock()
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	r.instancesLock.Lock()
	r.instances[req] = true
	r.instancesLock.Unlock()

	r.CreateOrUpdateRole(ctx, cr, r.openshiftSCCRole(cr, sccNonroot))
	r.CreateOrUpdateRole(ctx, cr, r.openshiftSCCRole(cr, sccPrivileged))

	r.CreateOrUpdateServiceAccount(ctx, cr, "node-agent", sccPrivileged)
	r.CreateOrUpdateDaemonSet(ctx, cr, r.nodeAgentDaemonSet(cr))

	r.CreateOrUpdateServiceAccount(ctx, cr, "cluster-agent", sccNonroot)
	r.CreateOrUpdateClusterRole(ctx, cr, r.clusterAgentClusterRole(cr))
	r.CreateOrUpdateClusterRoleBinding(ctx, cr, r.clusterAgentClusterRoleBinding(cr))
	r.CreateOrUpdateDeployment(ctx, cr, r.clusterAgentDeployment(cr))

	if cr.Spec.AgentsOnly != nil {
		// TODO: delete
		return ctrl.Result{}, nil
	}

	r.corootValidate(ctx, cr)

	r.CreateOrUpdateServiceAccount(ctx, cr, "coroot", sccNonroot)
	for _, pvc := range r.corootPVCs(cr) {
		r.CreateOrUpdatePVC(ctx, cr, pvc, cr.Spec.Storage.ReclaimPolicy)
	}
	r.CreateOrUpdateStatefulSet(ctx, cr, r.corootStatefulSet(cr))
	r.CreateOrUpdateService(ctx, cr, r.corootService(cr))
	if !r.deploymentDeleted {
		_ = r.Delete(ctx, r.corootDeployment(cr))
		r.deploymentDeleted = true
	}
	r.CreateOrUpdateIngress(ctx, cr, r.corootIngress(cr), cr.Spec.Ingress == nil)

	if cr.Spec.ExternalPrometheus == nil {
		r.CreateOrUpdateServiceAccount(ctx, cr, "prometheus", sccNonroot)
		r.CreateOrUpdatePVC(ctx, cr, r.prometheusPVC(cr), cr.Spec.Prometheus.Storage.ReclaimPolicy)
		r.CreateOrUpdateDeployment(ctx, cr, r.prometheusDeployment(cr))
		r.CreateOrUpdateService(ctx, cr, r.prometheusService(cr))
	} else {
		// TODO: delete
	}

	if cr.Spec.ExternalClickhouse == nil {
		r.CreateOrUpdateSecret(ctx, cr, "clickhouse", fmt.Sprintf("%s-clickhouse", cr.Name), "password", 16)

		r.CreateOrUpdateServiceAccount(ctx, cr, "clickhouse-keeper", sccNonroot)
		r.CreateOrUpdateService(ctx, cr, r.clickhouseKeeperServiceHeadless(cr))
		for _, pvc := range r.clickhouseKeeperPVCs(cr) {
			r.CreateOrUpdatePVC(ctx, cr, pvc, cr.Spec.Clickhouse.Keeper.Storage.ReclaimPolicy)
		}
		r.CreateOrUpdateStatefulSet(ctx, cr, r.clickhouseKeeperStatefulSet(cr))

		r.CreateOrUpdateServiceAccount(ctx, cr, "clickhouse", sccNonroot)
		r.CreateOrUpdateService(ctx, cr, r.clickhouseServiceHeadless(cr))
		for _, pvc := range r.clickhousePVCs(cr) {
			r.CreateOrUpdatePVC(ctx, cr, pvc, cr.Spec.Clickhouse.Storage.ReclaimPolicy)
		}
		for _, clickhouse := range r.clickhouseStatefulSets(cr) {
			r.CreateOrUpdateStatefulSet(ctx, cr, clickhouse)
		}
		r.CreateOrUpdateService(ctx, cr, r.clickhouseService(cr))
	} else {
		// TODO: delete
	}

	return ctrl.Result{}, nil
}

func (r *CorootReconciler) CreateOrUpdate(ctx context.Context, cr *corootv1.Coroot, obj client.Object, delete, retain bool, mutateF controllerutil.MutateFn) {
	logger := ctrl.Log.WithValues("namespace", obj.GetNamespace(), "name", obj.GetName(), "type", fmt.Sprintf("%T", obj))
	if delete {
		err := r.Delete(ctx, obj)
		if err == nil {
			logger.Info("deleted")
		}
		return
	}
	f := func() error {
		if retain {
			_ = controllerutil.RemoveControllerReference(cr, obj, r.Scheme)
		} else {
			_ = controllerutil.SetControllerReference(cr, obj, r.Scheme)
		}
		if mutateF != nil {
			return mutateF()
		}
		return nil
	}
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, obj, f)
	if err != nil {
		logger.Error(err, "failed to create or update")
		return
	}
	if res != controllerutil.OperationResultNone {
		logger.Info(fmt.Sprintf("%s", res))
	}
}

func (r *CorootReconciler) CreateOrUpdateSecret(ctx context.Context, cr *corootv1.Coroot, component, name, key string, length int) string {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    Labels(cr, component),
		},
	}
	var data string
	r.CreateOrUpdate(ctx, cr, s, false, false, func() error {
		if s.Data == nil {
			s.Data = map[string][]byte{}
		}
		if d, ok := s.Data[key]; ok {
			data = string(d)
		} else {
			data = RandomString(length)
			s.Data[key] = []byte(data)
		}
		return nil
	})
	return data
}

func (r *CorootReconciler) CreateOrUpdateDeployment(ctx context.Context, cr *corootv1.Coroot, d *appsv1.Deployment) {
	spec := d.Spec
	r.CreateOrUpdate(ctx, cr, d, false, false, func() error {
		return MergeSpecs(d, &d.Spec, spec, nil)
	})
}

func (r *CorootReconciler) CreateOrUpdateDaemonSet(ctx context.Context, cr *corootv1.Coroot, ds *appsv1.DaemonSet) {
	spec := ds.Spec
	r.CreateOrUpdate(ctx, cr, ds, false, false, func() error {
		return MergeSpecs(ds, &ds.Spec, spec, nil)
	})
}

func (r *CorootReconciler) CreateOrUpdateStatefulSet(ctx context.Context, cr *corootv1.Coroot, ss *appsv1.StatefulSet) {
	spec := ss.Spec
	r.CreateOrUpdate(ctx, cr, ss, false, false, func() error {
		volumeClaimTemplates := ss.Spec.VolumeClaimTemplates[:]
		err := MergeSpecs(ss, &ss.Spec, spec, nil)
		ss.Spec.VolumeClaimTemplates = volumeClaimTemplates
		return err
	})
}

func (r *CorootReconciler) CreateOrUpdatePVC(ctx context.Context, cr *corootv1.Coroot, pvc *corev1.PersistentVolumeClaim, reclaimPolicy corev1.PersistentVolumeReclaimPolicy) {
	spec := pvc.Spec
	retain := reclaimPolicy == corev1.PersistentVolumeReclaimRetain
	r.CreateOrUpdate(ctx, cr, pvc, false, retain, func() error {
		return MergeSpecs(pvc, &pvc.Spec, spec, nil)
	})
}

func (r *CorootReconciler) CreateOrUpdateService(ctx context.Context, cr *corootv1.Coroot, s *corev1.Service) {
	spec := s.Spec
	annotations := s.Annotations
	r.CreateOrUpdate(ctx, cr, s, false, false, func() error {
		err := MergeSpecs(s, &s.Spec, spec, annotations)
		s.Spec.Ports = spec.Ports
		return err
	})
}

func (r *CorootReconciler) CreateOrUpdateServiceAccount(ctx context.Context, cr *corootv1.Coroot, component, scc string) {
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
		Name:      cr.Name + "-" + component,
		Namespace: cr.Namespace,
		Labels:    Labels(cr, component),
	}}
	r.CreateOrUpdate(ctx, cr, sa, false, false, nil)
	r.CreateOrUpdate(ctx, cr, r.openshiftSCCRoleBinding(cr, component, scc), false, false, nil)
}

func (r *CorootReconciler) CreateOrUpdateRole(ctx context.Context, cr *corootv1.Coroot, role *rbacv1.Role) {
	rules := role.Rules
	r.CreateOrUpdate(ctx, cr, role, false, false, func() error {
		role.Rules = rules
		return nil
	})
}

func (r *CorootReconciler) CreateOrUpdateClusterRole(ctx context.Context, cr *corootv1.Coroot, role *rbacv1.ClusterRole) {
	rules := role.Rules
	r.CreateOrUpdate(ctx, cr, role, false, true, func() error {
		role.Rules = rules
		return nil
	})
}

func (r *CorootReconciler) CreateOrUpdateClusterRoleBinding(ctx context.Context, cr *corootv1.Coroot, b *rbacv1.ClusterRoleBinding) {
	r.CreateOrUpdate(ctx, cr, b, false, true, nil)
}

func (r *CorootReconciler) CreateOrUpdateIngress(ctx context.Context, cr *corootv1.Coroot, i *networkingv1.Ingress, delete bool) {
	spec := i.Spec
	annotations := i.Annotations
	r.CreateOrUpdate(ctx, cr, i, delete, false, func() error {
		return MergeSpecs(i, &i.Spec, spec, annotations)
	})
}

func (r *CorootReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corootv1.Coroot{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}

func Labels(cr *corootv1.Coroot, component string) map[string]string {
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	return map[string]string{
		"app.kubernetes.io/managed-by": "coroot-operator",
		"app.kubernetes.io/part-of":    cr.Name,
		"app.kubernetes.io/component":  component,
	}
}

var nonRootSecurityContext = &corev1.PodSecurityContext{
	RunAsNonRoot: ptr.To(true),
	RunAsUser:    ptr.To(int64(65534)),
	RunAsGroup:   ptr.To(int64(65534)),
	FSGroup:      ptr.To(int64(65534)),
}
