all: build

build:
	cd cmd && go build -v .

deps:
	dep ensure

fmt:
	go fmt ./cmd/...
	go fmt ./iopodman/...

generate:
	dep ensure
	go generate ./...

prep: deps generate fmt
