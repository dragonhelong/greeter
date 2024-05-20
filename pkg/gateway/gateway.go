package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime" // 注意v2版本
	helloworldpb "github.com/loonghe/grpc_greeter_helloworld/grpc/greeter/helloworld"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/swagger"
	"github.com/loonghe/grpc_greeter_helloworld/pkg/zaplog"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var grpcGatewayTag = opentracing.Tag{Key: string(ext.Component), Value: "grpc-gateway"}

// ProvideHTTP 把gRPC服务转成HTTP服务，让gRPC同时支持HTTP
func ProvideHTTP(ctx context.Context, endpoint string, grpcServer *grpc.Server) *http.Server {
	// gRPC-Gateway mux
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
	err := helloworldpb.RegisterGreeterHandlerFromEndpoint(ctx, gwmux, endpoint, dops)
	if err != nil {
		log.Fatalln("Failed to register gwmux:", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", gwmux)

	// 注册swagger
	mux.HandleFunc("/swagger/", swagger.ServeSwaggerFile)
	swagger.ServeSwaggerUI(mux)

	// 定义HTTP server配置
	return &http.Server{
		Addr:    endpoint,
		Handler: grpcHandlerFunc(grpcServer, mux), // 请求的统一入口
	}
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
