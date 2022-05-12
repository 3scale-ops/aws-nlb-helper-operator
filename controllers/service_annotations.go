package controllers

const (
	annotationPrefix                                   = "aws-nlb-helper.3scale.net"
	annotationLoadBalancerTerminationProtectionKey     = "aws-nlb-helper.3scale.net/loadbalanacer-termination-protection"
	annotationLoadBalancerTerminationProtectionDefault = false
	annotationTargetGroupsProxyProcotolKey             = "aws-nlb-helper.3scale.net/enable-targetgroups-proxy-protocol"
	annotationTargetGroupsProxyProcotolDefault         = false
	annotationTargetGroupsSticknessKey                 = "aws-nlb-helper.3scale.net/enable-targetgroups-stickness"
	annotationTargetGroupsSticknessDefault             = false
	annotationTargetGroupsDeregistrationDelayKey       = "/targetgroups-deregisration-delay"
	annotationTargetGroupsDeregistrationDelayDefault   = 300
	awsELBTypeAnnotationKey                            = "service.beta.kubernetes.io/aws-load-balancer-type"
	awsELBTypeNLBAnnotationValue                       = "nlb"
	awsELBTypeClassicAnnotationValue                   = "classic"
	awsELBNotReadyRetryInterval                        = 30
	reconcileInterval                                  = 60
)
