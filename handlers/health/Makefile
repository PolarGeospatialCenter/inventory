build: main.go
	GOOS=linux go build -o health

package: build
	zip health.zip health
