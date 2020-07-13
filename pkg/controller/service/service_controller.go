package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	aws_helper "github.com/3scale/aws-nlb-helper-operator/pkg/controller/aws"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	helperAnnotationPrefix                                   = "aws-nlb-helper.3scale.net"
	helperAnnotationLoadBalancerTerminationProtectionKey     = "aws-nlb-helper.3scale.net/loadbalanacer-termination-protection"
	helperAnnotationLoadBalancerTerminationProtectionDefault = false
	helperAnnotationTargetGroupsProxyProcotolKey             = "aws-nlb-helper.3scale.net/enable-targetgroups-proxy-protocol"
	helperAnnotationTargetGroupsProxyProcotolDefault         = false
	helperAnnotationTargetGroupsSticknessKey                 = "aws-nlb-helper.3scale.net/enable-targetgroups-stickness"
	helperAnnotationTargetGroupsSticknessDefault             = false
	helperAnnotationTargetGroupsDeregistrationDelayKey       = "aws-nlb-helper.3scale.net/targetgroups-deregisration-delay"
	helperAnnotationTargetGroupsDeregistrationDelayDefault   = 300
	awsLoadBalancerTypeAnnotationKey                         = "service.beta.kubernetes.io/aws-load-balancer-type"
	awsLoadBalancerTypeNLBAnnotationValue                    = "nlb"
	awsLoadBalancerTypeELBAnnotationValue                    = "elb"
	loadBalancerNotReadyRetryInterval                        = 30
	reconcileInterval                                        = 60
)

var log = logf.Log.WithName("controller_service")

// Add creates a new Service Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileService{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// hasHelperAnnotation returns true if the annotations list contains at
// least one aws-nlb-hepler annotation.
func hasHelperAnnotation(annotations map[string]string) bool {
	return len(getHelperAnnotations(annotations)) > 0
}

// getHelperAnnotations gets a map of strings with all the annotations matching
// the helperAnnotationPrefix prefix using getAnnotationsByPrefix()
func getHelperAnnotations(annotations map[string]string) map[string]string {
	return getAnnotationsByPrefix(annotations, helperAnnotationPrefix)
}

