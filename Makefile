test: deps
	go test ./pkg/...

deps: vendor
	go get github.com/hashicorp/consul/api

vendor: Gopkg.toml Gopkg.lock
	dep ensure
