// Package service implements the operations in interface helloworldpb.GreeterServer
package service

import (
	helloworldpb "github.com/loonghe/grpc_greeter_helloworld/grpc/greeter/helloworld"
	"github.com/loonghe/grpc_greeter_helloworld/internal/logic"
)

// serviceImpl implements helloworldpb.GreeterServer interface
type serviceImpl struct {
	userUseCase logic.UserUseCase
}

// NewService creates a new service.
func NewService(userUseCase logic.UserUseCase) helloworldpb.GreeterServer {
	return &serviceImpl{userUseCase: userUseCase}
}
