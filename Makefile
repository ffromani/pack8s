all: build

build: generate
	./hack/build/build.sh

deps:
	dep ensure

fmt:
	go fmt ./cmd/...
	go fmt ./iopodman/...

generate: deps
	go generate ./...

clean:
	rm -f pack8s
	rm -f varlink-go-interface-generator
	rm -rf _out

varlink-go-interface-generator:
	cd vendor/github.com/fromanirh/varlink-go/cmd/varlink-go-interface-generator && go build -v .
	cp vendor/github.com/fromanirh/varlink-go/cmd/varlink-go-interface-generator/varlink-go-interface-generator .

prep: generate fmt

release: build
	mkdir -p _out
	cp pack8s _out/pack8s-${VERSION}-linux-amd64
	hack/container/docker-push.sh ${VERSION}

.PHONY: all build deps fmt clean prep
