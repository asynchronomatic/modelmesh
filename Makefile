

build:
	go build -tags assert -o build/mesh modelmesh/cmd/mesh
.PHONY: build

build-all:
	GOARCH=amd64 GOOS=windows go build -o build/mesh.win modelmesh/cmd/mesh
	GOARCH=amd64 GOOS=linux   go build -o build/mesh.linux.amd64 modelmesh/cmd/mesh
	GOARCH=arm64 GOOS=linux   go build -o build/mesh.linux.arm64 modelmesh/cmd/mesh
	GOARCH=arm64 GOOS=darwin  go build -o build/mesh.darwin.arm64 modelmesh/cmd/mesh
.PHONY: build-all

run-admin:
	go run modelmesh/cmd/mesh admin
.PHONY: run-admin

run-proxy:
	go run -tags assert modelmesh/cmd/mesh proxy
.PHONY: run-proxy

run-hybrid:
	go run modelmesh/cmd/mesh hybrid
.PHONY: run-proxy


test:
	go test -v modelmesh/...
.PHONY: test

-include Makefile.local

