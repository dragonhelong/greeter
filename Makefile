# 从git克隆下来的原始代码，使用原生protoc命令行编译脚本，使用buf编译后就不需要make api命令编译，直接根目录下执行buf generate进行编译
.PHONY: api
# generate api proto
api:
	protoc -I ./proto \
   --go_out ./proto --go_opt paths=source_relative \
   --go-grpc_out ./proto --go-grpc_opt paths=source_relative \
   --grpc-gateway_out ./proto --grpc-gateway_opt paths=source_relative \
   ./proto/helloworld/hello_world.proto