package validator

import (
	"context"

	"github.com/loonghe/grpc_greeter_helloworld/pkg/zaplog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type validator interface {
	ValidateAll() error
}

// ServerValidationUnaryInterceptor 参数校验拦截插件
func ServerValidationUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	zaplog.Sugar.Infof("%+v", req)
	if r, ok := req.(validator); ok {
		if err := r.ValidateAll(); err != nil {
			zaplog.Sugar.Errorf("req validate error %v", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	return handler(ctx, req)
}
