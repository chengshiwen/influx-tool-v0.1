### Makefile ---

## Author: Shiwen Cheng
## Copyright: 2020, <chengshiwen0103@gmail.com>

all: build

build:
	mkdir -p bin
	go build -o bin/influx-tool -ldflags "-X main.GitCommit=$(shell git rev-parse HEAD | cut -c 1-7) -X 'main.BuildTime=$(shell date '+%Y-%m-%d %H:%M:%S')'" github.com/chengshiwen/influx-tool

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/influx-tool -ldflags "-s -X main.GitCommit=$(shell git rev-parse HEAD | cut -c 1-7) -X 'main.BuildTime=$(shell date '+%Y-%m-%d %H:%M:%S')'" github.com/chengshiwen/influx-tool

fmt:
	find . -name "*.go" -exec go fmt {} \;

clean:
	rm -rf bin


### Makefile ends here
