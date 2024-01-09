# Current Operator version
VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
VERSION_PKG ?= github.com/grafana/tempo-operator/internal/version
OPERATOR_VERSION ?= 0.7.0
TEMPO_VERSION ?= $(shell cat config/manager/manager.yaml | grep -oP "docker.io/grafana/tempo:\K.*")
TEMPO_QUERY_VERSION ?= $(shell cat config/manager/manager.yaml | grep -oP "docker.io/grafana/tempo-query:\K.*")
COMMIT_SHA = $(shell git rev-parse HEAD)
LD_FLAGS ?= "-X ${VERSION_PKG}.buildDate=${VERSION_DATE} \
			 -X ${VERSION_PKG}.revision=${COMMIT_SHA} \
			 -X ${VERSION_PKG}.operatorVersion=${OPERATOR_VERSION} \
			 -X ${VERSION_PKG}.tempoVersion=${TEMPO_VERSION} \
			 -X ${VERSION_PKG}.tempoQueryVersion=${TEMPO_QUERY_VERSION}"
ARCH ?= $(shell go env GOARCH)

# Image URL to use all building/pushing image targets
IMG_PREFIX ?= ghcr.io/grafana/tempo-operator
IMG_REPO ?= tempo-operator
IMG ?= ${IMG_PREFIX}/${IMG_REPO}:v${OPERATOR_VERSION}
BUNDLE_IMG ?= ${IMG_PREFIX}/${IMG_REPO}-bundle:v${OPERATOR_VERSION}

# When the VERBOSE variable is set to 1, all the commands are shown
ifeq ("$(VERBOSE)","true")
echo_prefix=">>>>"
else
VECHO = @
endif

ECHO ?= @echo $(echo_prefix)


