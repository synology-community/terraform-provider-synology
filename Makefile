
build:
	go build -o terraform-provider-synology

generate:
	go generate ./...

test-client:
	go test -v ./synology-go/...

test: test-client

lint-client:
	go vet ./synology-go/...

lint-provider:
	go vet ./internal/provider/...

lint: lint-client lint-provider
