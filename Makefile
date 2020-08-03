### Makefile ---

## Author: Shiwen Cheng
## Copyright: 2020, <chengshiwen0103@gmail.com>

export GO_BUILD=env GO111MODULE=on go build -o bin/influx-tool -ldflags "-X main.GitCommit=$(shell git rev-parse --short HEAD) -X 'main.BuildTime=$(shell date '+%Y-%m-%d %H:%M:%S')'"

all: build

build:
	mkdir -p bin
	$(GO_BUILD)

linux:
	$(GO_BUILD)

fmt:
	find . -name "*.go" -exec go fmt {} \;

clean:
	rm -rf bin


### Makefile ends here
