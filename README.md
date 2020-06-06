[![Go Report Card](https://goreportcard.com/badge/github.com/3scale/aws-nlb-helper-operator)](https://goreportcard.com/report/github.com/3scale/aws-nlb-helper-operator)
[![build status](https://circleci.com/gh/3scale/aws-nlb-helper-operator.svg?style=shield)](https://codecov.io/gh/3scale/aws-nlb-helper-operator/.circleci/config.yml)
[![release](https://badgen.net/github/release/3scale/aws-nlb-helper-operator)](https://github.com/3scale/aws-nlb-helper-operator/releases)
[![license](https://badgen.net/github/license/3scale/aws-nlb-helper-operator)](https://github.com/3scale/aws-nlb-helper-operator/blob/master/LICENSE)

# AWS NLB Helper operator

This operator allows to manage some settings for AWS Network Load Balanacer using
Kubernetes annotations in the service objects.

**Disclaimer**: This operator is in the early development stages, at v0.0.2 works
but lacks testing and some code improvements.

## Motivations

The current ingress controller for AWS Network Load Balanacers doesn't support
setting some attributes like enabling the termination protection, the proxy protocol
or the stickness. This operator adds support to change those settings by providing
some extra annotations to the kubernetes service objects.

## Annotations

| Setting                              | Annotations                                                    | Values      | Default |
| ------------------------------------ | -------------------------------------------------------------- | ----------- | ------- |
| Load Balancer Termination Protection | aws-nlb-helper.3scale.net/loadbalanacer-termination-protection | true, false | false   |
| Target Group Proxy Protocol          | aws-nlb-helper.3scale.net/enable-targetgroups-proxy-protocol   | true, false | false   |
| Target Group Stickness               | aws-nlb-helper.3scale.net/enable-targetgroups-stickness        | true, false | false   |
| Target Group Deregistration Delay    | aws-nlb-helper.3scale.net/targetgroups-deregisration-delay     | 0-3600      | 300     |

## Requirements

### Secret with IAM credentials

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

## Deployment

```bash
❯ make operator-deploy
kubectl create namespace aws-nlb-helper || true
namespace/aws-nlb-helper created
kubectl apply -n aws-nlb-helper -f deploy/aws_iam.yaml
secret/aws-nlb-helper-iam created
kubectl apply -n aws-nlb-helper -f deploy/service_account.yaml
serviceaccount/aws-nlb-helper-operator created
kubectl apply -n aws-nlb-helper -f deploy/role.yaml
role.rbac.authorization.k8s.io/aws-nlb-helper-operator created
kubectl apply -n aws-nlb-helper -f deploy/role_binding.yaml
rolebinding.rbac.authorization.k8s.io/aws-nlb-helper-operator created
sed -i "" 's@REPLACE_IMAGE@quay.io/3scale/aws-nlb-helper-operator:v0.0.2@g' deploy/operator.yaml
kubectl apply -n aws-nlb-helper -f deploy/operator.yaml
deployment.apps/aws-nlb-helper-operator created
sed -i "" 's@quay.io/3scale/aws-nlb-helper-operator:v0.0.2@REPLACE_IMAGE@g' deploy/operator.yaml
```

```bash
❯ kubectl logs -n aws-nlb-helper -f `k get -n aws-nlb-helper pods -l name=aws-nlb-helper-operator -o name`
{"level":"info","ts":1591382482.7026253,"logger":"cmd","msg":"Operator Version: 0.0.1"}
{"level":"info","ts":1591382482.7030494,"logger":"cmd","msg":"Go Version: go1.13.7"}
{"level":"info","ts":1591382482.7030537,"logger":"cmd","msg":"Go OS/Arch: linux/amd64"}
{"level":"info","ts":1591382482.7030568,"logger":"cmd","msg":"Version of operator-sdk: v0.17.1"}
{"level":"info","ts":1591382482.7043262,"logger":"leader","msg":"Trying to become the leader."}
{"level":"info","ts":1591382485.50146,"logger":"leader","msg":"No pre-existing lock was found."}
{"level":"info","ts":1591382485.5110912,"logger":"leader","msg":"Became the leader."}
{"level":"info","ts":1591382488.1714578,"logger":"controller-runtime.metrics","msg":"metrics server is starting to listen","addr":"0.0.0.0:8383"}
{"level":"info","ts":1591382488.1717732,"logger":"cmd","msg":"Registering Components."}
{"level":"info","ts":1591382488.1717985,"logger":"controller_service","msg":"Looking for Services with an aws-nlb-helper annotation"}
{"level":"info","ts":1591382490.8642957,"logger":"metrics","msg":"Metrics Service object created","Service.Name":"aws-nlb-helper-operator-metrics","Service.Namespace":"aws-nlb-helper"}
{"level":"info","ts":1591382493.6093283,"logger":"cmd","msg":"Starting the Cmd."}
{"level":"info","ts":1591382493.6095302,"logger":"controller-runtime.manager","msg":"starting metrics server","path":"/metrics"}
{"level":"info","ts":1591382493.6095517,"logger":"controller-runtime.controller","msg":"Starting EventSource","controller":"service-controller","source":"kind source: /, Kind="}
{"level":"info","ts":1591382493.7098615,"logger":"controller-runtime.controller","msg":"Starting Controller","controller":"service-controller"}
{"level":"info","ts":1591382493.7098913,"logger":"controller-runtime.controller","msg":"Starting workers","controller":"service-controller","worker count":1}
```

### Example service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: test-api
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    aws-nlb-helper.3scale.net/enable-targetgroups-proxy-protocol: "true"
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

* Raising any issues you find using operator
* Fixing issues by opening [Pull Requests](https://github.com/3scale/aws-nlb-helper-operator/pulls)
* Submitting a patch or opening a PR
* Improving documentation
* Talking about operator

All bugs, tasks or enhancements are tracked as [GitHub issues](https://github.com/3scale/aws-nlb-helper-operator/issues).
