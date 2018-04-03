test: deps
	go test ./pkg/...

deps: vendor
	go get github.com/hashicorp/consul/api
	go get github.com/aws/aws-sdk-go/aws
	go get github.com/aws/aws-sdk-go/service

vendor: Gopkg.toml Gopkg.lock
	dep ensure
