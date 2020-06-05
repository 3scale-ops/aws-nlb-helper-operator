CURRENT_GIT_TAG ?= $(shell git describe --tags --abbrev=0)
RELEASE ?= $(CURRENT_GIT_TAG)
BUILD_NAME ?= aws-nlb-helper-operator
BUILD_PATH ?= build/_output/bin
DOCKER_IMAGE ?= quay.io/3scale/aws-nlb-helper-operator
KUBECTL ?= kubectl
NAMESPACE ?= aws-nlb-helper

.PHONY: build

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S), Linux)
  INPLACE_SED := sed -i
else ifeq ($(UNAME_S), Darwin)
  INPLACE_SED := sed -i ""
endif

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

run-local: ## Run operator locally
	operator-sdk run --local --watch-namespace=$(NAMESPACE) --enable-delve

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o $(BUILD_PATH)/$(BUILD_NAME) cmd/manager/main.go

operator-image-build: build ## Build operator Docker image
	docker build . -f build/Dockerfile -t $(DOCKER_IMAGE):$(RELEASE)

operator-image-push: ## Push operator Docker image to remote registry
	docker push $(DOCKER_IMAGE):$(RELEASE)

operator-image-update: operator-image-build operator-image-push ## Build and Push Operator Docker image to remote registry

operator-create: ## Create/Update Operator objects
	$(KUBECTL) create namespace $(NAMESPACE) || true
	$(KUBECTL) apply -n $(NAMESPACE) -f deploy/service_account.yaml
	$(KUBECTL) apply -n $(NAMESPACE) -f deploy/role.yaml
	$(KUBECTL) apply -n $(NAMESPACE) -f deploy/role_binding.yaml
	$(INPLACE_SED) 's@REPLACE_IMAGE@$(DOCKER_IMAGE):$(RELEASE)@g' deploy/operator.yaml
	$(KUBECTL) apply -n $(NAMESPACE) -f deploy/operator.yaml
	$(INPLACE_SED) 's@$(DOCKER_IMAGE):$(RELEASE)@REPLACE_IMAGE@g' deploy/operator.yaml

operator-delete: ## Delete Operator objects
	$(KUBECTL) delete -n $(NAMESPACE) -f deploy/operator.yaml || true
	$(KUBECTL) delete -n $(NAMESPACE) -f deploy/role_binding.yaml || true
	$(KUBECTL) delete -n $(NAMESPACE) -f deploy/role.yaml || true
	$(KUBECTL) delete -n $(NAMESPACE) -f deploy/service_account.yaml || true
	$(KUBECTL) delete ns $(NAMESPACE) || true