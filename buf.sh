#！/bin/bash

# 在proto目录下已经通过过 buf mod init 初始化完成buf.yaml相关依赖前提下，可以通过本脚本进行一键编译

# 需要更新或安装buf相关组件，使用下面命令
cd ./proto
buf mod update

# 清理上一次编译文件
cd ../
rm -rf grpc/*

# 进行本次编译
buf generate