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
