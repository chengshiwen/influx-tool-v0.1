### Makefile ---

## Author: Shiwen Cheng
## Copyright: 2020, <chengshiwen0103@gmail.com>

export GO_BUILD=GO111MODULE=on CGO_ENABLED=0 go build -o bin/influx-tool -ldflags "-s -w -X main.GitCommit=$(shell git rev-parse --short HEAD) -X 'main.BuildTime=$(shell date '+%Y-%m-%d %H:%M:%S')'"

.PHONY: build linux lint down tidy clean

all: build

build:
	$(GO_BUILD)

linux:
	GOOS=linux GOARCH=amd64 $(GO_BUILD)

lint:
	golangci-lint run --enable=golint --disable=errcheck --disable=typecheck && goimports -l -w . && go fmt ./... && go vet ./...

down:
	go list ./... && go mod verify

tidy:
	head -n 3 go.mod > go.mod.tmp && mv go.mod.tmp go.mod && rm -f go.sum && go mod tidy -v

clean:
	rm -rf bin

### Makefile ends here
