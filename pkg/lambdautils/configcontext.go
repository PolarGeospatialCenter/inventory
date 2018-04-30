package lambdautils

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
)

type contextKey struct{}

var awsConfigContextKey = &contextKey{}

func NewAwsConfigContext(parentCtx context.Context, configs ...*aws.Config) context.Context {
	return context.WithValue(parentCtx, awsConfigContextKey, configs)
}

func AwsConfigFromContext(ctx context.Context) ([]*aws.Config, bool) {
	configs, ok := ctx.Value(awsConfigContextKey).([]*aws.Config)
	return configs, ok
}

func AwsContextConfigProvider(ctx context.Context) client.ConfigProvider {
	configs, ok := AwsConfigFromContext(ctx)
	if !ok {
		configs = []*aws.Config{}
	}
	return session.New(configs...)
}
