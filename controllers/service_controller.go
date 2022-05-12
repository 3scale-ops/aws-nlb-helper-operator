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

package controllers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/3scale-ops/aws-nlb-helper-operator/pkg/aws"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get

func (r *ServiceReconciler) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	rLogger := r.Log.WithValues("Namespace", req.Namespace, "Service", req.Name)
	rLogger.Info("Reconciling Service")

	// Fetch the Service svc
	svc := &corev1.Service{}
	err := r.Get(context.TODO(), req.NamespacedName, svc)
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
	serviceNameTagValue := req.Namespace + "/" + req.Name

	// Get the AWS Load Balancer type
	awsELBType := svc.GetAnnotations()[awsELBTypeAnnotationKey]
	if awsELBType == "" {
		rLogger.Info(
			"AWS elastic load balancer type annotation key is missing, defaulting",
			"awsELBTypeAnnotationKey", awsELBTypeAnnotationKey,
			"awsELBDefaultType", awsELBTypeClassicAnnotationValue,
		)
		awsELBType = awsELBTypeClassicAnnotationValue
	}
	rLogger.Info("AWS elastic load balancer type set", "awsELBType", awsELBType)

	if len(svc.Status.LoadBalancer.Ingress) < 1 {
		rLogger.V(2).Info(
			"AWS elastic load balancer DNS is not ready",
			"serviceNameTagValue", serviceNameTagValue,
			"loadBalancerNotReadyRetryInterval", awsELBNotReadyRetryInterval,
		)
		return reconcile.Result{
			RequeueAfter: awsELBNotReadyRetryInterval * time.Second,
		}, nil
	}
	awsELBIngressHostname := svc.Status.LoadBalancer.Ingress[0].Hostname
	rLogger.Info(
		"AWS elastic load balancer hostname",
		"awsELBDNS", awsELBIngressHostname,
	)

	if awsELBType == "nlb" {

		updated, err := aws.UpdateNetworkLoadBalancer(
			awsELBIngressHostname, serviceNameTagValue,
			r.getELBAttributesFromAnnotations(svc),
		)
		if err != nil {
			rLogger.Error(
				err, "unable to update the load balancer",
				"awsELBIngressHostname", awsELBIngressHostname,
			)
		}
		if updated {
			rLogger.Info("Load balancer updated",
				"awsELBIngressHostname", awsELBIngressHostname,
			)
		}
	}

	return ctrl.Result{}, nil
}

// getELBAttributesFromAnnotations generates the AWS network load balancer attributes from the
// annotations
func (r *ServiceReconciler) getELBAttributesFromAnnotations(
	svc *corev1.Service) aws.NetworkLoadBalancerAttributes {

	rLogger := r.Log.WithName("attribute")

	awsELBSettingsTerminationProtection, err := strconv.ParseBool(
		svc.GetAnnotations()[annotationLoadBalancerTerminationProtectionKey],
	)
	if err != nil {
		rLogger.V(2).Info(
			"unable to parse Termination Protection value, defaulting",
			"awsELBSettingsTerminationProtection", annotationLoadBalancerTerminationProtectionDefault,
		)
		awsELBSettingsTerminationProtection = annotationLoadBalancerTerminationProtectionDefault
	}

	awsELBSettingsDeregistrationDelay, err := strconv.Atoi(
		svc.GetAnnotations()[annotationTargetGroupsDeregistrationDelayKey],
	)
	if err != nil {
		rLogger.V(2).Info(
			"unable to parse Deregistration Delay value, defaulting",
			"awsELBSettingsDeregistrationDelay", annotationTargetGroupsDeregistrationDelayDefault,
		)
		awsELBSettingsDeregistrationDelay = annotationTargetGroupsDeregistrationDelayDefault
	}

	awsELBSettingsTargetGroupProxyProtocol, err := strconv.ParseBool(
		svc.GetAnnotations()[annotationTargetGroupsProxyProcotolKey],
	)
	if err != nil {
		rLogger.V(2).Info(
			"unable to parse Target Group Proxy Protocol value, defaulting",
			"awsELBSettingsTargetGroupProxyProtocol", annotationTargetGroupsProxyProcotolDefault,
		)
		awsELBSettingsTargetGroupProxyProtocol = annotationTargetGroupsProxyProcotolDefault
	}

	awsELBSettingsTargetGroupStickness, err := strconv.ParseBool(
		svc.GetAnnotations()[annotationTargetGroupsSticknessKey],
	)
	if err != nil {
		rLogger.V(2).Info(
			"unable to parse Target Group Sticknesss value, defaulting",
			"awsELBSettingsTargetGroupStickness", annotationTargetGroupsSticknessDefault,
		)
		awsELBSettingsTargetGroupStickness = annotationTargetGroupsSticknessDefault
	}

	return aws.NetworkLoadBalancerAttributes{
		LoadBalancerTerminationProtection: awsELBSettingsTerminationProtection,
		TargetGroupDeregistrationDelay:    awsELBSettingsDeregistrationDelay,
		TargetGroupStickness:              awsELBSettingsTargetGroupStickness,
		TargetGroupProxyProtocol:          awsELBSettingsTargetGroupProxyProtocol,
	}

}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithEventFilter(r.filterAnnotatedServices()).
		Complete(r)
}

// hasHelperAnnotation returns true if the annotations list contains at
// least one aws-nlb-hepler annotation.
func (r *ServiceReconciler) hasHelperAnnotation(annotations map[string]string) bool {
	return len(r.getHelperAnnotations(annotations)) > 0
}

// getHelperAnnotations gets a map of strings with all the annotations matching
// the annotationPrefix prefix using getAnnotationsByPrefix()
func (r *ServiceReconciler) getHelperAnnotations(annotations map[string]string) map[string]string {
	return r.getAnnotationsByPrefix(annotations, annotationPrefix)
}

// getAnnotationsByPrefix gets a map of strings with all the annotations matching
// the annotationPrefix.
func (r *ServiceReconciler) getAnnotationsByPrefix(annotations map[string]string, annotationPrefix string) map[string]string {
	matchingAnnotations := make(map[string]string)
	for key, value := range annotations {
		if strings.HasPrefix(key, annotationPrefix) {
			r.Log.WithName("filter").V(2).Info("Matching annotations found.",
				"AnnotationKey", key, "AnnotationValue", value,
			)
			matchingAnnotations[key] = value
		}
	}
	return matchingAnnotations
}

func (r *ServiceReconciler) filterAnnotatedServices() predicate.Funcs {

	r.Log.WithName("filter").Info(
		"Looking for Services with an aws-nlb-helper annotation",
	)

	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			switch o := e.Object.(type) {
			case *corev1.Service:
				if o.Spec.Type == "LoadBalancer" {
					return r.hasHelperAnnotation(o.GetAnnotations())
				}
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			switch o := e.ObjectNew.(type) {
			case *corev1.Service:
				if o.Spec.Type == "LoadBalancer" {
					return r.hasHelperAnnotation(o.GetAnnotations())
				}
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Ignore delete function as it will be deleted by the AWS controller
			return false
		},
	}

}
