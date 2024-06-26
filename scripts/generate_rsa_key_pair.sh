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
    openssl genrsa -out ./fixtures/private_key.pem 2048
    logger_print "[INFO]" "generated RSA private key."
    openssl rsa -inform PEM -outform PEM -in ./fixtures/private_key.pem -pubout -out ./fixtures/public_key.pem
    logger_print "[INFO]" "generated RSA public key."
}

run $@
