### Makefile ---

## Author: Shiwen Cheng
## Copyright: 2020, <chengshiwen0103@gmail.com>

export GO_BUILD=GO111MODULE=on go build -o bin/influx-tool -ldflags "-s -X main.GitCommit=$(shell git rev-parse --short HEAD) -X 'main.BuildTime=$(shell date '+%Y-%m-%d %H:%M:%S')'"

all: build

build:
	mkdir -p bin
	$(GO_BUILD)

linux:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO_BUILD)

fmt:
	go fmt ./...

clean:
	rm -rf bin


### Makefile ends here
