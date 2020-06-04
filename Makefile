CURRENT_GIT_TAG := $(shell git describe --tags --abbrev=0)
RELEASE := $(CURRENT_GIT_TAG)
DOCKER_IMAGE := quay.io/3scale/aws-nlb-heper-operator
KUBECTL ?= kubectl
NAMESPACE := aws-nlb-helper

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

run-local: ## Run operator locally
	operator-sdk run --local --watch-namespace=$(NAMESPACE) --enable-delve

operator-image-build: ## Build operator Docker image
	operator-sdk build $(DOCKER_IMAGE):$(RELEASE)

operator-image-push: ## Push operator Docker image to remote registry
	docker push $(DOCKER_IMAGE):$(RELEASE)

operator-image-update: operator-image-build operator-image-push ## Build and Push Operator Docker image to remote registry

operator-create: ## Create/Update Operator objects
	$(KUBECTL) create namespace $(NAMESPACE) || true
	$(KUBECTL) apply -n $(NAMESPACE) -f deploy/service_account.yaml
	$(KUBECTL) apply -n $(NAMESPACE) -f deploy/role.yaml
	$(KUBECTL) apply -n $(NAMESPACE) -f deploy/role_binding.yaml
	sed -i '' 's@REPLACE_IMAGE@$(DOCKER_IMAGE):$(RELEASE)@g' deploy/operator.yaml
	$(KUBECTL) apply -n $(NAMESPACE) -f deploy/operator.yaml
	sed -i '' 's@$(DOCKER_IMAGE):$(RELEASE)@REPLACE_IMAGE@g' deploy/operator.yaml

operator-delete: ## Delete Operator objects
	$(KUBECTL) delete -n $(NAMESPACE) -f deploy/operator.yaml || true
	$(KUBECTL) delete -n $(NAMESPACE) -f deploy/role_binding.yaml || true
	$(KUBECTL) delete -n $(NAMESPACE) -f deploy/role.yaml || true
	$(KUBECTL) delete -n $(NAMESPACE) -f deploy/service_account.yaml || true
	$(KUBECTL) delete ns $(NAMESPACE) || true