// getAnnotationsByPrefix gets a map of strings with all the annotations matching
// the annotationPrefix.
func getAnnotationsByPrefix(annotations map[string]string, annotationPrefix string) map[string]string {
	matchingAnnotations := make(map[string]string)
	gabpLogger := log.WithValues("AnnotationPrefix", annotationPrefix)
	for key, value := range annotations {
		if strings.HasPrefix(key, annotationPrefix) {
			gabpLogger.Info("Matching annotations found", "AnnotationKey", key, "AnnotationValue", value)
			matchingAnnotations[key] = value
		}
	}
	return matchingAnnotations
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("service-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	log.Info("Looking for Services with an aws-nlb-helper annotation")

	filter := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			switch o := e.Object.(type) {
			case *corev1.Service:
				if o.Spec.Type == "LoadBalancer" {
					return hasHelperAnnotation(e.Meta.GetAnnotations())
				}
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			switch o := e.ObjectNew.(type) {
			case *corev1.Service:
				if o.Spec.Type == "LoadBalancer" {
					return hasHelperAnnotation(e.MetaNew.GetAnnotations())
				}
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Ignore delete function as the LoadBalancer will be deleted by the AWS controller
			return false
		},
	}

	// Watch for changes to primary resource Service
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForObject{}, filter)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileService{}

// ReconcileService reconciles a Service object
type ReconcileService struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Service object and makes changes based on the state read
// and what is in the Service.Spec
func (r *ReconcileService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Namespace", request.Namespace, "Service", request.Name)
	reqLogger.Info("Reconciling Service")

	// Fetch the Service svc
	svc := &corev1.Service{}
	err := r.client.Get(context.TODO(), request.NamespacedName, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Get `kubernetes.io/service-name` tag value
	serviceNameTagValue := request.Namespace + "/" + request.Name

	// Get the AWS Load Balancer type
	awsLoadBalancerType := svc.GetAnnotations()[awsLoadBalancerTypeAnnotationKey]
	if awsLoadBalancerType == "" {
		reqLogger.Info("AWS load balancer annotation key is missing, defaulting to `elb`", "awsLoadBalancerTypeAnnotationKey", awsLoadBalancerTypeAnnotationKey)
		awsLoadBalancerType = awsLoadBalancerTypeELBAnnotationValue
	}
	reqLogger.Info("AWS load balancer type set", "awsLoadBalancerType", awsLoadBalancerType)

	if len(svc.Status.LoadBalancer.Ingress) < 1 {
		reqLogger.Info("AWS load balancer DNS not ready.", "serviceNameTagValue", serviceNameTagValue, "loadBalancerNotReadyRetryInterval", loadBalancerNotReadyRetryInterval)
		return reconcile.Result{RequeueAfter: loadBalancerNotReadyRetryInterval * time.Second}, nil
	}

	awsLoadBalancerIngressHostname := svc.Status.LoadBalancer.Ingress[0].Hostname
	reqLogger.Info("AWS load balancer type set", "awsLoadBalancerDNS", awsLoadBalancerIngressHostname)

	if awsLoadBalancerType == "nlb" {

		awsLoadBalancerSettingsTerminationProtection, err := strconv.ParseBool(svc.GetAnnotations()[helperAnnotationLoadBalancerTerminationProtectionKey])
		if err != nil {
			reqLogger.Info("Unable to parse Termination Protection value, defaulting.", "awsLoadBalancerSettingsTerminationProtection", helperAnnotationLoadBalancerTerminationProtectionDefault)
			awsLoadBalancerSettingsTerminationProtection = helperAnnotationLoadBalancerTerminationProtectionDefault
		}

		awsLoadBalancerSettingsDeregistrationDelay, err := strconv.Atoi(svc.GetAnnotations()[helperAnnotationTargetGroupsDeregistrationDelayKey])
		if err != nil {
			reqLogger.Info("Unable to parse Deregistration Delay value, defaulting.", "awsLoadBalancerSettingsDeregistrationDelay", helperAnnotationTargetGroupsDeregistrationDelayDefault)
			awsLoadBalancerSettingsDeregistrationDelay = helperAnnotationTargetGroupsDeregistrationDelayDefault
		}

		awsLoadBalancerSettingsTargetGroupProxyProtocol, err := strconv.ParseBool(svc.GetAnnotations()[helperAnnotationTargetGroupsProxyProcotolKey])
		if err != nil {
			reqLogger.Info("Unable to parse Target Group Proxy Protocol value, defaulting.", "awsLoadBalancerSettingsTargetGroupProxyProtocol", helperAnnotationTargetGroupsProxyProcotolDefault)
			awsLoadBalancerSettingsTargetGroupProxyProtocol = helperAnnotationTargetGroupsProxyProcotolDefault
		}

		awsLoadBalancerSettingsTargetGroupStickness, err := strconv.ParseBool(svc.GetAnnotations()[helperAnnotationTargetGroupsSticknessKey])
		if err != nil {
			reqLogger.Info("Unable to parse Target Group Sticknesss value, defaulting.", "awsLoadBalancerSettingsTargetGroupStickness", helperAnnotationTargetGroupsSticknessDefault)
			awsLoadBalancerSettingsTargetGroupStickness = helperAnnotationTargetGroupsSticknessDefault
		}

		awsLoadBalancerSettings := aws_helper.NetworkLoadBalancerAttributes{
			LoadBalancerTerminationProtection: awsLoadBalancerSettingsTerminationProtection,
			TargetGroupDeregistrationDelay:    awsLoadBalancerSettingsDeregistrationDelay,
			TargetGroupStickness:              awsLoadBalancerSettingsTargetGroupStickness,
			TargetGroupProxyProtocol:          awsLoadBalancerSettingsTargetGroupProxyProtocol,
		}
		updated, err := aws_helper.UpdateNetworkLoadBalancer(awsLoadBalancerIngressHostname, serviceNameTagValue, awsLoadBalancerSettings)
		if err != nil {
			reqLogger.Error(err, "Unable to update the load balancer", "awsLoadBalancerIngressHostname", awsLoadBalancerIngressHostname)
		}
		if updated {
			reqLogger.Info("Load balancer updated", "awsLoadBalancerIngressHostname", awsLoadBalancerIngressHostname)
		}
	}

	return reconcile.Result{RequeueAfter: reconcileInterval * time.Second}, nil

}
