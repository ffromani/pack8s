all: build

build:
	go build -v .

deps:
	dep ensure

fmt:
	go fmt ./cmd/...
	go fmt ./iopodman/...

generate:
	dep ensure
	go generate ./...

clean:
	rm -f pack8s

prep: deps generate fmt
