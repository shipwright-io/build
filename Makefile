GOCACHE ?= "$(shell echo ${PWD})/out/gocache"

.PHONY: build
## Build: compile the operator for Linux/AMD64.
build: out/operator

out/operator:
	$(Q)GOARCH=amd64 GOOS=linux go build -o out/operator cmd/manager/main.go


.PHONY: clean
clean:
	rm -rf out/operator

vendor: go.mod go.sum
	$(Q)GOCACHE=$(GOCACHE) go mod vendor ${V_FLAG}