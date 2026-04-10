default: build

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./internal/...

generate:
	cd tools && go generate ./...

.PHONY: build install lint test testacc generate
