package auth

import (
	"context"

	"github.com/loonghe/grpc_greeter_helloworld/pkg/zaplog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// token校验拦截插件, 即自定义校验方法
func ServerTokenValidationUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	zaplog.Sugar.Infof("%+v", req)
	// 拦截普通方法请求，验证Token
	if err := check(ctx); err != nil {
		return nil, err
	}
	// 继续处理请求
	return handler(ctx, req)
}

// check 验证token
func check(ctx context.Context) error {
	// 从上下文中获取元数据
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "获取Token失败")
	}
	var (
		appID     string
		appSecret string
	)
	if value, ok := md["app-id"]; ok {
		appID = value[0]
	}
	if value, ok := md["app-secret"]; ok {
		appSecret = value[0]
	}
	if appID != "grpc_token" || appSecret != "123456" {
		return status.Errorf(codes.Unauthenticated, "Token无效: app-id=%s, app-secret=%s", appID, appSecret)
	}
	return nil
}
