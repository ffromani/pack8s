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
	rm -f varlink-go-interface-generator

varlink-go-interface-generator:
	cd vendor/github.com/fromanirh/varlink-go/cmd/varlink-go-interface-generator && go build -v .
	cp vendor/github.com/fromanirh/varlink-go/cmd/varlink-go-interface-generator/varlink-go-interface-generator .

prep: deps generate fmt
