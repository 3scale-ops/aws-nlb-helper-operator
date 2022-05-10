[![Go Report Card](https://goreportcard.com/badge/github.com/3scale/aws-nlb-helper-operator)](https://goreportcard.com/report/github.com/3scale/aws-nlb-helper-operator)
[![build](https://github.com/3scale-ops/aws-nlb-helper-operator/workflows/build/badge.svg)](https://github.com/3scale-ops/aws-nlb-helper-operator/actions?query=workflow%3Abuild)
[![latest](https://github.com/3scale-ops/aws-nlb-helper-operator/workflows/latest/badge.svg)](https://github.com/3scale-ops/aws-nlb-helper-operator/actions?query=workflow%3Alatest)
[![release](https://github.com/3scale-ops/aws-nlb-helper-operator/workflows/release/badge.svg)](https://github.com/3scale-ops/aws-nlb-helper-operator/actions?query=workflow%3Arelease)
[![release](https://badgen.net/github/release/3scale/aws-nlb-helper-operator)](https://github.com/3scale/aws-nlb-helper-operator/releases)
[![license](https://badgen.net/github/license/3scale/aws-nlb-helper-operator)](https://github.com/3scale/aws-nlb-helper-operator/blob/master/LICENSE)

# AWS NLB Helper operator

This operator allows to manage some settings for AWS Network Load Balanacer using
Kubernetes annotations in the service objects.

**Disclaimer**: This operator is in the early development stages.

## Motivations

The current ingress controller for AWS Network Load Balanacers doesn't support
setting some attributes like enabling the termination protection, the proxy protocol
or the stickness. This operator adds support to change those settings by providing
some extra annotations to the kubernetes service objects.

## Annotations

| Setting                              | Annotations                                                      | Values          | Default |
| ------------------------------------ | ---------------------------------------------------------------- | --------------- | ------- |
| Load Balancer Termination Protection | `aws-nlb-helper.3scale.net/loadbalanacer-termination-protection` | `true`, `false` | `false` |
| Target Group Proxy Protocol          | `aws-nlb-helper.3scale.net/enable-targetgroups-proxy-protocol`   | `true`, `false` | `false` |
| Target Group Stickness               | `aws-nlb-helper.3scale.net/enable-targetgroups-stickness`        | `true`, `false` | `false` |
| Target Group Deregistration Delay    | `aws-nlb-helper.3scale.net/targetgroups-deregisration-delay`     | `0-3600`        | `300`   |

## AWS authentication

By default, the operator will use the role provided by the service acccount to
connect to the AWS API. The YAMLs for deploying using [IAM roles for service accounts](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) are available at [deploy/iam-service-account](deploy/iam-service-account).

Otherwise, if the environment variables `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are set,
the operator will use them to interact with the AWS API. You can find the YAMLs
for deploying the resources using the environment access keys at [deploy/iam-env-credentials](deploy/iam-env-credentials).

## OLM installation

At this stage, we use a custom `CatalogSource` specific for this operator:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: aws-nlb-helper-operator-catalog
  namespace: openshift-marketplace
  # Installed openshift-marketplace to enable multi-namespace support
  # - https://bugzilla.redhat.com/show_bug.cgi?id=1779080
spec:
  sourceType: grpc
  image: quay.io/3scale/aws-nlb-helper-operator-catalog:latest
  displayName: AWS NLB Helper Operator
  updateStrategy:
    registryPoll:
      interval: 30m
```

And can be installed via `Subscription`:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: aws-nlb-helper-operator
  namespace: aws-nlb-helper
spec:
  channel: alpha
  installPlanApproval: Automatic
  name: aws-nlb-helper-operator
  source: aws-nlb-helper-operator-catalog
  sourceNamespace: openshift-marketplace
  config:
    env:
      - name: WATCH_NAMESPACE
        value: aws-nlb-helper,3scale-saas
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-nlb-helper-iam
            key: AWS_ACCESS_KEY_ID
      - name: AWS_SECRET_ACCESS_KEY
        valueFrom:
          secretKeyRef:
            name: aws-nlb-helper-iam
            key: AWS_SECRET_ACCESS_KEY
      - name: AWS_REGION
        valueFrom:
          secretKeyRef:
            name: aws-nlb-helper-iam
            key: AWS_REGION
```

## Requirements

### Secret with IAM credentials (when using env based credentials)

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: aws-nlb-helper-iam
type: Opaque
data:
  AWS_ACCESS_KEY_ID: __AWS_ACCESS_KEY_ID__
  AWS_REGION: __AWS_REGION__
  AWS_SECRET_ACCESS_KEY: __AWS_SECRET_ACCESS_KEY__
```

The user needs the following permissions:

- tag:GetResources
- elasticloadbalancing:DescribeListeners
- elasticloadbalancing:DescribeLoadBalancers
- elasticloadbalancing:DescribeTags
- elasticloadbalancing:DescribeTargetGroupAttributes
- elasticloadbalancing:DescribeTargetGroups
- elasticloadbalancing:ModifyTargetGroupAttributes
- elasticloadbalancing:ModifyLoadBalancerAttributes

If you use Terraform, the following code will create the required user.

```terraform
data "aws_iam_policy_document" "this" {
  statement {
    actions = [
      "tag:GetResources",
      "elasticloadbalancing:DescribeListeners",
      "elasticloadbalancing:DescribeLoadBalancers",
      "elasticloadbalancing:DescribeTags",
      "elasticloadbalancing:DescribeTargetGroupAttributes",
      "elasticloadbalancing:DescribeTargetGroups",
      "elasticloadbalancing:ModifyTargetGroupAttributes",
      "elasticloadbalancing:ModifyLoadBalancerAttributes"
    ]
    resources = ["*"]
  }
}

resource "aws_iam_user_policy" "this" {
  name   = "aws-nlb-helper-user-policy"
  user   = aws_iam_user.this.name
  policy = data.aws_iam_policy_document.this.json
}

resource "aws_iam_user" "this" {
  name = "aws-nlb-helper"
}

resource "aws_iam_access_key" "this" {
  user = aws_iam_user.this.name
}
```

## Manual Deployment

For manualy deployment, check the available `Deployment` targets with `make help`.

### Example service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: test-api
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    aws-nlb-helper.3scale.net/enable-targetgroups-proxy-protocol: "true"
    aws-nlb-helper.3scale.net/enable-targetgroups-stickness: "true"
    aws-nlb-helper.3scale.net/loadbalanacer-termination-protection: "true"
    aws-nlb-helper.3scale.net/targetgroups-deregisration-delay: "450"
spec:
  type: LoadBalancer
  selector:
    deployment: kuard
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: http
```

### Controller logs

When the service is created, the `externalName` is not yet available (the load balancer is being provisioned), so it try to read the DNS until is available.

```log
{"level":"info","ts":1591461948.7908804,"logger":"controller_service","msg":"Matching annotations found","AnnotationPrefix":"aws-nlb-helper.3scale.net","AnnotationKey":"aws-nlb-helper.3scale.net/enable-targetgroups-proxy-protocol","AnnotationValue":"true"}
{"level":"info","ts":1591461948.7909887,"logger":"controller_service","msg":"Reconciling Service","Namespace":"aws-nlb-helper","Service":"test-api"}
{"level":"info","ts":1591461948.7910357,"logger":"controller_service","msg":"AWS load balancer type set","Namespace":"aws-nlb-helper","Service":"test-api","awsLoadBalancerType":"nlb"}
{"level":"info","ts":1591461948.7910464,"logger":"controller_service","msg":"AWS load balancer DNS not ready.","Namespace":"aws-nlb-helper","Service":"test-api","serviceNameTagValue":"aws-nlb-helper/test-api","loadBalancerNotReadyRetryInterval":30}
```

Next try after the `loadBalancerNotReadyRetryInterval` interval, the `externalName` is still not available.

```log
{"level":"info","ts":1591461948.8010314,"logger":"controller_service","msg":"Matching annotations found","AnnotationPrefix":"aws-nlb-helper.3scale.net","AnnotationKey":"aws-nlb-helper.3scale.net/enable-targetgroups-proxy-protocol","AnnotationValue":"true"}
{"level":"info","ts":1591461948.8010733,"logger":"controller_service","msg":"Reconciling Service","Namespace":"aws-nlb-helper","Service":"test-api"}
{"level":"info","ts":1591461948.801088,"logger":"controller_service","msg":"AWS load balancer type set","Namespace":"aws-nlb-helper","Service":"test-api","awsLoadBalancerType":"nlb"}
{"level":"info","ts":1591461948.8010914,"logger":"controller_service","msg":"AWS load balancer DNS not ready.","Namespace":"aws-nlb-helper","Service":"test-api","serviceNameTagValue":"aws-nlb-helper/test-api","loadBalancerNotReadyRetryInterval":30}
```

On the third try, the `externalName` is available. It will be used along the service name to identify the load balancer.

```log
{"level":"info","ts":1591461951.3948922,"logger":"controller_service","msg":"Matching annotations found","AnnotationPrefix":"aws-nlb-helper.3scale.net","AnnotationKey":"aws-nlb-helper.3scale.net/enable-targetgroups-proxy-protocol","AnnotationValue":"true"}
{"level":"info","ts":1591461951.3949463,"logger":"controller_service","msg":"Reconciling Service","Namespace":"aws-nlb-helper","Service":"test-api"}
{"level":"info","ts":1591461951.39497,"logger":"controller_service","msg":"AWS load balancer type set","Namespace":"aws-nlb-helper","Service":"test-api","awsLoadBalancerType":"nlb"}
{"level":"info","ts":1591461951.3949816,"logger":"controller_service","msg":"AWS load balancer type set","Namespace":"aws-nlb-helper","Service":"test-api","awsLoadBalancerDNS":"ac9e5c9af3c404884a410ab59ba57490-f738846f56f647b9.elb.us-east-1.amazonaws.com"}
```

All the annotations are read, the ones not defined are defaulted to the values defined in the table above.

```log
{"level":"info","ts":1591461951.394987,"logger":"controller_service","msg":"Unable to parse Deregistration Delay value, defaulting.","Namespace":"aws-nlb-helper","Service":"test-api","awsLoadBalancerSettingsDeregistrationDelay":300}
```

Now the controller looks for the load balancer, first using the tag `kubernetes.io/service-name` set by the AWS kubernetes controller.

```log
{"level":"info","ts":1591461951.3951344,"logger":"helper_aws","msg":"Looking for tagged resources","LoadBalancerDNS":"ac9e5c9af3c404884a410ab59ba57490-f738846f56f647b9.elb.us-east-1.amazonaws.com","ServiceName":"aws-nlb-helper/test-api","Tags":{"kubernetes.io/service-name":"aws-nlb-helper/test-api"}}
```

If the previous filter returns at least one load balancer, will look for a load balancer matching the `externalName`.

```log
{"level":"info","ts":1591461951.7107737,"logger":"helper_aws","msg":"Load balancer matching tags and DNS found","LoadBalancerDNS":"ac9e5c9af3c404884a410ab59ba57490-f738846f56f647b9.elb.us-east-1.amazonaws.com","ServiceName":"aws-nlb-helper/test-api","LoadBalancerARN":"arn:aws:elasticloadbalancing:us-east-1:170744112331:loadbalancer/net/ac9e5c9af3c404884a410ab59ba57490/f738846f56f647b9","LoadBalancerDNS":"ac9e5c9af3c404884a410ab59ba57490-f738846f56f647b9.elb.us-east-1.amazonaws.com"}
```

If there is a match, it found a load balancer with the `kubernetes.io/service-name` and `externalName`.

The first update will change the load balancer resource, enabling or disabling the termination protection.

```log
{"level":"info","ts":1591461951.7357097,"logger":"helper_aws","msg":"Load balancer updated","ModifyLoadBalancerAttributesOutput":{"Attributes":[{"Key":"deletion_protection.enabled","Value":"false"},{"Key":"access_logs.s3.enabled","Value":"false"},{"Key":"load_balancing.cross_zone.enabled","Value":"false"},{"Key":"access_logs.s3.prefix","Value":""},{"Key":"access_logs.s3.bucket","Value":""}]}}
```

After the load balancer attribute has been updated, will look for the Target Groups attached to the listeners and then update the stickness, proxy-protocol and deregistration delay attributes.

```log
{"level":"info","ts":1591461951.7687242,"logger":"helper_aws","msg":"Updating target group","targetGroupARN":"arn:aws:elasticloadbalancing:us-east-1:170744112331:targetgroup/k8s-awsnlbhe-testapi-b3a8421265/0522c9888807019c"}
{"level":"info","ts":1591461951.7973483,"logger":"helper_aws","msg":"Target groups updated","TargetGroupARN":"arn:aws:elasticloadbalancing:us-east-1:170744112331:targetgroup/k8s-awsnlbhe-testapi-b3a8421265/0522c9888807019c","ModifyLoadBalancerAttributesOutput":{"Attributes":[{"Key":"proxy_protocol_v2.enabled","Value":"true"},{"Key":"stickiness.enabled","Value":"false"},{"Key":"deregistration_delay.timeout_seconds","Value":"300"},{"Key":"stickiness.type","Value":"source_ip"}]}}
{"level":"info","ts":1591461951.7974193,"logger":"controller_service","msg":"Load balancer updated","Namespace":"aws-nlb-helper","Service":"test-api","awsLoadBalancerIngressHostname":"ac9e5c9af3c404884a410ab59ba57490-f738846f56f647b9.elb.us-east-1.amazonaws.com"}
```

If a service is added or updated, will run again. A part for those kind of events, the reconciliation loop will run again after the `reconcileInterval`, this way any manual change will be reverted to the state defined in the service.

## Contributing

You can contribute by:

* Raising any issues you find using the operator
* Fixing issues by opening [Pull Requests](https://github.com/3scale/aws-nlb-helper-operator/pulls)
* Submitting a patch or opening a PR
* Improving documentation
* Talking about the operator

All bugs, tasks or enhancements are tracked as [GitHub issues](https://github.com/3scale/aws-nlb-helper-operator/issues).
