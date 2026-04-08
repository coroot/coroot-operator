package main

import (
	"os"

	"github.io/coroot/operator/controller"
	"go.uber.org/zap/zapcore"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corootv1 "github.io/coroot/operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(corootv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Development: true, StacktraceLevel: zapcore.DPanicLevel})))
	logger := ctrl.Log

	mgrOptions := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: ":8081",
	}
	mgrOptions.Client.Cache = &client.CacheOptions{
		DisableFor: []client.Object{&corev1.Secret{}},
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
	if err != nil {
		logger.Error(err, "failed to start manager")
		os.Exit(1)
	}

	reconciler := controller.NewCorootReconciler(mgr)

	if err = reconciler.SetupWithManager(mgr); err != nil {
		logger.Error(err, "failed to create controller")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "failed to set up health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "failed to set up ready check")
		os.Exit(1)
	}

	logger.Info("starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "failed to start manager")
		os.Exit(1)
	}
}
