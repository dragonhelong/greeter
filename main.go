package main

import (
	"context"
	"flag"
	"log"
	"net"

	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	helloworldpb "github.com/loonghe/grpc_greeter_helloworld/grpc/greeter/helloworld"
	"github.com/loonghe/grpc_greeter_helloworld/internal/logic"
	"github.com/loonghe/grpc_greeter_helloworld/internal/repo/db"
	"github.com/loonghe/grpc_greeter_helloworld/internal/service"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/config"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/gateway"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/middleware/auth"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/middleware/validator"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/zaplog"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
)

func main() {
	// 实现加载静态配置文件并初始化日志
	configFilePath := flag.String("c", "./", "config file path")
	flag.Parse()
	if err := config.Load(*configFilePath); err != nil {
		panic(err)
	}
	zaplog.Init(config.Viper.GetString("zaplog.path"))
	defer zaplog.Sync()
	zaplog.Sugar.Info("server is running")
	// 初始化trace
	traceCfg := &jaegerconfig.Configuration{
		ServiceName: "MyService",
		Sampler: &jaegerconfig.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegerconfig.ReporterConfig{
			LocalAgentHostPort: "127.0.0.1:6831",
			LogSpans:           true,
		},
	}
	tracer, closer, err := traceCfg.NewTracer(jaegerconfig.Logger(jaeger.StdLogger))
	if err != nil {
		panic(err)
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	// 初始化上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Create a listener on TCP port
	lis, err := net.Listen("tcp", ":8091")
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	// 创建一个gRPC server对象，并且在grpc拦截器中加入各种拦截插件, 利用拦截器特性将opentracing的设置到grpc和grpc-gateway中
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpc_opentracing.UnaryServerInterceptor(
				grpc_opentracing.WithTracer(opentracing.GlobalTracer()),
			),
			auth.ServerTokenValidationUnaryInterceptor,
			validator.ServerValidationUnaryInterceptor,
		),
	)
	dbRegistry := db.NewMysql()
	userUseCase := logic.NewUserUseCase(dbRegistry)
	// 注册Greeter service到server
	helloworldpb.RegisterGreeterServer(s, service.NewService(userUseCase))

	log.Println("Serving on http://127.0.0.1:8091")
	// 使用gateway把grpcServer转成httpServer
	gwServer := gateway.ProvideHTTP(ctx, config.Viper.GetString("server.addr"), s)
	log.Fatalln(gwServer.Serve(lis)) // 启动HTTP服务
}
