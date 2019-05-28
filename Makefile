.PHONY: test deps

test: deps
	go test -cover -v ./pkg/...
	go test -cover -v ./handlers/...

deps:
	dep ensure -vendor-only
	go get -u github.com/aws/aws-sdk-go/aws
	go get -u github.com/aws/aws-sdk-go/service

build: deps
	@for dir in `ls handlers`; do \
		GOOS=linux go build -o bin/$$dir ./handlers/$$dir; \
	done
