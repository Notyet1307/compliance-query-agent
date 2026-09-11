.PHONY: verify test build demo smoke guest-binary
export GOTOOLCHAIN := local
export GOPROXY := off
export GOSUMDB := off
GOARCH ?= amd64
verify:
	./scripts/verify.sh
test:
	go test -count=1 ./...
build:
	mkdir -p bin
	go build -trimpath -o bin/compliance-agent ./cmd/compliance-agent
demo: build
	./bin/compliance-agent query --config configs/demo.json --input examples/mlps.json
smoke: build
	python3 scripts/smoke.py --binary bin/compliance-agent
guest-binary:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -trimpath -o bin/compliance-agent-guest ./cmd/compliance-agent
