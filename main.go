/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	goruntime "runtime"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/3scale-ops/aws-nlb-helper-operator/controllers"
	util "github.com/3scale-ops/aws-nlb-helper-operator/pkg/utils"
	"github.com/3scale-ops/aws-nlb-helper-operator/pkg/version"
	//+kubebuilder:scaffold:imports
)

const (
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	watchNamespaceEnvVar string = "WATCH_NAMESPACE"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger((util.Logger{}).New())

	printVersion()

	mgrOpts := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "804187e3.aws-nlb-helper.3scale.net",
	}

	SetOperatorScope(&mgrOpts)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ServiceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("Service"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Service")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

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

func SetOperatorScope(o *ctrl.Options) {

	watchNamespace, found := os.LookupEnv(watchNamespaceEnvVar)

	if !found {
		setupLog.Info("The manager will watch services from all Namespaces")
		return
	}

	if strings.Contains(watchNamespace, ",") {
		setupLog.Info(
			fmt.Sprintf(
				"Manager will be watching services from %q namespaces.",
				watchNamespace,
			),
		)
		o.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(watchNamespace, ","))
		return
	}

	setupLog.Info(fmt.Sprintf(
		"Manager will be watching the namespace %q", watchNamespace))
	o.Namespace = watchNamespace

}

// getWatchNamespace returns the Namespace the operator should be watching for changes
func printVersion() {
	setupLog.Info(fmt.Sprintf("AWS NLB Helper Operator Version: %s", version.Current()))
	setupLog.Info(fmt.Sprintf("Go Version: %s", goruntime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", goruntime.GOOS, goruntime.GOARCH))
}
