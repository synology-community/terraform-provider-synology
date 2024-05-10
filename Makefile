include .env

build:
	go build -o terraform-provider-synology

generate:
	go generate ./...

test:
	go test -v -cover -timeout=120s -parallel=4 ./...

testacc:
	SYNOLOGY_HOST=$(SYNOLOGY_HOST) SYNOLOGY_USER=$(SYNOLOGY_USER) SYNOLOGY_PASSWORD=$(SYNOLOGY_PASSWORD) TF_ACC=1 go test -v -cover -timeout 120m ./...

test: test testacc

lint-client:
	go vet ./synology/client/...

lint-provider:
	go vet ./synology/provider/...

lint: lint-client lint-provider

run-cmd-run:
	SYNOLOGY_HOST=$(SYNOLOGY_HOST) SYNOLOGY_USER=$(SYNOLOGY_USER) SYNOLOGY_PASSWORD=$(SYNOLOGY_PASSWORD) go run ./cmd/run

.PHONY: build generate test testacc lint-client lint-provider lint