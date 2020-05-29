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

// AWSClient is the struct implementing the lbprovider interface
type AWSClient struct {
	elbv2  *elbv2.ELBV2
	rgtapi *resourcegroupstaggingapi.ResourceGroupsTaggingAPI
}

// LoadBalancerAttributes struct
type LoadBalancerAttributes struct {
	LoadBalancerTerminationProtection bool
	TargetGroupDeregistrationDelay    int
	TargetGroupStickness              bool
	TargetGroupProxyProtocol          bool
}

// UpdateLoadBalancer updates an AWS load balancer
func UpdateLoadBalancer(clusterIDTagKey string, serviceNameTagValue string, loadBalancerAttributes LoadBalancerAttributes) (bool, error) {
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

	return awsClient != nil, nil
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
