package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime" // 注意v2版本
	helloworldpb "github.com/loonghe/grpc_greeter_helloworld/grpc/greeter/helloworld"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	helloworldpb.UnimplementedGreeterServer
}

type Validator interface {
	ValidateAll() error
}

func NewServer() *server {
	return &server{}
}

// register接口实现
func (s *server) SayHello(ctx context.Context, in *helloworldpb.HelloRequest) (*helloworldpb.HelloReply, error) {
	return &helloworldpb.HelloReply{Message: in.Name + " world"}, nil
}

// logout接口实现
func (s *server) Logout(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// 参数校验拦截插件
func ServerValidationUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if r, ok := req.(Validator); ok {
		if err := r.ValidateAll(); err != nil {
			return nil, err
		}
	}

	return handler(ctx, req)
}

func main() {
	// Create a listener on TCP port
	lis, err := net.Listen("tcp", ":8091")
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	// 创建一个gRPC server对象，并且在grpc拦截器中加入各种拦截插件
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(ServerValidationUnaryInterceptor))
	// 注册Greeter service到server
	helloworldpb.RegisterGreeterServer(s, &server{})

	// gRPC-Gateway mux
	gwmux := runtime.NewServeMux()
	dops := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = helloworldpb.RegisterGreeterHandlerFromEndpoint(context.Background(), gwmux, "127.0.0.1:8091", dops)
	if err != nil {
		log.Fatalln("Failed to register gwmux:", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", gwmux)

	// 定义HTTP server配置
	gwServer := &http.Server{
		Addr:    "127.0.0.1:8091",
		Handler: grpcHandlerFunc(s, mux), // 请求的统一入口
	}
	log.Println("Serving on http://127.0.0.1:8091")
	log.Fatalln(gwServer.Serve(lis)) // 启动HTTP服务
}

// grpcHandlerFunc 将gRPC请求和HTTP请求分别调用不同的handler处理
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}
