package service

import (
	"context"

	helloworldpb "github.com/loonghe/grpc_greeter_helloworld/grpc/greeter/helloworld"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/zaplog"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SayHello @alias=/api/v1/register
func (s *serviceImpl) SayHello(ctx context.Context, in *helloworldpb.HelloRequest) (*helloworldpb.HelloReply, error) {
	zaplog.WithTrace(ctx).Infof("register name is %d", in.Name)
	return &helloworldpb.HelloReply{Message: in.Name + " world"}, nil
}

// Logout @alias=/api/v1/logout
func (s *serviceImpl) Logout(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// GetUser @alias=/api/v1/GetUser
func (s *serviceImpl) GetUser(ctx context.Context, in *helloworldpb.UserReq) (*helloworldpb.UserRes, error) {
	zaplog.WithTrace(ctx).Infof("register name is %d", in.Id)
	user, err := s.userUseCase.GetUser(ctx, in.Id)
	if err != nil {
		zaplog.Sugar.Errorf("serviceImpl err: %v", err)
	}
	return &helloworldpb.UserRes{
		Id:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Phone: user.Phone,
	}, nil
}
