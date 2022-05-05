package controllers

const (
	helperAnnotationPrefix                                   = "aws-nlb-helper.3scale.net"
	helperAnnotationLoadBalancerTerminationProtectionKey     = "aws-nlb-helper.3scale.net/loadbalanacer-termination-protection"
	helperAnnotationLoadBalancerTerminationProtectionDefault = false
	helperAnnotationTargetGroupsProxyProcotolKey             = "aws-nlb-helper.3scale.net/enable-targetgroups-proxy-protocol"
	helperAnnotationTargetGroupsProxyProcotolDefault         = false
	helperAnnotationTargetGroupsSticknessKey                 = "aws-nlb-helper.3scale.net/enable-targetgroups-stickness"
	helperAnnotationTargetGroupsSticknessDefault             = false
	helperAnnotationTargetGroupsDeregistrationDelayKey       = "/targetgroups-deregisration-delay"
	helperAnnotationTargetGroupsDeregistrationDelayDefault   = 300
	awsLoadBalancerTypeAnnotationKey                         = "service.beta.kubernetes.io/aws-load-balancer-type"
	awsLoadBalancerTypeNLBAnnotationValue                    = "nlb"
	awsLoadBalancerTypeELBAnnotationValue                    = "elb"
	loadBalancerNotReadyRetryInterval                        = 30
	reconcileInterval                                        = 60
)
