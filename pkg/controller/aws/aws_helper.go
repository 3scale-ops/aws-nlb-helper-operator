package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("helper_aws")

const (
	awsLoadBalancerResourceTypeFilter            = "elasticloadbalancing"
	awsTargetGroupResourceTypeFilter             = "elasticloadbalancing:targetgroup"
	awsApplicationLoadBalancerResourceTypeFilter = "elasticloadbalancing:loadbalancer/app"
	awsApplicationLoadBalancerStickness          = "lb_cookie"
	awsNetworkLoadBalancerResourceTypeFilter     = "elasticloadbalancing:loadbalancer/net"
	awsNetworkLoadBalancerStickness              = "source_ip"
)

// AWSClient is the struct implementing the lbprovider interface
type AWSClient struct {
	elbv2  *elbv2.ELBV2
	rgtapi *resourcegroupstaggingapi.ResourceGroupsTaggingAPI
}

// NetworkLoadBalancerAttributes struct
type NetworkLoadBalancerAttributes struct {
	LoadBalancerTerminationProtection bool
	TargetGroupDeregistrationDelay    int
	TargetGroupStickness              bool
	TargetGroupProxyProtocol          bool
}

// UpdateNetworkLoadBalancer updates an AWS load balancer
func UpdateNetworkLoadBalancer(clusterIDTagKey string, serviceNameTagValue string, loadBalancerAttributes NetworkLoadBalancerAttributes) (bool, error) {
	ulbLogger := log.WithValues("ClusterId", clusterIDTagKey, "ServiceName", serviceNameTagValue)

	// Get AWS Clients for ELBV2 and ResourceGroupsTaggingAPI APIs
	awsClient, err := newAWSClient(
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		os.Getenv("AWS_REGION"),
	)

	if err != nil {
		ulbLogger.Error(err, "Unable to create AWS Client",
			"AWS_ACCESS_KEY_ID", os.Getenv("AWS_ACCESS_KEY_ID"),
			"AWS_REGION", os.Getenv("AWS_REGION"),
		)
		return false, err
	}

	// Generate resource tags map
	tags := map[string]string{
		"kubernetes.io/service-name":                             serviceNameTagValue,
		fmt.Sprintf("kubernetes.io/cluster/%s", clusterIDTagKey): "owned",
	}
	ulbLogger.Info("Looking for tagged resources", "Tags", tags)

	// Get tagged network load balancers
	networkLoadBalancerArns, err := awsClient.getNetworkLoadBalancerByTag(tags)
	if err != nil {
		ulbLogger.Error(err, "Unable to obtain load balancers matching the tags", "Tags", tags)
		return false, err
	}

	for _, loadBalancerARN := range networkLoadBalancerArns {

		awsClient.updateNetworkLoadBalancerAttributes(loadBalancerARN, loadBalancerAttributes)

		targetGroupARNs, err := awsClient.getTargetGroupsByLoadBalancer(loadBalancerARN)
		if err != nil {
			ulbLogger.Info("Unable to obtain load balancer target groups", "loadBalancerARN", loadBalancerARN)
			return false, err
		}

		for _, targetGroupARN := range targetGroupARNs {
			awsClient.updateNetworkTargetGroupAttribute(targetGroupARN, loadBalancerAttributes)
		}
	}

	return true, nil
}

// newAWSClient obtains an AWS session and initiates the needed AWS clients.
func newAWSClient(id string, secret string, region string) (*AWSClient, error) {

	// Get AWS config
	awsConfig := &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(id, secret, ""),
	}

	// Initialize an AWS session
	awsConfig = awsConfig.WithCredentialsChainVerboseErrors(true)
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize AWS session: %v", err)
	}

	// Return AWS clients for ELBV2 and ResourceGroupsTaggingAPI
	return &AWSClient{
		elbv2:  elbv2.New(sess),
		rgtapi: resourcegroupstaggingapi.New(sess),
	}, nil
}

// generateTagFilters generates a ResourceGroupsTaggingAPI TagFilter object from
// a tag maps list.
func generateTagFilters(tags map[string]string) []*resourcegroupstaggingapi.TagFilter {
	var tagFilters []*resourcegroupstaggingapi.TagFilter
	for k, v := range tags {
		tagFilters = append(
			tagFilters,
			&resourcegroupstaggingapi.TagFilter{
				Key:    aws.String(k),
				Values: []*string{aws.String(v)},
			})
	}
	return tagFilters
}

