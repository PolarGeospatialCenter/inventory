.PHONY: test deps

export GO111MODULE=on
export GOFLAGS=-mod=readonly
test: 
	go test -cover -v ./pkg/...
	go test -cover -v ./handlers/...

build: 
	@for dir in `ls handlers`; do \
		GOOS=linux go build -o bin/$$dir ./handlers/$$dir; \
	done
