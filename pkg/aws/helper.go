package aws

import (
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("aws")

const (
	awsDefaultRegion                         = "us-east-1"
	awsLoadBalancerResourceTypeFilter        = "elasticloadbalancing"
	awsTargetGroupResourceTypeFilter         = "elasticloadbalancing:targetgroup"
	awsNetworkLoadBalancerResourceTypeFilter = "elasticloadbalancing:loadbalancer/net"
	awsNetworkLoadBalancerStickness          = "source_ip"
)

// APIClient is the struct implementing the AWS provider interface
type APIClient struct {
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
func UpdateNetworkLoadBalancer(
	nlbDNS string,
	serviceNameTagValue string,
	nlbAttributes NetworkLoadBalancerAttributes) (bool, error) {

	ulbLog := log.WithValues(
		"LoadBalancerDNS", nlbDNS, "ServiceName", serviceNameTagValue,
	)

	// Get AWS Clients for ELBV2 and ResourceGroupsTaggingAPI APIs
	awsClient, err := newAPIClient()
	if err != nil {
		ulbLog.Error(err, "unable to initialize an AWS Client")
		return false, err
	}

	// Generate resource tags map
	tags := map[string]string{
		"kubernetes.io/service-name": serviceNameTagValue,
		// https://github.com/3scale/aws-nlb-helper-operator/issues/1
		// fmt.Sprintf("kubernetes.io/cluster/%s", clusterIDTagKey): "owned",
	}
	ulbLog.V(2).Info("Looking for tagged resources", "Tags", tags)

	// Get tagged network load balancers
	filteredLoadBalancers, err := awsClient.getNetworkLoadBalancerByTag(tags)
	if err != nil {
		ulbLog.Error(
			err, "unable to obtain load balancers matching the tags",
			"Tags", tags,
		)
		return false, err
	}

	// Second filtering using DNS name as clusterIDTagKey is not available
	// https://github.com/3scale/aws-nlb-helper-operator/issues/1

	nlbARN, err := awsClient.getLoadBalancerByDNS(filteredLoadBalancers, nlbDNS)
	if err != nil {
		ulbLog.Error(
			err, "unable to obtain load balancers matching the DNS",
			"Tags", tags,
		)
		return false, err
	}

	// update network load balancer attributes
	ulbLog.Info("elastic load balancer matching tags and DNS found",
		"NetowrkLoadBalancerARN", nlbARN, "NetowrkLoadBalancerDNS", nlbDNS,
	)
	awsClient.updateNetworkLoadBalancerAttributes(nlbARN, nlbAttributes)

	// update target group attributes
	targetGroupARNs, err := awsClient.getTargetGroupsByLoadBalancer(nlbARN)
	if err != nil {
		ulbLog.Error(
			err, "unable to obtain load balancer target groups",
			"NetowrkLoadBalancerARN", nlbARN,
		)
		return false, err
	}
	for _, targetGroupARN := range targetGroupARNs {
		awsClient.updateNetworkTargetGroupAttribute(targetGroupARN, nlbAttributes)
	}

	return true, nil
}

// newAWSConfig generates an AWS config.
func newAWSConfig() *aws.Config {

	awscfgLog := log.WithName("config")

	// set aws client region
	awsRegion, found := os.LookupEnv("AWS_REGION")
	if !found {
		awsRegion = awsDefaultRegion
		awscfgLog.Info("Empty AWS_REGION, defaulting",
			"awsRegion", awsRegion,
		)
	}

	// if key / access are set, set key / access authentication
	if (os.Getenv("AWS_ACCESS_KEY_ID") != "") &&
		(os.Getenv("AWS_SECRET_ACCESS_KEY") != "") {
		awscfgLog.V(2).Info(
			"Configuring AWS client using the environment credentials",
			"AWS_ACCESS_KEY_ID", os.Getenv("AWS_ACCESS_KEY_ID"),
		)
		return &aws.Config{
			Region: aws.String(awsRegion),
			Credentials: credentials.NewStaticCredentials(
				os.Getenv("AWS_ACCESS_KEY_ID"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"),
				"",
			),
		}
	}

	// set service account based authentication
	awscfgLog.V(2).Info("Configuring AWS client using the service account")
	return &aws.Config{Region: aws.String(awsRegion)}

}

// newAPIClient obtains an AWS session and initiates the needed AWS clients.
func newAPIClient() (*APIClient, error) {

	// Initialize an AWS session
	sess, err := session.NewSession(newAWSConfig())
	if err != nil {
		return nil, fmt.Errorf("unable to initialize AWS session: %v", err)
	}

	// Return AWS clients for ELBV2 and ResourceGroupsTaggingAPI
	return &APIClient{
		elbv2:  elbv2.New(sess),
		rgtapi: resourcegroupstaggingapi.New(sess),
	}, nil

}

// getLoadBalancerByDNS returns the load balancer DNS name
func (awsc *APIClient) getLoadBalancerByDNS(
	loadBalancerARNs []string, loadBalancerDNS string) (string, error) {

	dlbi := elbv2.DescribeLoadBalancersInput{}
	for _, arn := range loadBalancerARNs {
		dlbi.LoadBalancerArns = append(dlbi.LoadBalancerArns, aws.String(arn))
	}

	dlbo, err := awsc.elbv2.DescribeLoadBalancers(&dlbi)
	if err != nil {
		log.Error(
			err, "unable to describe load balancer",
			"LoadBalancerARNs", loadBalancerARNs, "DescribeTargetGroupsOutput", &dlbo,
		)
		return "", err
	}

	for _, lb := range dlbo.LoadBalancers {
		if *lb.DNSName == loadBalancerDNS {
			return *lb.LoadBalancerArn, nil
		}
	}

	return "", fmt.Errorf(
		"load balancer with DNS %s was not found", loadBalancerDNS,
	)

}

// generateTagFilters generates a ResourceGroupsTaggingAPI TagFilter object from
// a tag maps list.
func generateTagFilters(tags map[string]string,
) []*resourcegroupstaggingapi.TagFilter {
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
func (awsc *APIClient) getNetworkLoadBalancerByTag(tags map[string]string) ([]string, error) {
	return awsc.getResourcesByFilter(
		generateTagFilters(tags),
		[]*string{aws.String(awsNetworkLoadBalancerResourceTypeFilter)},
	)
}

// getResourcesByFilter returns a list of arn of resources matching the filters
func (awsc *APIClient) getResourcesByFilter(
	tagFilters []*resourcegroupstaggingapi.TagFilter, resourceTypeFilters []*string,
) ([]string, error) {

	getResourcesInput := &resourcegroupstaggingapi.GetResourcesInput{
		TagFilters:          tagFilters,
		ResourceTypeFilters: resourceTypeFilters,
	}

	resources, err := awsc.rgtapi.GetResources(getResourcesInput)
	if err != nil {
		println(err.Error())
		return nil, err
	}

	elbARNs := []string{}
	for _, resource := range resources.ResourceTagMappingList {
		elbARNs = append(elbARNs, *resource.ResourceARN)
	}
	return elbARNs, nil
}

// updateNetworkLoadBalancerAttributes returns the result of a nlb update
func (awsc *APIClient) updateNetworkLoadBalancerAttributes(
	nlbARN string, nlbAttributes NetworkLoadBalancerAttributes) (bool, error) {

	mlbai := elbv2.ModifyLoadBalancerAttributesInput{
		LoadBalancerArn: aws.String(nlbARN),
		Attributes: []*elbv2.LoadBalancerAttribute{
			{
				Key: aws.String("deletion_protection.enabled"),
				Value: aws.String(
					strconv.FormatBool(nlbAttributes.LoadBalancerTerminationProtection),
				),
			},
		},
	}

	mlbao, err := awsc.elbv2.ModifyLoadBalancerAttributes(&mlbai)
	log.V(2).Info("Modify load balancer aws command output",
		"ModifyLoadBalancerAttributesOutput", &mlbao,
	)

	if err != nil {
		log.Error(
			err, "unable to modify the network load balancer",
			"NetworkLoadBalancerARN", nlbARN,
		)
		return false, err
	}

	log.Info("Network load balancer updated", "NetworkLoadBalancerARN", nlbARN)
	return true, nil
}

// getTargetGroupsByLoadBalancer returns a list of target groups attached to a
// the load balancer defined by the loadBalancerARN parameter.
func (awsc *APIClient) getTargetGroupsByLoadBalancer(elbARN string) ([]string, error) {

	dlbi := elbv2.DescribeTargetGroupsInput{
		LoadBalancerArn: aws.String(elbARN),
	}

	dtgo, err := awsc.elbv2.DescribeTargetGroups(&dlbi)
	if err != nil {
		log.Error(err, "unable to describe load balancer target groups",
			"LoadBalancerARN", elbARN, "DescribeTargetGroupsOutput", &dtgo,
		)
		return nil, err
	}

	targetGroupARNs := []string{}
	for _, tg := range dtgo.TargetGroups {
		targetGroupARNs = append(targetGroupARNs, *tg.TargetGroupArn)
	}
	return targetGroupARNs, nil
}

// updateNetworkTargetGroupAttribute returns the result of updating the target groups
func (awsc *APIClient) updateNetworkTargetGroupAttribute(
	targetGroupARN string, nlbAttributes NetworkLoadBalancerAttributes) (bool, error) {

	log.V(2).Info("Updating target group", "targetGroupARN", targetGroupARN)

	mtgai := elbv2.ModifyTargetGroupAttributesInput{
		TargetGroupArn: aws.String(targetGroupARN),
		Attributes: []*elbv2.TargetGroupAttribute{
			{
				Key:   aws.String("stickiness.enabled"),
				Value: aws.String(strconv.FormatBool(nlbAttributes.TargetGroupStickness)),
			},
			{
				Key:   aws.String("stickiness.type"),
				Value: aws.String(awsNetworkLoadBalancerStickness),
			},
			{
				Key:   aws.String("proxy_protocol_v2.enabled"),
				Value: aws.String(strconv.FormatBool(nlbAttributes.TargetGroupProxyProtocol)),
			},
			{
				Key:   aws.String("deregistration_delay.timeout_seconds"),
				Value: aws.String(strconv.Itoa(nlbAttributes.TargetGroupDeregistrationDelay)),
			},
		},
	}

	mtgao, err := awsc.elbv2.ModifyTargetGroupAttributes(&mtgai)
	log.V(2).Info("Modify target group aws command output",
		"ModifyTargetGroupAttributesOutput", &mtgao,
	)

	if err != nil {
		log.Error(
			err, "unable to update the target groups",
			"TargetGroupARN", targetGroupARN,
		)
		return false, err
	}

	log.Info("Target groups succesfully updated",
		"TargetGroupARN", targetGroupARN,
	)
	return true, nil

}