# Default namespace of the operator
OPERATOR_NAMESPACE ?= tempo-operator-system

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(OPERATOR_VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by setup-envtest binary.
ENVTEST_K8S_VERSION = 1.24.2

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen  ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt setup-envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt ## Build manager binary.
	CGO_ENABLED=0 go build -o bin/manager -ldflags ${LD_FLAGS} main.go

.PHONY: run
run: manifests generate fmt ## Run a controller from your host.
	# Disabled webhooks only affects local runs, not the build or in-cluster deployments.
	@echo -e "\033[33mWebhooks are disabled! Use the normal deployment method to enable full operator functionality.\033[0m"
	ENABLE_WEBHOOKS=false go run ./main.go start

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker buildx build --load --platform linux/${ARCH} --build-arg OPERATOR_VERSION -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/overlays/$(BUNDLE_VARIANT) | kubectl apply -f -
	kubectl rollout --namespace $(OPERATOR_NAMESPACE) status deployment/tempo-operator-controller

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/overlays/$(BUNDLE_VARIANT) | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: olm-deploy
olm-deploy: operator-sdk ## Deploy operator via OLM
	$(OPERATOR_SDK) run bundle -n $(OPERATOR_NAMESPACE) $(BUNDLE_IMG)

.PHONY: olm-upgrade
olm-upgrade: operator-sdk ## Upgrade operator via OLM
	$(OPERATOR_SDK) run bundle-upgrade -n $(OPERATOR_NAMESPACE) $(BUNDLE_IMG)

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.5
CONTROLLER_TOOLS_VERSION ?= v0.9.2
GEN_CRD_VERSION ?= v0.0.5
ENVTEST_VERSION ?= latest
OPERATOR_SDK_VERSION ?= 1.27.0
CERTMANAGER_VERSION ?= 1.9.1

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
GEN_CRD = $(LOCALBIN)/gen-crd-api-reference-docs-$(GEN_CRD_VERSION)
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk-$(OPERATOR_SDK_VERSION)
KIND ?= $(LOCALBIN)/kind
KUTTL ?= $(LOCALBIN)/kubectl-kuttl

# Options for KIND version to use
export KUBE_VERSION ?= 1.27
KIND_CONFIG ?= kind-$(KUBE_VERSION).yaml

# Choose wich version to generate
BUNDLE_VARIANT ?= community
BUNDLE_DIR = ./bundle/$(BUNDLE_VARIANT)
MANIFESTS_DIR = config/manifests/$(BUNDLE_VARIANT)
BUNDLE_BUILD_GEN_FLAGS ?= $(BUNDLE_GEN_FLAGS) --output-dir . --kustomize-dir ../../$(MANIFESTS_DIR)

.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	test -s $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION) || $(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: setup-envtest
setup-envtest: ## Download envtest-setup locally if necessary.
	test -s $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION) || $(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: generate-bundle
generate-bundle: operator-sdk manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q --input-dir $(MANIFESTS_DIR) --output-dir $(MANIFESTS_DIR)
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	cd $(BUNDLE_DIR) && cp ../../PROJECT . && $(KUSTOMIZE) build ../../$(MANIFESTS_DIR) | $(OPERATOR_SDK) generate bundle $(BUNDLE_BUILD_GEN_FLAGS) && rm PROJECT
	$(OPERATOR_SDK) bundle validate $(BUNDLE_DIR)
	./hack/ignore-createdAt-bundle.sh

.PHONY: bundle
bundle: 
	BUNDLE_VARIANT=openshift $(MAKE) generate-bundle
	BUNDLE_VARIANT=community $(MAKE) generate-bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f $(BUNDLE_DIR)/bundle.Dockerfile -t $(BUNDLE_IMG) $(BUNDLE_DIR)

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.27.1/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= ${IMG_PREFIX}/${IMG_REPO}-catalog:v$(OPERATOR_VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

.PHONY: fbc-catalog-build
fbc-catalog-build: opm ## Build a File Based Catalog (FBC) catalog image
	mkdir -p catalog/tempo-operator
	$(OPM) generate dockerfile catalog
	$(OPM) render $(CATALOG_BASE_IMG) -o yaml > catalog/tempo-operator/base.yaml
	# $(OPM) init tempo-operator -c alpha -o yaml > catalog/tempo-operator/latest.yaml
	$(OPM) render $(BUNDLE_IMG) -o yaml >> catalog/tempo-operator/latest.yaml
	docker build -f catalog.Dockerfile -t $(CATALOG_IMG) .
	rm -r catalog.Dockerfile catalog

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

# Run CI steps
.PHONY: ci
ci: test ensure-generate-is-noop

# Run go lint against code
.PHONY: lint
lint:
	golangci-lint run


.PHONY: gen-crd-api-reference-docs
gen-crd-api-reference-docs: ## Download gen-crd-api-reference-docs locally if necessary.
	test -s $(GEN_CRD) || $(call go-get-tool,$(GEN_CRD),github.com/ViaQ/gen-crd-api-reference-docs,$(GEN_CRD_VERSION))


.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
		test -s $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION) || $(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4,$(KUSTOMIZE_VERSION))

.PHONY: kind
kind: $(KIND)
$(KIND): $(LOCALBIN)
	./hack/install/install-kind.sh

.PHONY: start-kind
start-kind: kind
	$(ECHO) Starting KIND cluster...
	$(VECHO)$(KIND) create cluster --config $(KIND_CONFIG) 2>&1 | grep -v "already exists" || true

stop-kind:
	$(ECHO)"Stopping the kind cluster"
	$(VECHO)kind delete cluster

.PHONY: deploy-minio
deploy-minio:
	$(ECHO) Installing minio
	$(VECHO) kubectl apply -f minio.yaml

# generic end-to-tests
.PHONY: prepare-e2e
prepare-e2e: kuttl start-kind cert-manager deploy-minio set-test-image-vars build docker-build load-image-operator deploy

.PHONY: e2e
e2e:
	$(KUTTL) test

# OpenShift end-to-tests
.PHONY: prepare-e2e-openshift
prepare-e2e-openshift: deploy-minio

.PHONY: e2e-openshift
e2e-openshift:
	$(KUTTL) test --config kuttl-test-openshift.yaml

# upgrade tests
e2e-upgrade:
	$(KUTTL) test --config kuttl-test-upgrade.yaml

.PHONY: scorecard-tests
scorecard-tests: operator-sdk
	$(OPERATOR_SDK) scorecard -w=5m bundle/$(BUNDLE_VARIANT) || (echo "scorecard test failed" && exit 1)

.PHONY: set-test-image-vars
set-test-image-vars:
	$(eval IMG=local/tempo-operator:e2e)

# Set the controller image parameters
.PHONY: set-image-controller
set-image-controller: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}

