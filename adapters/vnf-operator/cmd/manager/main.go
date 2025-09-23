package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	manov1alpha1 "github.com/o-ran/intent-mano/adapters/vnf-operator/api/v1alpha1"
	"github.com/o-ran/intent-mano/adapters/vnf-operator/controllers"
	"github.com/o-ran/intent-mano/adapters/vnf-operator/pkg/dms"
	"github.com/o-ran/intent-mano/adapters/vnf-operator/pkg/gitops"
	"github.com/o-ran/intent-mano/adapters/vnf-operator/pkg/translator"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(manov1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var dmsEndpoint string
	var porchRepo string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&dmsEndpoint, "dms-endpoint", "", "O2 DMS API endpoint")
	flag.StringVar(&porchRepo, "porch-repo", "", "Porch repository URL")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "vnf-operator.mano.oran.io",
		LeaderElectionNamespace: "kube-system",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize clients
	var dmsClient dms.Client
	if dmsEndpoint != "" {
		dmsClient = dms.NewO2DMSClient(dmsEndpoint, os.Getenv("DMS_TOKEN"))
	} else {
		setupLog.Info("Using mock DMS client")
		dmsClient = dms.NewMockDMSClient()
	}

	var gitOpsClient gitops.Client
	if porchRepo != "" {
		gitOpsClient = gitops.NewPorchClient(porchRepo, "default")
	} else {
		setupLog.Info("Using mock GitOps client")
		gitOpsClient = gitops.NewMockGitOpsClient()
	}

	// Setup VNF controller
	if err = (&controllers.VNFReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		PorchTranslator: translator.NewPorchTranslator(),
		DMSClient:       dmsClient,
		GitOpsClient:    gitOpsClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VNF")
		os.Exit(1)
	}

	// Setup health checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}