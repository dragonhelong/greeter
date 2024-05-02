package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime" // 注意v2版本
	helloworldpb "github.com/loonghe/grpc_greeter_helloworld/grpc/greeter/helloworld"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/config"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/zaplog"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
	zaplog.WithTrace(ctx).Infof("register name is %d", in.Name)
	return &helloworldpb.HelloReply{Message: in.Name + " world"}, nil
}

// logout接口实现
func (s *server) Logout(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

var grpcGatewayTag = opentracing.Tag{Key: string(ext.Component), Value: "grpc-gateway"}

// 参数校验拦截插件
func ServerValidationUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	zaplog.Sugar.Infof("%+v", req)
	if r, ok := req.(Validator); ok {
		if err := r.ValidateAll(); err != nil {
			zaplog.Sugar.Errorf("req validate error %v", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	return handler(ctx, req)
}

// token校验拦截插件, 即自定义校验方法
func ServerTokenValidationUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	zaplog.Sugar.Infof("%+v", req)
	// 拦截普通方法请求，验证Token
	if err := Check(ctx); err != nil {
		return nil, err
	}
	// 继续处理请求
	return handler(ctx, req)
}

// Check 验证token
func Check(ctx context.Context) error {
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
			ServerTokenValidationUnaryInterceptor,
			ServerValidationUnaryInterceptor,
		),
	)
	// 注册Greeter service到server
	helloworldpb.RegisterGreeterServer(s, &server{})

	// gRPC-Gateway mux
	// 初始化上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 请求时,将http header中某些字段转发到grpc上下文, 例如token验证场景，rpc请求直接将token塞进metadata, 服务端即可从ctx解析，如果是http请求，一般是将token塞进header，此时需要在grpc-gateway将指定header转发到grpc上下文
	inComingOpt := runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
		switch key {
		// 只需要对需要转发的header进行设置，转到context的字段命名不变
		case "App-Id", "App-Secret":
			return key, true
		default:
			return "", false
			// 使用默认处理会在http原始header加grpcgateway-前缀转发到上下文呢
			// return runtime.DefaultHeaderMatcher(s)
		}
	})
	// 响应后,grpc上下文转发到http头部
	outGoingOpt := runtime.WithOutgoingHeaderMatcher(func(key string) (string, bool) {
		return "", false
	})

	gwmux := runtime.NewServeMux(inComingOpt, outGoingOpt)
	dops := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(
			grpc_opentracing.UnaryClientInterceptor(
				grpc_opentracing.WithTracer(opentracing.GlobalTracer()),
			),
		),
	}
	err = helloworldpb.RegisterGreeterHandlerFromEndpoint(ctx, gwmux, "127.0.0.1:8091", dops)
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
			// rpc调用从这里直接到接口实现
			zaplog.Sugar.Info("match grpc call")
			grpcServer.ServeHTTP(w, r)
		} else {
			// http调用从这里转发到上面分支的rpc调用，向http请求响应头写入trace_id
			zaplog.Sugar.Info("match http web call")
			parentSpanContext, err := opentracing.GlobalTracer().Extract(
				opentracing.HTTPHeaders,
				opentracing.HTTPHeadersCarrier(r.Header))
			if err == nil || err == opentracing.ErrSpanContextNotFound {
				serverSpan := opentracing.GlobalTracer().StartSpan(
					"ServeHTTP",
					ext.RPCServerOption(parentSpanContext),
					grpcGatewayTag,
				)
				r = r.WithContext(opentracing.ContextWithSpan(r.Context(), serverSpan))

				trace, ok := serverSpan.Context().(jaeger.SpanContext)
				if ok {
					w.Header().Set(jaeger.TraceContextHeaderName, fmt.Sprint(trace.TraceID()))
				}

				defer serverSpan.Finish()
			}
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}