.PHONY: load-image-operator
load-image-operator:
	kind load docker-image local/tempo-operator:e2e

.PHONY: operator-sdk
operator-sdk: $(OPERATOR_SDK) ## Download operator-sdk locally if necessary.
$(OPERATOR_SDK): $(LOCALBIN)
	test -s $(OPERATOR_SDK) || curl -sLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION}/operator-sdk_`go env GOOS`_`go env GOARCH`
	@chmod +x $(OPERATOR_SDK)

.PHONY: olm-install
olm-install: operator-sdk ## Install Operator Lifecycle Manager (OLM)
	$(OPERATOR_SDK) olm install

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
go get -d $(2)@$(3) ;\
GOBIN=$(LOCALBIN) go install -mod=mod $(2) ;\
APP=$$(echo "$(LOCALBIN)/$@") ;\
APP_NAME=$$(echo "$$APP-$(3)") ;\
mv "$$APP" "$$APP_NAME" ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: kuttl
kuttl:
	./hack/install/install-kuttl.sh

.PHONY: generate-all
generate-all: generate api-docs bundle ## Update all generated files

.PHONY: ensure-generate-is-noop
ensure-generate-is-noop: generate-all ## Verify that all checked-in, generated code is up-to-date
	@# on make bundle config/manager/kustomization.yaml includes changes, which should be ignored for the below check
	@git restore config/manager/kustomization.yaml
	@git diff -s --exit-code apis/tempo/v1alpha1/zz_generated.*.go || (echo "Build failed: a model has been changed but the generated resources aren't up to date. Run 'make generate' and update your PR." && exit 1)
	@git diff -s --exit-code apis/config/v1alpha1/zz_generated.*.go || (echo "Build failed: a model has been changed but the generated resources aren't up to date. Run 'make generate' and update your PR." && exit 1)
	@git diff -s --exit-code bundle config || (echo "Build failed: the bundle, config files has been changed but the generated bundle, config files aren't up to date. Run 'make bundle' and update your PR." && git diff && exit 1)
	@git diff -s --exit-code bundle/community/bundle.Dockerfile || (echo "Build failed: the community bundle.Dockerfile file has been changed. The file should be the same as generated one. Run 'make bundle' and update your PR." && git diff && exit 1)
	@git diff -s --exit-code bundle/openshift/bundle.Dockerfile || (echo "Build failed: the OpenShift bundle.Dockerfile file has been changed. The file should be the same as generated one. Run 'make bundle' and update your PR." && git diff && exit 1)
	@git diff -s --exit-code docs/operator/api.md || (echo "Build failed: the api.md file has been changed but the generated api.md file isn't up to date. Run 'make api-docs' and update your PR." && git diff && exit 1)
	@git diff -s --exit-code docs/operator/feature-gates.md || (echo "Build failed: the feature-gates.md file has been changed but the generated feature-gates.md file isn't up to date. Run 'make api-docs' and update your PR." && git diff && exit 1)

reset: ## Reset all generated files to repository defaults
	unset IMG_PREFIX && unset OPERATOR_VERSION && $(MAKE) generate-all

