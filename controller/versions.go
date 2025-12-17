package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	corootv1 "github.io/coroot/operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	CorootImageRegistry = "ghcr.io/coroot"

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
		r.versions[app] = fmt.Sprintf("%s:%s", app, v)
	}
	r.versions[AppClickhouse] = ClickhouseImage
	r.versions[AppClickhouseKeeper] = ClickhouseImage
	r.versions[AppPrometheus] = PrometheusImage
	r.versions[AppKubeStateMetrics] = KubeStateMetricsImage
	for app, image := range r.versions {
		r.versions[app] = CorootImageRegistry + "/" + image
	}
}

func (r *CorootReconciler) fetchAppVersion(app App) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/coroot/%s/releases/latest", app))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err = json.Unmarshal(data, &release); err != nil {
		return "", err
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}
