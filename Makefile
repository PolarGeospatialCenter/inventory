.PHONY: test deps

test: deps
	go test -cover ./pkg/...

deps:
	dep ensure
	go get github.com/hashicorp/consul/api
	go get github.com/aws/aws-sdk-go/aws
	go get github.com/aws/aws-sdk-go/service
