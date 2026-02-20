package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	corootv1 "github.io/coroot/operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ClickhouseImage       = "clickhouse:25.11.2-ubi9-0"
	PrometheusImage       = "prometheus:2.55.1-ubi9-0"
	KubeStateMetricsImage = "kube-state-metrics:2.15.0-ubi9-0"
)

type App string

const (
	AppCorootCE         App = "coroot"
	AppCorootEE         App = "coroot-ee"
	AppNodeAgent        App = "coroot-node-agent"
	AppClusterAgent     App = "coroot-cluster-agent"
	AppClickhouse       App = "clickhouse"
	AppClickhouseKeeper App = "clickhouse-keeper"
	AppPrometheus       App = "prometheus"
	AppKubeStateMetrics App = "kube-state-metrics"
)

func (r *CorootReconciler) getAppImage(cr *corootv1.Coroot, app App) corootv1.ImageSpec {
	var image corootv1.ImageSpec
	switch app {
	case AppCorootCE:
		image = cr.Spec.CommunityEdition.Image
	case AppCorootEE:
		image = cr.Spec.EnterpriseEdition.Image
	case AppNodeAgent:
		image = cr.Spec.NodeAgent.Image
	case AppClusterAgent:
		image = cr.Spec.ClusterAgent.Image
	case AppKubeStateMetrics:
		image = cr.Spec.ClusterAgent.KubeStateMetrics.Image
	case AppClickhouse:
		image = cr.Spec.Clickhouse.Image
	case AppClickhouseKeeper:
		image = cr.Spec.Clickhouse.Keeper.Image
	case AppPrometheus:
		image = cr.Spec.Prometheus.Image
	}

	if image.Name != "" {
		return image
	}

	r.versionsLock.Lock()
	defer r.versionsLock.Unlock()
	image.Name = r.versions[app]
	if r.RegistryConfig.PullSecretName != "" {
		image.PullSecrets = []corev1.LocalObjectReference{{Name: r.RegistryConfig.PullSecretName}}
	}
	return image
}

func (r *CorootReconciler) fetchAppVersions() {
	logger := log.FromContext(nil)
	versions := map[App]string{}
	for _, app := range []App{AppCorootCE, AppCorootEE, AppNodeAgent, AppClusterAgent} {
		v, err := r.fetchAppVersion(app)
		if err != nil {
			logger.Error(err, "failed to get version", "app", app)
		}
		if v == "" {
			v = "latest"
		}
		versions[app] = v
	}
	logger.Info(fmt.Sprintf("got app versions: %v", versions))
	r.versionsLock.Lock()
	defer r.versionsLock.Unlock()
	for app, v := range versions {
		r.versions[app] = r.RegistryConfig.Image(fmt.Sprintf("%s:%s", app, v))
	}
	r.versions[AppClickhouse] = r.RegistryConfig.Image(ClickhouseImage)
	r.versions[AppClickhouseKeeper] = r.RegistryConfig.Image(ClickhouseImage)
	r.versions[AppPrometheus] = r.RegistryConfig.Image(PrometheusImage)
	r.versions[AppKubeStateMetrics] = r.RegistryConfig.Image(KubeStateMetricsImage)
}

func (r *CorootReconciler) fetchAppVersion(app App) (string, error) {
	repo, err := name.NewRepository(r.RegistryConfig.Image(string(app)))
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	opts := append(r.RegistryConfig.RemoteOptions(), remote.WithContext(ctx))
	tags, err := remote.List(repo, opts...)
	if err != nil {
		return "", err
	}

	type item struct {
		v   *semver.Version
		tag string
	}
	var items []item
	for _, t := range tags {
		if v, err := semver.NewVersion(t); err == nil {
			items = append(items, item{v: v, tag: t})
		}
	}
	if len(items) == 0 {
		return "", fmt.Errorf("no tags found")
	}
	sort.Slice(items, func(i, j int) bool { return items[i].v.LessThan(*items[j].v) })
	return items[len(items)-1].tag, nil
}
