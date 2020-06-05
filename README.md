[![Go Report Card](https://goreportcard.com/badge/github.com/3scale/aws-nlb-helper-operator)](https://goreportcard.com/report/github.com/3scale/aws-nlb-helper-operator)
[![codecov](https://codecov.io/gh/3scale/aws-nlb-helper-operator/branch/master/graph/badge.svg)](https://codecov.io/gh/3scale/aws-nlb-helper-operator)
[![build status](https://circleci.com/gh/3scale/aws-nlb-helper-operator.svg?style=shield)](https://codecov.io/gh/3scale/aws-nlb-helper-operator/.circleci/config.yml)
[![release](https://badgen.net/github/release/3scale/aws-nlb-helper-operator)](https://github.com/3scale/aws-nlb-helper-operator/releases)
[![license](https://badgen.net/github/license/3scale/aws-nlb-helper-operator)](https://github.com/3scale/aws-nlb-helper-operator/blob/master/LICENSE)

# AWS NLB Helper operator

This operator allows managing some settings for AWS Network Load Balanacer using
kubernets annotations on the service objects.

**Disclaimer**: This operator is in the early development stages, at v0.0.2 works
but lacks testing and some code improvements.

## Motivations

The current ingress controller for AWS Network Load Balanacers doesn't support
enabling some settings like enabling the termination protection, the proxy protocol
or the stickness. This operator adds support to change those settings by providing
some extra annotations to the kubernetes service.

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
  name   = "${local.workload}-user-policy"
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

- Example service

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
    deployment: kuardmake
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: http
```