.PHONY: cert-manager
cert-manager: cmctl
	# Consider using cmctl to install the cert-manager once install command is not experimental
	kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v${CERTMANAGER_VERSION}/cert-manager.yaml
	$(CMCTL) check api --wait=5m

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
CMCTL = $(shell pwd)/bin/cmctl
.PHONY: cmctl
cmctl:
	@{ \
	set -e ;\
	if (`pwd`/bin/cmctl version | grep ${CERTMANAGER_VERSION}) > /dev/null 2>&1 ; then \
		exit 0; \
	fi ;\
	TMP_DIR=$$(mktemp -d) ;\
	curl -L -o $$TMP_DIR/cmctl.tar.gz https://github.com/jetstack/cert-manager/releases/download/v$(CERTMANAGER_VERSION)/cmctl-`go env GOOS`-`go env GOARCH`.tar.gz ;\
	tar xzf $$TMP_DIR/cmctl.tar.gz -C $$TMP_DIR ;\
	[ -d bin ] || mkdir bin ;\
	mv $$TMP_DIR/cmctl $(CMCTL) ;\
	rm -rf $$TMP_DIR ;\
	}

.PHONY: api-docs
api-docs: docs/operator/api.md docs/operator/feature-gates.md

TYPES_TARGET := $(shell find apis/tempo -type f -iname "*_types.go")
docs/operator/api.md: $(TYPES_TARGET) gen-crd-api-reference-docs
	$(GEN_CRD) -api-dir "github.com/grafana/tempo-operator/apis/tempo/" -config "$(PWD)/config/docs/config.json" -template-dir "$(PWD)/config/docs/templates" -out-file "$(PWD)/$@"


FEATURE_GATES_TARGET := $(shell find apis/config -type f -iname "*_types.go")
docs/operator/feature-gates.md: $(FEATURE_GATES_TARGET) gen-crd-api-reference-docs
	$(GEN_CRD) -api-dir "github.com/grafana/tempo-operator/apis/config/v1alpha1/" -config "$(PWD)/config/docs/config.json" -template-dir "$(PWD)/config/docs/templates" -out-file "$(PWD)/$@"
	sed -i 's/title: "API"/title: "Feature Gates"/' $@
	sed -i 's/+docs:/  docs:/' $@
	sed -i 's/+parent:/    parent:/' $@
	sed -i 's/##/\n##/' $@
	sed -i 's/+newline/\n/' $@

##@ Release
CHLOGGEN_VERSION=v0.11.0
CHLOGGEN ?= $(LOCALBIN)/chloggen-$(CHLOGGEN_VERSION)
FILENAME?=$(shell git branch --show-current)

.PHONY: chloggen
chloggen:
	test -s $(CHLOGGEN) || $(call go-get-tool,$(CHLOGGEN),go.opentelemetry.io/build-tools/chloggen,$(CHLOGGEN_VERSION))

.PHONY: chlog-new
chlog-new: chloggen
	$(CHLOGGEN) new --filename $(FILENAME)

.PHONY: chlog-validate
chlog-validate: chloggen ## Validate changelog
	$(CHLOGGEN) validate

.PHONY: chlog-preview
chlog-preview: chloggen ## Preview changelog
	$(CHLOGGEN) update --dry --version $(OPERATOR_VERSION)
	@./hack/list-components.sh

.PHONY: chlog-update
chlog-update: chloggen ## Update changelog
	awk -i inplace '{print} /next version/{system("echo && ./hack/list-components.sh")}' CHANGELOG.md
	$(CHLOGGEN) update --version $(OPERATOR_VERSION)

.PHONY: release-artifacts
release-artifacts: set-image-controller ## Generate release artifacts
	mkdir -p dist
	$(KUSTOMIZE) build config/overlays/community -o dist/tempo-operator.yaml
	$(KUSTOMIZE) build config/overlays/openshift -o dist/tempo-operator-openshift.yaml
