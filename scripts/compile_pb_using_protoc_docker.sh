#!/usr/bin/env bash

# 遇到执行出错，直接终止脚本的执行
set -o errexit

function logger_print
{
    local prefix="[$(date +%Y/%m/%d\ %H:%M:%S)]"
    echo "${prefix}$@" >&2
}

function run
{
	code_root=github.com/amazingchow/tiny-but-powerful-websocket-gateway

	docker run -it --rm --privileged \
		-v ~/.${code_root}:/go/src/${code_root} \
		-u `id -u` \
		-e PROTOC_INSTALL=/go \
		-w /go/src/${code_root} \
		proto-tools:libprotoc-3.24.3_golang-1.21 bash /go/src/${code_root}/scripts/compile_pb.sh
}

run $@
