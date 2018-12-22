.PHONY: test deps

test: deps
	go test -cover ./pkg/...
	go test -cover ./handlers/...

deps:
	dep ensure -vendor-only
	go get -u github.com/aws/aws-sdk-go/aws
	go get -u github.com/aws/aws-sdk-go/service
	go get -u gopkg.in/src-d/go-git.v4

build: deps
	@for dir in `ls handlers`; do \
		GOOS=linux go build -o bin/$$dir ./handlers/$$dir; \
	done
