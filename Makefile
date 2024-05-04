
build:
	go build -o terraform-provider-synology

generate:
	go generate ./...

test:
	go test -v -cover -timeout=120s -parallel=4 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

test: test testacc

lint-client:
	go vet ./synology/client/...

lint-provider:
	go vet ./synology/provider/...

lint: lint-client lint-provider

.PHONY: build generate test testacc lint-client lint-provider lint