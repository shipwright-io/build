GOCACHE ?= "$(shell echo ${PWD})/out/gocache"

.PHONY: build
## Build: compile the operator for Linux/AMD64.
build: out/operator

out/operator:
	$(Q)GOARCH=amd64 GOOS=linux go build -o out/operator cmd/manager/main.go


.PHONY: local
local:

	- kubectl delete -f deploy/role.yaml
	- kubectl delete -f deploy/service_account.yaml
	- kubectl delete -f deploy/role_binding.yaml
	- kubectl delete -f deploy/operator.yaml
	- kubectl delete -f deploy/crds/build.dev_buildstrategies_crd.yaml
	- kubectl delete -f deploy/crds/build.dev_builds_crd.yaml
    
	kubectl apply -f deploy/role.yaml
	kubectl apply -f deploy/service_account.yaml
	kubectl apply -f deploy/role_binding.yaml
	kubectl apply -f deploy/operator.yaml
	kubectl apply -f deploy/crds/build.dev_buildstrategies_crd.yaml
	kubectl apply -f deploy/crds/build.dev_builds_crd.yaml

	operator-sdk run --local


.PHONY: clean
clean:
	rm -rf out/operator

vendor: go.mod go.sum
	$(Q)GOCACHE=$(GOCACHE) go mod vendor ${V_FLAG}