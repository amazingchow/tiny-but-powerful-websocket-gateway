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
	proto_gens_path=/go/src/${code_root}/internal/proto_gens
	cd /go/src
	proto_path=${code_root}/protos
	for i in $(ls /go/src/${proto_path}/*.proto); do
		logger_print "[INFO]" "to compile ${i}..."
		fn=${proto_path}/$(basename "${i}")
		${PROTOC_INSTALL}/bin/protoc -I${PROTOC_INSTALL}/include -I. \
			--go_out=plugins=grpc:. "${fn}"
		logger_print "[INFO]" "compiled ${i}."
	done
}

run $@