// getNetworkLoadBalancerByTag returns a list of network load balancers with
// the tag list defined by the tags parameter.
func (awsc *AWSClient) getNetworkLoadBalancerByTag(tags map[string]string) ([]string, error) {
	return awsc.getResourcesByFilter(
		generateTagFilters(tags),
		[]*string{aws.String(awsNetworkLoadBalancerResourceTypeFilter)},
	)
}

// getResourcesByFilter returns a list of arn of resources matching the filters
func (awsc *AWSClient) getResourcesByFilter(tagFilters []*resourcegroupstaggingapi.TagFilter, resourceTypeFilters []*string) ([]string, error) {

	getResourcesInput := &resourcegroupstaggingapi.GetResourcesInput{
		TagFilters:          tagFilters,
		ResourceTypeFilters: resourceTypeFilters,
	}

	resources, err := awsc.rgtapi.GetResources(getResourcesInput)
	if err != nil {
		println(err.Error())
		return nil, err
	}

	loadBalanerARNs := []string{}
	for _, resource := range resources.ResourceTagMappingList {
		loadBalanerARNs = append(loadBalanerARNs, *resource.ResourceARN)
	}
	return loadBalanerARNs, nil
}

func (awsc *AWSClient) updateNetworkLoadBalancerAttributes(loadBalancerARN string, loadBalancerAttributes NetworkLoadBalancerAttributes) (bool, error) {

	mlbai := elbv2.ModifyLoadBalancerAttributesInput{
		LoadBalancerArn: aws.String(loadBalancerARN),
		Attributes: []*elbv2.LoadBalancerAttribute{
			{
				Key:   aws.String("deletion_protection.enabled"),
				Value: aws.String(strconv.FormatBool(loadBalancerAttributes.LoadBalancerTerminationProtection)),
			},
		},
	}

	mlbao, err := awsc.elbv2.ModifyLoadBalancerAttributes(&mlbai)
	if err != nil {
		log.Error(err, "Unable to Modify the load balancer", "LoadBalancerARN", loadBalancerARN, "ModifyLoadBalancerAttributesOutput", &mlbao)
		return false, err
	}

	log.Info("Load balancer updated", "ModifyLoadBalancerAttributesOutput", &mlbao)
	return true, nil
}

// getTargetGroupsByLoadBalancer returns a list of target groups attached to a
// the load balancer defined by the loadBalancerARN parameter.
func (awsc *AWSClient) getTargetGroupsByLoadBalancer(loadBalancerARN string) ([]string, error) {

	dlbi := elbv2.DescribeTargetGroupsInput{
		LoadBalancerArn: aws.String(loadBalancerARN),
	}

	dtgo, err := awsc.elbv2.DescribeTargetGroups(&dlbi)
	if err != nil {
		log.Error(err, "Unable to describe load balancer target groups", "LoadBalancerARN", loadBalancerARN, "DescribeTargetGroupsOutput", &dtgo)
		return nil, err
	}

	targetGroupARNs := []string{}
	for _, tg := range dtgo.TargetGroups {
		targetGroupARNs = append(targetGroupARNs, *tg.TargetGroupArn)
	}
	return targetGroupARNs, nil
}

func (awsc *AWSClient) updateNetworkTargetGroupAttribute(targetGroupARN string, loadBalancerAttributes NetworkLoadBalancerAttributes) (bool, error) {

	log.Info("Updating target group", "targetGroupARN", targetGroupARN)

	mtgai := elbv2.ModifyTargetGroupAttributesInput{
		TargetGroupArn: aws.String(targetGroupARN),
		Attributes: []*elbv2.TargetGroupAttribute{
			{
				Key:   aws.String("stickiness.enabled"),
				Value: aws.String(strconv.FormatBool(loadBalancerAttributes.TargetGroupStickness)),
			},
			{
				Key:   aws.String("stickiness.type"),
				Value: aws.String(awsNetworkLoadBalancerStickness),
			},
			{
				Key:   aws.String("proxy_protocol_v2.enabled"),
				Value: aws.String(strconv.FormatBool(loadBalancerAttributes.TargetGroupProxyProtocol)),
			},
			{
				Key:   aws.String("deregistration_delay.timeout_seconds"),
				Value: aws.String(strconv.Itoa(loadBalancerAttributes.TargetGroupDeregistrationDelay)),
			},
		},
	}

	mtgao, err := awsc.elbv2.ModifyTargetGroupAttributes(&mtgai)
	if err != nil {
		log.Error(err, "Unable to modify the target groups", "TargetGroupARN", targetGroupARN, "ModifyLoadBalancerAttributesOutput", &mtgao)
		return false, err
	}

	log.Info("Target groups updated", "TargetGroupARN", targetGroupARN, "ModifyLoadBalancerAttributesOutput", &mtgao)
	return true, nil
}
